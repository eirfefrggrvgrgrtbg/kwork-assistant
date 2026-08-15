package proposal

import (
	"database/sql"
	"strings"
	"testing"

	"kwork-assistant/internal/domain"
)

func TestBuildPrompt_V5(t *testing.T) {
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
		if !strings.Contains(prompt, "Бюджет 5000 ₽ вижу") {
			t.Errorf("expected EXACT budget format, got: %s", prompt)
		}
		if !strings.Contains(prompt, "PROPOSAL-V5") {
			t.Errorf("expected V5 format")
		}
	})

	t.Run("RANGE", func(t *testing.T) {
		p := domain.Project{
			BudgetFrom: sql.NullFloat64{Float64: 5000, Valid: true},
			BudgetTo:   sql.NullFloat64{Float64: 15000, Valid: true},
			Currency:   sql.NullString{String: "RUB", Valid: true},
		}

		prompt := BuildPrompt(p, eval)
		if !strings.Contains(prompt, "Бюджет 5000–15000 ₽ вижу") {
			t.Errorf("expected RANGE budget format, got: %s", prompt)
		}
		if strings.Contains(prompt, "на какую сумму") {
			t.Errorf("should NOT ask how much client wants to pay in v5")
		}
	})

	t.Run("UNKNOWN", func(t *testing.T) {
		p := domain.Project{
			BudgetFrom: sql.NullFloat64{Valid: false},
			BudgetTo:   sql.NullFloat64{Valid: false},
			Currency:   sql.NullString{String: "RUB", Valid: true},
		}

		prompt := BuildPrompt(p, eval)
		if !strings.Contains(prompt, "БЮДЖЕТ не указан") {
			t.Errorf("expected UNKNOWN budget format, got: %s", prompt)
		}
		if !strings.Contains(prompt, "По стоимости смогу сориентировать") {
			t.Errorf("expected unknown v5 text, got: %s", prompt)
		}
	})
	
	t.Run("CONSTRAINTS", func(t *testing.T) {
		p := domain.Project{}
		prompt := BuildPrompt(p, eval)
		
		if !strings.Contains(prompt, "минимум 2 конкретные детали") {
			t.Error("should ask for project-specific details")
		}
		if !strings.Contains(prompt, "ОДНИМ сильным вопросом") {
			t.Error("should ask for single CTA")
		}
		if !strings.Contains(prompt, "Поле 'question' в JSON оставь пустым") {
			t.Error("should ask to leave question field empty")
		}
		if !strings.Contains(prompt, "НЕ спрашивай про дедлайн, если это не самая важная вещь") {
			t.Error("should not always ask deadline")
		}
		if !strings.Contains(prompt, "Мягкий максимум 650") || !strings.Contains(prompt, "ЖЁСТКИЙ МАКСИМУМ 1000") {
			t.Error("should have length limits")
		}
	})
}

