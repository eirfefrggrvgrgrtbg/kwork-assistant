package kwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// FetchCategories retrieves categories from Kwork.
func (c *Client) FetchCategories(ctx context.Context, token string) (map[int]string, error) {
	params := url.Values{}
	params.Set("token", token)
	params.Set("type", "1") // 1 means standard categories in kwork

	data, err := c.doRequest(ctx, "POST", "categories", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch categories: %w", err)
	}

	var categories []Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, fmt.Errorf("failed to parse categories response: %w", err)
	}

	result := make(map[int]string)
	for _, cat := range categories {
		result[cat.ID] = cat.Name
	}
	return result, nil
}
