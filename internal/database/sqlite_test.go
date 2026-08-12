package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestInitDB(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	// Verify tables are created
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table'")
	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("failed to scan table name: %v", err)
		}
		tables = append(tables, name)
	}

	hasAppMeta := false
	hasAiTestRuns := false
	for _, tName := range tables {
		if tName == "app_meta" {
			hasAppMeta = true
		}
		if tName == "ai_test_runs" {
			hasAiTestRuns = true
		}
	}

	if !hasAppMeta {
		t.Errorf("expected table app_meta to be created")
	}
	if !hasAiTestRuns {
		t.Errorf("expected table ai_test_runs to be created")
	}
}

func TestSaveAITestRun(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	err = db.SaveAITestRun(context.Background(), AITestRun{
		Model:      "test-model",
		Input:      "test input",
		Output:     "test output",
		DurationMs: 123,
		Success:    true,
	})

	if err != nil {
		t.Fatalf("failed to save test run: %v", err)
	}

	// Verify
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM ai_test_runs").Scan(&count)
	if err != nil {
		t.Fatalf("failed to query count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 row, got %d", count)
	}
}
