package database

import (
	"context"
	"path/filepath"
	"testing"
)

func TestGetAverageGenerationDuration(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	model := "test-model"

	// 1. 0 rows => 90
	avg, err := db.GetAverageGenerationDuration(ctx, model, 5)
	if err != nil {
		t.Fatal(err)
	}
	if avg != 90 {
		t.Errorf("Expected 90, got %d", avg)
	}

	// 2. Insert some unrelated AI rows and zero rows
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "some other task",
		Output:     "{}",
		DurationMs: 150000, // 150s
		Success:    true,
	})
	
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 1",
		Output:     "{}",
		DurationMs: 0,
		Success:    true,
	})

	avg, _ = db.GetAverageGenerationDuration(ctx, model, 5)
	if avg != 90 {
		t.Errorf("Expected 90 (ignored unrelated/zero), got %d", avg)
	}

	// 3. Insert 60/70/80 proposal durations
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 1",
		Output:     "{}",
		DurationMs: 60000,
		Success:    true,
	})
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 2",
		Output:     "{}",
		DurationMs: 70000,
		Success:    true,
	})
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 3",
		Output:     "{}",
		DurationMs: 80000,
		Success:    true,
	})

	avg, _ = db.GetAverageGenerationDuration(ctx, model, 5)
	if avg != 70 {
		t.Errorf("Expected 70 (median of 60, 70, 80), got %d", avg)
	}
	
	// 4. Test clamp bounds (20s - 180s)
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 4",
		Output:     "{}",
		DurationMs: 10000, // 10s
		Success:    true,
	})
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 5",
		Output:     "{}",
		DurationMs: 10000, // 10s
		Success:    true,
	})
	// Now we have 5 runs: 10, 10, 60, 70, 80 -> median is 60? 
	// Wait, we need all of them to be 10s to test the clamp.
	
	// Let's test clamp 20s
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 6",
		Output:     "{}",
		DurationMs: 10000, // 10s
		Success:    true,
	})
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 7",
		Output:     "{}",
		DurationMs: 10000, // 10s
		Success:    true,
	})
	_ = db.SaveAITestRun(ctx, AITestRun{
		Model:      model,
		Input:      "PROPOSAL_DRAFT for project 8",
		Output:     "{}",
		DurationMs: 10000, // 10s
		Success:    true,
	})
	
	avg, _ = db.GetAverageGenerationDuration(ctx, model, 5)
	// Last 5 runs are all 10s. Median should be 10s. But clamped to 20s.
	if avg != 20 {
		t.Errorf("Expected 20 (clamped from 10), got %d", avg)
	}
}
