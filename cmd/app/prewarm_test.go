package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"kwork-assistant/internal/config"
	"kwork-assistant/internal/domain"
)

// mockAIFallback mocks AIClient for prewarm logic
type mockAIFallback struct {
	prewarmErr error
	prewarmCalled bool
}

func (m *mockAIFallback) Generate(ctx context.Context, req domain.GenerateRequest) (domain.GenerateResponse, error) {
	return domain.GenerateResponse{}, nil
}

func (m *mockAIFallback) Health(ctx context.Context) error { return nil }

func (m *mockAIFallback) CheckModelExists(ctx context.Context, modelName string) (bool, error) { return true, nil }

func (m *mockAIFallback) Prewarm(ctx context.Context, modelName string, keepAlive string, timeoutSeconds int) error {
	m.prewarmCalled = true
	return m.prewarmErr
}

func TestPrewarmAI_Success(t *testing.T) {
	mockClient := &mockAIFallback{prewarmErr: nil}
	cfg := &config.Config{OllamaModel: "test-model", AIPrewarmTimeoutSeconds: 1}
	
	err := prewarmAI(context.Background(), mockClient, cfg)
	if err != nil {
		t.Fatalf("expected prewarm to succeed, got %v", err)
	}
	if !mockClient.prewarmCalled {
		t.Fatal("expected prewarm to be called")
	}
}

func TestPrewarmAI_Error(t *testing.T) {
	mockClient := &mockAIFallback{prewarmErr: fmt.Errorf("fake error")}
	cfg := &config.Config{OllamaModel: "test-model", AIPrewarmTimeoutSeconds: 1}
	
	err := prewarmAI(context.Background(), mockClient, cfg)
	if err == nil {
		t.Fatal("expected prewarm to fail")
	}
	if !strings.Contains(err.Error(), "fake error") {
		t.Fatalf("expected 'fake error' in error msg, got %v", err)
	}
}
