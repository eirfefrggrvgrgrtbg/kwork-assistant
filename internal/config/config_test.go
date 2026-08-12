package config

import (
	"os"
	"testing"
)

func TestLoadDefault(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AppEnv != "development" {
		t.Errorf("expected development, got %s", cfg.AppEnv)
	}
	if cfg.AITimeoutSeconds != 120 {
		t.Errorf("expected 120, got %d", cfg.AITimeoutSeconds)
	}
	if cfg.AIContextTokens != 8192 {
		t.Errorf("expected 8192, got %d", cfg.AIContextTokens)
	}
	if cfg.AIMaxOutputTokens != 700 {
		t.Errorf("expected 700, got %d", cfg.AIMaxOutputTokens)
	}
	if cfg.AITemperature != 0.3 {
		t.Errorf("expected 0.3, got %f", cfg.AITemperature)
	}
	if cfg.OllamaKeepAlive != "2m" {
		t.Errorf("expected 2m, got %s", cfg.OllamaKeepAlive)
	}
	if cfg.KworkPollLimit != 50 {
		t.Errorf("expected 50, got %d", cfg.KworkPollLimit)
	}
}

func TestLoadWithEnv(t *testing.T) {
	os.Clearenv()
	os.Setenv("AI_TIMEOUT_SECONDS", "60")
	os.Setenv("AI_CONTEXT_TOKENS", "4096")
	os.Setenv("AI_MAX_OUTPUT_TOKENS", "300")
	os.Setenv("AI_TEMPERATURE", "0.5")
	os.Setenv("OLLAMA_KEEP_ALIVE", "5m")
	os.Setenv("APP_ENV", "production")
	os.Setenv("KWORK_LOGIN", "user")
	os.Setenv("KWORK_POLL_LIMIT", "10")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.AITimeoutSeconds != 60 {
		t.Errorf("expected 60, got %d", cfg.AITimeoutSeconds)
	}
	if cfg.AIContextTokens != 4096 {
		t.Errorf("expected 4096, got %d", cfg.AIContextTokens)
	}
	if cfg.AIMaxOutputTokens != 300 {
		t.Errorf("expected 300, got %d", cfg.AIMaxOutputTokens)
	}
	if cfg.AITemperature != 0.5 {
		t.Errorf("expected 0.5, got %f", cfg.AITemperature)
	}
	if cfg.OllamaKeepAlive != "5m" {
		t.Errorf("expected 5m, got %s", cfg.OllamaKeepAlive)
	}
	if cfg.KworkLogin != "user" {
		t.Errorf("expected user, got %s", cfg.KworkLogin)
	}
	if cfg.KworkPollLimit != 10 {
		t.Errorf("expected 10, got %d", cfg.KworkPollLimit)
	}
}

func TestLoadInvalidNumeric(t *testing.T) {
	os.Clearenv()
	os.Setenv("AI_TIMEOUT_SECONDS", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
	
	os.Clearenv()
	os.Setenv("AI_TEMPERATURE", "invalid")
	if _, err := Load(); err == nil {
		t.Fatal("expected error")
	}
}
