package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"src/src/internal/audit"
	"src/src/internal/auth"
	"src/src/internal/db"
)

type TemplateVersion struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Runtime      string `json:"runtime"`
	Description  string `json:"description"`
	Changelog    string `json:"changelog"`
	Status       string `json:"status"`
	DeprecatedBy string `json:"deprecatedBy"`
	DeprecatedAt string `json:"deprecatedAt"`
	ReleasedBy   string `json:"releasedBy"`
	ReleasedAt   string `json:"releasedAt"`
	CreatedBy    string `json:"createdBy"`
	CreatedAt    string `json:"createdAt"`
	ExistsOnDisk bool   `json:"existsOnDisk"`
}

// ── templateRoot resolves the template_data directory on disk ───
func GetTemplateRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	candidates := []string{
		filepath.Join(wd, "src", "src", "internal", "template_data"), // ← your actual path
		filepath.Join(wd, "src", "internal", "template_data"),
		filepath.Join(wd, "internal", "template_data"),
		filepath.Join(wd, "..", "src", "src", "internal", "template_data"),
		filepath.Join(wd, "..", "src", "internal", "template_data"),
		filepath.Join(wd, "backend", "src", "src", "internal", "template_data"),
		filepath.Join(wd, "backend", "src", "internal", "template_data"),
	}

	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			log.Printf("[TEMPLATES] ✅ templateRoot resolved: %s", path)
			return path, nil
		}
	}

	log.Printf("[TEMPLATES] ❌ templateRoot not found. cwd=%s tried=%v", wd, candidates)
	return "", fmt.Errorf("template_data directory not found (cwd=%s)", wd)
}

// ── checkTemplateExists verifies runtime/version folder on disk ─
func checkTemplateExists(runtime, version string) bool {
	root, err := GetTemplateRoot()
	if err != nil {
		log.Printf("[TEMPLATES] ❌ templateRoot error: %v", err)
		return false
	}

	path := filepath.Join(root, runtime, version)
	log.Printf("[TEMPLATES] checking path: %s", path)

	info, err := os.Stat(path)
	if err != nil {
		log.Printf("[TEMPLATES] ❌ not found: %s", path)
		return false
	}
	if !info.IsDir() {
		log.Printf("[TEMPLATES] ❌ exists but not a directory: %s", path)
		return false
	}

	log.Printf("[TEMPLATES] ✅ exists: %s", path)
	return true
}

// ── scanDiskVersions returns all runtime/version folders on disk ─
// returns map[runtime][]version
func scanDiskVersions() map[string][]string {
	result := map[string][]string{}

	root, err := GetTemplateRoot()
	if err != nil {
		log.Printf("[TEMPLATES] scanDiskVersions: templateRoot error: %v", err)
		return result
	}

	// List runtime directories
	runtimeDirs, err := os.ReadDir(root)
	if err != nil {
		log.Printf("[TEMPLATES] scanDiskVersions: cannot read root: %v", err)
		return result
	}

	for _, rd := range runtimeDirs {
		if !rd.IsDir() {
			continue
		}
		runtime := rd.Name()
		runtimePath := filepath.Join(root, runtime)

		versionDirs, err := os.ReadDir(runtimePath)
		if err != nil {
			continue
		}
		for _, vd := range versionDirs {
			if vd.IsDir() {
				result[runtime] = append(result[runtime], vd.Name())
			}
		}
	}

	log.Printf("[TEMPLATES] scanDiskVersions found: %+v", result)
	return result
}

// ── GET /template-versions ──────────────────────────────────────
func GetTemplateVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status  := strings.TrimSpace(r.URL.Query().Get("status"))
	runtime := strings.TrimSpace(r.URL.Query().Get("runtime"))

	// Scan disk so we can annotate each version with existsOnDisk
	diskVersions := scanDiskVersions()

	query := `
		SELECT id, name, version, runtime,
		       COALESCE(description,''), COALESCE(changelog,''),
		       status,
		       COALESCE(deprecated_by,''), COALESCE(DATE_FORMAT(deprecated_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       COALESCE(released_by,''),   COALESCE(DATE_FORMAT(released_at,'%Y-%m-%dT%H:%i:%sZ'),''),
		       created_by, DATE_FORMAT(created_at,'%Y-%m-%dT%H:%i:%sZ')
		FROM template_versions
		WHERE 1=1`
	args := []interface{}{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if runtime != "" {
		query += " AND runtime = ?"
		args = append(args, runtime)
	}
	query += " ORDER BY name, created_at DESC"

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		log.Printf("[TEMPLATES][ERROR] query: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	versions := []TemplateVersion{}
	for rows.Next() {
		var t TemplateVersion
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Version, &t.Runtime,
			&t.Description, &t.Changelog, &t.Status,
			&t.DeprecatedBy, &t.DeprecatedAt,
			&t.ReleasedBy, &t.ReleasedAt,
			&t.CreatedBy, &t.CreatedAt,
		); err != nil {
			log.Printf("[TEMPLATES][ERROR] scan: %v", err)
			continue
		}

		// Annotate with disk existence
		t.ExistsOnDisk = false
		if versionList, ok := diskVersions[t.Runtime]; ok {
			for _, v := range versionList {
				if v == t.Version {
					t.ExistsOnDisk = true
					break
				}
			}
		}

		versions = append(versions, t)
	}

	log.Printf("[TEMPLATES] returning %d versions from DB", len(versions))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(versions)
}

// ── GET /template-versions/scan ─────────────────────────────────
// Returns what actually exists on disk — useful for debugging
func ScanTemplateVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	root, err := GetTemplateRoot()
	if err != nil {
		http.Error(w, fmt.Sprintf("template_data not found: %v", err), http.StatusInternalServerError)
		return
	}

	diskVersions := scanDiskVersions()

	type diskEntry struct {
		Runtime  string   `json:"runtime"`
		Versions []string `json:"versions"`
	}
	entries := []diskEntry{}
	for rt, vs := range diskVersions {
		entries = append(entries, diskEntry{Runtime: rt, Versions: vs})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"root":     root,
		"runtimes": entries,
	})
}

// ── POST /template-versions ─────────────────────────────────────
func CreateTemplateVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var body struct {
		Name        string `json:"name"`
		Version     string `json:"version"`
		Runtime     string `json:"runtime"`
		Description string `json:"description"`
		Changelog   string `json:"changelog"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if body.Name == "" || body.Version == "" || body.Runtime == "" {
		http.Error(w, "name, version, runtime are required", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	actor := "system"
	if claims != nil {
		actor = claims.GithubLogin
	}

	// ✅ Check filesystem before inserting into DB
	if !checkTemplateExists(body.Runtime, body.Version) {
		log.Printf("[TEMPLATES][WARN] create blocked — path not found runtime=%s version=%s", body.Runtime, body.Version)
		http.Error(w, fmt.Sprintf(
			"cannot create: template path template_data/%s/%s does not exist on disk",
			body.Runtime, body.Version,
		), http.StatusUnprocessableEntity)
		return
	}

	_, err := db.DB.Exec(`
		INSERT INTO template_versions
		  (name, version, runtime, description, changelog, status, created_by)
		VALUES (?, ?, ?, ?, ?, 'active', ?)
	`, body.Name, body.Version, body.Runtime,
		body.Description, body.Changelog, actor)
	if err != nil {
		if strings.Contains(err.Error(), "Duplicate") {
			http.Error(w, "version already exists for this template", http.StatusConflict)
			return
		}
		log.Printf("[TEMPLATES][ERROR] insert: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}

	audit.Log(r, audit.Entry{
		Actor:        actor,
		Action:       "template_version_created",
		ResourceType: "template",
		ResourceName: fmt.Sprintf("%s@%s", body.Name, body.Version),
		Status:       "success",
		Details:      fmt.Sprintf("runtime=%s path=template_data/%s/%s created_by=%s", body.Runtime, body.Runtime, body.Version, actor),
	})

	log.Printf("[TEMPLATES] created %s@%s runtime=%s by %s", body.Name, body.Version, body.Runtime, actor)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(`{"status":"created"}`))
}

// ── POST /template-versions/{id}/deprecate ──────────────────────
func DeprecateTemplateVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := extractTemplateID(r.URL.Path, "/deprecate")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	actor := "system"
	if claims != nil {
		actor = claims.GithubLogin
	}

	var name, version, runtime string
	err := db.DB.QueryRow(`
		SELECT name, version, runtime FROM template_versions WHERE id = ?
	`, id).Scan(&name, &version, &runtime)
	if err != nil {
		log.Printf("[TEMPLATES][ERROR] deprecate — not found id=%s: %v", id, err)
		http.Error(w, "template version not found", http.StatusNotFound)
		return
	}

	log.Printf("[TEMPLATES] deprecate requested id=%s name=%s version=%s runtime=%s", id, name, version, runtime)

	// ✅ Check filesystem before deprecating
	if !checkTemplateExists(runtime, version) {
		log.Printf("[TEMPLATES][WARN] deprecate blocked — path not found runtime=%s version=%s", runtime, version)
		http.Error(w, fmt.Sprintf(
			"cannot deprecate: template path template_data/%s/%s does not exist on disk",
			runtime, version,
		), http.StatusUnprocessableEntity)
		return
	}

	res, err := db.DB.Exec(`
		UPDATE template_versions
		SET status = 'deprecated', deprecated_by = ?, deprecated_at = NOW()
		WHERE id = ? AND status = 'active'
	`, actor, id)
	if err != nil {
		log.Printf("[TEMPLATES][ERROR] deprecate update: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "template version not found or already deprecated", http.StatusNotFound)
		return
	}

	audit.Log(r, audit.Entry{
		Actor:        actor,
		Action:       "template_version_deprecated",
		ResourceType: "template",
		ResourceName: fmt.Sprintf("%s@%s", name, version),
		Status:       "success",
		Details:      fmt.Sprintf("runtime=%s path=template_data/%s/%s deprecated_by=%s", runtime, runtime, version, actor),
	})

	log.Printf("[TEMPLATES] ✅ deprecated id=%s %s@%s runtime=%s by %s", id, name, version, runtime, actor)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"deprecated"}`))
}

// ── POST /template-versions/{id}/release ────────────────────────
func ReleaseTemplateVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := extractTemplateID(r.URL.Path, "/release")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	claims := auth.ClaimsFromContext(r.Context())
	actor := "system"
	if claims != nil {
		actor = claims.GithubLogin
	}

	var name, version, runtime string
	err := db.DB.QueryRow(`
		SELECT name, version, runtime FROM template_versions WHERE id = ?
	`, id).Scan(&name, &version, &runtime)
	if err != nil {
		log.Printf("[TEMPLATES][ERROR] release — not found id=%s: %v", id, err)
		http.Error(w, "template version not found", http.StatusNotFound)
		return
	}

	log.Printf("[TEMPLATES] release requested id=%s name=%s version=%s runtime=%s", id, name, version, runtime)

	// ✅ Check filesystem before re-releasing
	if !checkTemplateExists(runtime, version) {
		log.Printf("[TEMPLATES][WARN] release blocked — path not found runtime=%s version=%s", runtime, version)
		http.Error(w, fmt.Sprintf(
			"cannot release: template path template_data/%s/%s does not exist on disk",
			runtime, version,
		), http.StatusUnprocessableEntity)
		return
	}

	res, err := db.DB.Exec(`
		UPDATE template_versions
		SET status = 'active', released_by = ?, released_at = NOW(),
		    deprecated_by = NULL, deprecated_at = NULL
		WHERE id = ? AND status = 'deprecated'
	`, actor, id)
	if err != nil {
		log.Printf("[TEMPLATES][ERROR] release update: %v", err)
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		http.Error(w, "template version not found or already active", http.StatusNotFound)
		return
	}

	audit.Log(r, audit.Entry{
		Actor:        actor,
		Action:       "template_version_released",
		ResourceType: "template",
		ResourceName: fmt.Sprintf("%s@%s", name, version),
		Status:       "success",
		Details:      fmt.Sprintf("runtime=%s path=template_data/%s/%s re-released_by=%s", runtime, runtime, version, actor),
	})

	log.Printf("[TEMPLATES] ✅ released id=%s %s@%s runtime=%s by %s", id, name, version, runtime, actor)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"released"}`))
}

// ── extractTemplateID strips suffix and returns the ID segment ──
func extractTemplateID(path, suffix string) string {
	path = strings.TrimSuffix(path, suffix)
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}