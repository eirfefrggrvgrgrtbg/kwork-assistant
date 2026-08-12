package proposal

import (
	"context"
	"fmt"
	"time"

	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
)

type AIClient interface {
	Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error)
}

type Generator struct {
	db        *database.DB
	aiClient  AIClient
	modelName string
}

func NewGenerator(db *database.DB, aiClient AIClient, modelName string) *Generator {
	return &Generator{
		db:        db,
		aiClient:  aiClient,
		modelName: modelName,
	}
}

func (g *Generator) Generate(ctx context.Context, p domain.Project, eval domain.ProjectEvaluation, promptVersion string) (domain.ProposalDraft, error) {
	if !eval.Suitable {
		return domain.ProposalDraft{}, fmt.Errorf("project is not suitable for generation")
	}
	if eval.Category != "website" && eval.Category != "telegram" {
		return domain.ProposalDraft{}, fmt.Errorf("project category %s is not supported for auto-generation", eval.Category)
	}
	if eval.Score < 60 {
		return domain.ProposalDraft{}, fmt.Errorf("project score %d is too low (minimum 60)", eval.Score)
	}

	prompt := BuildPrompt(p, eval)

	draft, aiResponse, err := g.attemptGeneration(ctx, p, eval, prompt, promptVersion)
	if err != nil {
		return domain.ProposalDraft{}, err
	}

	// Validation
	if err := Validate(draft); err != nil {
		g.saveRun(ctx, p, aiResponse.Response, fmt.Sprintf("validation failed: %v", err), 0)
		
		// AUTO-RETRY (1 time)
		feedbackPrompt := prompt + fmt.Sprintf("\n\nВНИМАНИЕ! Твой предыдущий ответ был отклонен валидатором по причине: %s.\nИсправь ответ, строго соблюдая правила. Убери выдуманную техническую уверенность или выдуманные примеры.", err.Error())
		
		draftRetry, aiResponseRetry, errRetry := g.attemptGeneration(ctx, p, eval, feedbackPrompt, promptVersion)
		if errRetry != nil {
			return domain.ProposalDraft{}, fmt.Errorf("retry generation failed: %w", errRetry)
		}
		
		if errRetryVal := Validate(draftRetry); errRetryVal != nil {
			g.saveRun(ctx, p, aiResponseRetry.Response, fmt.Sprintf("retry validation failed: %v", errRetryVal), 0)
			return domain.ProposalDraft{}, fmt.Errorf("proposal validation failed after retry: %w", errRetryVal)
		}

		draft = draftRetry
		aiResponse = aiResponseRetry
	}

	g.saveRun(ctx, p, aiResponse.Response, "", 0)
	return draft, nil
}

func (g *Generator) attemptGeneration(ctx context.Context, p domain.Project, eval domain.ProjectEvaluation, prompt string, promptVersion string) (domain.ProposalDraft, *domain.GenerateResponse, error) {
	start := time.Now()

	req := domain.GenerateRequest{
		Model:           g.modelName,
		Prompt:          prompt,
		Format:          "json",
		ContextTokens:   8192,
		MaxOutputTokens: 700,
		Temperature:     0.3,
		KeepAlive:       "2m",
	}

	aiResponse, err := g.aiClient.Generate(ctx, req)
	if err != nil {
		g.saveRun(ctx, p, "ERROR", err.Error(), time.Since(start))
		return domain.ProposalDraft{}, nil, fmt.Errorf("AI generation failed: %w", err)
	}

	draft, err := ParseJSON(aiResponse.Response, p.ID, eval.ID, g.modelName, promptVersion)
	if err != nil {
		g.saveRun(ctx, p, aiResponse.Response, err.Error(), time.Since(start))
		return domain.ProposalDraft{}, &aiResponse, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return draft, &aiResponse, nil
}

func (g *Generator) saveRun(ctx context.Context, p domain.Project, output, errStr string, duration time.Duration) {
	run := database.AITestRun{
		Model:      g.modelName,
		Input:      fmt.Sprintf("PROPOSAL_DRAFT for Project ID: %d", p.ID),
		Output:     output,
		DurationMs: duration.Milliseconds(),
		Success:    errStr == "",
		Error:      errStr,
	}
	_ = g.db.SaveAITestRun(ctx, run)
}
