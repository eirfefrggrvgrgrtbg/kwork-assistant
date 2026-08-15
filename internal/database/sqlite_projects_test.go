package database

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"kwork-assistant/internal/domain"
)

func TestGetSuitableProjects(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := InitDB(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	defer db.Close()
	ctx := context.Background()

	// Insert project
	p := domain.Project{
		ExternalID:   123,
		Source:       "kwork",
		Title:        "Test project",
		CategoryName: sql.NullString{String: "website", Valid: true},
		PublishedAt:  time.Now(),
		FetchedAt:    time.Now(),
	}
	isNew, err := db.UpsertProject(ctx, p)
	if err != nil || !isNew {
		t.Fatalf("UpsertProject failed: %v", err)
	}

	savedProj, _ := db.GetProjectByExternalID(ctx, "kwork", 123)

	// Insert evaluation 1 (stale, high score)
	eval1 := domain.ProjectEvaluation{
		ProjectID:     savedProj.ID,
		Category:      "website",
		Score:         90,
		Suitable:      true,
		Model:         "m",
		PromptVersion: "v1",
		InputHash:     "hash1",
	}
	db.SaveEvaluation(ctx, eval1)

	// Insert evaluation 2 (current, low score)
	eval2 := domain.ProjectEvaluation{
		ProjectID:     savedProj.ID,
		Category:      "website",
		Score:         50,
		Suitable:      false,
		Model:         "m",
		PromptVersion: "v1",
		InputHash:     "hash2",
	}
	db.SaveEvaluation(ctx, eval2)

	// Result should be empty because latest eval is score=50
	projs, err := db.GetSuitableProjects(ctx, 80, 10)
	if err != nil {
		t.Fatalf("GetSuitableProjects: %v", err)
	}
	if len(projs) != 0 {
		t.Fatalf("expected 0, got %d", len(projs))
	}

	// Insert evaluation 3 (current, high score)
	eval3 := domain.ProjectEvaluation{
		ProjectID:     savedProj.ID,
		Category:      "website",
		Score:         95,
		Suitable:      true,
		Model:         "m",
		PromptVersion: "v1",
		InputHash:     "hash3",
	}
	db.SaveEvaluation(ctx, eval3)

	// Result should contain it now
	projs, err = db.GetSuitableProjects(ctx, 80, 10)
	if err != nil {
		t.Fatalf("GetSuitableProjects: %v", err)
	}
	if len(projs) != 1 {
		t.Fatalf("expected 1, got %d", len(projs))
	}
	if projs[0].Title != "Test project" {
		t.Errorf("wrong title: %s", projs[0].Title)
	}
	if projs[0].Score != 95 {
		t.Errorf("wrong score: %d", projs[0].Score)
	}
}
