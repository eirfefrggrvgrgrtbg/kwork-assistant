package evaluation

import (
	"strings"
	"testing"
	"kwork-assistant/internal/domain"
)

func TestBuildPrompt(t *testing.T) {
	p := domain.Project{
		Title: "Test Project",
		Description: "Need a simple website",
	}

	prompt := BuildPrompt(p)

	if !strings.Contains(prompt, "Test Project") {
		t.Errorf("prompt missing title")
	}
	if !strings.Contains(prompt, "Need a simple website") {
		t.Errorf("prompt missing description")
	}
	if !strings.Contains(prompt, "WEBSITE:") {
		t.Errorf("prompt missing instructions")
	}
}
