package projectwatch

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/kwork"
)

type mockTransport struct {
	roundTripFunc func(req *http.Request) (*http.Response, error)
}
func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTripFunc(req)
}

type mockSource struct {
	projects []domain.Project
}
func (m *mockSource) FetchProjects(ctx context.Context, limit int) ([]domain.Project, error) {
	return m.projects, nil
}

type mockCooldownEvaluator struct{}
func (m *mockCooldownEvaluator) EvaluateProject(ctx context.Context, p domain.Project) (domain.ProjectEvaluation, error) {
	return domain.ProjectEvaluation{}, nil
}

func TestWatcher_URLCooldown(t *testing.T) {
	// Mock URL resolver globally for this test
	resolveCalls := 0
	transport := &mockTransport{
		roundTripFunc: func(req *http.Request) (*http.Response, error) {
			resolveCalls++
			return &http.Response{StatusCode: 404, Body: http.NoBody, Request: req}, nil
		},
	}
	kwork.SetResolverClient(&http.Client{Transport: transport})

	db, _ := database.InitDB(":memory:")
	defer db.Close()

	source := &mockSource{
		projects: []domain.Project{
			{Source: "kwork", ExternalID: 999111, Title: "Cooldown Test"},
		},
	}
	
	// Ensure we don't crash from nil logger
	cfg := &config.Config{KworkProjectWatchEnabled: true}
	watcher := NewWatcher(cfg, db, source, &mockCooldownEvaluator{}, "model", "v", nil, slog.Default())
	ctx := context.Background()

	// 1. New project (F. new project => resolver called)
	_, _ = watcher.Run(ctx)
	if resolveCalls != 2 { // 2 because 2 candidates are checked (view, and root)
		t.Errorf("Expected 2 resolve calls for new project, got %d", resolveCalls)
	}

	// 2. Second poll < 30min (G. unresolved project second poll <30min => resolver NOT called again)
	resolveCalls = 0
	_, _ = watcher.Run(ctx)
	if resolveCalls != 0 {
		t.Errorf("Expected 0 resolve calls during cooldown, got %d", resolveCalls)
	}

	// 3. After cooldown (H. after cooldown => resolver can retry)
	resolveCalls = 0
	// For watcher to find the project in its map, we need its ID. Let's just find it.
	projs, _ := db.GetRecentProjects(ctx, 1)
	if len(projs) > 0 {
		watcher.urlResolveCooldowns[projs[0].ID] = time.Now().Add(-31 * time.Minute)
	}
	_, _ = watcher.Run(ctx)
	if resolveCalls != 2 {
		t.Errorf("Expected 2 resolve calls after cooldown, got %d", resolveCalls)
	}
}
