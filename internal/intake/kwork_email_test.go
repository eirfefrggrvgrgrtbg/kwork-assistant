package intake

import (
	"testing"
	"time"

	"kwork-assistant/internal/domain"
)

func TestParseKworkEmail(t *testing.T) {
	email := domain.InboundEmail{
		Sender:   "notify@kwork.ru",
		Subject:  "Новый проект: Доработка сайта на WordPress",
		TextBody: "Описание:\nНужно доработать сайт, поправить верстку.\nБюджет: до 5 000 руб\n\nПосмотреть проект\nhttps://kwork.ru/projects/999123/view",
		ReceivedAt: time.Now(),
	}

	candidate, isKwork := ParseKworkEmail(email)
	if !isKwork {
		t.Fatalf("expected to detect as Kwork email")
	}

	if candidate.ExternalID != 999123 {
		t.Errorf("expected ID 999123, got %d", candidate.ExternalID)
	}
	if candidate.Title != "Доработка сайта на WordPress" {
		t.Errorf("expected title 'Доработка сайта на WordPress', got %q", candidate.Title)
	}
	if candidate.BudgetTo != 5000 {
		t.Errorf("expected budget 5000, got %f", candidate.BudgetTo)
	}
	if candidate.Currency != "RUB" {
		t.Errorf("expected currency RUB, got %q", candidate.Currency)
	}
	if candidate.Description != "Нужно доработать сайт, поправить верстку." {
		t.Errorf("expected description 'Нужно доработать сайт, поправить верстку.', got %q", candidate.Description)
	}
	if candidate.ParseConfidence != domain.ParseConfidenceHigh {
		t.Errorf("expected confidence High, got %s", candidate.ParseConfidence)
	}
}

func TestParseKworkEmail_MissingFields(t *testing.T) {
	email := domain.InboundEmail{
		Sender:   "notify@kwork.ru",
		Subject:  "Kwork",
		TextBody: "Просто текст без явного бюджета и ID",
		HTMLBody: "<a href=\"https://kwork.ru/projects/111/view\">Link</a>",
	}

	candidate, isKwork := ParseKworkEmail(email)
	if !isKwork {
		t.Fatalf("expected to detect as Kwork email")
	}

	if candidate.ExternalID != 111 {
		t.Errorf("expected ID 111 from HTML fallback, got %d", candidate.ExternalID)
	}
	if len(candidate.MissingFields) == 0 {
		t.Errorf("expected missing fields")
	}
	if candidate.ParseConfidence == domain.ParseConfidenceHigh {
		t.Errorf("expected lower confidence")
	}
}
