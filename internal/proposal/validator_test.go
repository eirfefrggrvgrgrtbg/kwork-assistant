package proposal

import (
	"testing"
	
	"kwork-assistant/internal/domain"
)

func TestValidateProposal(t *testing.T) {
	tests := []struct {
		name      string
		proposal  string
		wantError bool
	}{
		{
			name:      "Unsupported certainty (решение: заменить)",
			proposal:  "Проблема связана с regex. Решение: заменить его.",
			wantError: true,
		},
		{
			name:      "Unsupported certainty (проблема вызвана)",
			proposal:  "Проблема вызвана неверной настройкой PHP.",
			wantError: true,
		},
		{
			name:      "Fake data format",
			proposal:  "Например формат user..domain",
			wantError: true,
		},
		{
			name:      "Code access hallucination",
			proposal:  "начну с анализа кода во вложении",
			wantError: true,
		},
		{
			name:      "Valid hypothesis",
			proposal:  "По описанию возможная причина — ограничение в валидации. Первым делом проверю, где именно оно находится.",
			wantError: false,
		},
		{
			name:      "Valid hypothesis 2",
			proposal:  "Если используется кастомный regex, обновлю правило.",
			wantError: false,
		},
		{
			name:      "Synthetic email",
			proposal:  "Проверю адрес user@sub.domain",
			wantError: true,
		},
		{
			name:      "General email mention",
			proposal:  "Проверю обработку email с многоуровневыми доменами",
			wantError: false,
		},
		{
			name:      "Synthetic URL",
			proposal:  "Отправлю данные на https://example.com",
			wantError: true,
		},
		{
			name:      "Synthetic Telegram",
			proposal:  "Свяжемся в тг @username_bot",
			wantError: true,
		},
		{
			name:      "Synthetic Phone",
			proposal:  "Мой номер +7 999 123-45-67",
			wantError: true,
		},
		{
			name:      "Certainty with 'связана с'",
			proposal:  "Проблема с валидацией связана с жестким regex.",
			wantError: true,
		},
		{
			name:      "Possible cause without certainty",
			proposal:  "По описанию возможная причина может быть в правилах валидации.",
			wantError: false,
		},
		{
			name:      "Conditional check",
			proposal:  "Если ограничение находится в regex, скорректирую правило проверки.",
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			draft := domain.ProposalDraft{
				Proposal:   tt.proposal,
				Confidence: "high",
			}
			err := Validate(draft)
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}
