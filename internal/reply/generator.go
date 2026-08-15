package reply

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
)

type Generator struct {
	aiClient ai.AIClient
	db       *database.DB
	model    string
	logger   *slog.Logger
}

func NewGenerator(aiClient ai.AIClient, db *database.DB, model string, logger *slog.Logger) *Generator {
	return &Generator{
		aiClient: aiClient,
		db:       db,
		model:    model,
		logger:   logger.With("component", "reply_generator"),
	}
}

// GenerateDraft produces a reply draft for the given conversation and the latest unreplied message.
func (g *Generator) GenerateDraft(ctx context.Context, conversationID int64, latestMessageID int64, project *domain.Project, history []domain.KworkMessage) (*domain.ReplyDraft, error) {
	prompt := BuildPrompt(project, history)

	req := domain.GenerateRequest{
		Model:  g.model,
		Prompt: prompt,
		Format: "json",
	}

	rawResponse, err := g.aiClient.Generate(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ai generation failed: %w", err)
	}

	result, err := ParseJSONResponse(rawResponse.Response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	draft := &domain.ReplyDraft{
		ConversationID:   conversationID,
		ContextMessageID: sql.NullInt64{Int64: latestMessageID, Valid: true},
		ModelVersion:     g.model,
		DraftText:        result.DraftText,
		Status:           domain.ReplyDraftStatusGenerated,
	}

	if err := ValidateDraft(result.DraftText); err != nil {
		g.logger.Warn("Generated draft failed validation", "error", err, "conversation_id", conversationID)
		draft.Status = domain.ReplyDraftStatusRejected
		// We can still return it, or maybe return an error. Let's return it so we can save it as rejected.
	}

	if len(result.Warnings) > 0 {
		g.logger.Warn("AI generated warnings for draft", "warnings", result.Warnings, "conversation_id", conversationID)
	}

	return draft, nil
}
