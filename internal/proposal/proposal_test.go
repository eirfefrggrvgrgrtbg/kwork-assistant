package proposal

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"kwork-assistant/internal/domain"
)

// ---- mock AI client ----

type mockAICall struct {
	text string
	err  error
}

type mockAIClient struct {
	calls    []mockAICall
	callIdx  int
	callsN   int // total calls made
}

func (m *mockAIClient) Generate(_ context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	m.callsN++
	if m.callIdx >= len(m.calls) {
		return domain.GenerateResponse{}, fmt.Errorf("unexpected call %d", m.callIdx)
	}
	c := m.calls[m.callIdx]
	m.callIdx++
	if c.err != nil {
		return domain.GenerateResponse{}, c.err
	}
	return domain.GenerateResponse{Response: c.text}, nil
}

// validJSON returns a JSON proposal of exactly n runes.
func validJSON(n int) string {
	text := strings.Repeat("А", n)
	return fmt.Sprintf(`{"proposal":%q,"confidence":"high","estimated_days":3}`, text)
}

func newTestGenerator(client *mockAIClient) *Generator {
	return &Generator{
		db:        nil, // saveRun is a no-op when db is nil
		aiClient:  client,
		modelName: "test-model",
	}
}

func testProject() domain.Project {
	return domain.Project{ID: 1, Title: "T", Description: "D"}
}
func testEval() domain.ProjectEvaluation {
	return domain.ProjectEvaluation{Score: 90, Suitable: true, Category: "telegram", Complexity: "low", RiskLevel: "low"}
}

// saveRun must not panic on nil db — add guard in generator.go
// (already guarded via _ = g.db.SaveAITestRun when db is nil would panic;
// we route around by overriding saveRun to be a no-op in test via the nil check below)

// TestGenerator_NormalSuccess — 700-char proposal succeeds in 1 call.
func TestGenerator_NormalSuccess(t *testing.T) {
	ai := &mockAIClient{calls: []mockAICall{{text: validJSON(700)}}}
	g := newTestGenerator(ai)

	draft, err := g.generateInternal(context.Background(), testProject(), testEval(), "v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len([]rune(draft.Proposal)) != 700 {
		t.Errorf("expected 700 chars, got %d", len([]rune(draft.Proposal)))
	}
	if ai.callsN != 1 {
		t.Errorf("expected 1 call, got %d", ai.callsN)
	}
}

// TestGenerator_RetrySuccess — 1400 first, 850 second → success, 2 calls.
func TestGenerator_RetrySuccess(t *testing.T) {
	ai := &mockAIClient{calls: []mockAICall{
		{text: validJSON(1400)},
		{text: validJSON(850)},
	}}
	g := newTestGenerator(ai)

	draft, err := g.generateInternal(context.Background(), testProject(), testEval(), "v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len([]rune(draft.Proposal)) != 850 {
		t.Errorf("expected 850 chars, got %d", len([]rune(draft.Proposal)))
	}
	if ai.callsN != 2 {
		t.Errorf("expected 2 calls, got %d", ai.callsN)
	}
}

// TestGenerator_CompressionSuccess — 1400, 1350, 800 → success, 3 calls.
func TestGenerator_CompressionSuccess(t *testing.T) {
	ai := &mockAIClient{calls: []mockAICall{
		{text: validJSON(1400)},
		{text: validJSON(1350)},
		{text: validJSON(800)},
	}}
	g := newTestGenerator(ai)

	draft, err := g.generateInternal(context.Background(), testProject(), testEval(), "v3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len([]rune(draft.Proposal)) != 800 {
		t.Errorf("expected 800 chars, got %d", len([]rune(draft.Proposal)))
	}
	if ai.callsN != 3 {
		t.Errorf("expected 3 calls, got %d", ai.callsN)
	}
}

// TestGenerator_AllFail — all 3 > 1000 → error after 3 calls.
func TestGenerator_AllFail(t *testing.T) {
	ai := &mockAIClient{calls: []mockAICall{
		{text: validJSON(1400)},
		{text: validJSON(1350)},
		{text: validJSON(1200)},
	}}
	g := newTestGenerator(ai)

	_, err := g.generateInternal(context.Background(), testProject(), testEval(), "v3")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if ai.callsN != 3 {
		t.Errorf("expected 3 calls, got %d", ai.callsN)
	}
}

// ---- existing tests preserved ----

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
		Proposal:   "```\nsome code\n```",
		Confidence: "high",
	}
	err := Validate(d)
	if err == nil || !strings.Contains(err.Error(), "markdown") {
		t.Errorf("expected markdown error, got %v", err)
	}
}

func TestValidate_Forbidden(t *testing.T) {
	d := domain.ProposalDraft{
		Proposal:   "Уважаемый заказчик, я специализируюсь на ботах.",
		Confidence: "high",
	}
	err := Validate(d)
	if err == nil || (!strings.Contains(err.Error(), "forbidden") && !strings.Contains(err.Error(), "hallucinated")) {
		t.Errorf("expected forbidden error, got %v", err)
	}
}

func TestBuildPrompt(t *testing.T) {
	p := domain.Project{
		Title:       "Need a bot",
		Description: "Bot for TG",
		BudgetTo:    sql.NullFloat64{Float64: 1000, Valid: true},
		Currency:    sql.NullString{String: "RUB", Valid: true},
	}
	e := domain.ProjectEvaluation{
		Complexity: "low",
		RiskLevel:  "low",
		Summary:    "Simple bot",
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
