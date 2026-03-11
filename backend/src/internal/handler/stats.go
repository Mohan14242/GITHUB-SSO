package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"src/src/internal/db"
)

type PlatformStats struct {
	TotalServices           int `json:"totalServices"`
	DeploymentsToday        int `json:"deploymentsToday"`
	PendingDeployments      int `json:"pendingDeployments"`
	PendingServiceCreations int `json:"pendingServiceCreations"`
	ActivePipelines         int `json:"activePipelines"`
}

// ── 60-second in-memory cache ──
var (
	cachedStats PlatformStats
	cacheExpiry time.Time
	cacheMu     sync.Mutex
)

func GetPlatformStats(w http.ResponseWriter, r *http.Request) {
	log.Println("[STATS] Fetching platform stats")

	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cacheMu.Lock()
	defer cacheMu.Unlock()

	// Return cached value if still fresh
	if time.Now().Before(cacheExpiry) {
		log.Println("[STATS] Returning cached stats")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cachedStats)
		return
	}

	var stats PlatformStats

	// 1️⃣ Total services
	err := db.DB.QueryRow(`
		SELECT COUNT(*) FROM services WHERE status = 'ready'
	`).Scan(&stats.TotalServices)
	if err != nil {
		log.Printf("[STATS][ERROR] Failed to count services: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	log.Printf("[STATS] totalServices=%d", stats.TotalServices)

	// 2️⃣ Deployments today
	today := time.Now().Format("2006-01-02")
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM artifacts
		WHERE DATE(created_at) = ? AND action = 'deploy'
	`, today).Scan(&stats.DeploymentsToday)
	if err != nil {
		log.Printf("[STATS][ERROR] Failed to count deployments today: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	log.Printf("[STATS] deploymentsToday=%d", stats.DeploymentsToday)

	// 3️⃣ Pending deployment approvals
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM deployment_approvals WHERE status = 'pending'
	`).Scan(&stats.PendingDeployments)
	if err != nil {
		log.Printf("[STATS][ERROR] Failed to count pending deployment approvals: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	log.Printf("[STATS] pendingDeployments=%d", stats.PendingDeployments)

	// 4️⃣ Pending service creation requests
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM service_creation_requests WHERE status = 'pending'
	`).Scan(&stats.PendingServiceCreations)
	if err != nil {
		log.Printf("[STATS][ERROR] Failed to count pending service creation requests: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	log.Printf("[STATS] pendingServiceCreations=%d", stats.PendingServiceCreations)

	// 5️⃣ Active pipelines
	err = db.DB.QueryRow(`
		SELECT COUNT(*) FROM environment_state
		WHERE status = 'deploying'
		AND deployed_at >= NOW() - INTERVAL 30 MINUTE
	`).Scan(&stats.ActivePipelines)
	if err != nil {
		log.Printf("[STATS][ERROR] Failed to count active pipelines: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	log.Printf("[STATS] activePipelines=%d", stats.ActivePipelines)

	// Update cache
	cachedStats = stats
	cacheExpiry = time.Now().Add(60 * time.Second)

	log.Printf("[STATS][SUCCESS] stats=%+v (cache refreshed)", stats)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
