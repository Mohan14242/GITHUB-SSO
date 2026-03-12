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
	log.Printf("[PIPELINE] CreatePipelineRun called serviceName=%s environment=%s triggeredBy=%s cicdType=%s",
		serviceName, environment, triggeredBy, cicdType)

	res, err := db.DB.Exec(`
		INSERT INTO pipeline_runs (service_name, environment, status, triggered_by, cicd_type)
		VALUES (?, ?, 'pending', ?, ?)
	`, serviceName, environment, triggeredBy, cicdType)
	if err != nil {
		log.Printf("[PIPELINE] CreatePipelineRun insert error: %v", err)
		return 0, err
	}

	runID, err := res.LastInsertId()
	if err != nil {
		log.Printf("[PIPELINE] CreatePipelineRun LastInsertId error: %v", err)
		return 0, err
	}

	log.Printf("[PIPELINE] run row inserted id=%d", runID)

	stages := defaultStages(cicdType)
	for i, name := range stages {
		_, err := db.DB.Exec(`
			INSERT INTO pipeline_stages (run_id, stage_name, stage_order, status)
			VALUES (?, ?, ?, 'pending')
		`, runID, name, i)
		if err != nil {
			log.Printf("[PIPELINE] failed to insert stage %s: %v", name, err)
		} else {
			log.Printf("[PIPELINE] stage inserted runId=%d order=%d name=%s", runID, i, name)
		}
	}

	log.Printf("[PIPELINE] created run id=%d service=%s env=%s stages=%d",
		runID, serviceName, environment, len(stages))
	return runID, nil
}

// ── loadRun loads a full PipelineRun with stages from DB ─────────
func loadRun(runID string) (*PipelineRun, error) {
	log.Printf("[PIPELINE] loadRun called runID=%s", runID)

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
		log.Printf("[PIPELINE] loadRun query error runID=%s: %v", runID, err)
		return nil, err
	}

	log.Printf("[PIPELINE] loadRun run found id=%d service=%s env=%s status=%s",
		run.ID, run.ServiceName, run.Environment, run.Status)

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
		log.Printf("[PIPELINE] loadRun stages query error runID=%s: %v", runID, err)
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
			log.Printf("[PIPELINE] loadRun stage scan error: %v", err)
			continue
		}
		log.Printf("[PIPELINE] loadRun stage id=%d name=%s status=%s", s.ID, s.StageName, s.Status)
		run.Stages = append(run.Stages, s)
	}

	log.Printf("[PIPELINE] loadRun complete runID=%s totalStages=%d", runID, len(run.Stages))
	return &run, nil
}

// ── GET /pipeline/:runId — regular snapshot ───────────────────────
func GetPipelineRun(w http.ResponseWriter, r *http.Request) {
	log.Printf("[PIPELINE] GetPipelineRun called path=%s", r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runID := parts[len(parts)-1]

	log.Printf("[--------------------------------------------------] GetPipelineRun runID=%s", runID)

	run, err := loadRun(runID)
	if err != nil {
		log.Printf("[PIPELINE] GetPipelineRun run not found id=%s: %v", runID, err)
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	log.Printf("[PIPELINE] GetPipelineRun responding runID=%s status=%s stages=%d",
		runID, run.Status, len(run.Stages))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// ── GET /pipeline/:runId/stream — SSE stream ─────────────────────
func StreamPipelineRun(w http.ResponseWriter, r *http.Request) {
	log.Printf("[SSE] StreamPipelineRun called path=%s remoteAddr=%s", r.URL.Path, r.RemoteAddr)

	parts    := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runIDStr := parts[len(parts)-2]
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		log.Printf("[SSE] invalid run id path=%s err=%v", r.URL.Path, err)
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	log.Printf("[SSE] loading run for stream runID=%d", runID)

	run, err := loadRun(runIDStr)
	if err != nil {
		log.Printf("[SSE] run not found runID=%d err=%v", runID, err)
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	log.Printf("[SSE] run loaded runID=%d status=%s stages=%d", runID, run.Status, len(run.Stages))

	// SSE headers
	w.Header().Set("Content-Type",                "text/event-stream")
	w.Header().Set("Cache-Control",               "no-cache")
	w.Header().Set("Connection",                  "keep-alive")
	w.Header().Set("X-Accel-Buffering",           "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Printf("[SSE] streaming not supported runID=%d", runID)
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
	log.Printf("[SSE] snapshot sent runID=%d bytes=%d", runID, len(initialData))

	// If already terminal — send completed event and close
	if run.Status == "success" || run.Status == "failed" || run.Status == "cancelled" {
		log.Printf("[SSE] run already terminal runID=%d status=%s — sending completed and closing", runID, run.Status)
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
	log.Printf("[SSE] subscribing to hub runID=%d", runID)
	ch := cicd.GlobalHub.Subscribe(runID)
	defer cicd.GlobalHub.Unsubscribe(runID, ch)

	ctx := r.Context()
	msgCount := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("[SSE] client disconnected runID=%d messagesDelivered=%d", runID, msgCount)
			return

		case msg, ok := <-ch:
			if !ok {
				log.Printf("[SSE] channel closed runID=%d", runID)
				return
			}
			msgCount++
			log.Printf("[SSE] broadcasting msg #%d runID=%d bytes=%d", msgCount, runID, len(msg))
			fmt.Fprint(w, msg)
			flusher.Flush()

			// Check if run is now terminal — close stream
			if strings.Contains(msg, cicd.EventRunCompleted) {
				log.Printf("[SSE] run completed, closing stream runID=%d totalMessages=%d", runID, msgCount)
				return
			}
		}
	}
}

// ── GET /pipeline/service/:serviceName/:env — latest run ─────────
func GetLatestPipelineRun(w http.ResponseWriter, r *http.Request) {
	log.Printf("[PIPELINE] GetLatestPipelineRun called path=%s", r.URL.Path)

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	parts       := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	environment := parts[len(parts)-1]
	serviceName := parts[len(parts)-2]

	log.Printf("[-----------------------------------------------------] GetLatestPipelineRun serviceName=%s environment=%s", serviceName, environment)

	var runID int64
	err := db.DB.QueryRow(`
		SELECT id FROM pipeline_runs
		WHERE service_name = ? AND environment = ?
		ORDER BY started_at DESC LIMIT 1
	`, serviceName, environment).Scan(&runID)
	if err != nil {
		log.Printf("[PIPELINE] GetLatestPipelineRun no runs found serviceName=%s env=%s: %v",
			serviceName, environment, err)
		http.Error(w, "no pipeline runs found", http.StatusNotFound)
		return
	}
    
	log.Printf("[PIPELINE] GetLatestPipelineRun found runID=%d", runID)

	run, err := loadRun(fmt.Sprintf("%d", runID))
	if err != nil {
		log.Printf("[PIPELINE] GetLatestPipelineRun loadRun error runID=%d: %v", runID, err)
		http.Error(w, "pipeline run not found", http.StatusNotFound)
		return
	}

	log.Printf("[PIPELINE] GetLatestPipelineRun responding runID=%d status=%s stages=%d",
		runID, run.Status, len(run.Stages))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}

// ── POST /pipeline/:runId/stage — CI/CD updates a stage ──────────
func UpdatePipelineStage(w http.ResponseWriter, r *http.Request) {
	log.Printf("[PIPELINE] UpdatePipelineStage called path=%s", r.URL.Path)

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	parts    := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	runIDStr := parts[len(parts)-2]
	runID, err := strconv.ParseInt(runIDStr, 10, 64)
	if err != nil {
		log.Printf("[PIPELINE] UpdatePipelineStage invalid run id path=%s err=%v", r.URL.Path, err)
		http.Error(w, "invalid run id", http.StatusBadRequest)
		return
	}

	var body struct {
		StageName string `json:"stageName"`
		Status    string `json:"status"`
		Logs      string `json:"logs"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		log.Printf("[PIPELINE] UpdatePipelineStage invalid JSON runID=%d: %v", runID, err)
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.StageName == "" || body.Status == "" {
		log.Printf("[PIPELINE] UpdatePipelineStage missing fields runID=%d stageName=%q status=%q",
			runID, body.StageName, body.Status)
		http.Error(w, "stageName and status required", http.StatusBadRequest)
		return
	}

	log.Printf("[PIPELINE] UpdatePipelineStage runID=%d stageName=%s status=%s logsLen=%d",
		runID, body.StageName, body.Status, len(body.Logs))

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
	result, err := db.DB.Exec(query, body.Status, body.Logs, runID, body.StageName)
	if err != nil {
		log.Printf("[PIPELINE] UpdatePipelineStage db exec error runID=%d stage=%s: %v",
			runID, body.StageName, err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	rowsAffected, _ := result.RowsAffected()
	log.Printf("[PIPELINE] UpdatePipelineStage db updated runID=%d stage=%s rowsAffected=%d",
		runID, body.StageName, rowsAffected)

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
	log.Printf("[PIPELINE] UpdatePipelineStage stage fetched id=%d name=%s status=%s",
		stage.ID, stage.StageName, stage.Status)

	// Broadcast stage update to SSE subscribers
	log.Printf("[PIPELINE] broadcasting stage_updated runID=%d stage=%s", runID, body.StageName)
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
	log.Printf("[PIPELINE] run status recalculated runID=%d newStatus=%s", runID, runStatus)

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
	log.Printf("[PIPELINE] broadcasting run_updated runID=%d status=%s", runID, runStatus)
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
		log.Printf("[PIPELINE] run terminal — broadcasting run_completed runID=%d status=%s", runID, runStatus)
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

	log.Printf("[PIPELINE] stage updated complete runId=%d stage=%s status=%s runStatus=%s",
		runID, body.StageName, body.Status, runStatus)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"updated"}`))
}

// ── updateRunStatus recalculates run status from all its stages ──
func updateRunStatus(runID string) string {
	log.Printf("[PIPELINE] updateRunStatus called runID=%s", runID)

	var total, pending, running, failed, success, skipped int

	rows, err := db.DB.Query(`SELECT status FROM pipeline_stages WHERE run_id=?`, runID)
	if err != nil {
		log.Printf("[PIPELINE] updateRunStatus query error runID=%s: %v", runID, err)
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

	log.Printf("[PIPELINE] updateRunStatus runID=%s total=%d pending=%d running=%d success=%d failed=%d skipped=%d",
		runID, total, pending, running, success, failed, skipped)

	var status string
	switch {
	case failed > 0:
		status = "failed"
	case running > 0:
		status = "running"
	case pending == total:
		status = "pending"
	case success+skipped == total:
		status = "success"
	default:
		status = "running"
	}

	log.Printf("[PIPELINE] updateRunStatus result runID=%s status=%s", runID, status)

	extra := ""
	if status == "success" || status == "failed" {
		extra = ", completed_at=NOW()"
	}
	db.DB.Exec(`UPDATE pipeline_runs SET status=?`+extra+` WHERE id=?`, status, runID)
	return status
}