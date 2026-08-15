package main

import (
	"context"
	"path/filepath"
	"testing"

	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/proposal"
)

type mockAIClient struct {
	calls int
}

func (m *mockAIClient) Prewarm(ctx context.Context, modelName string, keepAlive string, timeoutSeconds int) error { return nil }

func (m *mockAIClient) Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	m.calls++
	return domain.GenerateResponse{
		Response: `{"proposal": "mock response", "confidence": "high", "question": ""}`,
	}, nil
}

func TestProposalAdapter_CacheLogic(t *testing.T) {
	tempDir := t.TempDir()
	db, err := database.InitDB(filepath.Join(tempDir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	proj1 := domain.Project{
		Source:     "kwork",
		ExternalID: 123,
		Title:      "Test",
		URL:        "https://kwork.ru/projects/123",
	}
	db.UpsertProject(ctx, proj1)
	savedProj, _ := db.GetProjectByExternalID(ctx, "kwork", 123)

	eval := domain.ProjectEvaluation{
		ProjectID:     savedProj.ID,
		Suitable:      true,
		Category:      "website",
		Score:         90,
		Model:         "test-model",
		PromptVersion: "evaluation-v2",
	}
	err = db.SaveEvaluation(ctx, eval)
	if err != nil {
		t.Fatal(err)
	}
	
	// Ensure we have eval.ID for the test
	evalP, err := db.GetEvaluationByExternalID(ctx, "kwork", 123)
	if err != nil {
		t.Fatal(err)
	}
	evalID := evalP.ID

	mockAI := &mockAIClient{}
	gen := proposal.NewGenerator(db, mockAI, "test-model")

	adapter := &ProposalGeneratorAdapter{
		Gen:           gen,
		DB:            db,
		Model:         "test-model",
		PromptVersion: "proposal-v5",
	}

	// ---------------------------------------------------------
	// TEST A: Existing draft = proposal-v4, eval = 100
	// ---------------------------------------------------------
	t.Run("A: Old version cache rejected", func(t *testing.T) {
		mockAI.calls = 0
		db.SaveProposalDraft(ctx, domain.ProposalDraft{
			EvaluationID:  evalID,
			Proposal:      "v4 old text",
			PromptVersion: "proposal-v4",
			Model:         "test-model",
		})

		_, err := adapter.GenerateDraft(ctx, savedProj.ID, false)
		if err != nil {
			t.Fatal(err)
		}
		if mockAI.calls != 1 {
			t.Errorf("expected 1 AI inference call for stale version, got %d", mockAI.calls)
		}
		
		draft, _ := db.GetProposalDraftByExternalID(ctx, savedProj.Source, savedProj.ExternalID)
		if draft.PromptVersion != "proposal-v5" {
			t.Errorf("expected new draft to be saved as proposal-v5, got %s", draft.PromptVersion)
		}
	})

	// ---------------------------------------------------------
	// TEST B: Existing draft = proposal-v5, eval = 100
	// ---------------------------------------------------------
	t.Run("B: Current version cache reused", func(t *testing.T) {
		mockAI.calls = 0
		// Db already has the v5 draft from previous step, let's just make sure.
		draft, _ := db.GetProposalDraftByExternalID(ctx, savedProj.Source, savedProj.ExternalID)
		if draft.PromptVersion != "proposal-v5" {
			t.Fatalf("expected v5 draft to be present")
		}

		_, err := adapter.GenerateDraft(ctx, savedProj.ID, false)
		if err != nil {
			t.Fatal(err)
		}
		if mockAI.calls != 0 {
			t.Errorf("expected 0 AI inference calls (cache reuse), got %d", mockAI.calls)
		}
	})

	// ---------------------------------------------------------
	// TEST C: Existing draft = proposal-v5, eval = 99
	// Current evaluation = 100
	// ---------------------------------------------------------
	t.Run("C: Stale evaluation cache rejected", func(t *testing.T) {
		mockAI.calls = 0
		
		// Change the evaluation by inserting a new evaluation with a different ID (sqlite autoincrements)
		// But SaveEvaluation updates if project_id exists. Wait! SaveEvaluation does ON CONFLICT REPLACE.
		// So it will overwrite the old evaluation but might get a new ID or keep the same ID.
		// Actually let's just create a completely new project and eval.
		
		proj2 := domain.Project{
			Source:     "kwork",
			ExternalID: 124,
			Title:      "Test2",
			URL:        "https://kwork.ru/projects/124",
		}
		db.UpsertProject(ctx, proj2)
		savedProj2, _ := db.GetProjectByExternalID(ctx, "kwork", 124)
		
		eval2 := domain.ProjectEvaluation{
			ProjectID:     savedProj2.ID,
			Suitable:      true,
			Category:      "website",
			Score:         95,
			Model:         "test-model",
			PromptVersion: "evaluation-v2",
		}
		_ = db.SaveEvaluation(ctx, eval2)
		evalP2, _ := db.GetEvaluationByExternalID(ctx, "kwork", 124)
		
		// Insert a stale draft for this project (EvaluationID = 99)
		db.SaveProposalDraft(ctx, domain.ProposalDraft{
			EvaluationID:  99,
			Proposal:      "v5 text but old eval",
			PromptVersion: "proposal-v5",
			Model:         "test-model",
		})
		
		_, err := adapter.GenerateDraft(ctx, savedProj2.ID, false)
		if err != nil {
			t.Fatal(err)
		}
		
		if mockAI.calls != 1 {
			t.Errorf("expected 1 AI inference call for stale evaluation, got %d", mockAI.calls)
		}
		
		draft, _ := db.GetProposalDraftByExternalID(ctx, savedProj2.Source, savedProj2.ExternalID)
		if draft.EvaluationID != evalP2.ID {
			t.Errorf("expected new draft to link to eval %d, got %d", evalP2.ID, draft.EvaluationID)
		}
	})
}
