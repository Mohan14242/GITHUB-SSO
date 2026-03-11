package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"src/src/internal/audit"
	"src/src/internal/auth"
	"src/src/internal/cicd"
	"src/src/internal/db"
)

/* ===================== MODELS ===================== */

type Approval struct {
	ID          int64      `json:"id"`
	ServiceName string     `json:"serviceName"`
	Environment string     `json:"environment"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	ApprovedAt  *time.Time `json:"approvedAt,omitempty"`
}

/* ===================== GET APPROVALS ===================== */

func GetApprovals(w http.ResponseWriter, r *http.Request) {
	log.Println("[APPROVAL] Fetching approvals (pending + history)")

	env := r.URL.Query().Get("environment")
	if env == "" {
		http.Error(w, "environment is required", http.StatusBadRequest)
		return
	}

	rows, err := db.DB.Query(`
		SELECT id, service_name, environment, status, created_at, approved_at
		FROM deployment_approvals
		WHERE environment = ?
		ORDER BY created_at DESC
	`, env)

	if err != nil {
		log.Println("[APPROVAL][ERROR]", err)
		http.Error(w, "failed to fetch approvals", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var approvals []Approval
	for rows.Next() {
		var a Approval
		if err := rows.Scan(
			&a.ID, &a.ServiceName, &a.Environment,
			&a.Status, &a.CreatedAt, &a.ApprovedAt,
		); err != nil {
			log.Println("[APPROVAL][ERROR]", err)
			continue
		}
		approvals = append(approvals, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(approvals)
}

/* ===================== APPROVE ===================== */

func ApproveDeployment(w http.ResponseWriter, r *http.Request) {
	log.Println("[APPROVAL] Approve request received")

	id, err := extractApprovalID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid approval id", http.StatusBadRequest)
		return
	}

	// get actor for audit log
	actor := "unknown"
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		actor = claims.GithubLogin
	}

	tx, err := db.DB.Begin()
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	var serviceName, environment string
	err = tx.QueryRow(`
		SELECT service_name, environment
		FROM deployment_approvals
		WHERE id = ? AND status = 'pending'
	`, id).Scan(&serviceName, &environment)

	if err == sql.ErrNoRows {
		http.Error(w, "approval not found or already processed", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	log.Printf("[APPROVAL] Approving service=%s env=%s by=%s", serviceName, environment, actor)

	_, err = tx.Exec(`
		UPDATE deployment_approvals
		SET status='approved', approved_at=NOW()
		WHERE id=?
	`, id)
	if err != nil {
		http.Error(w, "failed to update approval", http.StatusInternalServerError)
		return
	}

	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit approval", http.StatusInternalServerError)
		return
	}

	// ── Audit log ──
	audit.Log(r, audit.Entry{
		Action:       "deployment_approved",
		ResourceType: "deployment",
		ResourceName: serviceName,
		Environment:  environment,
		Status:       "success",
		Details:      fmt.Sprintf("Deployment approved by %s for env=%s", actor, environment),
	})

	/* ===== Lookup CI/CD type before triggering ===== */
	var cicdType, repo string
	err = db.DB.QueryRow(`
		SELECT cicd_type, repo_name FROM services WHERE service_name = ?
	`, serviceName).Scan(&cicdType, &repo)

	if err != nil {
		http.Error(w, "service not found", http.StatusNotFound)
		return
	}

	/* ===== Create pipeline run ===== */
	runID, err := CreatePipelineRun(serviceName, environment, actor, cicdType)
	if err != nil {
		log.Printf("[APPROVAL][ERROR] failed to create pipeline run service=%s env=%s: %v",
			serviceName, environment, err)
		http.Error(w, "failed to create pipeline run", http.StatusInternalServerError)
		return
	}
	log.Printf("[APPROVAL] pipeline run created runID=%d service=%s env=%s", runID, serviceName, environment)

	/* ===== Trigger CI/CD after approval ===== */
	branch := "master"
	log.Printf("[APPROVAL] Triggering prod deployment via %s runID=%d", cicdType, runID)

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
		audit.Log(r, audit.Entry{
			Action:       "deployment_trigger_failed",
			ResourceType: "deployment",
			ResourceName: serviceName,
			Environment:  environment,
			Status:       "failed",
			Details:      fmt.Sprintf("CICD trigger failed after approval: %s", err.Error()),
		})
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "production deployment approved and triggered",
		"runId":   runID,
	})
}

/* ===================== REJECT ===================== */

func RejectDeployment(w http.ResponseWriter, r *http.Request) {
	log.Println("[APPROVAL] Reject request received")

	id, err := extractApprovalID(r.URL.Path)
	if err != nil {
		http.Error(w, "invalid approval id", http.StatusBadRequest)
		return
	}

	// get actor + optional reason from body
	actor := "unknown"
	if claims := auth.ClaimsFromContext(r.Context()); claims != nil {
		actor = claims.GithubLogin
	}

	var body struct {
		Reason string `json:"reason"`
	}
	json.NewDecoder(r.Body).Decode(&body)
	reason := body.Reason
	if reason == "" {
		reason = "No reason provided"
	}

	// fetch service name before update for audit log
	var serviceName, environment string
	err = db.DB.QueryRow(`
		SELECT service_name, environment
		FROM deployment_approvals
		WHERE id = ? AND status = 'pending'
	`, id).Scan(&serviceName, &environment)

	if err == sql.ErrNoRows {
		http.Error(w, "approval not found or already processed", http.StatusNotFound)
		return
	}
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	res, err := db.DB.Exec(`
		UPDATE deployment_approvals
		SET status='rejected', approved_at=NOW()
		WHERE id=? AND status='pending'
	`, id)

	if err != nil {
		http.Error(w, "failed to reject approval", http.StatusInternalServerError)
		return
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		http.Error(w, "approval not found or already processed", http.StatusNotFound)
		return
	}

	log.Printf("[APPROVAL] Rejected service=%s env=%s by=%s reason=%s",
		serviceName, environment, actor, reason)

	// ── Audit log ──
	audit.Log(r, audit.Entry{
		Action:       "deployment_rejected",
		ResourceType: "deployment",
		ResourceName: serviceName,
		Environment:  environment,
		Status:       "rejected",
		Details:      fmt.Sprintf("Rejected by %s. Reason: %s", actor, reason),
	})

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message":"production deployment rejected"}`))
}

/* ===================== HELPERS ===================== */

func extractApprovalID(path string) (int64, error) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	idStr := parts[len(parts)-2]
	return strconv.ParseInt(idStr, 10, 64)
}