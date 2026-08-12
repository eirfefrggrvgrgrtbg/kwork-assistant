package telegram

import (
	"database/sql"
	"strings"
	"testing"
	"kwork-assistant/internal/domain"
)

func TestFormatSuitableProject(t *testing.T) {
	p := domain.Project{
		Title: "Test <script>alert(1)</script>",
		BudgetTo: sql.NullFloat64{Float64: 5000, Valid: true},
		Currency: sql.NullString{String: "RUB", Valid: true},
	}
	
	eval := domain.ProjectEvaluation{
		Score: 92,
		Category: "website",
		Complexity: "medium",
		RiskLevel: "low",
		Summary: "Test summary & stuff",
	}
	
	draft := domain.ProposalDraft{
		Proposal: "Hello > world",
		Question: "Any questions?",
	}
	
	result := FormatSuitableProject(p, eval, draft)
	
	if !strings.Contains(result, "Test &lt;script&gt;alert(1)&lt;/script&gt;") {
		t.Errorf("Expected escaped title, got %s", result)
	}
	if !strings.Contains(result, "Hello &gt; world") {
		t.Errorf("Expected escaped proposal, got %s", result)
	}
}
