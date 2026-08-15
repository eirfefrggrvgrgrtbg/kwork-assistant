package evaluation

import (
	"context"
	"fmt"
	
	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/domain"
)

type Evaluator struct {
	client        ai.AIClient
	model         string
	promptVersion string
}

func NewEvaluator(client ai.AIClient, model string, promptVersion string) *Evaluator {
	return &Evaluator{
		client:        client,
		model:         model,
		promptVersion: promptVersion,
	}
}

func (e *Evaluator) EvaluateProject(ctx context.Context, p domain.Project) (domain.ProjectEvaluation, error) {
	prompt := BuildPrompt(p)

	req := domain.GenerateRequest{
		Model:           e.model,
		Prompt:          prompt,
		ContextTokens:   8192,
		MaxOutputTokens: 700,
		Temperature:     0.3,
		KeepAlive:       "2m",
		Format:          "json", // Ollama supports JSON format enforcement
	}

	resp, err := e.client.Generate(ctx, req)
	if err != nil {
		return domain.ProjectEvaluation{}, fmt.Errorf("AI generation failed: %w", err)
	}

	eval, err := ParseJSON(resp.Response, p.ID, e.model, e.promptVersion)
	if err != nil {
		return domain.ProjectEvaluation{}, fmt.Errorf("failed to parse AI response: %w", err)
	}
	
	eval.InputHash = p.InputHash()

	return eval, nil
}
