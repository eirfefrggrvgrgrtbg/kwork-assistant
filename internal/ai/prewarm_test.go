package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestOllamaClient_Prewarm_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/generate" {
			t.Errorf("expected path /api/generate, got %s", r.URL.Path)
		}
		var reqBody ollamaGenerateRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal("failed to decode request body")
		}
		if reqBody.Model != "test-model" {
			t.Errorf("expected model 'test-model', got '%s'", reqBody.Model)
		}
		if reqBody.KeepAlive != "5m" {
			t.Errorf("expected keep_alive '5m', got '%s'", reqBody.KeepAlive)
		}
		
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"response": "OK"}`))
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, 1)
	err := client.Prewarm(context.Background(), "test-model", "5m", 1)
	if err != nil {
		t.Fatalf("expected prewarm to succeed, got %v", err)
	}
}

func TestOllamaClient_Prewarm_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewOllamaClient(server.URL, 10)
	
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	
	err := client.Prewarm(ctx, "test-model", "5m", 1)
	if err == nil {
		t.Fatal("expected prewarm to fail with timeout/cancellation, got nil")
	}
}
