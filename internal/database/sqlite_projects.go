package database

import (
	"context"
	"database/sql"
	"fmt"
)

type SuitableProject struct {
	ID          int64
	Title       string
	BudgetFrom  sql.NullFloat64
	BudgetTo    sql.NullFloat64
	Score       int
	Category    string
}

func (db *DB) GetSuitableProjects(ctx context.Context, minScore int, limit int) ([]SuitableProject, error) {
	query := `
		SELECT p.id, p.title, p.budget_from, p.budget_to, e.score, e.category
		FROM projects p
		JOIN project_evaluations e ON p.id = e.project_id
		WHERE e.id = (
			SELECT MAX(id) FROM project_evaluations e2 
			WHERE e2.project_id = p.id
		)
		AND e.suitable = 1 
		AND e.category IN ('website', 'telegram')
		AND e.score >= ?
		ORDER BY e.created_at DESC
		LIMIT ?
	`
	rows, err := db.QueryContext(ctx, query, minScore, limit)
	if err != nil {
		return nil, fmt.Errorf("QueryContext: %w", err)
	}
	defer rows.Close()

	var result []SuitableProject
	for rows.Next() {
		var p SuitableProject
		if err := rows.Scan(&p.ID, &p.Title, &p.BudgetFrom, &p.BudgetTo, &p.Score, &p.Category); err != nil {
			continue
		}
		result = append(result, p)
	}
	return result, nil
}
