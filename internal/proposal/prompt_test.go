package proposal

import (
	"database/sql"
	"strings"
	"testing"

	"kwork-assistant/internal/domain"
)

func TestBuildPrompt_Budget(t *testing.T) {
	eval := domain.ProjectEvaluation{
		Category: "website",
		Summary:  "test summary",
	}

	t.Run("EXACT", func(t *testing.T) {
		p := domain.Project{
			BudgetFrom: sql.NullFloat64{Float64: 5000, Valid: true},
			BudgetTo:   sql.NullFloat64{Valid: false},
			Currency:   sql.NullString{String: "RUB", Valid: true},
		}

		prompt := BuildPrompt(p, eval)
		if !strings.Contains(prompt, "Вижу бюджет 5000 ₽") {
			t.Errorf("expected EXACT budget format, got: %s", prompt)
		}
		if strings.Contains(prompt, "Бюджет не указан") {
			t.Errorf("expected NO 'Бюджет не указан', got: %s", prompt)
		}
		if strings.Contains(prompt, "на какую сумму в этом диапазоне") {
			t.Errorf("expected NO range question, got: %s", prompt)
		}
	})

	t.Run("RANGE", func(t *testing.T) {
		p := domain.Project{
			BudgetFrom: sql.NullFloat64{Float64: 5000, Valid: true},
			BudgetTo:   sql.NullFloat64{Float64: 15000, Valid: true},
			Currency:   sql.NullString{String: "RUB", Valid: true},
		}

		prompt := BuildPrompt(p, eval)
		if !strings.Contains(prompt, "Вижу бюджет 5000–15000 ₽") {
			t.Errorf("expected RANGE budget format, got: %s", prompt)
		}
		if !strings.Contains(prompt, "на какую сумму в этом диапазоне") {
			t.Errorf("expected range question, got: %s", prompt)
		}
	})

	t.Run("UNKNOWN", func(t *testing.T) {
		p := domain.Project{
			BudgetFrom: sql.NullFloat64{Valid: false},
			BudgetTo:   sql.NullFloat64{Valid: false},
			Currency:   sql.NullString{String: "RUB", Valid: true},
		}

		prompt := BuildPrompt(p, eval)
		if !strings.Contains(prompt, "Бюджет не указан") {
			t.Errorf("expected UNKNOWN budget format, got: %s", prompt)
		}
		if !strings.Contains(prompt, "какой бюджет вы закладываете") {
			t.Errorf("expected unknown question, got: %s", prompt)
		}
	})
}
