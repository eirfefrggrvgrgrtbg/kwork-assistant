package projectwatch

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"database/sql"

	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
)

type mockNotifier struct {
	calls int
}

func (m *mockNotifier) SendSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation) (int, error) {
	m.calls++
	return 12345, nil
}

func (m *mockNotifier) UpdateSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation, messageID int) error {
	m.calls++
	return nil
}

type mockEvaluator struct {
	model         string
	promptVersion string
	evals         map[int64]*domain.ProjectEvaluation
	evalCalls     int
}

func (m *mockEvaluator) EvaluateProject(ctx context.Context, p domain.Project) (domain.ProjectEvaluation, error) {
	m.evalCalls++
	eval := *m.evals[p.ExternalID]
	eval.ProjectID = p.ID
	eval.Model = m.model
	eval.PromptVersion = m.promptVersion
	eval.InputHash = p.InputHash()
	return eval, nil
}

func (m *mockEvaluator) GetModel() string {
	return m.model
}

func (m *mockEvaluator) GetPromptVersion() string {
	return m.promptVersion
}

type mockProjectSource struct {
	projects []domain.Project
}

func (m *mockProjectSource) FetchProjects(ctx context.Context, limit int) ([]domain.Project, error) {
	return m.projects, nil
}

func TestProjectWatcher_Run(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory db: %v", err)
	}
	defer db.Close()

	cfg := &config.Config{
		KworkProjectWatchEnabled: true,
		KworkPollLimit:           20,
		KworkSuitableScore:       80,
	}

	notifier := &mockNotifier{}
	evalMock := &mockEvaluator{
		model:         "test_model",
		promptVersion: "v1",
		evals:         make(map[int64]*domain.ProjectEvaluation),
	}
	
	sourceMock := &mockProjectSource{}

	watcher := NewWatcher(cfg, db, sourceMock, evalMock, "test_model", "v1", notifier, slog.Default())

	ctx := context.Background()

	// Setup 12 dummy projects
	var batch []domain.Project
	for i := 1; i <= 12; i++ {
		p := domain.Project{
			ExternalID: int64(i),
			Source:     "kwork",
			Title:      fmt.Sprintf("Project %d", i),
		}
		batch = append(batch, p)
		
		evalMock.evals[int64(i)] = &domain.ProjectEvaluation{
			Score:    85,
			Suitable: true,
			Category: "website",
		}
	}
	
	sourceMock.projects = batch

	// A. FIRST FETCH - 12 unseen projects
	stats, err := watcher.Run(ctx)
	if err != nil {
		t.Fatalf("Run A failed: %v", err)
	}
	if stats.NewProjects != 12 || stats.AlreadyKnown != 0 {
		t.Errorf("A failed: expected 12 new, 0 known. Got %d new, %d known", stats.NewProjects, stats.AlreadyKnown)
	}
	if evalMock.evalCalls != 12 {
		t.Errorf("A failed: expected 12 eval calls, got %d", evalMock.evalCalls)
	}
	if notifier.calls != 12 {
		t.Errorf("A failed: expected 12 notifs, got %d", notifier.calls)
	}

	// Reset counters for next run
	evalMock.evalCalls = 0
	notifier.calls = 0

	// B. SECOND IDENTICAL FETCH
	stats, err = watcher.Run(ctx)
	if err != nil {
		t.Fatalf("Run B failed: %v", err)
	}
	if stats.NewProjects != 0 || stats.AlreadyKnown != 12 {
		t.Errorf("B failed: expected 0 new, 12 known. Got %d new, %d known", stats.NewProjects, stats.AlreadyKnown)
	}
	if stats.SkippedEvaluation != 12 || evalMock.evalCalls != 0 {
		t.Errorf("B failed: expected 12 skipped, 0 eval calls. Got %d skipped, %d calls", stats.SkippedEvaluation, evalMock.evalCalls)
	}
	if stats.TelegramDupesSkipped != 12 {
		t.Errorf("B failed: expected 12 dupes skipped, got %d", stats.TelegramDupesSkipped)
	}

	// C. KNOWN BUT UNEVALUATED
	// We'll insert a new project directly into DB to simulate "known", but no evaluation exists
	p13 := domain.Project{ExternalID: 13, Source: "kwork", Title: "Project 13"}
	db.UpsertProject(ctx, p13)
	evalMock.evals[13] = &domain.ProjectEvaluation{Score: 90, Suitable: true, Category: "telegram"}
	
	sourceMock.projects = []domain.Project{p13}
	evalMock.evalCalls = 0
	notifier.calls = 0
	
	stats, err = watcher.Run(ctx)
	if err != nil {
		t.Fatalf("Run C failed: %v", err)
	}
	if stats.AlreadyKnown != 1 || stats.NewProjects != 0 {
		t.Errorf("C failed: expected 1 known, got %d known", stats.AlreadyKnown)
	}
	if evalMock.evalCalls != 1 {
		t.Errorf("C failed: expected 1 eval call, got %d", evalMock.evalCalls)
	}
	if notifier.calls != 1 {
		t.Errorf("C failed: expected 1 notif, got %d", notifier.calls)
	}

	// D. KNOWN AND EVALUATED
	evalMock.evalCalls = 0
	notifier.calls = 0
	stats, err = watcher.Run(ctx)
	if err != nil {
		t.Fatalf("Run D failed: %v", err)
	}
	if stats.AlreadyKnown != 1 || stats.SkippedEvaluation != 1 || evalMock.evalCalls != 0 {
		t.Errorf("D failed: expected 1 known, 1 skipped eval, 0 eval calls. Got %d known, %d skipped, %d calls", stats.AlreadyKnown, stats.SkippedEvaluation, evalMock.evalCalls)
	}
	
	// E. SUITABLE EVALUATED BUT NOT NOTIFIED
	// To simulate this, we evaluate but delete the notification.
	p13Saved, _ := db.GetProjectByExternalID(ctx, "kwork", 13)
	db.ExecContext(ctx, "DELETE FROM telegram_notifications WHERE project_id = ?", p13Saved.ID)
	
	// F. TEST CACHE INVALIDATION
	// Change project budget and check if it gets re-evaluated
	p13Saved.BudgetFrom = sql.NullFloat64{Float64: 2000, Valid: true}
	p13Saved.BudgetTo = sql.NullFloat64{Float64: 6000, Valid: true}
	db.UpsertProject(ctx, p13Saved) // Update project in DB

	// Make source return the updated project so Watcher fetches it
	sourceMock.projects[0] = p13Saved

	evalMock.evalCalls = 0
	notifier.calls = 0
	stats, err = watcher.Run(ctx)
	if err != nil {
		t.Fatalf("Run F failed: %v", err)
	}

	if evalMock.evalCalls != 1 {
		t.Errorf("F failed: expected 1 eval call due to cache invalidation (hash changed), got %d", evalMock.evalCalls)
	}
	if notifier.calls != 1 {
		t.Errorf("F failed: expected 1 notif update call, got %d", notifier.calls)
	}
}
