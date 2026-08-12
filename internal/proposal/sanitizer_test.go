package proposal

import (
	"testing"
)

func TestSanitizeText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Email",
			input:    "Contact me at test@example.com please.",
			expected: "Contact me at [EMAIL УДАЛЕН] please.",
		},
		{
			name:     "HTTPS URL",
			input:    "Check https://example.com/test for details",
			expected: "Check [URL УДАЛЕН] for details",
		},
		{
			name:     "WWW URL",
			input:    "Site www.example.ru is down",
			expected: "Site [URL УДАЛЕН] is down",
		},
		{
			name:     "Plain domain",
			input:    "Domain example.ru",
			expected: "Domain [САЙТ УДАЛЕН]",
		},
		{
			name:     "Telegram username",
			input:    "My tg @username",
			expected: "My tg [TELEGRAM УДАЛЕН]",
		},
		{
			name:     "Telegram link",
			input:    "Link https://t.me/test_user",
			expected: "Link [URL УДАЛЕН]",
		},
		{
			name:     "Telegram short link",
			input:    "Link t.me/test_user",
			expected: "Link [TELEGRAM УДАЛЕН]",
		},
		{
			name:     "Phone number RU",
			input:    "Call +7 999 123-45-67 now",
			expected: "Call [ТЕЛЕФОН УДАЛЕН] now",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeText(tt.input)
			if got != tt.expected {
				t.Errorf("SanitizeText() = %v, want %v", got, tt.expected)
			}
		})
	}
}
