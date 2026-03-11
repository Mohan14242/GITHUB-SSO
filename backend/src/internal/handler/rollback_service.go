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
		log.Printf("[ROLLBACK][WARN] Invalid method: %s\n", r.Method)
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
		log.Printf("[ROLLBACK][ERROR] Invalid request body: %v\n", err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[ROLLBACK] Request payload: service=%s env=%s version=%s\n",
		serviceName, req.Environment, req.Version)

	if req.Environment == "" || req.Version == "" {
		log.Printf("[ROLLBACK][WARN] Missing environment or version\n")
		http.Error(w, "environment and version are required", http.StatusBadRequest)
		return
	}

	// 🔍 Validate artifact exists
	var exists bool
	err := db.DB.QueryRow(`
		SELECT EXISTS (
		  SELECT 1 FROM artifacts
		  WHERE service_name = ? AND environment = ? AND version = ?
		)`,
		serviceName, req.Environment, req.Version,
	).Scan(&exists)
	if err != nil {
		log.Printf("[ROLLBACK][ERROR] DB error checking artifact: %v\n", err)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("DB error checking artifact: %v", err),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !exists {
		log.Printf("[ROLLBACK][WARN] Artifact not found: service=%s env=%s version=%s\n",
			serviceName, req.Environment, req.Version)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("Artifact not found for version=%s", req.Version),
		})
		http.Error(w, "invalid version for environment", http.StatusBadRequest)
		return
	}

	// 🔍 Get current running version
	var currentVersion string
	err = db.DB.QueryRow(`
		SELECT version FROM environment_state
		WHERE service_name = ? AND environment = ?
	`, serviceName, req.Environment).Scan(&currentVersion)
	if err != nil {
		log.Printf("[ROLLBACK][ERROR] Failed to fetch current version: service=%s env=%s err=%v\n",
			serviceName, req.Environment, err)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("Failed to fetch current environment state: %v", err),
		})
		http.Error(w, "failed to fetch current environment state", http.StatusInternalServerError)
		return
	}

	log.Printf("[ROLLBACK] Current version: %s | Requested version: %s\n", currentVersion, req.Version)

	// 🚫 Prevent rollback to same version
	if currentVersion == req.Version {
		log.Printf("[ROLLBACK][WARN] Rollback blocked (same version): service=%s env=%s version=%s\n",
			serviceName, req.Environment, req.Version)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("Blocked: %s is already the running version", req.Version),
		})
		http.Error(w, "this is the current running version", http.StatusBadRequest)
		return
	}

	// 🔍 Get CICD type & repo
	var cicdType, repo string
	err = db.DB.QueryRow(`
		SELECT cicd_type, repo_name FROM services WHERE service_name = ?
	`, serviceName).Scan(&cicdType, &repo)
	if err != nil {
		log.Printf("[ROLLBACK][ERROR] Service not found: %s\n", serviceName)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      "Service not found in database",
		})
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	log.Printf("[ROLLBACK] CICD type=%s repo=%s\n", cicdType, repo)

	// get actor for pipeline run
	actor := "unknown"
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		actor = claims.GithubLogin
	}

	// 🏗️ Create pipeline run
	runID, err := CreatePipelineRun(serviceName, req.Environment, actor, cicdType)
	if err != nil {
		log.Printf("[ROLLBACK][ERROR] Failed to create pipeline run: service=%s env=%s err=%v\n",
			serviceName, req.Environment, err)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("Failed to create pipeline run: %v", err),
		})
		http.Error(w, "failed to create pipeline run", http.StatusInternalServerError)
		return
	}
	log.Printf("[ROLLBACK] pipeline run created runID=%d service=%s env=%s\n", runID, serviceName, req.Environment)

	// 🚀 Trigger rollback via CICD
	switch cicdType {
	case "jenkins":
		log.Printf("[ROLLBACK] Triggering Jenkins rollback: service=%s env=%s version=%s runID=%d\n",
			serviceName, req.Environment, req.Version, runID)
		err = cicd.TriggerJenkinsRollback(serviceName, req.Environment, req.Version, runID)

	case "github":
		log.Printf("[ROLLBACK] Triggering GitHub rollback: repo=%s env=%s version=%s runID=%d\n",
			repo, req.Environment, req.Version, runID)
		err = cicd.TriggerGitHubRollback(repo, req.Environment, req.Version, runID)

	default:
		log.Printf("[ROLLBACK][WARN] Unsupported CICD type: %s\n", cicdType)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("Unsupported cicd type: %s", cicdType),
		})
		http.Error(w, "unsupported cicd type", http.StatusBadRequest)
		return
	}

	if err != nil {
		log.Printf("[ROLLBACK][ERROR] CICD trigger failed: runID=%d err=%v\n", runID, err)
		audit.Log(r, audit.Entry{
			Action:       "rollback",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  req.Environment,
			Status:       "failed",
			Details:      fmt.Sprintf("CICD trigger failed via %s runID=%d: %v", cicdType, runID, err),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// ✅ Success
	log.Printf("[ROLLBACK][SUCCESS] Rollback triggered: service=%s env=%s from=%s to=%s runID=%d\n",
		serviceName, req.Environment, currentVersion, req.Version, runID)

	audit.Log(r, audit.Entry{
		Action:       "rollback",
		ResourceType: "deployment",
		ResourceName: serviceName,
		Environment:  req.Environment,
		Status:       "success",
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