package cicd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"src/src/internal/aws"
)

type GitHubClient struct {
	Token string
	Org   string
}

// ── getOrg returns GITHUB_ORG from env ──────────────────────────
func getOrg() (string, error) {
	org := os.Getenv("GITHUB_ORG")
	if org == "" {
		log.Println("[GITHUB][ERROR] GITHUB_ORG environment variable is not set")
		return "", fmt.Errorf("GITHUB_ORG environment variable is not set")
	}
	log.Printf("[GITHUB] org resolved: %s", org)
	return org, nil
}

// ── NewGitHubClient creates a client with org + token ───────────
func NewGitHubClient() (*GitHubClient, error) {
	log.Println("[GITHUB][NEW-CLIENT] fetching GitHub token from AWS Secrets Manager")

	token, err := aws.GetGitToken("git-token")
	if err != nil {
		log.Printf("[GITHUB][NEW-CLIENT][ERROR] failed to fetch GitHub token: %v", err)
		return nil, err
	}
	if token == "" {
		log.Println("[GITHUB][NEW-CLIENT][ERROR] GitHub token is empty")
		return nil, fmt.Errorf("github token is empty")
	}
	log.Println("[GITHUB][NEW-CLIENT] ✅ GitHub token fetched successfully")

	org, err := getOrg()
	if err != nil {
		log.Printf("[GITHUB][NEW-CLIENT][ERROR] failed to resolve org: %v", err)
		return nil, err
	}

	log.Printf("[GITHUB][NEW-CLIENT] ✅ client ready org=%s", org)
	return &GitHubClient{Token: token, Org: org}, nil
}

// ── CreateWebhook creates a webhook on an org repo ──────────────
func (g *GitHubClient) CreateWebhook(repo, webhookURL string) error {
	startTotal := time.Now()

	log.Println("[GITHUB][CREATE-WEBHOOK] ──────────────────────────────────")
	log.Printf("[GITHUB][CREATE-WEBHOOK] starting org=%s repo=%s webhookURL=%s",
		g.Org, repo, webhookURL)

	// 1️⃣ Build payload
	payload := map[string]interface{}{
		"name":   "web",
		"active": true,
		"events": []string{"push"},
		"config": map[string]string{
			"url":          webhookURL,
			"content_type": "json",
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal webhook payload: %w", err)
	}
	log.Printf("[GITHUB][CREATE-WEBHOOK] payload ready size=%d bytes", len(body))

	// 2️⃣ Build request — always against org repo
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/hooks", g.Org, repo)
	log.Printf("[GITHUB][CREATE-WEBHOOK] POST %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] failed to build request: %v", err)
		return fmt.Errorf("failed to build webhook request: %w", err)
	}

	req.Header.Set("Authorization",        "token "+g.Token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("Content-Type",         "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	// 3️⃣ Execute
	client := &http.Client{Timeout: 15 * time.Second}
	startHTTP := time.Now()

	resp, err := client.Do(req)
	httpDuration := time.Since(startHTTP)
	if err != nil {
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] HTTP request failed duration=%s: %v",
			httpDuration, err)
		return fmt.Errorf("webhook HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	log.Printf("[GITHUB][CREATE-WEBHOOK] response status=%s duration=%s org=%s repo=%s",
		resp.Status, httpDuration, g.Org, repo)
	log.Printf("[GITHUB][CREATE-WEBHOOK] response body=%s",
		strings.TrimSpace(string(respBody)))

	// 4️⃣ Handle status codes explicitly
	switch resp.StatusCode {
	case http.StatusCreated:
		log.Printf("[GITHUB][CREATE-WEBHOOK] ✅ webhook created successfully org=%s repo=%s totalTime=%s",
			g.Org, repo, time.Since(startTotal))
		log.Println("[GITHUB][CREATE-WEBHOOK] ──────────────────────────────────")
		return nil

	case http.StatusUnprocessableEntity: // 422
		log.Printf("[GITHUB][CREATE-WEBHOOK] ⚠️ webhook already exists or validation failed org=%s repo=%s",
			g.Org, repo)
		return nil // treat as success — idempotent

	case http.StatusUnauthorized:
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] 401 unauthorized — token invalid or expired org=%s repo=%s",
			g.Org, repo)
		return fmt.Errorf("github webhook: token is invalid or expired (401)")

	case http.StatusForbidden:
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] 403 forbidden — token missing admin:repo_hook scope org=%s repo=%s",
			g.Org, repo)
		return fmt.Errorf("github webhook: token missing admin:repo_hook permission (403)")

	case http.StatusNotFound:
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] 404 not found — repo does not exist in org org=%s repo=%s",
			g.Org, repo)
		return fmt.Errorf("github webhook: repo '%s' not found in org '%s' (404)", repo, g.Org)

	default:
		log.Printf("[GITHUB][CREATE-WEBHOOK][ERROR] unexpected status=%s org=%s repo=%s body=%s",
			resp.Status, g.Org, repo, strings.TrimSpace(string(respBody)))
		return fmt.Errorf("github webhook creation failed: status=%s org=%s repo=%s",
			resp.Status, g.Org, repo)
	}
}

// ── TriggerGitHubDeploy dispatches a workflow on an org repo ────
func TriggerGitHubDeploy(repo, branch string) error {
	log.Printf("[GITHUB][DEPLOY] starting repo=%s branch=%s", repo, branch)

	token, err := aws.GetGitToken("git-token")
	if err != nil {
		log.Printf("[GITHUB][DEPLOY][ERROR] failed to fetch GitHub token: %v", err)
		return fmt.Errorf("failed to fetch github token: %w", err)
	}
	log.Println("[GITHUB][DEPLOY] ✅ token fetched")

	org, err := getOrg()
	if err != nil {
		log.Printf("[GITHUB][DEPLOY][ERROR] failed to resolve org: %v", err)
		return err
	}

	workflow := "cicd.yaml"

	payload := map[string]interface{}{
		"ref": branch,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[GITHUB][DEPLOY][ERROR] failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal deploy payload: %w", err)
	}
	log.Printf("[GITHUB][DEPLOY] payload ready ref=%s", branch)

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches",
		org, repo, workflow,
	)
	log.Printf("[GITHUB][DEPLOY] POST %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[GITHUB][DEPLOY][ERROR] failed to build request: %v", err)
		return fmt.Errorf("failed to build deploy request: %w", err)
	}

	req.Header.Set("Authorization",        "Bearer "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("Content-Type",         "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	startHTTP := time.Now()
	resp, err := http.DefaultClient.Do(req)
	httpDuration := time.Since(startHTTP)
	if err != nil {
		log.Printf("[GITHUB][DEPLOY][ERROR] HTTP request failed duration=%s: %v",
			httpDuration, err)
		return fmt.Errorf("deploy HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[GITHUB][DEPLOY] response status=%s duration=%s org=%s repo=%s branch=%s",
		resp.Status, httpDuration, org, repo, branch)

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[GITHUB][DEPLOY][ERROR] dispatch rejected status=%s org=%s repo=%s body=%s",
			resp.Status, org, repo, strings.TrimSpace(string(bodyBytes)))
		return fmt.Errorf(
			"github deploy dispatch failed: org=%s repo=%s status=%s body=%s",
			org, repo, resp.Status, strings.TrimSpace(string(bodyBytes)),
		)
	}

	log.Printf("[GITHUB][DEPLOY] ✅ workflow dispatched successfully org=%s repo=%s branch=%s",
		org, repo, branch)
	return nil
}

// ── TriggerGitHubRollback dispatches a rollback workflow ────────
func TriggerGitHubRollback(repo, environment, version string) error {
	log.Printf("[GITHUB][ROLLBACK] starting repo=%s environment=%s version=%s",
		repo, environment, version)

	token, err := aws.GetGitToken("git-token")
	if err != nil {
		log.Printf("[GITHUB][ROLLBACK][ERROR] failed to fetch GitHub token: %v", err)
		return fmt.Errorf("failed to fetch github token: %w", err)
	}
	log.Println("[GITHUB][ROLLBACK] ✅ token fetched")

	org, err := getOrg()
	if err != nil {
		log.Printf("[GITHUB][ROLLBACK][ERROR] failed to resolve org: %v", err)
		return err
	}

	workflow := "cicd.yaml"
	ref      := environmentBranch(environment)

	log.Printf("[GITHUB][ROLLBACK] resolved ref=%s for environment=%s org=%s repo=%s",
		ref, environment, org, repo)

	payload := map[string]interface{}{
		"ref": ref,
		"inputs": map[string]string{
			"rollback":         "true",
			"rollback_version": version,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[GITHUB][ROLLBACK][ERROR] failed to marshal payload: %v", err)
		return fmt.Errorf("failed to marshal rollback payload: %w", err)
	}
	log.Printf("[GITHUB][ROLLBACK] payload ready body=%s", string(body))

	url := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/actions/workflows/%s/dispatches",
		org, repo, workflow,
	)
	log.Printf("[GITHUB][ROLLBACK] POST %s", url)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		log.Printf("[GITHUB][ROLLBACK][ERROR] failed to build request: %v", err)
		return fmt.Errorf("failed to build rollback request: %w", err)
	}

	req.Header.Set("Authorization",        "Bearer "+token)
	req.Header.Set("Accept",               "application/vnd.github+json")
	req.Header.Set("Content-Type",         "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent",           "platform-backend")

	startHTTP := time.Now()
	resp, err := http.DefaultClient.Do(req)
	httpDuration := time.Since(startHTTP)
	if err != nil {
		log.Printf("[GITHUB][ROLLBACK][ERROR] HTTP request failed duration=%s: %v",
			httpDuration, err)
		return fmt.Errorf("rollback HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	log.Printf("[GITHUB][ROLLBACK] response status=%s duration=%s org=%s repo=%s",
		resp.Status, httpDuration, org, repo)

	if resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Printf("[GITHUB][ROLLBACK][ERROR] dispatch rejected status=%s org=%s repo=%s body=%s",
			resp.Status, org, repo, strings.TrimSpace(string(bodyBytes)))
		return fmt.Errorf(
			"github rollback dispatch failed: org=%s repo=%s status=%s body=%s",
			org, repo, resp.Status, strings.TrimSpace(string(bodyBytes)),
		)
	}

	log.Printf("[GITHUB][ROLLBACK] ✅ rollback workflow dispatched successfully org=%s repo=%s environment=%s version=%s",
		org, repo, environment, version)
	return nil
}

// ── environmentBranch maps environment name to git branch ───────
func environmentBranch(env string) string {
	switch env {
	case "dev":
		return "dev"
	case "test":
		return "test"
	case "prod":
		return "master"
	default:
		log.Printf("[GITHUB] ⚠️ unknown environment=%s defaulting to master", env)
		return "master"
	}
}
