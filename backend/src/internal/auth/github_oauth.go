package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"src/src/internal/audit" // ← add this import
)

const githubOrg = "mohans-organization"

var teamRoles = map[string]string{
	"platform-admins":   "admin",
	"sre":               "operator",
	"developers":        "developer",
	"platform-readonly": "readonly",
}

var rolePriority = map[string]int{
	"admin":     4,
	"operator":  3,
	"developer": 2,
	"readonly":  1,
}

type githubUser struct {
	Login string `json:"login"`
	ID    int64  `json:"id"`
}

/* ─────────────────────────────────────────
   GET /auth/login
───────────────────────────────────────── */

func HandleLogin(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	redirectURI := os.Getenv("GITHUB_REDIRECT_URI")

	log.Println("[AUTH][LOGIN] Login endpoint hit")
	log.Printf("[AUTH][LOGIN] GITHUB_CLIENT_ID present=%v", clientID != "")
	log.Printf("[AUTH][LOGIN] GITHUB_REDIRECT_URI=%s", redirectURI)

	if clientID == "" {
		log.Println("[AUTH][LOGIN][ERROR] GITHUB_CLIENT_ID is not set — cannot initiate OAuth")
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	if redirectURI == "" {
		log.Println("[AUTH][LOGIN][ERROR] GITHUB_REDIRECT_URI is not set — cannot initiate OAuth")
		http.Error(w, "OAuth not configured", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:org",
		clientID, redirectURI,
	)

	log.Printf("[AUTH][LOGIN] Redirecting user to GitHub OAuth → %s", url)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

/* ─────────────────────────────────────────
   GET /auth/callback
───────────────────────────────────────── */

func HandleCallback(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	log.Println("[AUTH][CALLBACK] ──────────────────────────────────────")
	log.Println("[AUTH][CALLBACK] OAuth callback received")

	// ── Step 0: extract code ──
	code := r.URL.Query().Get("code")
	if code == "" {
		log.Println("[AUTH][CALLBACK][ERROR] No 'code' param in callback — OAuth may have been cancelled or failed")
		redirectError(w, r, "oauth_failed")
		return
	}
	log.Printf("[AUTH][CALLBACK][STEP 1] OAuth code received (length=%d)", len(code))

	// ── Step 1: exchange code → access token ──
	log.Println("[AUTH][CALLBACK][STEP 2] Exchanging code for GitHub access token")
	accessToken, err := exchangeCode(code)
	if err != nil {
		log.Printf("[AUTH][CALLBACK][ERROR] Code exchange failed: %v", err)
		redirectError(w, r, "oauth_failed")
		return
	}
	log.Printf("[AUTH][CALLBACK][STEP 2][SUCCESS] Access token received (length=%d)", len(accessToken))

	// ── Step 2: fetch GitHub user ──
	log.Println("[AUTH][CALLBACK][STEP 3] Fetching GitHub user identity")
	user, err := getGithubUser(accessToken)
	if err != nil {
		log.Printf("[AUTH][CALLBACK][ERROR] GitHub /user call failed: %v", err)
		redirectError(w, r, "user_fetch_failed")
		return
	}
	log.Printf("[AUTH][CALLBACK][STEP 3][SUCCESS] GitHub user → login=%s id=%d", user.Login, user.ID)

	// ── Step 3: org membership gate ──
	log.Printf("[AUTH][CALLBACK][STEP 4] Checking org membership → org=%s login=%s", githubOrg, user.Login)
	isMember, err := checkOrgMembership(accessToken, user.Login)
	if err != nil {
		log.Printf("[AUTH][CALLBACK][ERROR] Org membership API call failed for %s: %v", user.Login, err)
		// ── audit: failed login ──
		audit.Log(r, audit.Entry{
			Action:       "login",
			ResourceType: "auth",
			ResourceName: user.Login,
			Status:       "failed",
			Details:      "Org membership check failed: " + err.Error(),
		})
		redirectError(w, r, "org_check_failed")
		return
	}
	if !isMember {
		log.Printf("[AUTH][CALLBACK][DENY] %s is NOT a member of org=%s — rejecting login", user.Login, githubOrg)
		// ── audit: denied login ──
		audit.Log(r, audit.Entry{
			Action:       "login",
			ResourceType: "auth",
			ResourceName: user.Login,
			Status:       "rejected",
			Details:      fmt.Sprintf("Not a member of org=%s", githubOrg),
		})
		redirectError(w, r, "not_org_member")
		return
	}
	log.Printf("[AUTH][CALLBACK][STEP 4][SUCCESS] %s confirmed as org member", user.Login)

	// ── Step 4: determine role from teams ──
	log.Printf("[AUTH][CALLBACK][STEP 5] Determining role for login=%s across %d teams", user.Login, len(teamRoles))
	role, err := determineRole(accessToken, user.Login)
	if err != nil {
		log.Printf("[AUTH][CALLBACK][ERROR] Role determination failed for %s: %v", user.Login, err)
		redirectError(w, r, "role_check_failed")
		return
	}
	if role == "" {
		log.Printf("[AUTH][CALLBACK][DENY] %s has no matching team in org — rejecting login", user.Login)
		// ── audit: no role assigned ──
		audit.Log(r, audit.Entry{
			Action:       "login",
			ResourceType: "auth",
			ResourceName: user.Login,
			Status:       "rejected",
			Details:      "No team role assigned in org",
		})
		redirectError(w, r, "no_role_assigned")
		return
	}
	log.Printf("[AUTH][CALLBACK][STEP 5][SUCCESS] Role assigned → login=%s role=%s", user.Login, role)

	// ── Step 5: issue JWT ──
	log.Printf("[AUTH][CALLBACK][STEP 6] Generating JWT → login=%s role=%s", user.Login, role)
	jwtToken, err := GenerateJWT(user.Login, user.ID, role)
	if err != nil {
		log.Printf("[AUTH][CALLBACK][ERROR] JWT generation failed for %s: %v", user.Login, err)
		redirectError(w, r, "token_failed")
		return
	}
	log.Println("[AUTH][CALLBACK][STEP 6][SUCCESS] JWT generated")

	// ── audit: successful login ──
	audit.Log(r, audit.Entry{
		Action:       "login",
		ResourceType: "auth",
		ResourceName: user.Login,
		Status:       "success",
		Details:      fmt.Sprintf("GitHub SSO login successful role=%s", role),
	})

	// ── Step 6: redirect to frontend ──
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	dest := fmt.Sprintf(
		"%s/auth/callback?token=%s&role=%s&login=%s",
		frontendURL, jwtToken, role, user.Login,
	)

	elapsed := time.Since(start)
	log.Printf("[AUTH][CALLBACK][SUCCESS] Login complete → login=%s role=%s totalTime=%s",
		user.Login, role, elapsed)
	log.Println("[AUTH][CALLBACK] ──────────────────────────────────────")

	http.Redirect(w, r, dest, http.StatusTemporaryRedirect)
}

/* ─────────────────────────────────────────
   GET /auth/me
───────────────────────────────────────── */

func HandleMe(w http.ResponseWriter, r *http.Request) {
	log.Println("[AUTH][ME] /auth/me called")

	claims := ClaimsFromContext(r.Context())
	if claims == nil {
		log.Println("[AUTH][ME][ERROR] No claims in context — Authenticate middleware may not be applied")
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	log.Printf("[AUTH][ME][SUCCESS] Returning identity → login=%s role=%s", claims.GithubLogin, claims.Role)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"login":     claims.GithubLogin,
		"role":      claims.Role,
		"githubId":  claims.GithubID,
		"expiresAt": claims.ExpiresAt.Time.Unix(),
	})
}

/* ─────────────────────────────────────────
   Private helpers
───────────────────────────────────────── */

func exchangeCode(code string) (string, error) {
	log.Println("[AUTH][EXCHANGE] Posting code to GitHub token endpoint")

	payload := fmt.Sprintf(
		"client_id=%s&client_secret=%s&code=%s",
		os.Getenv("GITHUB_CLIENT_ID"),
		os.Getenv("GITHUB_CLIENT_SECRET"),
		code,
	)

	req, err := http.NewRequest(
		"POST",
		"https://github.com/login/oauth/access_token",
		strings.NewReader(payload),
	)
	if err != nil {
		log.Printf("[AUTH][EXCHANGE][ERROR] Failed to build request: %v", err)
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	log.Printf("[AUTH][EXCHANGE] GitHub responded in %s", time.Since(start))

	if err != nil {
		log.Printf("[AUTH][EXCHANGE][ERROR] HTTP request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("[AUTH][EXCHANGE] Response status: %s", resp.Status)

	var result struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Error != "" {
		log.Printf("[AUTH][EXCHANGE][ERROR] GitHub returned error=%s description=%s",
			result.Error, result.ErrorDesc)
		return "", fmt.Errorf("github oauth: %s — %s", result.Error, result.ErrorDesc)
	}

	log.Println("[AUTH][EXCHANGE][SUCCESS] Access token obtained")
	return result.AccessToken, nil
}

func getGithubUser(token string) (*githubUser, error) {
	log.Println("[AUTH][USER] Calling GitHub /user")

	req, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "platform-backend")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	log.Printf("[AUTH][USER] GitHub /user responded in %s", time.Since(start))

	if err != nil {
		log.Printf("[AUTH][USER][ERROR] HTTP request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	log.Printf("[AUTH][USER] Response status: %s", resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AUTH][USER][ERROR] Non-200 response from GitHub /user: %s", resp.Status)
		return nil, fmt.Errorf("github /user failed: %s", resp.Status)
	}

	var user githubUser
	json.NewDecoder(resp.Body).Decode(&user)

	if user.Login == "" {
		log.Println("[AUTH][USER][ERROR] GitHub /user returned empty login field")
		return nil, fmt.Errorf("empty github login")
	}

	log.Printf("[AUTH][USER][SUCCESS] login=%s id=%d", user.Login, user.ID)
	return &user, nil
}

func checkOrgMembership(token, login string) (bool, error) {
	url := fmt.Sprintf(
		"https://api.github.com/user/memberships/orgs/%s",
		githubOrg,
	)
	log.Printf("[AUTH][ORG] Checking org membership → GET %s login=%s", url, login)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "platform-backend")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[AUTH][ORG][ERROR] HTTP request failed: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	log.Printf("[AUTH][ORG] GitHub responded in %s status=%s", time.Since(start), resp.Status)

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AUTH][ORG][DENY] login=%s not in org=%s (status=%s)", login, githubOrg, resp.Status)
		return false, nil
	}

	var result struct {
		State string `json:"state"`
		Role  string `json:"role"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	log.Printf("[AUTH][ORG] login=%s state=%s role=%s", login, result.State, result.Role)

	isMember := result.State == "active"
	log.Printf("[AUTH][ORG] login=%s isMember=%v", login, isMember)
	return isMember, nil
}

func determineRole(token, login string) (string, error) {
	log.Printf("[AUTH][ROLE] Checking team memberships for login=%s across %d teams",
		login, len(teamRoles))

	bestRole := ""
	bestPriority := 0

	for team, role := range teamRoles {
		log.Printf("[AUTH][ROLE] Checking team=%s → candidate role=%s", team, role)

		isMember, err := checkTeamMembership(token, team, login)
		if err != nil {
			log.Printf("[AUTH][ROLE][WARN] Team check error for team=%s login=%s: %v — skipping",
				team, login, err)
			continue
		}

		log.Printf("[AUTH][ROLE] login=%s team=%s isMember=%v", login, team, isMember)

		if isMember && rolePriority[role] > bestPriority {
			log.Printf("[AUTH][ROLE] Upgrading role → %s (priority %d → %d)",
				role, bestPriority, rolePriority[role])
			bestRole = role
			bestPriority = rolePriority[role]
		}
	}

	if bestRole == "" {
		log.Printf("[AUTH][ROLE][WARN] No team match found for login=%s", login)
	} else {
		log.Printf("[AUTH][ROLE][SUCCESS] Final role for login=%s → %s (priority=%d)",
			login, bestRole, bestPriority)
	}

	return bestRole, nil
}

func checkTeamMembership(token, team, login string) (bool, error) {
	url := fmt.Sprintf(
		"https://api.github.com/orgs/%s/teams/%s/memberships/%s",
		githubOrg, team, login,
	)
	log.Printf("[AUTH][TEAM] GET %s", url)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "platform-backend")

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	log.Printf("[AUTH][TEAM] Responded in %s status=%s", time.Since(start), resp.Status)

	if err != nil {
		log.Printf("[AUTH][TEAM][ERROR] HTTP request failed team=%s login=%s: %v", team, login, err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[AUTH][TEAM] team=%s login=%s → not a member (status=%s)", team, login, resp.Status)
		return false, nil
	}

	var result struct {
		State string `json:"state"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	active := result.State == "active"
	log.Printf("[AUTH][TEAM] team=%s login=%s state=%s active=%v", team, login, result.State, active)
	return active, nil
}

func redirectError(w http.ResponseWriter, r *http.Request, reason string) {
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	dest := fmt.Sprintf("%s/login?error=%s", frontendURL, reason)
	log.Printf("[AUTH][REDIRECT_ERROR] Reason=%s → redirecting to %s", reason, dest)
	http.Redirect(w, r, dest, http.StatusTemporaryRedirect)
}
