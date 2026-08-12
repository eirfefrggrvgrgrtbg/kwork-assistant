package kwork

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL = "https://api.kwork.ru"
	// mobileAppKey is a public, static client identifier required by the Kwork Mobile API.
	// It is NOT a user credential. It is globally shared across all mobile app installations.
	mobileAppKey   = "bW9iaWxlX2FwaTpxRnZmUmw3dw=="
)

// Client represents the Kwork API client.
type Client struct {
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Kwork API client.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    defaultBaseURL,
	}
}

// doRequest performs an HTTP request to the Kwork API.
// endpoint should be the path, e.g., "signIn" or "categories".
func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, withAuth bool) ([]byte, error) {
	reqURL := fmt.Sprintf("%s/%s", c.baseURL, endpoint)
	
	var reqBody io.Reader
	if method == "POST" && params != nil {
		reqBody = strings.NewReader(params.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	if method == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	if withAuth {
		req.Header.Set("Authorization", "Basic "+mobileAppKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &ErrorResponse{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("HTTP %d on %s", resp.StatusCode, endpoint),
		}
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode generic response on %s: %w", endpoint, err)
	}

	if !apiResp.Success {
		return nil, &ErrorResponse{
			Code:    resp.StatusCode,
			Message: apiResp.Error,
		}
	}

	return apiResp.Data, nil
}
