package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type githubUser struct {
	Login string `json:"login"`
}

var githubClient = &http.Client{
	Timeout: 15 * time.Second,
}

// CreateRepo creates a repository inside the organization only
func CreateRepo(token, repoName string) (string, error) {
	log.Printf("[GIT][CREATE-REPO] starting repo creation repoName=%s", repoName)

	org, err := getOrgName()
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] failed to get org name: %v", err)
		return "", err
	}

	log.Printf("[GIT][CREATE-REPO] checking if repo already exists org=%s repo=%s", org, repoName)
	exists, err := RepoExists(token, org, repoName)
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] repo existence check failed org=%s repo=%s: %v",
			org, repoName, err)
		return "", err
	}

	repoURL := fmt.Sprintf("https://github.com/%s/%s", org, repoName)

	if exists {
		log.Printf("[GIT][CREATE-REPO] ⚠️ repo already exists — skipping creation org=%s repo=%s url=%s",
			org, repoName, repoURL)
		return repoURL, nil
	}

	log.Printf("[GIT][CREATE-REPO] repo does not exist — creating under org=%s repo=%s", org, repoName)

	payload, err := json.Marshal(map[string]interface{}{
		"name":        repoName,
		"private":     true,
		"auto_init":   true,
		"description": fmt.Sprintf("Service repository for %s", repoName),
	})
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] failed to marshal payload: %v", err)
		return "", fmt.Errorf("failed to marshal create repo payload: %w", err)
	}

	// Use org repos endpoint — not /user/repos
	url := fmt.Sprintf("https://api.github.com/orgs/%s/repos", org)
	log.Printf("[GIT][CREATE-REPO] POST %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] failed to build request: %v", err)
		return "", fmt.Errorf("failed to build create repo request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Content-Type",         "application/json")
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	client := &http.Client{Timeout: 20 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] HTTP request failed org=%s repo=%s: %v",
			org, repoName, err)
		return "", fmt.Errorf("failed to create repo %s in org %s: %w", repoName, org, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][CREATE-REPO] GitHub response status=%d org=%s repo=%s",
		resp.StatusCode, org, repoName)

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[GIT][CREATE-REPO][ERROR] creation failed status=%d org=%s repo=%s body=%s",
			resp.StatusCode, org, repoName, string(body))
		return "", fmt.Errorf(
			"repo creation failed: org=%s repo=%s status=%d body=%s",
			org, repoName, resp.StatusCode, string(body),
		)
	}

	log.Printf("[GIT][CREATE-REPO] ✅ repo created successfully org=%s repo=%s url=%s",
		org, repoName, repoURL)
	return repoURL, nil
}

// GetAuthenticatedUser returns the GitHub login of the token owner
func GetAuthenticatedUser(token string) (string, error) {
	log.Println("[GIT][GET-USER] fetching authenticated GitHub user")

	req, err := http.NewRequest("GET", "https://api.github.com/user", nil)
	if err != nil {
		log.Printf("[GIT][GET-USER][ERROR] failed to build request: %v", err)
		return "", fmt.Errorf("failed to build get user request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	resp, err := githubClient.Do(req)
	if err != nil {
		log.Printf("[GIT][GET-USER][ERROR] HTTP request failed: %v", err)
		return "", fmt.Errorf("failed to fetch authenticated user: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][GET-USER] GitHub response status=%d", resp.StatusCode)

	if resp.StatusCode == http.StatusUnauthorized {
		log.Println("[GIT][GET-USER][ERROR] token is invalid or expired")
		return "", fmt.Errorf("github token is invalid or expired (status 401)")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[GIT][GET-USER][ERROR] unexpected status=%d body=%s",
			resp.StatusCode, string(body))
		return "", fmt.Errorf(
			"github api /user failed: status=%d body=%s",
			resp.StatusCode, string(body),
		)
	}

	var user githubUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		log.Printf("[GIT][GET-USER][ERROR] failed to decode response: %v", err)
		return "", fmt.Errorf("failed to decode github user response: %w", err)
	}

	if user.Login == "" {
		log.Println("[GIT][GET-USER][ERROR] github user login is empty")
		return "", fmt.Errorf("github user login is empty")
	}

	log.Printf("[GIT][GET-USER] ✅ authenticated user resolved login=%s", user.Login)
	return user.Login, nil
}

// RepoExists checks whether a repo exists under the organization
func RepoExists(token, owner, repoName string) (bool, error) {
	log.Printf("[GIT][REPO-EXISTS] checking org=%s repo=%s", owner, repoName)

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repoName)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("[GIT][REPO-EXISTS][ERROR] failed to build request: %v", err)
		return false, fmt.Errorf("failed to build repo exists request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	resp, err := githubClient.Do(req)
	if err != nil {
		log.Printf("[GIT][REPO-EXISTS][ERROR] HTTP request failed org=%s repo=%s: %v",
			owner, repoName, err)
		return false, fmt.Errorf("failed to check repo existence %s/%s: %w", owner, repoName, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][REPO-EXISTS] GitHub response status=%d org=%s repo=%s",
		resp.StatusCode, owner, repoName)

	switch resp.StatusCode {
	case http.StatusOK:
		log.Printf("[GIT][REPO-EXISTS] ✅ repo exists org=%s repo=%s", owner, repoName)
		return true, nil

	case http.StatusNotFound:
		log.Printf("[GIT][REPO-EXISTS] repo does not exist org=%s repo=%s", owner, repoName)
		return false, nil

	case http.StatusUnauthorized:
		log.Printf("[GIT][REPO-EXISTS][ERROR] token unauthorized org=%s repo=%s", owner, repoName)
		return false, fmt.Errorf("github token is invalid or expired (status 401)")

	case http.StatusForbidden:
		log.Printf("[GIT][REPO-EXISTS][ERROR] access forbidden org=%s repo=%s — check token scopes",
			owner, repoName)
		return false, fmt.Errorf(
			"access forbidden to org repo %s/%s — check token scopes",
			owner, repoName,
		)

	default:
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[GIT][REPO-EXISTS][ERROR] unexpected status=%d org=%s repo=%s body=%s",
			resp.StatusCode, owner, repoName, string(body))
		return false, fmt.Errorf(
			"repo existence check failed: org=%s repo=%s status=%d body=%s",
			owner, repoName, resp.StatusCode, string(body),
		)
	}
}


