package kwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// FetchDialogMessages fetches the messages of a specific dialog.
func (c *Client) FetchDialogMessages(ctx context.Context, token, username string) ([]InboxMessage, error) {
	params := url.Values{}
	params.Set("token", token)
	params.Set("username", username)
	params.Set("page", "1")

	data, err := c.doRequest(ctx, "POST", "inboxes", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch messages for %s: %w", username, err)
	}

	var messages []InboxMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages array (raw length: %d bytes): %w", len(data), err)
	}

	return messages, nil
}
