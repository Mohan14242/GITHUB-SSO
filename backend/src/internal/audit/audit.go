package audit

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"src/src/internal/db"
)

type AuditLog struct {
	ID           int64  `json:"id"`
	Action       string `json:"action"`
	Actor        string `json:"actor"`
	ResourceType string `json:"resourceType"`
	ResourceName string `json:"resourceName"`
	Environment  string `json:"environment"`
	Status       string `json:"status"`
	Details      string `json:"details"`
	IPAddress    string `json:"ipAddress"`
	CreatedAt    string `json:"createdAt"`
}

type AuditResponse struct {
	Logs  []AuditLog `json:"logs"`
	Total int        `json:"total"`
	Page  int        `json:"page"`
}

func GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	log.Println("[AUDIT-HANDLER] Fetching audit logs")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	q := r.URL.Query()
	actor        := strings.TrimSpace(q.Get("actor"))
	environment  := strings.TrimSpace(q.Get("environment"))
	resourceName := strings.TrimSpace(q.Get("resourceName"))
	action       := strings.TrimSpace(q.Get("action"))
	status       := strings.TrimSpace(q.Get("status"))
	from         := strings.TrimSpace(q.Get("from"))
	to           := strings.TrimSpace(q.Get("to"))

	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * 100

	// ── build WHERE ──
	where := " WHERE 1=1"
	args  := []interface{}{}

	if actor != "" {
		where += " AND actor LIKE ?"
		args = append(args, "%"+actor+"%")
	}
	if environment != "" {
		where += " AND environment = ?"
		args = append(args, environment)
	}
	if resourceName != "" {
		where += " AND resource_name LIKE ?"
		args = append(args, "%"+resourceName+"%")
	}
	if action != "" {
		actions := strings.Split(action, ",")
		placeholders := make([]string, len(actions))
		for i, a := range actions {
			placeholders[i] = "?"
			args = append(args, strings.TrimSpace(a))
		}
		where += " AND action IN (" + strings.Join(placeholders, ",") + ")"
	}
	if status != "" {
		where += " AND status = ?"
		args = append(args, status)
	}
	if from != "" {
		where += " AND created_at >= ?"
		args = append(args, from)
	}
	if to != "" {
		where += " AND created_at <= ?"
		args = append(args, to+" 23:59:59")
	}

	// ── total count ──
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := db.DB.QueryRow("SELECT COUNT(*) FROM audit_logs"+where, countArgs...).Scan(&total)
	if err != nil {
		log.Printf("[AUDIT-HANDLER][ERROR] count query failed: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	// ── paginated rows ──
	query := `
		SELECT id, action, actor, resource_type, resource_name,
		       environment, status, details, ip_address,
		       DATE_FORMAT(created_at, '%Y-%m-%dT%H:%i:%sZ')
		FROM audit_logs` + where + ` ORDER BY created_at DESC LIMIT 100 OFFSET ?`
	args = append(args, offset)

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("[AUDIT-HANDLER][ERROR] query failed: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	logs := []AuditLog{}
	for rows.Next() {
		var l AuditLog
		if err := rows.Scan(
			&l.ID, &l.Action, &l.Actor, &l.ResourceType,
			&l.ResourceName, &l.Environment, &l.Status,
			&l.Details, &l.IPAddress, &l.CreatedAt,
		); err != nil {
			log.Printf("[AUDIT-HANDLER][ERROR] scan: %v", err)
			continue
		}
		logs = append(logs, l)
	}

	log.Printf("[AUDIT-HANDLER] returning %d/%d logs page=%d", len(logs), total, page)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AuditResponse{
		Logs:  logs,
		Total: total,
		Page:  page,
	})
}