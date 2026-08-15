package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"kwork-assistant/internal/domain"
	_ "modernc.org/sqlite"
)

type DB struct {
	*sql.DB
}

func InitDB(dbPath string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return &DB{db}, nil
}

func migrate(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS app_meta (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS ai_test_runs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			model TEXT NOT NULL,
			input TEXT NOT NULL,
			output TEXT,
			duration_ms INTEGER,
			success BOOLEAN,
			error TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			external_id INTEGER NOT NULL,
			source TEXT NOT NULL,
			url TEXT,
			title TEXT,
			description TEXT,
			budget_from REAL,
			budget_to REAL,
			currency TEXT,
			category_id INTEGER,
			category_name TEXT,
			buyer_id INTEGER,
			buyer_name TEXT,
			offers_count INTEGER,
			published_at DATETIME,
			fetched_at DATETIME,
			raw_json TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(source, external_id)
		);`,
		`CREATE TABLE IF NOT EXISTS project_evaluations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			category TEXT NOT NULL,
			score INTEGER NOT NULL,
			suitable BOOLEAN NOT NULL,
			complexity TEXT,
			budget_assessment TEXT,
			risk_level TEXT,
			estimated_effort TEXT,
			summary TEXT,
			reasons_json TEXT,
			warnings_json TEXT,
			model TEXT NOT NULL,
			prompt_version TEXT NOT NULL,
			input_hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(project_id) REFERENCES projects(id),
			UNIQUE(project_id, model, prompt_version, input_hash)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_pe_project_id ON project_evaluations(project_id);`,
		`CREATE INDEX IF NOT EXISTS idx_pe_category ON project_evaluations(category);`,
		`CREATE INDEX IF NOT EXISTS idx_pe_score ON project_evaluations(score);`,
		`CREATE TABLE IF NOT EXISTS proposal_drafts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			evaluation_id INTEGER NOT NULL,
			proposal TEXT NOT NULL,
			question TEXT,
			approach TEXT NOT NULL,
			confidence TEXT NOT NULL,
			warnings_json TEXT,
			model TEXT NOT NULL,
			prompt_version TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(project_id) REFERENCES projects(id),
			FOREIGN KEY(evaluation_id) REFERENCES project_evaluations(id),
			UNIQUE(evaluation_id, model, prompt_version)
		);`,
		`CREATE TABLE IF NOT EXISTS inbound_emails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_uid TEXT,
			message_id TEXT NOT NULL,
			sender TEXT,
			subject TEXT,
			text_body TEXT,
			html_body TEXT,
			received_at DATETIME,
			raw_hash TEXT,
			processing_status TEXT,
			processing_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(message_id)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_inbound_emails_status ON inbound_emails(processing_status);`,
		`CREATE TABLE IF NOT EXISTS telegram_notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id INTEGER NOT NULL,
			evaluation_id INTEGER,
			proposal_draft_id INTEGER,
			notification_type TEXT NOT NULL,
			chat_id INTEGER NOT NULL,
			telegram_message_id INTEGER,
			status TEXT NOT NULL,
			error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			sent_at DATETIME,
			FOREIGN KEY(project_id) REFERENCES projects(id),
			UNIQUE(project_id, notification_type)
		);`,
		`CREATE TABLE IF NOT EXISTS kwork_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			counterparty_user_id INTEGER NOT NULL,
			counterparty_username TEXT NOT NULL,
			counterparty_display_name TEXT,
			project_id INTEGER,
			last_external_message_id INTEGER,
			last_message_at DATETIME,
			unread_count INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(counterparty_user_id)
		);`,
		`CREATE TABLE IF NOT EXISTS kwork_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			external_message_id INTEGER,
			sender_user_id INTEGER NOT NULL,
			sender_username TEXT NOT NULL,
			direction TEXT NOT NULL,
			text TEXT NOT NULL,
			sent_at DATETIME NOT NULL,
			raw_hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(conversation_id) REFERENCES kwork_conversations(id),
			UNIQUE(external_message_id)
		);`,
		`CREATE TABLE IF NOT EXISTS reply_drafts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			context_message_id INTEGER,
			model_version TEXT NOT NULL,
			draft_text TEXT NOT NULL,
			status TEXT NOT NULL,
			sent_message_id INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(conversation_id) REFERENCES kwork_conversations(id),
			FOREIGN KEY(context_message_id) REFERENCES kwork_messages(id)
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	// MIGRATION: Add input_hash to project_evaluations if not exists
	var cols int
	err := db.QueryRow(`SELECT count(*) FROM pragma_table_info('project_evaluations') WHERE name='input_hash'`).Scan(&cols)
	if err == nil && cols == 0 {
		// input_hash column is missing, perform migration
		_, err = db.Exec(`
			PRAGMA foreign_keys=off;
			BEGIN TRANSACTION;
			ALTER TABLE project_evaluations RENAME TO project_evaluations_old;
			CREATE TABLE project_evaluations (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				project_id INTEGER NOT NULL,
				category TEXT NOT NULL,
				score INTEGER NOT NULL,
				suitable BOOLEAN NOT NULL,
				complexity TEXT,
				budget_assessment TEXT,
				risk_level TEXT,
				estimated_effort TEXT,
				summary TEXT,
				reasons_json TEXT,
				warnings_json TEXT,
				model TEXT NOT NULL,
				prompt_version TEXT NOT NULL,
				input_hash TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY(project_id) REFERENCES projects(id),
				UNIQUE(project_id, model, prompt_version, input_hash)
			);
			INSERT INTO project_evaluations (id, project_id, category, score, suitable, complexity, budget_assessment, risk_level, estimated_effort, summary, reasons_json, warnings_json, model, prompt_version, created_at, input_hash)
			SELECT id, project_id, category, score, suitable, complexity, budget_assessment, risk_level, estimated_effort, summary, reasons_json, warnings_json, model, prompt_version, created_at, NULL FROM project_evaluations_old;
			DROP TABLE project_evaluations_old;
			CREATE INDEX IF NOT EXISTS idx_pe_project_id ON project_evaluations(project_id);
			CREATE INDEX IF NOT EXISTS idx_pe_category ON project_evaluations(category);
			CREATE INDEX IF NOT EXISTS idx_pe_score ON project_evaluations(score);
			COMMIT;
			PRAGMA foreign_keys=on;
		`)
		if err != nil {
			return err
		}
	}

	return nil
}

type AITestRun struct {
	Model      string
	Input      string
	Output     string
	DurationMs int64
	Success    bool
	Error      string
}

func (db *DB) SaveAITestRun(ctx context.Context, run AITestRun) error {
	query := `
		INSERT INTO ai_test_runs (model, input, output, duration_ms, success, error)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.ExecContext(ctx, query,
		run.Model,
		run.Input,
		run.Output,
		run.DurationMs,
		run.Success,
		run.Error,
	)
	return err
}

func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

func (db *DB) UpsertProject(ctx context.Context, p domain.Project) (bool, error) {
	var exists bool
	err := db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM projects WHERE source = ? AND external_id = ?)", p.Source, p.ExternalID).Scan(&exists)
	if err != nil {
		return false, err
	}

	query := `
		INSERT INTO projects (
			external_id, source, url, title, description,
			budget_from, budget_to, currency, category_id, category_name,
			buyer_id, buyer_name, offers_count, published_at, fetched_at, raw_json
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?
		)
		ON CONFLICT(source, external_id) DO UPDATE SET
			url = excluded.url,
			title = excluded.title,
			description = excluded.description,
			budget_from = excluded.budget_from,
			budget_to = excluded.budget_to,
			currency = excluded.currency,
			category_id = excluded.category_id,
			category_name = excluded.category_name,
			buyer_id = excluded.buyer_id,
			buyer_name = excluded.buyer_name,
			offers_count = excluded.offers_count,
			published_at = excluded.published_at,
			fetched_at = excluded.fetched_at,
			raw_json = excluded.raw_json,
			updated_at = CURRENT_TIMESTAMP
	`

	res, err := db.ExecContext(ctx, query,
		p.ExternalID, p.Source, p.URL, p.Title, p.Description,
		p.BudgetFrom, p.BudgetTo, p.Currency, p.CategoryID, p.CategoryName,
		p.BuyerID, p.BuyerName, p.OffersCount, p.PublishedAt, p.FetchedAt, p.RawJSON,
	)
	if err != nil {
		return false, err
	}

	_, err = res.RowsAffected()
	if err != nil {
		return false, err
	}

	return !exists, nil
}

func (db *DB) IsProjectNew(ctx context.Context, source string, externalID int64) (bool, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM projects WHERE source = ? AND external_id = ?", source, externalID).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return true, nil
		}
		return false, err
	}
	return false, nil
}

func (db *DB) GetRecentProjects(ctx context.Context, limit int) ([]domain.Project, error) {
	query := `
		SELECT id, external_id, source, url, title, description,
		budget_from, budget_to, currency, category_id, category_name,
		buyer_id, buyer_name, offers_count, published_at, fetched_at, raw_json
		FROM projects
		ORDER BY id DESC LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		err := rows.Scan(
			&p.ID, &p.ExternalID, &p.Source, &p.URL, &p.Title, &p.Description,
			&p.BudgetFrom, &p.BudgetTo, &p.Currency, &p.CategoryID, &p.CategoryName,
			&p.BuyerID, &p.BuyerName, &p.OffersCount, &p.PublishedAt, &p.FetchedAt, &p.RawJSON,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *DB) GetProject(ctx context.Context, id int64) (*domain.Project, error) {
	query := `
		SELECT id, external_id, source, url, title, description,
		       budget_from, budget_to, currency, category_id, category_name,
		       buyer_id, buyer_name, offers_count, published_at, fetched_at, raw_json
		FROM projects
		WHERE id = ?
	`
	var p domain.Project
	err := db.QueryRowContext(ctx, query, id).Scan(
		&p.ID, &p.ExternalID, &p.Source, &p.URL, &p.Title, &p.Description,
		&p.BudgetFrom, &p.BudgetTo, &p.Currency, &p.CategoryID, &p.CategoryName,
		&p.BuyerID, &p.BuyerName, &p.OffersCount, &p.PublishedAt, &p.FetchedAt, &p.RawJSON,
	)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (db *DB) GetProjectByExternalID(ctx context.Context, source string, extID int64) (domain.Project, error) {
	query := `
		SELECT id, external_id, source, url, title, description,
		budget_from, budget_to, currency, category_id, category_name,
		buyer_id, buyer_name, offers_count, published_at, fetched_at, raw_json
		FROM projects
		WHERE source = ? AND external_id = ?
	`
	var p domain.Project
	err := db.QueryRowContext(ctx, query, source, extID).Scan(
		&p.ID, &p.ExternalID, &p.Source, &p.URL, &p.Title, &p.Description,
		&p.BudgetFrom, &p.BudgetTo, &p.Currency, &p.CategoryID, &p.CategoryName,
		&p.BuyerID, &p.BuyerName, &p.OffersCount, &p.PublishedAt, &p.FetchedAt, &p.RawJSON,
	)
	return p, err
}

func (db *DB) SaveEvaluation(ctx context.Context, eval domain.ProjectEvaluation) error {
	reasonsJSON, _ := json.Marshal(eval.Reasons)
	warningsJSON, _ := json.Marshal(eval.Warnings)

	query := `
		INSERT INTO project_evaluations (
			project_id, category, score, suitable, complexity,
			budget_assessment, risk_level, estimated_effort, summary,
			reasons_json, warnings_json, model, prompt_version, input_hash
		) VALUES (
			?, ?, ?, ?, ?,
			?, ?, ?, ?,
			?, ?, ?, ?, ?
		)
		ON CONFLICT(project_id, model, prompt_version, input_hash) DO UPDATE SET
			category = excluded.category,
			score = excluded.score,
			suitable = excluded.suitable,
			complexity = excluded.complexity,
			budget_assessment = excluded.budget_assessment,
			risk_level = excluded.risk_level,
			estimated_effort = excluded.estimated_effort,
			summary = excluded.summary,
			reasons_json = excluded.reasons_json,
			warnings_json = excluded.warnings_json
	`
	_, err := db.ExecContext(ctx, query,
		eval.ProjectID, eval.Category, eval.Score, eval.Suitable, eval.Complexity,
		eval.BudgetAssessment, eval.RiskLevel, eval.EstimatedEffort, eval.Summary,
		string(reasonsJSON), string(warningsJSON), eval.Model, eval.PromptVersion, eval.InputHash,
	)
	return err
}

func (db *DB) DeleteEvaluation(ctx context.Context, projectID int64, model, promptVersion string) error {
	query := `DELETE FROM project_evaluations WHERE project_id = ? AND model = ? AND prompt_version = ?`
	_, err := db.ExecContext(ctx, query, projectID, model, promptVersion)
	return err
}

func (db *DB) HasEvaluation(ctx context.Context, projectID int64, model, promptVersion string) (bool, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM project_evaluations WHERE project_id = ? AND model = ? AND prompt_version = ?", projectID, model, promptVersion).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (db *DB) GetUnevaluatedProjects(ctx context.Context, model, promptVersion string, limit int) ([]domain.Project, error) {
	query := `
		SELECT p.id, p.external_id, p.source, p.url, p.title, p.description,
		p.budget_from, p.budget_to, p.currency, p.category_id, p.category_name,
		p.buyer_id, p.buyer_name, p.offers_count, p.published_at, p.fetched_at, p.raw_json
		FROM projects p
		LEFT JOIN project_evaluations pe ON p.id = pe.project_id AND pe.model = ? AND pe.prompt_version = ?
		WHERE pe.id IS NULL
		ORDER BY p.id DESC LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, model, promptVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		err := rows.Scan(
			&p.ID, &p.ExternalID, &p.Source, &p.URL, &p.Title, &p.Description,
			&p.BudgetFrom, &p.BudgetTo, &p.Currency, &p.CategoryID, &p.CategoryName,
			&p.BuyerID, &p.BuyerName, &p.OffersCount, &p.PublishedAt, &p.FetchedAt, &p.RawJSON,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *DB) GetEvaluations(ctx context.Context, filter domain.EvaluationFilter) ([]domain.ProjectEvaluation, error) {
	query := `SELECT id, project_id, category, score, suitable, complexity, budget_assessment, risk_level, estimated_effort, summary, reasons_json, warnings_json, model, prompt_version, input_hash, created_at FROM project_evaluations WHERE 1=1`
	var args []interface{}

	if filter.Category != "" {
		query += ` AND category = ?`
		args = append(args, filter.Category)
	}
	if filter.MinScore > 0 {
		query += ` AND score >= ?`
		args = append(args, filter.MinScore)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evals []domain.ProjectEvaluation
	for rows.Next() {
		var e domain.ProjectEvaluation
		var reasonsStr, warningsStr string
		var inputHash sql.NullString
		err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Category, &e.Score, &e.Suitable, &e.Complexity,
			&e.BudgetAssessment, &e.RiskLevel, &e.EstimatedEffort, &e.Summary,
			&reasonsStr, &warningsStr, &e.Model, &e.PromptVersion, &inputHash, &e.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		e.InputHash = inputHash.String
		if reasonsStr != "" {
			json.Unmarshal([]byte(reasonsStr), &e.Reasons)
		}
		if warningsStr != "" {
			json.Unmarshal([]byte(warningsStr), &e.Warnings)
		}
		evals = append(evals, e)
	}
	return evals, rows.Err()
}

func (db *DB) GetEvaluationByExternalID(ctx context.Context, source string, extID int64) (*domain.ProjectEvaluation, error) {
	p, err := db.GetProjectByExternalID(ctx, source, extID)
	if err != nil {
		return nil, err
	}

	query := `
		SELECT id, project_id, category, score, suitable, complexity, budget_assessment, risk_level, estimated_effort, summary, reasons_json, warnings_json, model, prompt_version, input_hash, created_at
		FROM project_evaluations
		WHERE project_id = ?
		ORDER BY id DESC LIMIT 1
	`
	var e domain.ProjectEvaluation
	var reasonsStr, warningsStr string
	var inputHash sql.NullString
	err = db.QueryRowContext(ctx, query, p.ID).Scan(
		&e.ID, &e.ProjectID, &e.Category, &e.Score, &e.Suitable, &e.Complexity,
		&e.BudgetAssessment, &e.RiskLevel, &e.EstimatedEffort, &e.Summary,
		&reasonsStr, &warningsStr, &e.Model, &e.PromptVersion, &inputHash, &e.CreatedAt,
	)
	if err == nil {
		e.InputHash = inputHash.String
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Return nil if no evaluation found, rather than error
		}
		return nil, err
	}
	if reasonsStr != "" {
		json.Unmarshal([]byte(reasonsStr), &e.Reasons)
	}
	if warningsStr != "" {
		json.Unmarshal([]byte(warningsStr), &e.Warnings)
	}
	return &e, nil
}

func (db *DB) SaveProposalDraft(ctx context.Context, draft domain.ProposalDraft) error {
	warningsJSON, _ := json.Marshal(draft.Warnings)

	query := `
		INSERT INTO proposal_drafts (
			project_id, evaluation_id, proposal, question, approach,
			confidence, warnings_json, model, prompt_version
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(evaluation_id, model, prompt_version) DO UPDATE SET
			proposal = excluded.proposal,
			question = excluded.question,
			approach = excluded.approach,
			confidence = excluded.confidence,
			warnings_json = excluded.warnings_json
	`
	_, err := db.ExecContext(ctx, query,
		draft.ProjectID, draft.EvaluationID, draft.Proposal, draft.Question, draft.Approach,
		draft.Confidence, string(warningsJSON), draft.Model, draft.PromptVersion,
	)
	return err
}

func (db *DB) DeleteProposalDraft(ctx context.Context, evaluationID int64, model, promptVersion string) error {
	query := `DELETE FROM proposal_drafts WHERE evaluation_id = ? AND model = ? AND prompt_version = ?`
	_, err := db.ExecContext(ctx, query, evaluationID, model, promptVersion)
	return err
}

func (db *DB) HasProposalDraft(ctx context.Context, evaluationID int64, model, promptVersion string) (bool, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM proposal_drafts WHERE evaluation_id = ? AND model = ? AND prompt_version = ?", evaluationID, model, promptVersion).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (db *DB) GetProjectsForProposal(ctx context.Context, model, evalPromptVersion, proposalPromptVersion string, limit int) ([]domain.Project, error) {
	query := `
		SELECT p.id, p.external_id, p.source, p.url, p.title, p.description,
		p.budget_from, p.budget_to, p.currency, p.category_id, p.category_name,
		p.buyer_id, p.buyer_name, p.offers_count, p.published_at, p.fetched_at, p.raw_json
		FROM projects p
		INNER JOIN project_evaluations pe ON p.id = pe.project_id AND pe.model = ? AND pe.prompt_version = ?
		LEFT JOIN proposal_drafts pd ON pe.id = pd.evaluation_id AND pd.model = ? AND pd.prompt_version = ?
		WHERE pe.suitable = 1 
		  AND pe.score >= 60 
		  AND pe.category IN ('website', 'telegram')
		  AND pd.id IS NULL
		ORDER BY p.id DESC LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, model, evalPromptVersion, model, proposalPromptVersion, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []domain.Project
	for rows.Next() {
		var p domain.Project
		err := rows.Scan(
			&p.ID, &p.ExternalID, &p.Source, &p.URL, &p.Title, &p.Description,
			&p.BudgetFrom, &p.BudgetTo, &p.Currency, &p.CategoryID, &p.CategoryName,
			&p.BuyerID, &p.BuyerName, &p.OffersCount, &p.PublishedAt, &p.FetchedAt, &p.RawJSON,
		)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (db *DB) GetProposalDraftByExternalID(ctx context.Context, source string, extID int64) (*domain.ProposalDraft, error) {
	query := `
		SELECT pd.id, pd.project_id, pd.evaluation_id, pd.proposal, pd.question, pd.approach, pd.confidence, pd.warnings_json, pd.model, pd.prompt_version, pd.created_at
		FROM proposal_drafts pd
		INNER JOIN projects p ON pd.project_id = p.id
		WHERE p.source = ? AND p.external_id = ?
		ORDER BY pd.id DESC LIMIT 1
	`
	var d domain.ProposalDraft
	var warningsStr string
	err := db.QueryRowContext(ctx, query, source, extID).Scan(
		&d.ID, &d.ProjectID, &d.EvaluationID, &d.Proposal, &d.Question, &d.Approach,
		&d.Confidence, &warningsStr, &d.Model, &d.PromptVersion, &d.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if warningsStr != "" {
		json.Unmarshal([]byte(warningsStr), &d.Warnings)
	}
	return &d, nil
}

func (db *DB) GetProposalDrafts(ctx context.Context, filter domain.EvaluationFilter) ([]domain.ProposalDraft, error) {
	query := `
		SELECT pd.id, pd.project_id, pd.evaluation_id, pd.proposal, pd.question, pd.approach, pd.confidence, pd.warnings_json, pd.model, pd.prompt_version, pd.created_at
		FROM proposal_drafts pd
		INNER JOIN project_evaluations pe ON pd.evaluation_id = pe.id
		WHERE 1=1
	`
	var args []interface{}

	if filter.Category != "" {
		query += ` AND pe.category = ?`
		args = append(args, filter.Category)
	}
	if filter.MinScore > 0 {
		query += ` AND pe.score >= ?`
		args = append(args, filter.MinScore)
	}
	query += ` ORDER BY pd.id DESC LIMIT ?`

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drafts []domain.ProposalDraft
	for rows.Next() {
		var d domain.ProposalDraft
		var warningsStr string
		err := rows.Scan(
			&d.ID, &d.ProjectID, &d.EvaluationID, &d.Proposal, &d.Question, &d.Approach,
			&d.Confidence, &warningsStr, &d.Model, &d.PromptVersion, &d.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		if warningsStr != "" {
			json.Unmarshal([]byte(warningsStr), &d.Warnings)
		}
		drafts = append(drafts, d)
	}
	return drafts, rows.Err()
}

func (db *DB) GetAppMeta(ctx context.Context, key string) (string, error) {
	var value string
	err := db.QueryRowContext(ctx, "SELECT value FROM app_meta WHERE key = ?", key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (db *DB) SetAppMeta(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO app_meta (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.ExecContext(ctx, query, key, value)
	return err
}

func (db *DB) SaveInboundEmail(ctx context.Context, email *domain.InboundEmail) error {
	query := `
		INSERT INTO inbound_emails (
			provider_uid, message_id, sender, subject, text_body, html_body,
			received_at, raw_hash, processing_status, processing_error
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(message_id) DO NOTHING
	`
	res, err := db.ExecContext(ctx, query,
		email.ProviderUID, email.MessageID, email.Sender, email.Subject, email.TextBody, email.HTMLBody,
		email.ReceivedAt, email.RawHash, email.ProcessingStatus, email.ProcessingError,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	// If id is 0, it means conflict DO NOTHING happened, meaning the email was a duplicate.
	// We'll leave email.ID as 0 in this case.
	if id > 0 {
		email.ID = id
	}
	return nil
}

func (db *DB) GetInboundEmailByID(ctx context.Context, id int64) (*domain.InboundEmail, error) {
	query := `
		SELECT id, provider_uid, message_id, sender, subject, text_body, html_body,
		       received_at, raw_hash, processing_status, processing_error, created_at, updated_at
		FROM inbound_emails WHERE id = ?
	`
	var e domain.InboundEmail
	err := db.QueryRowContext(ctx, query, id).Scan(
		&e.ID, &e.ProviderUID, &e.MessageID, &e.Sender, &e.Subject, &e.TextBody, &e.HTMLBody,
		&e.ReceivedAt, &e.RawHash, &e.ProcessingStatus, &e.ProcessingError, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &e, nil
}

func (db *DB) GetPendingInboundEmails(ctx context.Context, limit int) ([]domain.InboundEmail, error) {
	query := `
		SELECT id, provider_uid, message_id, sender, subject, text_body, html_body,
		       received_at, raw_hash, processing_status, processing_error, created_at, updated_at
		FROM inbound_emails WHERE processing_status IN (?, ?)
		ORDER BY id ASC LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, domain.ProcessingStatusNew, domain.ProcessingStatusRetryableError, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []domain.InboundEmail
	for rows.Next() {
		var e domain.InboundEmail
		err := rows.Scan(
			&e.ID, &e.ProviderUID, &e.MessageID, &e.Sender, &e.Subject, &e.TextBody, &e.HTMLBody,
			&e.ReceivedAt, &e.RawHash, &e.ProcessingStatus, &e.ProcessingError, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func (db *DB) GetRecentInboundEmails(ctx context.Context, limit int) ([]domain.InboundEmail, error) {
	query := `
		SELECT id, provider_uid, message_id, sender, subject, text_body, html_body,
		       received_at, raw_hash, processing_status, processing_error, created_at, updated_at
		FROM inbound_emails
		ORDER BY id DESC LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []domain.InboundEmail
	for rows.Next() {
		var e domain.InboundEmail
		err := rows.Scan(
			&e.ID, &e.ProviderUID, &e.MessageID, &e.Sender, &e.Subject, &e.TextBody, &e.HTMLBody,
			&e.ReceivedAt, &e.RawHash, &e.ProcessingStatus, &e.ProcessingError, &e.CreatedAt, &e.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, rows.Err()
}

func (db *DB) UpdateInboundEmailStatus(ctx context.Context, id int64, status domain.ProcessingStatus, processingError string) error {
	query := `UPDATE inbound_emails SET processing_status = ?, processing_error = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.ExecContext(ctx, query, status, processingError, id)
	return err
}

func (db *DB) SaveTelegramNotification(ctx context.Context, n *domain.TelegramNotification) error {
	var evaluationID sql.NullInt64
	if n.EvaluationID > 0 {
		evaluationID = sql.NullInt64{Int64: n.EvaluationID, Valid: true}
	}
	var proposalDraftID sql.NullInt64
	if n.ProposalDraftID > 0 {
		proposalDraftID = sql.NullInt64{Int64: n.ProposalDraftID, Valid: true}
	}
	var msgID sql.NullInt64
	if n.TelegramMessageID > 0 {
		msgID = sql.NullInt64{Int64: n.TelegramMessageID, Valid: true}
	}

	query := `
		INSERT INTO telegram_notifications (
			project_id, evaluation_id, proposal_draft_id, notification_type,
			chat_id, telegram_message_id, status, error, sent_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(project_id, notification_type) DO UPDATE SET
			evaluation_id = excluded.evaluation_id,
			proposal_draft_id = excluded.proposal_draft_id,
			chat_id = excluded.chat_id,
			telegram_message_id = excluded.telegram_message_id,
			status = excluded.status,
			error = excluded.error,
			sent_at = excluded.sent_at
	`
	
	var sentAt interface{}
	if !n.SentAt.IsZero() {
		sentAt = n.SentAt
	}

	res, err := db.ExecContext(ctx, query,
		n.ProjectID, evaluationID, proposalDraftID, n.NotificationType,
		n.ChatID, msgID, n.Status, n.Error, sentAt,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	if id > 0 {
		n.ID = id
	}
	return nil
}

func (db *DB) HasTelegramNotification(ctx context.Context, projectID int64, notificationType domain.NotificationType) (bool, error) {
	var id int64
	err := db.QueryRowContext(ctx, "SELECT id FROM telegram_notifications WHERE project_id = ? AND notification_type = ?", projectID, notificationType).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (db *DB) GetTelegramNotificationMessageID(ctx context.Context, projectID int64, notificationType domain.NotificationType) (bool, int) {
	var msgID sql.NullInt64
	err := db.QueryRowContext(ctx, "SELECT telegram_message_id FROM telegram_notifications WHERE project_id = ? AND notification_type = ?", projectID, notificationType).Scan(&msgID)
	if err != nil || !msgID.Valid {
		return false, 0
	}
	return true, int(msgID.Int64)
}

