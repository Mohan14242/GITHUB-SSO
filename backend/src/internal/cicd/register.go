package cicd

import (
	"fmt"
	"log"
	"os"
	"strings"
)

// RegisterJenkins creates a Jenkins multibranch job and optionally
// creates a GitHub webhook — always scoped to the organization
func RegisterJenkins(
	repoURL, serviceName string,
	enableWebhook bool,
) (string, error) {
	log.Printf("[CICD][REGISTER-JENKINS] starting serviceName=%s repoURL=%s enableWebhook=%v",
		serviceName, repoURL, enableWebhook)

	// ── Jenkins client ────────────────────────────────────────────
	log.Printf("[CICD][REGISTER-JENKINS] initialising Jenkins client")
	jenkins := NewJenkinsClient()
	log.Printf("[CICD][REGISTER-JENKINS] ✅ Jenkins client ready")

	// ── GitHub client (only if webhook needed) ────────────────────
	var github *GitHubClient
	if enableWebhook {
		log.Printf("[CICD][REGISTER-JENKINS] webhook enabled — creating GitHub client")
		var err error
		github, err = NewGitHubClient()
		if err != nil {
			log.Printf("[CICD][REGISTER-JENKINS][ERROR] failed to create GitHub client: %v", err)
			return "", fmt.Errorf("failed to create GitHub client: %w", err)
		}
		log.Printf("[CICD][REGISTER-JENKINS] ✅ GitHub client ready org=%s", github.Org)
	} else {
		log.Printf("[CICD][REGISTER-JENKINS] webhook disabled — skipping GitHub client creation")
	}

	// ── Generate webhook token ────────────────────────────────────
	log.Printf("[CICD][REGISTER-JENKINS] generating webhook token")
	webhookToken, err := GenerateWebhookToken()
	if err != nil {
		log.Printf("[CICD][REGISTER-JENKINS][ERROR] failed to generate webhook token: %v", err)
		return "", fmt.Errorf("failed to generate webhook token: %w", err)
	}
	log.Printf("[CICD][REGISTER-JENKINS] ✅ webhook token generated")

	// ── Step 1: Create Jenkins multibranch job ────────────────────
	credentialsID := os.Getenv("JENKINS_GITHUB_CREDENTIALS_ID")
	if credentialsID == "" {
		log.Printf("[CICD][REGISTER-JENKINS][ERROR] JENKINS_GITHUB_CREDENTIALS_ID env var is not set")
		return "", fmt.Errorf("JENKINS_GITHUB_CREDENTIALS_ID environment variable is not set")
	}

	log.Printf("[CICD][REGISTER-JENKINS] creating Jenkins multibranch job serviceName=%s repoURL=%s credentialsID=%s",
		serviceName, repoURL, credentialsID)

	if err := jenkins.CreateMultibranchJob(
		serviceName,
		repoURL,
		credentialsID,
		webhookToken,
	); err != nil {
		log.Printf("[CICD][REGISTER-JENKINS][ERROR] Jenkins job creation failed serviceName=%s: %v",
			serviceName, err)
		return "", fmt.Errorf("jenkins job creation failed for service %s: %w", serviceName, err)
	}
	log.Printf("[CICD][REGISTER-JENKINS] ✅ Jenkins job created serviceName=%s", serviceName)

	// ── Step 2: Create GitHub webhook (optional) ──────────────────
	if enableWebhook {
		jenkinsURL := strings.TrimRight(os.Getenv("JENKINS_URL"), "/")
		if jenkinsURL == "" {
			log.Printf("[CICD][REGISTER-JENKINS][ERROR] JENKINS_URL env var is not set")
			return "", fmt.Errorf("JENKINS_URL environment variable is not set")
		}

		webhookURL := fmt.Sprintf(
			"%s/multibranch-webhook-trigger/invoke?token=%s",
			jenkinsURL,
			webhookToken,
		)

		// Extract repo name from repoURL — org comes from GITHUB_ORG inside CreateWebhook
		repo := extractRepo(repoURL)

		log.Printf("[CICD][REGISTER-JENKINS] creating GitHub webhook org=%s repo=%s webhookURL=%s",
			github.Org, repo, webhookURL)

		if err := github.CreateWebhook(repo, webhookURL); err != nil {
			log.Printf("[CICD][REGISTER-JENKINS][ERROR] GitHub webhook creation failed org=%s repo=%s: %v",
				github.Org, repo, err)
			return "", fmt.Errorf("github webhook creation failed for repo %s: %w", repo, err)
		}
		log.Printf("[CICD][REGISTER-JENKINS] ✅ GitHub webhook created org=%s repo=%s",
			github.Org, repo)
	}

	log.Printf("[CICD][REGISTER-JENKINS] ✅ registration complete serviceName=%s enableWebhook=%v",
		serviceName, enableWebhook)

	// Return token so caller can persist it
	return webhookToken, nil
}
