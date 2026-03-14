package git

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// DeleteRepo deletes a repository from the organization only
func DeleteRepo(token, repoName string) error {
	log.Printf("[GIT][DELETE-REPO] starting deletion repoName=%s", repoName)

	org, err := GetOrgName()
	if err != nil {
		log.Printf("[GIT][DELETE-REPO][ERROR] failed to get org name: %v", err)
		return err
	}

	log.Printf("[GIT][DELETE-REPO] verifying repo belongs to org=%s repo=%s", org, repoName)
	if err := verifyOrgRepo(token, org, repoName); err != nil {
		log.Printf("[GIT][DELETE-REPO][ERROR] org repo verification failed org=%s repo=%s: %v",
			org, repoName, err)
		return err
	}
	log.Printf("[GIT][DELETE-REPO] ✅ repo verified org=%s repo=%s", org, repoName)

	// Check repo exists before attempting deletion
	log.Printf("[GIT][DELETE-REPO] checking repo existence org=%s repo=%s", org, repoName)
	exists, err := RepoExistsInOrg(token, org, repoName)
	if err != nil {
		log.Printf("[GIT][DELETE-REPO][ERROR] existence check failed org=%s repo=%s: %v",
			org, repoName, err)
		return err
	}
	if !exists {
		log.Printf("[GIT][DELETE-REPO] ⚠️ repo does not exist — skipping deletion org=%s repo=%s",
			org, repoName)
		return nil
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", org, repoName)
	log.Printf("[GIT][DELETE-REPO] DELETE %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		log.Printf("[GIT][DELETE-REPO][ERROR] failed to build request org=%s repo=%s: %v",
			org, repoName, err)
		return fmt.Errorf("failed to build delete repo request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[GIT][DELETE-REPO][ERROR] HTTP request failed org=%s repo=%s: %v",
			org, repoName, err)
		return fmt.Errorf("failed to delete repo %s in org %s: %w", repoName, org, err)
	}
	defer resp.Body.Close()

	log.Printf("[GIT][DELETE-REPO] GitHub response status=%d org=%s repo=%s",
		resp.StatusCode, org, repoName)

	switch resp.StatusCode {
	case http.StatusNoContent:
		log.Printf("[GIT][DELETE-REPO] ✅ repo deleted successfully org=%s repo=%s",
			org, repoName)
		return nil

	case http.StatusNotFound:
		log.Printf("[GIT][DELETE-REPO] ⚠️ repo not found (already deleted) org=%s repo=%s",
			org, repoName)
		return nil

	case http.StatusUnauthorized:
		log.Printf("[GIT][DELETE-REPO][ERROR] token unauthorized org=%s repo=%s",
			org, repoName)
		return fmt.Errorf("github token is invalid or expired (status 401)")

	case http.StatusForbidden:
		log.Printf("[GIT][DELETE-REPO][ERROR] access forbidden org=%s repo=%s — token needs delete_repo scope",
			org, repoName)
		return fmt.Errorf(
			"access forbidden — token needs 'delete_repo' scope to delete org repo %s/%s",
			org, repoName,
		)

	default:
		body, _ := io.ReadAll(resp.Body)
		log.Printf("[GIT][DELETE-REPO][ERROR] unexpected status=%d org=%s repo=%s body=%s",
			resp.StatusCode, org, repoName, string(body))
		return fmt.Errorf(
			"repo deletion failed: org=%s repo=%s status=%d body=%s",
			org, repoName, resp.StatusCode, string(body),
		)
	}
}


