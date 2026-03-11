package cicd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type JenkinsClient struct {
	BaseURL string
	User    string
	Token   string
}

// ─────────────────────────────────────────────
// Create Jenkins client with normalized base URL
// ─────────────────────────────────────────────

func NewJenkinsClient() *JenkinsClient {
	baseURL := os.Getenv("JENKINS_URL")
	user    := os.Getenv("JENKINS_USER")
	token   := os.Getenv("JENKINS_API_TOKEN")

	log.Println("[JENKINS] Initializing Jenkins client")
	log.Println("[JENKINS] Raw JENKINS_URL:", baseURL)
	log.Println("[JENKINS] Jenkins user:", user)

	if baseURL == "" || user == "" || token == "" {
		log.Fatal("[JENKINS] Missing required Jenkins environment variables")
	}

	client := &JenkinsClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		User:    user,
		Token:   token,
	}

	log.Println("[JENKINS] Normalized BaseURL:", client.BaseURL)
	return client
}

// ─────────────────────────────────────────────
// CSRF CRUMB (method on JenkinsClient)
// ─────────────────────────────────────────────

func (j *JenkinsClient) getCrumb() (string, string, error) {
	crumbURL := j.BaseURL + "/crumbIssuer/api/json"
	log.Println("[JENKINS] Fetching CSRF crumb from:", crumbURL)

	req, err := http.NewRequest("GET", crumbURL, nil)
	if err != nil {
		log.Println("[JENKINS][ERROR] Failed to create crumb request:", err)
		return "", "", err
	}
	req.SetBasicAuth(j.User, j.Token)

	start    := time.Now()
	resp, err := http.DefaultClient.Do(req)
	log.Println("[JENKINS] Crumb request latency:", time.Since(start))

	if err != nil {
		log.Println("[JENKINS][ERROR] Crumb request failed:", err)
		return "", "", err
	}
	defer resp.Body.Close()

	log.Println("[JENKINS] Crumb response status:", resp.Status)

	if resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("crumb fetch failed: %s", resp.Status)
	}

	var data struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		log.Println("[JENKINS][ERROR] Failed to decode crumb response:", err)
		return "", "", err
	}

	log.Println("[JENKINS] Crumb field:", data.CrumbRequestField)
	return data.CrumbRequestField, data.Crumb, nil
}

// ─────────────────────────────────────────────
// getCrumb as standalone helper (used by Trigger functions)
// ─────────────────────────────────────────────

func getCrumb(client *http.Client, jenkinsURL, user, apiToken string) (string, string, error) {
	crumbURL := fmt.Sprintf("%s/crumbIssuer/api/json", jenkinsURL)

	req, err := http.NewRequest("GET", crumbURL, nil)
	if err != nil {
		return "", "", err
	}
	req.SetBasicAuth(user, apiToken)

	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("crumb fetch failed: %s - %s", resp.Status, string(body))
	}

	var data struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	return data.CrumbRequestField, data.Crumb, nil
}

// ─────────────────────────────────────────────
// CREATE MULTIBRANCH JOB
// ─────────────────────────────────────────────

func (j *JenkinsClient) CreateMultibranchJob(
	jobName, repoURL, credentialsID, webhookToken string,
) error {
	log.Println("[JENKINS] Creating multibranch job:", jobName)

	configXML := fmt.Sprintf(`
<org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProject plugin="workflow-multibranch">
  <description>Auto-created by Platform</description>

  <properties>
    <com.igalg.jenkins.plugins.mswt.trigger.ComputedFolderWebHookTrigger>
      <token>%s</token>
    </com.igalg.jenkins.plugins.mswt.trigger.ComputedFolderWebHookTrigger>
  </properties>

  <orphanedItemStrategy class="com.cloudbees.hudson.plugins.folder.computed.DefaultOrphanedItemStrategy">
    <pruneDeadBranches>true</pruneDeadBranches>
    <daysToKeep>-1</daysToKeep>
    <numToKeep>-1</numToKeep>
  </orphanedItemStrategy>

  <sources class="jenkins.branch.MultiBranchProject$BranchSourceList">
    <data>
      <jenkins.branch.BranchSource>
        <source class="org.jenkinsci.plugins.github_branch_source.GitHubSCMSource">
          <id>%s</id>
          <repoOwner>%s</repoOwner>
          <repository>%s</repository>
          <credentialsId>%s</credentialsId>
        </source>
      </jenkins.branch.BranchSource>
    </data>
  </sources>

  <factory class="org.jenkinsci.plugins.workflow.multibranch.WorkflowBranchProjectFactory">
    <scriptPath>Jenkinsfile</scriptPath>
  </factory>
</org.jenkinsci.plugins.workflow.multibranch.WorkflowMultiBranchProject>
`,
		webhookToken,
		jobName,
		extractOwner(repoURL),
		extractRepo(repoURL),
		credentialsID,
	)

	endpoint := fmt.Sprintf("%s/createItem?name=%s", j.BaseURL, url.QueryEscape(jobName))

	req, _ := http.NewRequest("POST", endpoint, bytes.NewBuffer([]byte(configXML)))
	req.SetBasicAuth(j.User, j.Token)
	req.Header.Set("Content-Type", "application/xml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("jenkins job creation failed: %s - %s", resp.Status, string(body))
	}

	log.Println("[JENKINS] Job created:", jobName)
	return nil
}

// ─────────────────────────────────────────────
// TRIGGER DEPLOY
// runID is passed so Jenkins can send stage
// updates back to the platform pipeline screen
// ─────────────────────────────────────────────

func TriggerJenkinsDeploy(jobName, branch string, runID int64) error {
	jenkinsURL := strings.TrimRight(os.Getenv("JENKINS_URL"), "/")
	user       := os.Getenv("JENKINS_USER")
	apiToken   := os.Getenv("JENKINS_API_TOKEN")

	if jenkinsURL == "" || user == "" || apiToken == "" {
		return fmt.Errorf("jenkins environment variables not set")
	}

	log.Printf("[JENKINS] TriggerJenkinsDeploy job=%s branch=%s runID=%d", jobName, branch, runID)

	client := &http.Client{}

	/* 1️⃣ GET CRUMB */
	crumbField, crumb, err := getCrumb(client, jenkinsURL, user, apiToken)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	/* 2️⃣ PREPARE PARAMETERS — include RUN_ID */
	formData := url.Values{}
	formData.Set("ROLLBACK",         "false")
	formData.Set("ROLLBACK_VERSION", "")
	formData.Set("RUN_ID",           fmt.Sprintf("%d", runID)) // ← passed to Jenkinsfile

	/* 3️⃣ BUILD MULTIBRANCH URL */
	buildURL := fmt.Sprintf(
		"%s/job/%s/job/%s/buildWithParameters",
		jenkinsURL,
		url.PathEscape(jobName),
		url.PathEscape(branch),
	)
	log.Println("[JENKINS] Build URL:", buildURL)

	/* 4️⃣ TRIGGER BUILD */
	req, err := http.NewRequest("POST", buildURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(crumbField, crumb)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 && resp.StatusCode != 302 {
		return fmt.Errorf("jenkins trigger failed: %s - %s", resp.Status, string(body))
	}

	log.Printf("[JENKINS] Deploy triggered successfully job=%s branch=%s runID=%d", jobName, branch, runID)
	return nil
}

// ─────────────────────────────────────────────
// TRIGGER ROLLBACK
// runID is passed so Jenkins can send stage
// updates back to the platform pipeline screen
// ─────────────────────────────────────────────

func TriggerJenkinsRollback(serviceName, branch, version string, runID int64) error {
	jenkinsURL := strings.TrimRight(os.Getenv("JENKINS_URL"), "/")
	user       := os.Getenv("JENKINS_USER")
	apiToken   := os.Getenv("JENKINS_API_TOKEN")

	if jenkinsURL == "" || user == "" || apiToken == "" {
		return fmt.Errorf("jenkins environment variables not set")
	}

	log.Printf("[JENKINS] TriggerJenkinsRollback service=%s branch=%s version=%s runID=%d",
		serviceName, branch, version, runID)

	client := &http.Client{}

	/* 1️⃣ GET CRUMB */
	crumbField, crumb, err := getCrumb(client, jenkinsURL, user, apiToken)
	if err != nil {
		return fmt.Errorf("failed to get crumb: %w", err)
	}

	/* 2️⃣ PREPARE PARAMETERS — include RUN_ID */
	formData := url.Values{}
	formData.Set("ROLLBACK",         "true")
	formData.Set("ROLLBACK_VERSION", version)
	formData.Set("RUN_ID",           fmt.Sprintf("%d", runID)) // ← passed to Jenkinsfile

	/* 3️⃣ BUILD MULTIBRANCH URL */
	buildURL := fmt.Sprintf(
		"%s/job/%s/job/%s/buildWithParameters",
		jenkinsURL,
		url.PathEscape(serviceName),
		url.PathEscape(branch),
	)
	log.Println("[JENKINS] Rollback URL:", buildURL)

	/* 4️⃣ TRIGGER ROLLBACK */
	req, err := http.NewRequest("POST", buildURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return err
	}
	req.SetBasicAuth(user, apiToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(crumbField, crumb)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 201 && resp.StatusCode != 302 {
		return fmt.Errorf("jenkins rollback failed: %s - %s", resp.Status, string(body))
	}

	log.Printf("[JENKINS] Rollback triggered successfully service=%s branch=%s version=%s runID=%d",
		serviceName, branch, version, runID)
	return nil
}