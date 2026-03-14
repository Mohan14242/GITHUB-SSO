package service

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"src/src/internal/aws"
	"src/src/internal/cicd"
	"src/src/internal/db"
	"src/src/internal/git"
	"src/src/internal/model"
	"src/src/internal/templates"
)

var ErrServiceAlreadyExists = errors.New("service already exists")

// ============================================================
// CreateService – PRODUCTION-GRADE IMPLEMENTATION
// ============================================================
func CreateService(req model.CreateServiceRequest) (string, error) {
	log.Println("🚀 CreateService started:", req.ServiceName)

	// ============================================================
	// PHASE 1: DB RESERVATION (FAST, SAFE)
	// ============================================================
	ctxDB, cancelDB := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB()

	tx, err := db.DB.BeginTx(ctxDB, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRowContext(
		ctxDB,
		`SELECT EXISTS (SELECT 1 FROM services WHERE service_name = ?)`,
		req.ServiceName,
	).Scan(&exists)
	if err != nil {
		return "", err
	}

	if exists {
		log.Println("⚠️ Service already exists:", req.ServiceName)
		return "", ErrServiceAlreadyExists
	}

	// Reserve service row
	_, err = tx.ExecContext(
		ctxDB,
		`INSERT INTO services (service_name, status)
		 VALUES (?, 'creating')`,
		req.ServiceName,
	)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	log.Println("✅ Service reserved in DB")

	// ============================================================
	// PHASE 2: EXTERNAL PROVISIONING (NO DB TX)
	// ============================================================

	// 1️⃣ Fetch GitHub token
	log.Println("🔐 Fetching GitHub token")
	token, err := aws.GetGitToken("git-token")
	if err != nil {
		return "", err
	}

	// 2️⃣ GitHub owner
	org, err := git.GetOrgName()
	if err != nil {
		log.Printf("[GIT][CREATE-REPO][ERROR] failed to get org name: %v", err)
		return "", err
	}

	// 3️⃣ Repo existence check
	repoExists, err := git.RepoExistsInOrg(token, org, req.RepoName)
	if err != nil {
		return "", err
	}
	if repoExists {
		return "", errors.New("repository already exists")
	}
	// 4️⃣ Create repo
	log.Println("📦 Creating GitHub repo:", req.RepoName)
	repoURL, err := git.CreateRepo(token, req.RepoName)
	if err != nil {
		return "", err
	}

	// Cleanup on failure
	cleanupRepo := func() {
		log.Println("🗑️ Cleaning up GitHub repo:", req.RepoName)
		_ = git.DeleteRepo(token, req.RepoName)
	}

	repoPath := "/tmp/" + req.RepoName
	defer os.RemoveAll(repoPath)

	// 5️⃣ Apply golden template
	log.Println("📐 Applying golden template")
	err = templates.CreateServiceFromTemplate(
		templates.TemplateRequest{
			Language:   req.Runtime,
			Version:    req.TemplateVersion,
			CICD:       req.CICDType,
			DeployType: req.DeployType,
		},
		repoPath,
	)
	if err != nil {
		cleanupRepo()
		return "", err
	}

	// After template copy
	err = UpdateConfigJSON(repoPath, req.ServiceName, repoURL)
	if err != nil {
		cleanupRepo()
		return "", err
	}

	// 6️⃣ Push code
	log.Println("⬆️ Pushing code")
	err = git.PushRepo(token, req.RepoName, repoPath,"dev")
	if err != nil {
		cleanupRepo()
		return "", err
	}

	// 7️⃣ Jenkins (optional)
	var webhookToken string
	if req.CICDType == "jenkins" {
		log.Println("🏗️ Registering Jenkins job")

		webhookToken, err = cicd.RegisterJenkins(
			repoURL,
			req.ServiceName,
			req.EnableWebhook,
		)
		if err != nil {
			cleanupRepo()
			return "", err
		}
	}

	// ============================================================
	// PHASE 3: FINAL DB UPDATE + DEPLOYMENTS
	// ============================================================
	ctxDB2, cancelDB2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelDB2()

	tx2, err := db.DB.BeginTx(ctxDB2, nil)
	if err != nil {
		return "", err
	}
	defer tx2.Rollback()

	_, err = tx2.ExecContext(
		ctxDB2,
		`UPDATE services
		 SET repo_url=?,
		     repo_name=?,
		     owner_team=?,
		     runtime=?,
		     cicd_type=?,
		     template_version=?,
		     deploy_type=?,
		     environments=?,
		     enablewebhook=?,
		     webhook_token=?,
		     status='ready'
		 WHERE service_name=?`,
		repoURL,
		req.RepoName,
		req.OwnerTeam,
		req.Runtime,
		req.CICDType,
		req.TemplateVersion,
		req.DeployType,
		mustJSON(req.Environments),
		req.EnableWebhook,
		webhookToken,
		req.ServiceName,
	)
	if err != nil {
		return "", err
	}

	// 🔥 Correct way to fetch service_id
	var serviceID int64
	err = tx2.QueryRowContext(
		ctxDB2,
		`SELECT id FROM services WHERE service_name = ?`,
		req.ServiceName,
	).Scan(&serviceID)
	if err != nil {
		return "", err
	}

	// Insert deployments
	stmt, err := tx2.PrepareContext(
		ctxDB2,
		`INSERT INTO deployments (service_id, environment, status)
		 VALUES (?, ?, ?)`,
	)
	if err != nil {
		return "", err
	}
	defer stmt.Close()

	for _, env := range req.Environments {
		_, err := stmt.ExecContext(ctxDB2, serviceID, env, "not_deployed")
		if err != nil {
			return "", err
		}
	}

	if err := tx2.Commit(); err != nil {
		return "", err
	}

	log.Println("🎉 CreateService completed successfully:", repoURL)
	return repoURL, nil
}

// ------------------------------------------------------------
// Helper
// ------------------------------------------------------------
func mustJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}