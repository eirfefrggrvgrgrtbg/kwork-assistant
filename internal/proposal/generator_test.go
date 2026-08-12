package proposal

import (
	"context"
	"testing"
	"fmt"

	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/database"
)

type MockAIClient struct {
	responses []string
	callCount int
}

func (m *MockAIClient) Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	if m.callCount >= len(m.responses) {
		return domain.GenerateResponse{}, fmt.Errorf("unexpected call %d", m.callCount)
	}
	resp := m.responses[m.callCount]
	m.callCount++
	return domain.GenerateResponse{Response: resp}, nil
}

// A stub DB just to avoid panics (if saveRun ignores or can handle it). 
// Actually, saveRun might panic if db is nil. We can create an empty in-memory db.
func getTestDB(t *testing.T) *database.DB {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	// Migrate is called implicitly inside InitDB now
	return db
}

func TestGenerator_RetryLogic_Success(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	mockAI := &MockAIClient{
		responses: []string{
			`{
				"proposal": "Проблема вызвана неправильным regex.",
				"question": "test",
				"internal_approach": "test",
				"confidence": "high",
				"warnings": []
			}`,
			`{
				"proposal": "По описанию возможное ограничение находится в валидации. Первым делом проверю, где именно выполняется проверка.",
				"question": "test",
				"internal_approach": "test",
				"confidence": "high",
				"warnings": []
			}`,
		},
	}

	gen := NewGenerator(db, mockAI, "test-model")
	
	p := domain.Project{
		ID: 1,
		Title: "Test",
		Description: "Test",
	}
	eval := domain.ProjectEvaluation{
		ID: 1,
		Suitable: true,
		Category: "website",
		Score: 80,
	}

	draft, err := gen.Generate(context.Background(), p, eval, "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mockAI.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mockAI.callCount)
	}

	if draft.Proposal != "По описанию возможное ограничение находится в валидации. Первым делом проверю, где именно выполняется проверка." {
		t.Errorf("unexpected final draft: %s", draft.Proposal)
	}
}

func TestGenerator_RetryLogic_Failure(t *testing.T) {
	db := getTestDB(t)
	defer db.Close()

	mockAI := &MockAIClient{
		responses: []string{
			`{
				"proposal": "Проблема вызвана неправильным regex.",
				"question": "test",
				"internal_approach": "test",
				"confidence": "high",
				"warnings": []
			}`,
			`{
				"proposal": "Проблема вызвана неправильным regex. Опять.",
				"question": "test",
				"internal_approach": "test",
				"confidence": "high",
				"warnings": []
			}`,
		},
	}

	gen := NewGenerator(db, mockAI, "test-model")
	
	p := domain.Project{
		ID: 1,
		Title: "Test",
		Description: "Test",
	}
	eval := domain.ProjectEvaluation{
		ID: 1,
		Suitable: true,
		Category: "website",
		Score: 80,
	}

	_, err := gen.Generate(context.Background(), p, eval, "v1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if mockAI.callCount != 2 {
		t.Errorf("expected 2 calls, got %d", mockAI.callCount)
	}
}
