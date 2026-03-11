package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"src/src/internal/audit"
	"src/src/internal/auth"
	"src/src/internal/cicd"
	"src/src/internal/db"
)

type DeployRequest struct {
	Environment string `json:"environment"`
}

func DeployServices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /services/{serviceName}/deploy
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[2] != "deploy" {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	serviceName := parts[1]
	log.Printf("[DEPLOY] Request received for service=%s", serviceName)

	var req DeployRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	if req.Environment == "" {
		http.Error(w, "environment required", http.StatusBadRequest)
		return
	}

	// env → branch mapping
	var branch string
	switch req.Environment {
	case "dev":
		branch = "dev"
	case "test":
		branch = "test"
	case "prod":
		branch = "master"
	default:
		http.Error(w, "invalid environment", http.StatusBadRequest)
		return
	}

	// ── prod → approval gate ──
	if req.Environment == "prod" {
		_, err := db.DB.Exec(`
			INSERT INTO deployment_approvals
			  (service_name, environment, status, created_at)
			VALUES (?, ?, 'pending', NOW())
		`, serviceName, req.Environment)

		if err != nil {
			log.Printf("[DEPLOY][ERROR] Failed to create approval for service=%s: %v", serviceName, err)
			http.Error(w, "failed to create approval", http.StatusInternalServerError)
			return
		}

		audit.Log(r, audit.Entry{
			Action:       "deployment_triggered",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  "prod",
			Status:       "pending",
			Details:      fmt.Sprintf("Production deployment queued for approval service=%s", serviceName),
		})

		log.Printf("[DEPLOY] Prod deployment queued for approval service=%s", serviceName)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte(`{"status":"pending_approval"}`))
		return
	}

	log.Printf("[DEPLOY] Triggering deployment service=%s environment=%s branch=%s",
		serviceName, req.Environment, branch)

	// 🔍 Get CICD type & repo info
	var cicdType, repo string
	err := db.DB.QueryRow(`
		SELECT cicd_type, repo_name FROM services WHERE service_name = ?
	`, serviceName).Scan(&cicdType, &repo)
	if err != nil {
		log.Printf("[DEPLOY][ERROR] Service not found service=%s: %v", serviceName, err)
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	log.Printf("[DEPLOY] cicdType=%s repo=%s", cicdType, repo)

	// get actor for pipeline run
	actor := "unknown"
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		actor = claims.GithubLogin
	}

	// 🏗️ Create pipeline run
	runID, err := CreatePipelineRun(serviceName, req.Environment, actor, cicdType)
	if err != nil {
		log.Printf("[DEPLOY][ERROR] Failed to create pipeline run service=%s: %v", serviceName, err)
		http.Error(w, "failed to create pipeline run", http.StatusInternalServerError)
		return
	}
	log.Printf("[DEPLOY] pipeline run created runID=%d service=%s env=%s", runID, serviceName, req.Environment)

	// 🚀 Trigger CICD
	switch cicdType {
	case "jenkins":
		err = cicd.TriggerJenkinsDeploy(serviceName, branch, runID)
	case "github":
		err = cicd.TriggerGitHubDeploy(repo, branch, runID)
	default:
		http.Error(w, "unsupported cicd type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[DEPLOY][ERROR] CICD trigger failed service=%s runID=%d: %v", serviceName, runID, err)

		audit.Log(r, audit.Entry{
			Action:       "deployment_triggered",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("CICD trigger failed cicdType=%s runID=%d error=%s", cicdType, runID, err.Error()),
		})

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	audit.Log(r, audit.Entry{
		Action:       "deployment_triggered",
		ResourceType: "deployment",
		ResourceName: serviceName,
		Environment:  req.Environment,
		Status:       "success",
		Details:      fmt.Sprintf("Deployment triggered cicdType=%s repo=%s branch=%s runID=%d", cicdType, repo, branch, runID),
	})

	log.Printf("[DEPLOY][SUCCESS] Deployment triggered service=%s environment=%s runID=%d",
		serviceName, req.Environment, runID)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "deployment triggered",
		"runId":   runID,
	})
}