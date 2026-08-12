package kwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// FetchDialogs fetches all dialogs (or the first page of them).
func (c *Client) FetchDialogs(ctx context.Context, token string) ([]Dialog, error) {
	params := url.Values{}
	params.Set("token", token)
	params.Set("filter", "all")
	params.Set("page", "1")

	data, err := c.doRequest(ctx, "POST", "dialogs", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch dialogs: %w", err)
	}

	var dialogs []Dialog
	if err := json.Unmarshal(data, &dialogs); err != nil {
		// Tolerant fallback: maybe it's not an array of Dialogs, or maybe we just want to see the raw JSON in debug.
		return nil, fmt.Errorf("failed to unmarshal dialogs array (raw length: %d bytes): %w", len(data), err)
	}

	return dialogs, nil
}
