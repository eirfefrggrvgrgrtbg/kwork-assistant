package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/email"
	"kwork-assistant/internal/pipeline"
	"kwork-assistant/internal/telegram"
)

func buildEmailConfig(cfg *config.Config) email.Config {
	return email.Config{
		Host:          cfg.IMAPHost,
		Port:          cfg.IMAPPort,
		Username:      cfg.IMAPUsername,
		Password:      cfg.IMAPPassword,
		UseTLS:        cfg.IMAPUseTLS,
		Folder:        cfg.EmailFolder,
		LookbackHours: cfg.EmailLookback,
	}
}

func buildTelegramConfig(cfg *config.Config) telegram.Config {
	return telegram.Config{
		Token:       cfg.TelegramBotToken,
		OwnerChatID: cfg.TelegramOwnerChatID,
	}
}

func runEmailHealth(ctx context.Context, cfg *config.Config) {
	fmt.Println("Testing IMAP Email Health...")
	client := email.NewIMAPClient(buildEmailConfig(cfg))
	if err := client.Health(); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("IMAP AUTH OK")
}

func runTelegramHealth(ctx context.Context, cfg *config.Config, db *database.DB) {
	fmt.Println("Testing Telegram Bot Health...")
	bot, err := telegram.NewBot(buildTelegramConfig(cfg), db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}
	if err := bot.Health(); err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("TELEGRAM API OK")
}

func fetchEmails(ctx context.Context, cfg *config.Config, db *database.DB) error {
	client := email.NewIMAPClient(buildEmailConfig(cfg))
	emails, err := client.FetchRecentEmails(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch emails: %w", err)
	}
	
	slog.Info("Fetched emails", "count", len(emails))
	
	for _, e := range emails {
		err := db.SaveInboundEmail(ctx, &e)
		if err != nil {
			slog.Error("Failed to upsert email", "msg_id", e.MessageID, "error", err)
		}
	}
	return nil
}

func runPipelineOnce(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient) {
	fmt.Println("Running Pipeline Once...")
	
	err := fetchEmails(ctx, cfg, db)
	if err != nil {
		fmt.Printf("Error fetching emails: %v\n", err)
		// We can continue to process already stored emails
	}

	bot, err := telegram.NewBot(buildTelegramConfig(cfg), db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}
	
	pipeCfg := pipeline.Config{
		Model:          cfg.OllamaModel,
		SystemPrompt:   "evaluation-v2",
		ProposalPrompt: "proposal-v3",
	}
	
	svc := pipeline.NewService(db, aiClient, bot, pipeCfg, slog.Default())
	if err := svc.ProcessPending(ctx, 10); err != nil {
		fmt.Printf("Error processing pending emails: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Pipeline run finished.")
}

func runDaemon(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient) {
	fmt.Println("Starting Daemon mode (IMAP -> AI -> TG)")

	bot, err := telegram.NewBot(buildTelegramConfig(cfg), db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}

	// Start bot polling in background
	go bot.StartPolling(ctx)

	pipeCfg := pipeline.Config{
		Model:          cfg.OllamaModel,
		SystemPrompt:   "evaluation-v2",
		ProposalPrompt: "proposal-v3",
	}
	
	svc := pipeline.NewService(db, aiClient, bot, pipeCfg, slog.Default())

	ticker := time.NewTicker(30 * time.Second) // Could be configured
	defer ticker.Stop()
	
	slog.Info("Daemon loop started")

	for {
		select {
		case <-ctx.Done():
			slog.Info("Daemon stopping")
			return
		case <-ticker.C:
			// Check if auto_processing is enabled
			status, err := db.GetAppMeta(ctx, "auto_processing_enabled")
			if err != nil || status != "true" {
				continue // Paused
			}
			
			// 1. Fetch
			_ = fetchEmails(ctx, cfg, db) // Ignore errors, keep trying
			
			// 2. Process
			_ = svc.ProcessPending(ctx, 10)
		}
	}
}
