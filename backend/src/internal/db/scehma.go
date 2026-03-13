package db

import "log"

func EnsureSchema() error {
	log.Println("🧱 Ensuring database schema exists")

	/* ===================== SERVICES ===================== */
	servicesTable := `
	CREATE TABLE IF NOT EXISTS services (
		id               BIGINT AUTO_INCREMENT PRIMARY KEY,
		service_name     VARCHAR(150) NOT NULL UNIQUE,
		status           VARCHAR(30)  NOT NULL DEFAULT 'creating',
		last_error       TEXT         NULL,
		provisioned_at   TIMESTAMP    NULL,
		repo_url         VARCHAR(255) NULL,
		repo_name        VARCHAR(255) NULL,
		webhook_token    VARCHAR(64)  NULL,
		owner_team       VARCHAR(100) NULL,
		runtime          VARCHAR(50)  NULL,
		cicd_type        VARCHAR(50)  NULL,
		template_version VARCHAR(50)  NULL,
		deploy_type      VARCHAR(50)  NULL,
		environments     JSON         NULL,
		enablewebhook    BOOLEAN      NOT NULL DEFAULT FALSE,
		created_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		updated_at       TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
			ON UPDATE CURRENT_TIMESTAMP
	);`

	/* ===================== DEPLOYMENTS ===================== */
	deploymentsTable := `
	CREATE TABLE IF NOT EXISTS deployments (
		id               BIGINT      AUTO_INCREMENT PRIMARY KEY,
		service_id       BIGINT      NOT NULL,
		environment      VARCHAR(20) NOT NULL,
		status           VARCHAR(20) NOT NULL DEFAULT 'not_deployed',
		last_deployed_at TIMESTAMP   NULL,
		updated_at       TIMESTAMP   DEFAULT CURRENT_TIMESTAMP
			ON UPDATE CURRENT_TIMESTAMP,
		UNIQUE KEY uniq_service_env (service_id, environment),
		FOREIGN KEY (service_id)
			REFERENCES services(id)
			ON DELETE CASCADE
	);`

	/* ===================== ARTIFACTS ===================== */
	artifactsTable := `
	CREATE TABLE IF NOT EXISTS artifacts (
		id            BIGINT       AUTO_INCREMENT PRIMARY KEY,
		service_name  VARCHAR(150) NOT NULL,
		environment   VARCHAR(20)  NOT NULL,
		version       VARCHAR(255) NOT NULL,
		artifact_type VARCHAR(20)  NOT NULL,
		commit_sha    VARCHAR(40)  NULL,
		pipeline      VARCHAR(30)  NULL,
		action        VARCHAR(20)  NOT NULL,
		created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_artifacts_service_env (service_name, environment),
		INDEX idx_artifacts_version     (version),
		INDEX idx_artifacts_created_at  (created_at)
	);`

	/* ===================== ENVIRONMENT STATE ===================== */
	environmentStateTable := `
	CREATE TABLE IF NOT EXISTS environment_state (
		service_name VARCHAR(150) NOT NULL,
		environment  VARCHAR(20)  NOT NULL,
		version      VARCHAR(255) NOT NULL,
		status       VARCHAR(20)  NOT NULL DEFAULT 'success',
		deployed_at  TIMESTAMP    NOT NULL,
		PRIMARY KEY (service_name, environment),
		INDEX idx_env_state_service (service_name)
	);`

	/* ===================== DEPLOYMENT APPROVALS ===================== */
	approvalsTable := `
	CREATE TABLE IF NOT EXISTS deployment_approvals (
		id            BIGINT       AUTO_INCREMENT PRIMARY KEY,
		service_name  VARCHAR(150) NOT NULL,
		environment   VARCHAR(50)  NOT NULL,
		status        ENUM('pending','approved','rejected') NOT NULL DEFAULT 'pending',
		requested_by  VARCHAR(150) NULL,
		reviewed_by   VARCHAR(150) NULL,
		reject_reason TEXT         NULL,
		run_id        BIGINT       NULL,
		created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		approved_at   TIMESTAMP    NULL,
		INDEX idx_approvals_env_status (environment, status),
		INDEX idx_approvals_service    (service_name),
		INDEX idx_approvals_created_at (created_at)
	);`

	/* ===================== SERVICE CREATION REQUESTS ===================== */
	serviceCreationRequestsTable := `
	CREATE TABLE IF NOT EXISTS service_creation_requests (
		id            BIGINT       AUTO_INCREMENT PRIMARY KEY,
		service_name  VARCHAR(150) NOT NULL,
		requested_by  VARCHAR(150) NOT NULL,
		yaml_payload  TEXT         NOT NULL,
		status        ENUM('pending','approved','rejected') NOT NULL DEFAULT 'pending',
		reviewed_by   VARCHAR(150) NULL,
		reject_reason TEXT         NULL,
		created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		reviewed_at   TIMESTAMP    NULL,
		INDEX idx_scr_status    (status),
		INDEX idx_scr_service   (service_name),
		INDEX idx_scr_requester (requested_by),
		INDEX idx_scr_created   (created_at)
	);`

	/* ===================== AUDIT LOGS ===================== */
	auditLogsTable := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id            BIGINT       AUTO_INCREMENT PRIMARY KEY,
		action        VARCHAR(100) NOT NULL,
		actor         VARCHAR(100) NOT NULL,
		resource_type VARCHAR(50)  NOT NULL,
		resource_name VARCHAR(200) NOT NULL,
		environment   VARCHAR(50)  NULL,
		status        VARCHAR(50)  NOT NULL,
		details       TEXT         NULL,
		ip_address    VARCHAR(50)  NULL,
		created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		INDEX idx_audit_actor_time    (actor, created_at),
		INDEX idx_audit_resource_time (resource_name, created_at),
		INDEX idx_audit_action        (action),
		INDEX idx_audit_environment   (environment),
		INDEX idx_audit_created_at    (created_at)
	);`

	/* ===================== TEMPLATE VERSIONS ===================== */
	templateVersionsTable := `
	CREATE TABLE IF NOT EXISTS template_versions (
		id            BIGINT       AUTO_INCREMENT PRIMARY KEY,
		name          VARCHAR(100) NOT NULL,
		version       VARCHAR(50)  NOT NULL,
		runtime       VARCHAR(50)  NOT NULL,
		description   TEXT         NULL,
		changelog     TEXT         NULL,
		status        ENUM('active','deprecated') NOT NULL DEFAULT 'active',
		deprecated_by VARCHAR(150) NULL,
		deprecated_at TIMESTAMP    NULL,
		released_by   VARCHAR(150) NULL,
		released_at   TIMESTAMP    NULL,
		created_by    VARCHAR(150) NOT NULL,
		created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		UNIQUE KEY uniq_name_version (name, version),
		INDEX idx_tv_status     (status),
		INDEX idx_tv_runtime    (runtime),
		INDEX idx_tv_created_at (created_at)
	);`

	/* ===================== PIPELINE RUNS ===================== */
	pipelineRunsTable := `
	CREATE TABLE IF NOT EXISTS pipeline_runs (
		id              BIGINT       AUTO_INCREMENT PRIMARY KEY,
		service_name    VARCHAR(150) NOT NULL,
		environment     VARCHAR(20)  NOT NULL,
		status          ENUM('pending','running','success','failed','cancelled')
		                             NOT NULL DEFAULT 'pending',
		triggered_by    VARCHAR(150) NOT NULL,
		cicd_type       VARCHAR(50)  NULL,
		external_run_id VARCHAR(100) NULL,
		started_at      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
		completed_at    TIMESTAMP    NULL,
		INDEX idx_pr_service_env (service_name, environment),
		INDEX idx_pr_status      (status),
		INDEX idx_pr_started     (started_at),
		INDEX idx_pr_triggered   (triggered_by)
	);`

	/* ===================== PIPELINE STAGES ===================== */
	pipelineStagesTable := `
	CREATE TABLE IF NOT EXISTS pipeline_stages (
		id           BIGINT       AUTO_INCREMENT PRIMARY KEY,
		run_id       BIGINT       NOT NULL,
		stage_name   VARCHAR(100) NOT NULL,
		stage_order  INT          NOT NULL DEFAULT 0,
		status       ENUM('pending','running','success','failed','skipped')
		                          NOT NULL DEFAULT 'pending',
		started_at   TIMESTAMP    NULL,
		completed_at TIMESTAMP    NULL,
		logs         TEXT         NULL,
		FOREIGN KEY (run_id)
			REFERENCES pipeline_runs(id)
			ON DELETE CASCADE,
		INDEX idx_ps_run_id    (run_id),
		INDEX idx_ps_run_order (run_id, stage_order)
	);`

	tables := []struct {
		name string
		sql  string
	}{
		{"services",                  servicesTable},
		{"deployments",               deploymentsTable},
		{"artifacts",                 artifactsTable},
		{"environment_state",         environmentStateTable},
		{"deployment_approvals",      approvalsTable},
		{"service_creation_requests", serviceCreationRequestsTable},
		{"audit_logs",                auditLogsTable},
		{"template_versions",         templateVersionsTable},
		{"pipeline_runs",             pipelineRunsTable},
		{"pipeline_stages",           pipelineStagesTable},
	}

	for _, t := range tables {
		if _, err := DB.Exec(t.sql); err != nil {
			log.Printf("❌ Failed to ensure %s table: %v\n", t.name, err)
			return err
		}
		log.Printf("✅ Table ready: %s", t.name)
	}

	/* ===================== INDEXES (MYSQL SAFE) ===================== */
	indexes := []string{
		`CREATE INDEX idx_services_owner_team ON services(owner_team);`,
		`CREATE INDEX idx_services_status ON services(status);`,
		`CREATE INDEX idx_services_created_at ON services(created_at);`,
		`CREATE INDEX idx_deployments_service_id ON deployments(service_id);`,
	}

	for _, idx := range indexes {
		if _, err := DB.Exec(idx); err != nil {
			log.Println("ℹ️ Index already exists or skipped:", err)
		}
	}

	/* ===================== MIGRATIONS (safe ALTER for existing DBs) ===================== */
	migrations := []string{
		`ALTER TABLE deployment_approvals ADD COLUMN IF NOT EXISTS run_id BIGINT NULL;`,
	}

	for _, m := range migrations {
		if _, err := DB.Exec(m); err != nil {
			log.Println("ℹ️ Migration skipped (column may already exist):", err)
		}
	}

	log.Println("✅ Database schema is ready (production-grade)")
	return nil
}
