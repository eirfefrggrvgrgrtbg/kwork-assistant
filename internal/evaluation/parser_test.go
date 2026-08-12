package evaluation

import (
	"strings"
	"testing"
	"time"
)

func TestParseJSON_Valid(t *testing.T) {
	rawJSON := `
	{
		"category": "website",
		"score": 95,
		"suitable": true,
		"complexity": "medium",
		"budget_assessment": "good",
		"risk_level": "low",
		"estimated_effort": "3-5 дней",
		"summary": "Разработка сайта",
		"reasons": ["Хороший стек", "Адекватный бюджет"],
		"warnings": []
	}`

	eval, err := ParseJSON(rawJSON, 123, "gemma4:e4b", "v1")
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if eval.ProjectID != 123 {
		t.Errorf("expected ProjectID 123, got %d", eval.ProjectID)
	}
	if eval.Category != "website" {
		t.Errorf("expected Category website, got %s", eval.Category)
	}
	if eval.Score != 95 {
		t.Errorf("expected Score 95, got %d", eval.Score)
	}
	if eval.Suitable != true {
		t.Errorf("expected Suitable true, got %t", eval.Suitable)
	}
	if eval.Complexity != "medium" {
		t.Errorf("expected Complexity medium, got %s", eval.Complexity)
	}
	if eval.BudgetAssessment != "good" {
		t.Errorf("expected BudgetAssessment good, got %s", eval.BudgetAssessment)
	}
	if eval.RiskLevel != "low" {
		t.Errorf("expected RiskLevel low, got %s", eval.RiskLevel)
	}
	if eval.EstimatedEffort != "3-5 дней" {
		t.Errorf("expected EstimatedEffort '3-5 дней', got %s", eval.EstimatedEffort)
	}
	if eval.Summary != "Разработка сайта" {
		t.Errorf("expected Summary 'Разработка сайта', got %s", eval.Summary)
	}
	if len(eval.Reasons) != 2 {
		t.Errorf("expected 2 reasons, got %d", len(eval.Reasons))
	}
	if len(eval.Warnings) != 0 {
		t.Errorf("expected 0 warnings, got %d", len(eval.Warnings))
	}
	if eval.Model != "gemma4:e4b" {
		t.Errorf("expected Model gemma4:e4b, got %s", eval.Model)
	}
	if eval.PromptVersion != "v1" {
		t.Errorf("expected PromptVersion v1, got %s", eval.PromptVersion)
	}
	if time.Since(eval.CreatedAt) > time.Second {
		t.Errorf("CreatedAt seems wrong")
	}
}

func TestParseJSON_MarkdownBlock(t *testing.T) {
	rawJSON := "```json\n{\"category\": \"telegram\", \"score\": 80, \"suitable\": true, \"complexity\": \"low\", \"budget_assessment\": \"acceptable\", \"risk_level\": \"medium\", \"summary\": \"Тест\", \"reasons\": [\"1\"]}\n```"
	
	eval, err := ParseJSON(rawJSON, 1, "test", "v1")
	if err != nil {
		t.Fatalf("expected no error parsing markdown block, got: %v", err)
	}
	if eval.Category != "telegram" {
		t.Errorf("expected telegram, got %s", eval.Category)
	}
}

func TestParseJSON_InvalidScore(t *testing.T) {
	rawJSON := `{"category": "website", "score": 105, "suitable": true, "complexity": "low", "budget_assessment": "acceptable", "risk_level": "medium", "summary": "Тест", "reasons": ["1"]}`
	
	_, err := ParseJSON(rawJSON, 1, "test", "v1")
	if err == nil || !strings.Contains(err.Error(), "invalid score") {
		t.Errorf("expected invalid score error, got: %v", err)
	}
}

func TestParseJSON_InvalidCategory(t *testing.T) {
	rawJSON := `{"category": "desktop", "score": 80, "suitable": true, "complexity": "low", "budget_assessment": "acceptable", "risk_level": "medium", "summary": "Тест", "reasons": ["1"]}`
	
	_, err := ParseJSON(rawJSON, 1, "test", "v1")
	if err == nil || !strings.Contains(err.Error(), "invalid category") {
		t.Errorf("expected invalid category error, got: %v", err)
	}
}

func TestParseJSON_EmptySummary(t *testing.T) {
	rawJSON := `{"category": "website", "score": 80, "suitable": true, "complexity": "low", "budget_assessment": "acceptable", "risk_level": "medium", "summary": "   ", "reasons": ["1"]}`
	
	_, err := ParseJSON(rawJSON, 1, "test", "v1")
	if err == nil || !strings.Contains(err.Error(), "summary cannot be empty") {
		t.Errorf("expected empty summary error, got: %v", err)
	}
}

func TestParseJSON_EmptyReasons(t *testing.T) {
	rawJSON := `{"category": "website", "score": 80, "suitable": true, "complexity": "low", "budget_assessment": "acceptable", "risk_level": "medium", "summary": "Тест", "reasons": []}`
	
	_, err := ParseJSON(rawJSON, 1, "test", "v1")
	if err == nil || !strings.Contains(err.Error(), "reasons list cannot be empty") {
		t.Errorf("expected empty reasons error, got: %v", err)
	}
}
