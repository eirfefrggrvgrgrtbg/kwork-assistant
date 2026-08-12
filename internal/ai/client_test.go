package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kwork-assistant/internal/domain"
)

func TestParseJSONResponse_Valid(t *testing.T) {
	validJSON := `{
		"score": 85,
		"suitable": true,
		"category": "telegram",
		"reasons": ["matches go stack"],
		"draft_response": "Hello, I can do this."
	}`

	eval, err := ParseJSONResponse(validJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if eval.Score != 85 {
		t.Errorf("expected score 85, got %d", eval.Score)
	}
	if !eval.Suitable {
		t.Errorf("expected suitable true")
	}
	if len(eval.Reasons) != 1 {
		t.Errorf("expected 1 reason, got %d", len(eval.Reasons))
	}
	if eval.Category != "telegram" {
		t.Errorf("expected category telegram, got %s", eval.Category)
	}
}

func TestParseJSONResponse_InvalidScore(t *testing.T) {
	invalidJSON := `{"score": 150, "suitable": true, "category": "telegram", "draft_response": "Hi"}`
	_, err := ParseJSONResponse(invalidJSON)
	if err == nil {
		t.Fatal("expected error for score > 100")
	}
}

func TestParseJSONResponse_MissingDraft(t *testing.T) {
	invalidJSON := `{"score": 90, "suitable": true, "category": "website"}`
	_, err := ParseJSONResponse(invalidJSON)
	if err == nil {
		t.Fatal("expected error for missing draft response when suitable")
	}
}

func TestParseJSONResponse_InvalidCategory(t *testing.T) {
	invalidJSON := `{
		"score": 80,
		"suitable": true,
		"category": "mobile",
		"reasons": ["test"],
		"draft_response": "Hi"
	}`
	_, err := ParseJSONResponse(invalidJSON)
	if err == nil {
		t.Fatal("expected error for invalid category")
	}
}

func TestOllamaClient_Generate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected path /api/generate, got %s", r.URL.Path)
		}
		
		var req ollamaGenerateRequest
		json.NewDecoder(r.Body).Decode(&req)
		
		if req.Format != "json" {
			t.Errorf("expected format json, got %s", req.Format)
		}
		
		if req.KeepAlive != "2m" {
			t.Errorf("expected keep_alive 2m, got %s", req.KeepAlive)
		}

		if req.Options["num_ctx"].(float64) != 8192 {
			t.Errorf("expected num_ctx 8192, got %v", req.Options["num_ctx"])
		}

		resp := ollamaGenerateResponse{
			Response: `{"score":100}`,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	client := NewOllamaClient(srv.URL, 5)
	
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	resp, err := client.Generate(ctx, domain.GenerateRequest{
		Model:           "test-model",
		Prompt:          "Test",
		Format:          "json",
		KeepAlive:       "2m",
		ContextTokens:   8192,
		MaxOutputTokens: 700,
		Temperature:     0.3,
	})
	
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Response != `{"score":100}` {
		t.Errorf("unexpected response: %s", resp.Response)
	}
}
