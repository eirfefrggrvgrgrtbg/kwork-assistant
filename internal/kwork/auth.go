package kwork

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// Authenticate logs into Kwork and returns the API token.
func (c *Client) Authenticate(ctx context.Context, login, password, phoneLast string) (string, error) {
	params := url.Values{}
	params.Set("login", login)
	params.Set("password", password)
	if phoneLast != "" {
		params.Set("phone_last", phoneLast)
	}

	data, err := c.doRequest(ctx, "POST", "signIn", params, true)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	var resp struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("failed to parse auth response: %w", err)
	}

	if resp.Token == "" {
		return "", fmt.Errorf("auth response did not contain token")
	}

	return resp.Token, nil
}

// GetMe retrieves the currently authenticated user's details.
func (c *Client) GetMe(ctx context.Context, token string) (*Actor, error) {
	params := url.Values{}
	params.Set("token", token)

	data, err := c.doRequest(ctx, "POST", "actor", params, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get actor: %w", err)
	}

	var actor Actor
	if err := json.Unmarshal(data, &actor); err != nil {
		return nil, fmt.Errorf("failed to parse actor response: %w", err)
	}

	return &actor, nil
}
