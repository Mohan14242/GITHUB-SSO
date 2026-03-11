package audit

import (
	"log"
	"net"
	"net/http"
	"strings"
	"src/src/internal/db"
)

type Entry struct {
	Actor        string
	Action       string
	ResourceType string
	ResourceName string
	Environment  string
	Status       string
	Details      string
}

func getIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return strings.TrimSpace(strings.Split(ip, ",")[0])
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func Log(r *http.Request, e Entry) {
    actor := e.Actor        // ← just use what caller passed in
    if actor == "" {
        actor = "system"
    }

    ip := getIP(r)

    _, err := db.DB.Exec(`
        INSERT INTO audit_logs
          (action, actor, resource_type, resource_name, environment, status, details, ip_address)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
    `, e.Action, actor, e.ResourceType, e.ResourceName,
        e.Environment, e.Status, e.Details, ip)

    if err != nil {
        log.Printf("[AUDIT][ERROR] Failed to write audit log: %v", err)
    } else {
        log.Printf("[AUDIT] action=%s actor=%s resource=%s/%s env=%s status=%s",
            e.Action, actor, e.ResourceType, e.ResourceName, e.Environment, e.Status)
    }
}
