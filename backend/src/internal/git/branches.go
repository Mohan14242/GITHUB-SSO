package git

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

// getOrgName returns the GitHub organization name from env
func getOrgName() (string, error) {
	org := os.Getenv("GITHUB_ORG")
	if org == "" {
		log.Println("[GIT][ERROR] GITHUB_ORG environment variable is not set")
		return "", fmt.Errorf("GITHUB_ORG environment variable is not set")
	}
	log.Printf("[GIT] org resolved: %s", org)
	return org, nil
}

// CreateBranch creates a new branch in the organization repo only
func CreateBranch(token, repo, newBranch, sourceBranch string) error {
	log.Printf("[GIT][CREATE-BRANCH] repo=%s newBranch=%s sourceBranch=%s", repo, newBranch, sourceBranch)

	org, err := getOrgName()
	if err != nil {
		return err
	}

	// 1️⃣ Verify repo belongs to the organization
	log.Printf("[GIT][CREATE-BRANCH] verifying repo belongs to org=%s", org)
	if err := verifyOrgRepo(token, org, repo); err != nil {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] org repo verification failed: %v", err)
		return err
	}
	log.Printf("[GIT][CREATE-BRANCH] ✅ repo verified org=%s repo=%s", org, repo)

	// 2️⃣ Get source branch SHA
	log.Printf("[GIT][CREATE-BRANCH] fetching SHA for sourceBranch=%s", sourceBranch)
	sha, err := getOrgBranchSHA(token, org, repo, sourceBranch)
	if err != nil {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] failed to get SHA for branch=%s: %v", sourceBranch, err)
		return err
	}
	log.Printf("[GIT][CREATE-BRANCH] ✅ SHA resolved sourceBranch=%s sha=%s", sourceBranch, sha)

	// 3️⃣ Create new branch ref
	log.Printf("[GIT][CREATE-BRANCH] creating branch ref=%s from sha=%s in org=%s repo=%s",
		newBranch, sha, org, repo)

	payload := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", newBranch),
		"sha": sha,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal create branch payload: %w", err)
	}

	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", org, repo),
		bytes.NewBuffer(body),
	)
	if err != nil {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] failed to build request: %v", err)
		return fmt.Errorf("failed to build create branch request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Content-Type",         "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] HTTP request failed: %v", err)
		return fmt.Errorf("failed to create branch %s in org %s: %w", newBranch, org, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][CREATE-BRANCH] GitHub response status=%d org=%s repo=%s branch=%s",
		resp.StatusCode, org, repo, newBranch)

	if resp.StatusCode == 422 {
		log.Printf("[GIT][CREATE-BRANCH] branch already exists — skipping org=%s repo=%s branch=%s",
			org, repo, newBranch)
		return nil
	}

	if resp.StatusCode >= 300 {
		log.Printf("[GIT][CREATE-BRANCH][ERROR] failed to create branch=%s org=%s repo=%s status=%d",
			newBranch, org, repo, resp.StatusCode)
		return fmt.Errorf("failed to create branch %s in org %s/%s — status %d",
			newBranch, org, repo, resp.StatusCode)
	}

	log.Printf("[GIT][CREATE-BRANCH] ✅ branch created successfully org=%s repo=%s branch=%s",
		org, repo, newBranch)
	return nil
}

// getOrgBranchSHA returns the SHA of a branch in the org repo
func getOrgBranchSHA(token, org, repo, branch string) (string, error) {
	log.Printf("[GIT][GET-SHA] org=%s repo=%s branch=%s", org, repo, branch)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s/git/ref/heads/%s", org, repo, branch),
		nil,
	)
	if err != nil {
		log.Printf("[GIT][GET-SHA][ERROR] failed to build request: %v", err)
		return "", fmt.Errorf("failed to build get SHA request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[GIT][GET-SHA][ERROR] HTTP request failed org=%s repo=%s branch=%s: %v",
			org, repo, branch, err)
		return "", fmt.Errorf("failed to get branch SHA for %s/%s/%s: %w", org, repo, branch, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][GET-SHA] GitHub response status=%d org=%s repo=%s branch=%s",
		resp.StatusCode, org, repo, branch)

	if resp.StatusCode == 404 {
		log.Printf("[GIT][GET-SHA][ERROR] branch not found org=%s repo=%s branch=%s", org, repo, branch)
		return "", fmt.Errorf("branch '%s' not found in org repo %s/%s", branch, org, repo)
	}
	if resp.StatusCode >= 300 {
		log.Printf("[GIT][GET-SHA][ERROR] unexpected status=%d org=%s repo=%s branch=%s",
			resp.StatusCode, org, repo, branch)
		return "", fmt.Errorf("failed to get branch SHA — org=%s repo=%s branch=%s status=%d",
			org, repo, branch, resp.StatusCode)
	}

	var res struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		log.Printf("[GIT][GET-SHA][ERROR] failed to decode response: %v", err)
		return "", fmt.Errorf("failed to decode branch SHA response: %w", err)
	}
	if res.Object.SHA == "" {
		log.Printf("[GIT][GET-SHA][ERROR] empty SHA returned org=%s repo=%s branch=%s", org, repo, branch)
		return "", fmt.Errorf("empty SHA returned for branch %s in %s/%s", branch, org, repo)
	}

	log.Printf("[GIT][GET-SHA] ✅ SHA resolved org=%s repo=%s branch=%s sha=%s",
		org, repo, branch, res.Object.SHA)
	return res.Object.SHA, nil
}

// verifyOrgRepo confirms the repo exists and belongs to the organization
func verifyOrgRepo(token, org, repo string) error {
	log.Printf("[GIT][VERIFY-REPO] verifying org=%s repo=%s", org, repo)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/%s", org, repo),
		nil,
	)
	if err != nil {
		log.Printf("[GIT][VERIFY-REPO][ERROR] failed to build request: %v", err)
		return fmt.Errorf("failed to build verify repo request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[GIT][VERIFY-REPO][ERROR] HTTP request failed org=%s repo=%s: %v", org, repo, err)
		return fmt.Errorf("failed to verify org repo %s/%s: %w", org, repo, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][VERIFY-REPO] GitHub response status=%d org=%s repo=%s",
		resp.StatusCode, org, repo)

	if resp.StatusCode == 404 {
		log.Printf("[GIT][VERIFY-REPO][ERROR] repo not found org=%s repo=%s", org, repo)
		return fmt.Errorf("repo '%s' does not exist in organization '%s'", repo, org)
	}
	if resp.StatusCode == 403 {
		log.Printf("[GIT][VERIFY-REPO][ERROR] access denied org=%s repo=%s — check token permissions",
			org, repo)
		return fmt.Errorf("access denied to org repo %s/%s — check token permissions", org, repo)
	}
	if resp.StatusCode >= 300 {
		log.Printf("[GIT][VERIFY-REPO][ERROR] unexpected status=%d org=%s repo=%s",
			resp.StatusCode, org, repo)
		return fmt.Errorf("unexpected status %d verifying org repo %s/%s", resp.StatusCode, org, repo)
	}

	var repoInfo struct {
		Owner struct {
			Login string `json:"login"`
			Type  string `json:"type"`
		} `json:"owner"`
		FullName string `json:"full_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		log.Printf("[GIT][VERIFY-REPO][ERROR] failed to decode repo info: %v", err)
		return fmt.Errorf("failed to decode repo info: %w", err)
	}

	log.Printf("[GIT][VERIFY-REPO] repo info fullName=%s ownerLogin=%s ownerType=%s",
		repoInfo.FullName, repoInfo.Owner.Login, repoInfo.Owner.Type)

	if repoInfo.Owner.Type != "Organization" {
		log.Printf("[GIT][VERIFY-REPO][ERROR] repo owner is not an org — ownerLogin=%s ownerType=%s",
			repoInfo.Owner.Login, repoInfo.Owner.Type)
		return fmt.Errorf(
			"repo '%s' is owned by '%s' (type=%s) — only organization repos are allowed",
			repo, repoInfo.Owner.Login, repoInfo.Owner.Type,
		)
	}

	if repoInfo.Owner.Login != org {
		log.Printf("[GIT][VERIFY-REPO][ERROR] org mismatch — expected=%s actual=%s",
			org, repoInfo.Owner.Login)
		return fmt.Errorf(
			"repo '%s' belongs to org '%s' but expected org '%s'",
			repo, repoInfo.Owner.Login, org,
		)
	}

	log.Printf("[GIT][VERIFY-REPO] ✅ repo verified org=%s repo=%s fullName=%s",
		org, repo, repoInfo.FullName)
	return nil
}
