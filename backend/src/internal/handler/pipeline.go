package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"src/src/internal/cicd"
	"src/src/internal/db"
)

type PipelineStage struct {
	ID          int64  `json:"id"`
	StageName   string `json:"stageName"`
	StageOrder  int    `json:"stageOrder"`
	Status      string `json:"status"`
	StartedAt   string `json:"startedAt"`
	CompletedAt string `json:"completedAt"`
	Logs        string `json:"logs"`
}

type PipelineRun struct {
	ID            int64           `json:"id"`
	ServiceName   string          `json:"serviceName"`
	Environment   string          `json:"environment"`
	Status        string          `json:"status"`
	TriggeredBy   string          `json:"triggeredBy"`
	CICDType      string          `json:"cicdType"`
	ExternalRunID string          `json:"externalRunId"`
	StartedAt     string          `json:"startedAt"`
	CompletedAt   string          `json:"completedAt"`
	Stages        []PipelineStage `json:"stages"`
}

func defaultStages(cicdType string) []string {
	// Stage names must match exactly what the CI/CD pipeline sends
	// in its notifyStage() / notify-platform calls
	switch strings.ToLower(cicdType) {
	case "jenkins":
		return []string{
			"Checkout", "Setup", "Build", "Test",
			"Push Image", "Deploy", "Health Check",
		}
	default:
		return []string{
			"Checkout", "Setup", "Build", "Test",
			"Push Image", "Deploy", "Health Check",
		}
	}
}

// ── CreatePipelineRun ────────────────────────────────────────────
func CreatePipelineRun(serviceName, environment, triggeredBy, cicdType string) (int64, error) {
	res, err := db.DB.Exec(`
		INSERT INTO pipeline_runs (service_name, environment, status, triggered_by, cicd_type)
		VALUES (?, ?, 'pending', ?, ?)
	`, serviceName, environment, triggeredBy, cicdType)
	if err != nil {
		return 0, err
	}

	runID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	stages := defaultStages(cicdType)
	for i, name := range stages {
		_, err := db.DB.Exec(`
			INSERT INTO pipeline_stages (run_id, stage_name, stage_order, status)
			VALUES (?, ?, ?, 'pending')
		`, runID, name, i)
		if err != nil {
			log.Printf("[PIPELINE] failed to insert stage %s: %v", name, err)
		}
	}

	log.Printf("[PIPELINE] created run id=%d service=%s env=%s stages=%d",
		runID, serviceName, environment, len(stages))
	return runID, nil
}

// ── loadRun loads a full PipelineRun with stages from DB ─────────
func loadRun(runID string) (*PipelineRun, error) {
	var run PipelineRun
	err := db.DB.QueryRow(`
		SELECT id, service_name, environment, status,
		       triggered_by,
		       COALESCE(cicd_type,''),
		       COALESCE(external_run_id,''),
		       DATE_FORMAT(started_at,'%Y-%m-%dT%H:%i:%sZ'),
		       COALESCE(DATE_FORMAT(completed_at,'%Y-%m-%dT%H:%i:%sZ'),'')
		FROM pipeline_runs WHERE id = ?
	`, runID).Scan(
		&run.ID, &run.ServiceName, &run.Environment, &run.Status,
		&run.TriggeredBy, &run.CICDType, &run.ExternalRunID,
		&run.StartedAt, &run.CompletedAt,
	)
	if err != nil {
		return nil, err
	}

	rows, err := db.DB.Query(`
		SELECT id, stage_name, stage_order, status,
		       COALESCE(DATE_FORMAT(started_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       COALESCE(DATE_FORMAT(completed_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       COALESCE(logs,'')
		FROM pipeline_stages
		WHERE run_id = ?
		ORDER BY stage_order ASC
	`, run.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	run.Stages = []PipelineStage{}
	for rows.Next() {
		var s PipelineStage
		if err := rows.Scan(
			&s.ID, &s.StageName, &s.StageOrder, &s.Status,
			&s.StartedAt, &s.CompletedAt, &s.Logs,
		); err != nil {
			continue
		}
		run.Stages = append(run.Stages, s)
	}
	return &run, nil
}

// ── GET /pipeline/:runId — regular snapshot ───────────────────────
func GetPipelineRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runID := parts[len(parts)-1]

	run, err := loadRun(runID)
	if err != nil {
		log.Printf("[PIPELINE] run not found id=%s: %v", runID, err)
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// ── GET /pipeline/:runId/stream — SSE stream ─────────────────────
func StreamPipelineRun(w http.ResponseWriter, r *http.Request) {
	// extract runID from /pipeline/{runId}/stream
	parts    := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runIDStr := parts[len(parts)-2]
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	// Check run exists
	run, err := loadRun(runIDStr)
	if err != nil {
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	// SSE headers
	w.Header().Set("Content-Type",                "text/event-stream")
	w.Header().Set("Cache-Control",               "no-cache")
	w.Header().Set("Connection",                  "keep-alive")
	w.Header().Set("X-Accel-Buffering",           "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	log.Printf("[SSE] client connected runID=%d remoteAddr=%s", runID, r.RemoteAddr)

	// Send current snapshot immediately so UI renders right away
	initialData, _ := json.Marshal(cicd.Event{
		Type:    "run_snapshot",
		Payload: run,
	})
	fmt.Fprintf(w, "event: run_snapshot\ndata: %s\n\n", initialData)
	flusher.Flush()

	// If already terminal — send completed event and close
	if run.Status == "success" || run.Status == "failed" || run.Status == "cancelled" {
		completedData, _ := json.Marshal(cicd.Event{
			Type: cicd.EventRunCompleted,
			Payload: cicd.RunPayload{
				ID:          run.ID,
				ServiceName: run.ServiceName,
				Environment: run.Environment,
				Status:      run.Status,
				StartedAt:   run.StartedAt,
				CompletedAt: run.CompletedAt,
			},
		})
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", cicd.EventRunCompleted, completedData)
		flusher.Flush()
		return
	}

	// Subscribe to live events
	ch := cicd.GlobalHub.Subscribe(runID)
	defer cicd.GlobalHub.Unsubscribe(runID, ch)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SSE] client disconnected runID=%d", runID)
			return

		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprint(w, msg)
			flusher.Flush()

			// Check if run is now terminal — close stream
			if strings.Contains(msg, cicd.EventRunCompleted) {
				log.Printf("[SSE] run completed, closing stream runID=%d", runID)
				return
			}
		}
	}
}

// ── GET /pipeline/service/:serviceName/:env — latest run ─────────
func GetLatestPipelineRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts       := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	environment := parts[len(parts)-1]
	serviceName := parts[len(parts)-2]

	var runID int64
	err := db.DB.QueryRow(`
		SELECT id FROM pipeline_runs
		WHERE service_name = ? AND environment = ?
		ORDER BY started_at DESC LIMIT 1
	`, serviceName, environment).Scan(&runID)
	if err != nil {
		http.Error(w, "no pipeline runs found", http.StatusNotFound)
		return
	}

	run, err := loadRun(fmt.Sprintf("%d", runID))
	if err != nil {
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// ── POST /pipeline/:runId/stage — CI/CD updates a stage ──────────
func UpdatePipelineStage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts    := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runIDStr := parts[len(parts)-2]
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	var body struct {
		StageName string `json:"stageName"`
		Status    string `json:"status"`
		Logs      string `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.StageName == "" || body.Status == "" {
		http.Error(w, "stageName and status required", http.StatusBadRequest)
		return
	}

	// Update stage in DB
	var query string
	switch body.Status {
	case "running":
		query = `UPDATE pipeline_stages SET status=?, logs=?, started_at=NOW()
		          WHERE run_id=? AND stage_name=?`
	case "success", "failed", "skipped":
		query = `UPDATE pipeline_stages SET status=?, logs=?, completed_at=NOW()
		          WHERE run_id=? AND stage_name=?`
	default:
		query = `UPDATE pipeline_stages SET status=?, logs=?
		          WHERE run_id=? AND stage_name=?`
	}
	_, err = db.DB.Exec(query, body.Status, body.Logs, runID, body.StageName)
	if err != nil {
		log.Printf("[PIPELINE] stage update error: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// Fetch updated stage for SSE broadcast
	var stage PipelineStage
	db.DB.QueryRow(`
		SELECT id, stage_name, stage_order, status,
		       COALESCE(DATE_FORMAT(started_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       COALESCE(DATE_FORMAT(completed_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       COALESCE(logs,'')
		FROM pipeline_stages
		WHERE run_id=? AND stage_name=?
	`, runID, body.StageName).Scan(
		&stage.ID, &stage.StageName, &stage.StageOrder, &stage.Status,
		&stage.StartedAt, &stage.CompletedAt, &stage.Logs,
	)

	// Broadcast stage update to SSE subscribers
	cicd.GlobalHub.Broadcast(runID, cicd.Event{
		Type: cicd.EventStageUpdated,
		Payload: cicd.StagePayload{
			ID:          stage.ID,
			RunID:       runID,
			StageName:   stage.StageName,
			StageOrder:  stage.StageOrder,
			Status:      stage.Status,
			StartedAt:   stage.StartedAt,
			CompletedAt: stage.CompletedAt,
			Logs:        body.Logs,
		},
	})

	// Update overall run status
	runStatus := updateRunStatus(runIDStr)

	// Fetch updated run for broadcast
	var run PipelineRun
	var completedAt sql.NullString
	var startedAt string
	db.DB.QueryRow(`
		SELECT id, service_name, environment, status,
		       DATE_FORMAT(started_at,'%Y-%m-%dT%H:%i:%sZ'),
		       COALESCE(DATE_FORMAT(completed_at,'%Y-%m-%dT%H:%i:%sZ'),'')
		FROM pipeline_runs WHERE id=?
	`, runID).Scan(
		&run.ID, &run.ServiceName, &run.Environment, &run.Status,
		&startedAt, &completedAt,
	)

	// Broadcast run status update
	cicd.GlobalHub.Broadcast(runID, cicd.Event{
		Type: cicd.EventRunUpdated,
		Payload: cicd.RunPayload{
			ID:          runID,
			ServiceName: run.ServiceName,
			Environment: run.Environment,
			Status:      runStatus,
			StartedAt:   startedAt,
			CompletedAt: completedAt.String,
		},
	})

	// If terminal — broadcast completed event so SSE stream closes
	if runStatus == "success" || runStatus == "failed" {
		cicd.GlobalHub.Broadcast(runID, cicd.Event{
			Type: cicd.EventRunCompleted,
			Payload: cicd.RunPayload{
				ID:          runID,
				ServiceName: run.ServiceName,
				Environment: run.Environment,
				Status:      runStatus,
				StartedAt:   startedAt,
				CompletedAt: completedAt.String,
			},
		})
	}

	log.Printf("[PIPELINE] stage updated runId=%d stage=%s status=%s runStatus=%s",
		runID, body.StageName, body.Status, runStatus)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}

// ── updateRunStatus recalculates run status from all its stages ──
func updateRunStatus(runID string) string {
	var total, pending, running, failed, success, skipped int

	rows, err := db.DB.Query(`SELECT status FROM pipeline_stages WHERE run_id=?`, runID)
	if err != nil {
		return "running"
	}
	defer rows.Close()

	for rows.Next() {
		var s string
		rows.Scan(&s)
		total++
		switch s {
		case "pending":  pending++
		case "running":  running++
		case "failed":   failed++
		case "success":  success++
		case "skipped":  skipped++
		}
	}

	// A run is complete when every stage is in a terminal state
	// (success or skipped) with no failures
	var status string
	switch {
	case failed > 0:
		status = "failed"
	case running > 0:
		status = "running"
	case pending == total:
		status = "pending"
	case success+skipped == total:
		// All stages done — success even if some were skipped (rollback)
		status = "success"
	default:
		status = "running"
	}

	extra := ""
	if status == "success" || status == "failed" {
		extra = ", completed_at=NOW()"
	}
	db.DB.Exec(`UPDATE pipeline_runs SET status=?`+extra+` WHERE id=?`, status, runID)
	return status
}