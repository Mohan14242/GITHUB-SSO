package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"src/src/internal/audit"
	"src/src/internal/auth"
	"src/src/internal/db"
	"src/src/internal/handler"
)

// ── seedTemplateVersions auto-registers folders found on disk ───
func seedTemplateVersions() {
	root, err := handler.GetTemplateRoot()
	if err != nil {
		log.Printf("[SEED] templateRoot not found, skipping seed: %v", err)
		return
	}

	runtimeDirs, err := os.ReadDir(root)
	if err != nil {
		log.Printf("[SEED] cannot read template_data: %v", err)
		return
	}

	count := 0
	for _, rd := range runtimeDirs {
		if !rd.IsDir() {
			continue
		}
		runtime := rd.Name()
		versionDirs, err := os.ReadDir(filepath.Join(root, runtime))
		if err != nil {
			continue
		}
		for _, vd := range versionDirs {
			if !vd.IsDir() {
				continue
			}
			version := vd.Name()
			name := runtime + "-service"

			_, err := db.DB.Exec(`
				INSERT IGNORE INTO template_versions
				  (name, version, runtime, description, status, created_by)
				VALUES (?, ?, ?, ?, 'active', 'system')
			`, name, version, runtime,
				fmt.Sprintf("Auto-seeded %s %s template", runtime, version))
			if err != nil {
				log.Printf("[SEED] failed to seed %s/%s: %v", runtime, version, err)
			} else {
				count++
				log.Printf("[SEED] ✅ seeded template %s@%s runtime=%s", name, version, runtime)
			}
		}
	}
	log.Printf("[SEED] template seed complete — %d entries processed", count)
}

func main() {
	db.InitMySQL()
	if err := db.EnsureSchema(); err != nil {
		log.Fatal("❌ Database schema initialization failed:", err)
	}

	// ✅ Auto-seed template versions from disk
	seedTemplateVersions()

	mux := http.NewServeMux()

	// ─────────────────────────────────────────
	// PUBLIC — no JWT required
	// ─────────────────────────────────────────
	mux.HandleFunc("/auth/login", auth.HandleLogin)
	mux.HandleFunc("/auth/callback", auth.HandleCallback)

	// ─────────────────────────────────────────
	// AUTHENTICATED — any valid JWT
	// ─────────────────────────────────────────
	mux.HandleFunc("/auth/me", auth.Authenticate(auth.HandleMe))

	// ─────────────────────────────────────────
	// READONLY+ — every logged-in user
	// ─────────────────────────────────────────
	mux.HandleFunc("/services", auth.RequireRole("readonly", handler.GetServices))
	mux.HandleFunc("/servicesdashboard/", auth.RequireRole("readonly", handler.GetServiceDashboard))
	mux.HandleFunc("/artifact-by-env/", auth.RequireRole("readonly", handler.GetServiceArtifacts))
	mux.HandleFunc("/service-by-env/", auth.RequireRole("readonly", handler.GetServiceEnvironments))

	// ─────────────────────────────────────────
	// DEVELOPER+ — developers, operators, admins
	// ─────────────────────────────────────────
	mux.HandleFunc("/create-service", auth.RequireRole("developer", handler.CreateService))
	mux.HandleFunc("/deploy-services/", auth.RequireRole("developer", handler.DeployServices))

	// ─────────────────────────────────────────
	// OPERATOR+ — sre, admins only
	// ─────────────────────────────────────────
	mux.HandleFunc("/rollback-services/", auth.RequireRole("operator", handler.RollbackService))
	mux.HandleFunc("/approvals", auth.RequireRole("operator", handler.GetApprovals))
	mux.HandleFunc("/approvals/", auth.RequireRole("operator", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			handler.ApproveDeployment(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/reject") {
			handler.RejectDeployment(w, r)
			return
		}
		http.NotFound(w, r)
	}))

	// ─────────────────────────────────────────
	// PIPELINE KEY — CI/CD callbacks only
	// ─────────────────────────────────────────
	mux.HandleFunc("/artifacts", auth.RequirePipelineKey(handler.RegisterArtifact))
	mux.HandleFunc("/stats", auth.RequireRole("readonly", handler.GetPlatformStats))
	mux.HandleFunc("/audit-logs", auth.RequireRole("operator", audit.GetAuditLogs))

	// ── Service Creation Requests ──
	mux.HandleFunc("/service-creation-requests", auth.RequireRole("operator", handler.GetServiceCreationRequests))
	mux.HandleFunc("/service-creation-requests/", auth.RequireRole("operator", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/approve") {
			handler.ApproveServiceCreation(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/reject") {
			handler.RejectServiceCreation(w, r)
			return
		}
		http.NotFound(w, r)
	}))

	// ── Template Versions ──
	mux.HandleFunc("/template-versions/scan", func(w http.ResponseWriter, r *http.Request) {
		auth.RequireRole("admin", handler.ScanTemplateVersions)(w, r)
	})
	mux.HandleFunc("/template-versions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			auth.RequireRole("admin", handler.CreateTemplateVersion)(w, r)
			return
		}
		auth.RequireRole("readonly", handler.GetTemplateVersions)(w, r)
	})
	mux.HandleFunc("/template-versions/", auth.RequireRole("admin", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/deprecate") {
			handler.DeprecateTemplateVersion(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/release") {
			handler.ReleaseTemplateVersion(w, r)
			return
		}
		http.NotFound(w, r)
	}))
	// Pipeline routes — order matters: specific before wildcard
	mux.HandleFunc("/pipeline/service/", auth.RequireRole("readonly", handler.GetLatestPipelineRun))

	mux.HandleFunc("/pipeline/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			auth.RequireRole("readonly", handler.StreamPipelineRun)(w, r)
			return
		}
		if strings.HasSuffix(r.URL.Path, "/stage") {
			auth.RequirePipelineKey(handler.UpdatePipelineStage)(w, r)
			return
		}
		auth.RequireRole("readonly", handler.GetPipelineRun)(w, r)
	})

	log.Println("🚀 Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", auth.WithCORS(mux)))
}
