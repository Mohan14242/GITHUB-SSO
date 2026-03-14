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

type RollbackRequest struct {
	Environment string `json:"environment"`
	Version     string `json:"version"`
}

func RollbackService(w http.ResponseWriter, r *http.Request) {
	log.Println("[ROLLBACK] Incoming request")

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// /rollback-services/{serviceName}/rollback
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) != 3 || parts[0] != "rollback-services" || parts[2] != "rollback" {
		log.Printf("[ROLLBACK][WARN] Invalid path: %s\n", r.URL.Path)
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	serviceName := parts[1]
	log.Printf("[ROLLBACK] Service: %s\n", serviceName)

	var req RollbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[ROLLBACK] Request: service=%s env=%s version=%s\n", serviceName, req.Environment, req.Version)

	if req.Environment == "" || req.Version == "" {
		http.Error(w, "environment and version are required", http.StatusBadRequest)
		return
	}

	// Validate artifact exists
	var exists bool
	err := db.DB.QueryRow(`
		SELECT EXISTS (
		  SELECT 1 FROM artifacts
		  WHERE service_name = ? AND environment = ? AND version = ?
		)`,
		serviceName, req.Environment, req.Version,
	).Scan(&exists)
	if err != nil {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status: "failed", Details:"failed to check the DB",
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details: fmt.Sprintf("Artifact not found for version=%s", req.Version),
		})
		http.Error(w, "invalid version for environment", http.StatusBadRequest)
		return
	}

	// Get current running version
	var currentVersion string
	err = db.DB.QueryRow(`
		SELECT version FROM environment_state
		WHERE service_name = ? AND environment = ?
	`, serviceName, req.Environment).Scan(&currentVersion)
	if err != nil {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details: "Failed to fetch current environment state",
		})
		http.Error(w, "failed to fetch current environment state", http.StatusInternalServerError)
		return
	}

	// Prevent rollback to same version
	if currentVersion == req.Version {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details: fmt.Sprintf("Blocked: %s is already the running version", req.Version),
		})
		http.Error(w, "this is the current running version", http.StatusBadRequest)
		return
	}

	// Get CICD type & repo
	var cicdType, repo string
	err = db.DB.QueryRow(`
		SELECT cicd_type, repo_name FROM services WHERE service_name = ?
	`, serviceName).Scan(&cicdType, &repo)
	if err != nil {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status: "failed", Details: "Service not found in database",
		})
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	log.Printf("[ROLLBACK] cicdType=%s repo=%s\n", cicdType, repo)

	actor := "unknown"
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		actor = claims.GithubLogin
	}

	// ── ONLY CHANGE: CreatePipelineRun → CreateRollbackPipelineRun ──
	// CreateRollbackPipelineRun creates only 2 stages in pipeline_stages:
	//   "Rollback" and "Health Check"
	// These match exactly what Jenkins sends via notifyStage() when
	// ROLLBACK=true — no phantom pending stages are left in the DB.
	runID, err := CreateRollbackPipelineRun(serviceName, req.Environment, actor, cicdType)
	if err != nil {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details:"failed to create the rollback pipeline",
		})
		http.Error(w, "failed to create pipeline run", http.StatusInternalServerError)
		return
	}
	log.Printf("[ROLLBACK] rollback pipeline run created runID=%d service=%s env=%s stages=Rollback,HealthCheck\n",
		runID, serviceName, req.Environment)
	
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

	log.Printf("[ROLLBACK] env=%s mappedBranch=%s\n", req.Environment, branch)
	// Trigger rollback via CICD
	switch cicdType {
	case "jenkins":
		err = cicd.TriggerJenkinsRollback(serviceName, req.Environment, req.Version, runID)
	case "github":
		err = cicd.TriggerGitHubRollback(repo, req.Environment, req.Version, runID)
	default:
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details: fmt.Sprintf("Unsupported cicd type: %s", cicdType),
		})
		http.Error(w, "unsupported cicd type", http.StatusBadRequest)
		return
	}

	if err != nil {
		audit.Log(r, audit.Entry{
			Action: "rollback", ResourceType: "deployment",
			ResourceName: serviceName, Environment: req.Environment,
			Status:  "failed",
			Details: fmt.Sprintf("CICD trigger failed via %s runID=%d", cicdType, runID),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("[ROLLBACK][SUCCESS] service=%s env=%s from=%s to=%s runID=%d\n",
		serviceName, req.Environment, currentVersion, req.Version, runID)

	audit.Log(r, audit.Entry{
		Action: "rollback", ResourceType: "deployment",
		ResourceName: serviceName, Environment: req.Environment,
		Status: "success",
		Details: fmt.Sprintf("Rolled back from version=%s to version=%s via %s runID=%d",
			currentVersion, req.Version, cicdType, runID),
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "rollback triggered",
		"runId":   runID,
	})
}