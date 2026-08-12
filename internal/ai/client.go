package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kwork-assistant/internal/domain"
)

type AIClient interface {
	Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error)
	Health(ctx context.Context) error
	CheckModelExists(ctx context.Context, modelName string) (bool, error)
}

type OllamaClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewOllamaClient(baseURL string, timeoutSeconds int) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(timeoutSeconds) * time.Second,
		},
	}
}

type ollamaGenerateRequest struct {
	Model     string                 `json:"model"`
	Prompt    string                 `json:"prompt"`
	Stream    bool                   `json:"stream"`
	Format    string                 `json:"format,omitempty"`
	KeepAlive string                 `json:"keep_alive,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Error    string `json:"error,omitempty"`
}

func (c *OllamaClient) Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	apiURL := fmt.Sprintf("%s/api/generate", c.baseURL)

	options := make(map[string]interface{})
	if req.ContextTokens > 0 {
		options["num_ctx"] = req.ContextTokens
	}
	if req.MaxOutputTokens > 0 {
		options["num_predict"] = req.MaxOutputTokens
	}
	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}

	ollamaReq := ollamaGenerateRequest{
		Model:     req.Model,
		Prompt:    req.Prompt,
		Stream:    false,
		Format:    req.Format,
		KeepAlive: req.KeepAlive,
		Options:   options,
	}

	bodyBytes, err := json.Marshal(ollamaReq)
	if err != nil {
		return domain.GenerateResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return domain.GenerateResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return domain.GenerateResponse{}, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return domain.GenerateResponse{}, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(respBody))
	}

	var ollamaResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return domain.GenerateResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if ollamaResp.Error != "" {
		return domain.GenerateResponse{}, fmt.Errorf("ollama API error: %s", ollamaResp.Error)
	}

	return domain.GenerateResponse{Response: ollamaResp.Response}, nil
}

func (c *OllamaClient) Health(ctx context.Context) error {
	apiURL := fmt.Sprintf("%s/api/tags", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

type ollamaTagsResponse struct {
	Models []struct {
		Name string `json:"name"`
	} `json:"models"`
}

func (c *OllamaClient) CheckModelExists(ctx context.Context, modelName string) (bool, error) {
	apiURL := fmt.Sprintf("%s/api/tags", c.baseURL)
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var tags ollamaTagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return false, fmt.Errorf("failed to decode tags: %w", err)
	}

	for _, m := range tags.Models {
		if m.Name == modelName {
			return true, nil
		}
	}
	return false, nil
}

func ParseJSONResponse(rawJSON string) (domain.AIJobEvaluation, error) {
	var eval domain.AIJobEvaluation
	if err := json.Unmarshal([]byte(rawJSON), &eval); err != nil {
		return domain.AIJobEvaluation{}, fmt.Errorf("failed to parse AI response as JSON: %w", err)
	}

	if eval.Score < 0 || eval.Score > 100 {
		return domain.AIJobEvaluation{}, fmt.Errorf("invalid score: %d (must be 0-100)", eval.Score)
	}

	if eval.DraftResponse == "" && eval.Suitable {
		return domain.AIJobEvaluation{}, fmt.Errorf("missing draft_response for a suitable job")
	}

	if eval.Category != "website" && eval.Category != "telegram" && eval.Category != "skip" {
		return domain.AIJobEvaluation{}, fmt.Errorf("invalid category: %s (must be website, telegram, or skip)", eval.Category)
	}

	return eval, nil
}
