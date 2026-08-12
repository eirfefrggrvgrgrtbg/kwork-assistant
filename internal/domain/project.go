package domain

import (
	"context"
	"database/sql"
	"time"
)

type Project struct {
	ID           int64
	ExternalID   int64
	Source       string
	URL          string
	Title        string
	Description  string
	BudgetFrom   sql.NullFloat64
	BudgetTo     sql.NullFloat64
	Currency     sql.NullString
	CategoryID   sql.NullInt64
	CategoryName sql.NullString
	BuyerID      sql.NullInt64
	BuyerName    sql.NullString
	OffersCount  int
	PublishedAt  time.Time
	FetchedAt    time.Time
	RawJSON      string
}

type ProjectSource interface {
	Health(ctx context.Context) error
	GetMe(ctx context.Context) (int64, string, error)
	FetchCategories(ctx context.Context) (map[int]string, error)
	FetchProjects(ctx context.Context, limit int) ([]Project, error)
}

type ProjectEvaluation struct {
	ID               int64
	ProjectID        int64
	Category         string
	Score            int
	Suitable         bool
	Complexity       string
	BudgetAssessment string
	RiskLevel        string
	EstimatedEffort  string
	Summary          string
	Reasons          []string
	Warnings         []string
	Model            string
	PromptVersion    string    `json:"prompt_version"`
	CreatedAt        time.Time `json:"created_at"`
}

type ProposalDraft struct {
	ID            int64     `json:"id"`
	ProjectID     int64     `json:"project_id"`
	EvaluationID  int64     `json:"evaluation_id"`
	Proposal      string    `json:"proposal"`
	Question      string    `json:"question"`
	Approach      string    `json:"approach"`
	Confidence    string    `json:"confidence"`
	Warnings      []string  `json:"warnings"`
	Model         string    `json:"model"`
	PromptVersion string    `json:"prompt_version"`
	CreatedAt     time.Time `json:"created_at"`
}

type EvaluationFilter struct {
	Category string
	MinScore int
	Limit    int
}
