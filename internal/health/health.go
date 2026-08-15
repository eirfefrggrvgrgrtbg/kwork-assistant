package health

import (
	"context"
	"fmt"
	"log/slog"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
)

func Check(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient) error {
	slog.Info("Starting health check", "event", "health_check_started")

	fmt.Println("APP OK")

	if err := db.Health(ctx); err != nil {
		slog.Error("SQLite health check failed", "event", "sqlite_health_failed", "error", err)
		return fmt.Errorf("SQLITE FAILED: %w", err)
	}
	slog.Info("SQLite health check passed", "event", "db_ready")
	fmt.Println("SQLITE OK")

	if err := aiClient.Health(ctx); err != nil {
		slog.Error("Ollama health check failed", "event", "ollama_health_failed", "error", err)
		return fmt.Errorf("OLLAMA NOT RUNNING: Is Ollama started? (error: %w)", err)
	}
	slog.Info("Ollama health check passed", "event", "ollama_health_ok")
	fmt.Println("OLLAMA OK")

	hasModel, err := aiClient.CheckModelExists(ctx, cfg.OllamaModel)
	if err != nil {
		slog.Error("Failed to check model existence", "event", "ollama_model_check_failed", "error", err)
		return fmt.Errorf("MODEL CHECK FAILED: %w", err)
	}

	if !hasModel {
		fmt.Printf("MODEL %s NOT INSTALLED\n", cfg.OllamaModel)
		fmt.Printf("To install run: ollama pull %s\n", cfg.OllamaModel)
	} else {
		fmt.Printf("MODEL %s OK\n", cfg.OllamaModel)
	}

	return nil
}
