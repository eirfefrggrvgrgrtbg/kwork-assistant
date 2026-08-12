package evaluation

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kwork-assistant/internal/domain"
)

type AIResponse struct {
	Category         string   `json:"category"`
	Score            int      `json:"score"`
	Suitable         bool     `json:"suitable"`
	Complexity       string   `json:"complexity"`
	BudgetAssessment string   `json:"budget_assessment"`
	RiskLevel        string   `json:"risk_level"`
	EstimatedEffort  string   `json:"estimated_effort"`
	Summary          string   `json:"summary"`
	Reasons          []string `json:"reasons"`
	Warnings         []string `json:"warnings"`
}

func ParseJSON(rawJSON string, projectID int64, model string, promptVersion string) (domain.ProjectEvaluation, error) {
	// Clean up potential markdown formatting block
	rawJSON = strings.TrimSpace(rawJSON)
	if strings.HasPrefix(rawJSON, "```json") {
		rawJSON = strings.TrimPrefix(rawJSON, "```json")
	} else if strings.HasPrefix(rawJSON, "```") {
		rawJSON = strings.TrimPrefix(rawJSON, "```")
	}
	if strings.HasSuffix(rawJSON, "```") {
		rawJSON = strings.TrimSuffix(rawJSON, "```")
	}
	rawJSON = strings.TrimSpace(rawJSON)

	var aiResp AIResponse
	if err := json.Unmarshal([]byte(rawJSON), &aiResp); err != nil {
		return domain.ProjectEvaluation{}, fmt.Errorf("failed to parse JSON: %w\nRaw: %s", err, rawJSON)
	}

	// Validate Score
	if aiResp.Score < 0 || aiResp.Score > 100 {
		return domain.ProjectEvaluation{}, fmt.Errorf("invalid score: %d", aiResp.Score)
	}

	// Validate Category
	cat := strings.ToLower(aiResp.Category)
	if cat != "website" && cat != "telegram" && cat != "skip" {
		return domain.ProjectEvaluation{}, fmt.Errorf("invalid category: %s", aiResp.Category)
	}

	// Validate Complexity
	comp := strings.ToLower(aiResp.Complexity)
	if comp != "low" && comp != "medium" && comp != "high" && comp != "unknown" {
		return domain.ProjectEvaluation{}, fmt.Errorf("invalid complexity: %s", aiResp.Complexity)
	}

	// Validate Budget
	budg := strings.ToLower(aiResp.BudgetAssessment)
	if budg != "good" && budg != "acceptable" && budg != "low" && budg != "unknown" {
		return domain.ProjectEvaluation{}, fmt.Errorf("invalid budget_assessment: %s", aiResp.BudgetAssessment)
	}

	// Validate Risk
	risk := strings.ToLower(aiResp.RiskLevel)
	if risk != "low" && risk != "medium" && risk != "high" {
		return domain.ProjectEvaluation{}, fmt.Errorf("invalid risk_level: %s", aiResp.RiskLevel)
	}

	// Validate Strings
	if strings.TrimSpace(aiResp.Summary) == "" {
		return domain.ProjectEvaluation{}, fmt.Errorf("summary cannot be empty")
	}
	if len(aiResp.Reasons) == 0 {
		return domain.ProjectEvaluation{}, fmt.Errorf("reasons list cannot be empty")
	}

	eval := domain.ProjectEvaluation{
		ProjectID:        projectID,
		Category:         cat,
		Score:            aiResp.Score,
		Suitable:         aiResp.Suitable,
		Complexity:       comp,
		BudgetAssessment: budg,
		RiskLevel:        risk,
		EstimatedEffort:  aiResp.EstimatedEffort,
		Summary:          aiResp.Summary,
		Reasons:          aiResp.Reasons,
		Warnings:         aiResp.Warnings,
		Model:            model,
		PromptVersion:    promptVersion,
		CreatedAt:        time.Now(),
	}

	return eval, nil
}
