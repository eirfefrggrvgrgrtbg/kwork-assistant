package proposal

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kwork-assistant/internal/domain"
)

type rawProposalDraft struct {
	Proposal   string   `json:"proposal"`
	Question   string   `json:"question"`
	Approach   string   `json:"approach"`
	Confidence string   `json:"confidence"`
	Warnings   []string `json:"warnings"`
}

func ParseJSON(raw string, projectID int64, evaluationID int64, model, promptVersion string) (domain.ProposalDraft, error) {
	// Clean markdown block if present
	cleanJSON := strings.TrimSpace(raw)
	if strings.HasPrefix(cleanJSON, "```json") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
	} else if strings.HasPrefix(cleanJSON, "```") {
		cleanJSON = strings.TrimPrefix(cleanJSON, "```")
	}
	if strings.HasSuffix(cleanJSON, "```") {
		cleanJSON = strings.TrimSuffix(cleanJSON, "```")
	}
	cleanJSON = strings.TrimSpace(cleanJSON)

	var parsed rawProposalDraft
	if err := json.Unmarshal([]byte(cleanJSON), &parsed); err != nil {
		return domain.ProposalDraft{}, fmt.Errorf("failed to parse JSON: %w\nRaw: %s", err, cleanJSON)
	}

	draft := domain.ProposalDraft{
		ProjectID:     projectID,
		EvaluationID:  evaluationID,
		Proposal:      strings.TrimSpace(parsed.Proposal),
		Question:      strings.TrimSpace(parsed.Question),
		Approach:      strings.TrimSpace(parsed.Approach),
		Confidence:    strings.ToLower(strings.TrimSpace(parsed.Confidence)),
		Warnings:      parsed.Warnings,
		Model:         model,
		PromptVersion: promptVersion,
		CreatedAt:     time.Now(),
	}

	if draft.Warnings == nil {
		draft.Warnings = []string{}
	}

	return draft, nil
}
