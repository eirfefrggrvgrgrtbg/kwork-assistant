package reply

import (
	"encoding/json"
	"fmt"
	"strings"
)

type ReplyResult struct {
	DraftText string   `json:"draft_text"`
	Warnings  []string `json:"warnings"`
}

func ParseJSONResponse(raw string) (*ReplyResult, error) {
	// Clean up markdown code block if present
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```json") {
		raw = strings.TrimPrefix(raw, "```json")
	} else if strings.HasPrefix(raw, "```") {
		raw = strings.TrimPrefix(raw, "```")
	}
	if strings.HasSuffix(raw, "```") {
		raw = strings.TrimSuffix(raw, "```")
	}

	var res ReplyResult
	if err := json.Unmarshal([]byte(raw), &res); err != nil {
		return nil, fmt.Errorf("failed to parse reply json: %w", err)
	}

	res.DraftText = strings.TrimSpace(res.DraftText)

	return &res, nil
}
