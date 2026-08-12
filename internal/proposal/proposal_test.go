package proposal

import (
	"database/sql"
	"strings"
	"testing"
	"kwork-assistant/internal/domain"
)

func TestParseJSON_Valid(t *testing.T) {
	raw := `
	{
		"proposal": "Здравствуйте. Посмотрел задачу. Сделаю телеграм-бота для приема заявок.",
		"question": "Нужна ли админка?",
		"approach": "Простая задача, делаю акцент на ТГ боте.",
		"confidence": "high",
		"warnings": ["Нет точного ТЗ"]
	}`

	draft, err := ParseJSON(raw, 1, 2, "model1", "v1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if draft.Proposal != "Здравствуйте. Посмотрел задачу. Сделаю телеграм-бота для приема заявок." {
		t.Errorf("wrong proposal: %s", draft.Proposal)
	}
	if draft.Confidence != "high" {
		t.Errorf("wrong confidence: %s", draft.Confidence)
	}
}

func TestValidate_Empty(t *testing.T) {
	d := domain.ProposalDraft{Proposal: "   ", Confidence: "high"}
	err := Validate(d)
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty error, got %v", err)
	}
}

func TestValidate_TooLong(t *testing.T) {
	d := domain.ProposalDraft{Proposal: strings.Repeat("A", 1001), Confidence: "high"}
	err := Validate(d)
	if err == nil || !strings.Contains(err.Error(), "too long") {
		t.Errorf("expected too long error, got %v", err)
	}
}

func TestValidate_Markdown(t *testing.T) {
	d := domain.ProposalDraft{
		Proposal: "```\nsome code\n```",
		Confidence: "high",
	}
	err := Validate(d)
	if err == nil || !strings.Contains(err.Error(), "markdown") {
		t.Errorf("expected markdown error, got %v", err)
	}
}

func TestValidate_Forbidden(t *testing.T) {
	d := domain.ProposalDraft{
		Proposal: "Уважаемый заказчик, я специализируюсь на ботах.",
		Confidence: "high",
	}
	err := Validate(d)
	if err == nil || (!strings.Contains(err.Error(), "forbidden") && !strings.Contains(err.Error(), "hallucinated")) {
		t.Errorf("expected forbidden error, got %v", err)
	}
}

func TestBuildPrompt(t *testing.T) {
	p := domain.Project{
		Title: "Need a bot",
		Description: "Bot for TG",
		BudgetTo: sql.NullFloat64{Float64: 1000, Valid: true},
		Currency: sql.NullString{String: "RUB", Valid: true},
	}
	e := domain.ProjectEvaluation{
		Complexity: "low",
		RiskLevel: "low",
		Summary: "Simple bot",
	}

	prompt := BuildPrompt(p, e)
	if !strings.Contains(prompt, "Need a bot") {
		t.Errorf("missing title")
	}
	if !strings.Contains(prompt, "1000 RUB") {
		t.Errorf("missing budget")
	}
	if !strings.Contains(prompt, "Simple bot") {
		t.Errorf("missing evaluation summary")
	}
}
