package pipeline

import (
	"context"
	"testing"
	"time"

	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"log/slog"
)

type mockAI struct{}
func (m *mockAI) Prewarm(ctx context.Context, modelName string, keepAlive string, timeoutSeconds int) error { return nil }

func (m *mockAI) Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	return domain.GenerateResponse{
		Response: `{"category": "website", "score": 85, "reasoning": "Test reasoning", "complexity": "low", "risk_level": "low", "budget_assessment": "good", "summary": "Test summary", "reasons": ["Test"], "suitable": true, "proposal": "My test proposal", "confidence": "high"}`,
	}, nil
}

func (m *mockAI) CheckModelExists(ctx context.Context, model string) (bool, error) {
	return true, nil
}

func (m *mockAI) Health(ctx context.Context) error {
	return nil
}

type mockBot struct{}

func (m *mockBot) SendMessageWithButton(chatID int64, text string, buttonText, url string) (int64, error) {
	return 1, nil
}

func (m *mockBot) GetOwnerChatID() int64 {
	return 12345
}

func setupTestDB(t *testing.T) *database.DB {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to init in-memory db: %v", err)
	}
	return db
}

func TestPipelineIdempotency(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	cfg := Config{
		Model:          "test-model",
		SystemPrompt:   "v1",
		ProposalPrompt: "v1",
	}

	bot := &mockBot{}
	svc := NewService(db, &mockAI{}, bot, cfg, slog.Default())
	ctx := context.Background()

	// 1. Insert InboundEmail
	email := domain.InboundEmail{
		ProviderUID:      "100",
		MessageID:        "msg-123",
		Sender:           "notify@kwork.ru",
		Subject:          "Новый проект: Создать сайт",
		TextBody:         "Ссылка: https://kwork.ru/projects/12345/view\nБюджет: 5000 руб\nОписание: test",
		ReceivedAt:       time.Now(),
		RawHash:          "hash",
		ProcessingStatus: domain.ProcessingStatusNew,
	}

	err := db.SaveInboundEmail(ctx, &email)
	if err != nil {
		t.Fatalf("SaveInboundEmail failed: %v", err)
	}

	// Run process pending
	err = svc.ProcessPending(ctx, 1)
	if err != nil {
		t.Fatalf("ProcessPending failed: %v", err)
	}

	// Verify email is processed
	emails, _ := db.GetPendingInboundEmails(ctx, 10)
	if len(emails) != 0 {
		t.Errorf("Expected 0 pending emails, got %d", len(emails))
	}

	// Verify project created
	_, err = db.GetProjectByExternalID(ctx, "kwork_email", 12345)
	if err != nil {
		t.Fatalf("Project not created: %v", err)
	}

	// Run process pending AGAIN with the same email reset to New
	db.UpdateInboundEmailStatus(ctx, 1, domain.ProcessingStatusNew, "")
	err = svc.ProcessPending(ctx, 1)
	if err != nil {
		t.Fatalf("ProcessPending retry failed: %v", err)
	}

	// Ensure no duplicates
	count := 0
	row := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM projects WHERE external_id = 12345")
	row.Scan(&count)
	if count != 1 {
		t.Errorf("Expected 1 project, got %d", count)
	}
}
