package proposal

import (
	"context"
	"fmt"
	"strings"
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
	return g.generateInternal(ctx, p, eval, promptVersion)
}

// generateInternal contains the 3-attempt generation logic without the eligibility guards.
// Exposed for unit-testing with a mock AI client (no real DB needed).
func (g *Generator) generateInternal(ctx context.Context, p domain.Project, eval domain.ProjectEvaluation, promptVersion string) (domain.ProposalDraft, error) {
	generationStart := time.Now()
	basePrompt := BuildPrompt(p, eval)

	// Attempt 1: normal generation
	draft, aiResponse, err := g.attemptGeneration(ctx, p, eval, basePrompt, promptVersion)
	
	// Validation attempt 1
	var valErr error
	if err != nil {
		valErr = err // treat parse error as validation error to trigger retry
	} else {
		valErr = Validate(draft)
	}

	if valErr == nil {
		g.saveRun(ctx, p, aiResponse.Response, "", time.Since(generationStart))
		return draft, nil
	}
	
	if aiResponse != nil {
		g.saveRun(ctx, p, aiResponse.Response, fmt.Sprintf("attempt1 validation failed: %v", valErr), time.Since(generationStart))
	} else {
		g.saveRun(ctx, p, "ERROR", fmt.Sprintf("attempt1 failed: %v", valErr), time.Since(generationStart))
	}

	// Attempt 2: targeted retry.
	// If the failure is "too long", give the model explicit char count and instructions to shorten.
	// Otherwise give generic correction feedback.
	var retryPrompt string
	if strings.HasPrefix(valErr.Error(), "proposal too long:") {
		charCount := len([]rune(draft.Proposal))
		retryPrompt = basePrompt + fmt.Sprintf(
			"\n\nВНИМАНИЕ! Предыдущий отклик слишком длинный: %d символов (максимум 1000).\n"+
				"Сократи его до 400–600 символов.\n"+
				"Удали второстепенные объяснения.\n"+
				"Оставь:\n"+
				"1. понимание задачи;\n"+
				"2. конкретный подход;\n"+
				"3. один полезный вопрос.\n"+
				"НЕ добавляй новые детали.",
			charCount,
		)
	} else {
		retryPrompt = basePrompt + fmt.Sprintf(
			"\n\nВНИМАНИЕ! Твой предыдущий ответ был отклонен валидатором по причине: %s.\n"+
				"Исправь ответ, строго соблюдая правила. Убери выдуманную техническую уверенность или выдуманные примеры.",
			valErr.Error(),
		)
	}

	draftRetry, aiResponseRetry, errRetry := g.attemptGeneration(ctx, p, eval, retryPrompt, promptVersion)
	
	var valErrRetry error
	if errRetry != nil {
		valErrRetry = errRetry
	} else {
		valErrRetry = Validate(draftRetry)
	}

	if valErrRetry == nil {
		g.saveRun(ctx, p, aiResponseRetry.Response, "", time.Since(generationStart))
		return draftRetry, nil
	}
	
	if aiResponseRetry != nil {
		g.saveRun(ctx, p, aiResponseRetry.Response, fmt.Sprintf("attempt2 validation failed: %v", valErrRetry), time.Since(generationStart))
	} else {
		g.saveRun(ctx, p, "ERROR", fmt.Sprintf("attempt2 failed: %v", valErrRetry), time.Since(generationStart))
	}

	// Attempt 3: hard compression pass (only for length failures).
	// Ask the LLM to shorten the *text* of the proposal directly without adding info.
	if strings.HasPrefix(valErrRetry.Error(), "proposal too long:") {
		compressed, err := g.compressionPass(ctx, p, eval, draftRetry.Proposal, promptVersion)
		if err != nil {
			return domain.ProposalDraft{}, fmt.Errorf("compression pass failed: %w", err)
		}
		if valErrComp := Validate(compressed); valErrComp != nil {
			g.saveRun(ctx, p, compressed.Proposal, fmt.Sprintf("attempt3 compression validation failed: %v", valErrComp), time.Since(generationStart))
			return domain.ProposalDraft{}, fmt.Errorf("proposal validation failed after all 3 attempts: %w", valErrComp)
		}
		g.saveRun(ctx, p, compressed.Proposal, "", time.Since(generationStart))
		return compressed, nil
	}

	return domain.ProposalDraft{}, fmt.Errorf("proposal validation failed after retry: %w", valErrRetry)
}

// compressionPass asks the LLM to shorten an already-generated proposal text.
// It does NOT regenerate from the project brief — it only compresses the existing text.
func (g *Generator) compressionPass(ctx context.Context, p domain.Project, eval domain.ProjectEvaluation, tooLongText string, promptVersion string) (domain.ProposalDraft, error) {
	compressionPrompt := fmt.Sprintf(
		"Сократи следующий готовый отклик до 400–600 символов, не добавляя новой информации и не меняя смысл. "+
			"Верни результат ТОЛЬКО в JSON с полями proposal, question, internal_approach, confidence.\n\n"+
			"Отклик:\n%s",
		tooLongText,
	)

	req := domain.GenerateRequest{
		Model:           g.modelName,
		Prompt:          compressionPrompt,
		Format:          "json",
		ContextTokens:   4096,
		MaxOutputTokens: 1500,
		Temperature:     0.2,
		KeepAlive:       "2m",
	}

	aiResponse, err := g.aiClient.Generate(ctx, req)
	if err != nil {
		g.saveRun(ctx, p, "ERROR", err.Error(), 0)
		return domain.ProposalDraft{}, fmt.Errorf("AI compression failed: %w", err)
	}

	draft, err := ParseJSON(aiResponse.Response, p.ID, eval.ID, g.modelName, promptVersion)
	if err != nil {
		g.saveRun(ctx, p, aiResponse.Response, err.Error(), 0)
		return domain.ProposalDraft{}, fmt.Errorf("failed to parse compression response: %w", err)
	}

	return draft, nil
}

func (g *Generator) attemptGeneration(ctx context.Context, p domain.Project, eval domain.ProjectEvaluation, prompt string, promptVersion string) (domain.ProposalDraft, *domain.GenerateResponse, error) {
	start := time.Now()

	req := domain.GenerateRequest{
		Model:           g.modelName,
		Prompt:          prompt,
		Format:          "json",
		ContextTokens:   8192,
		MaxOutputTokens: 1500,
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
	if g.db == nil {
		return // no-op in unit tests
	}
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
