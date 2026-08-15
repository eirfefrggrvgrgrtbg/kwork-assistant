package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/chat"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/email"
	"kwork-assistant/internal/evaluation"
	"kwork-assistant/internal/pipeline"
	"kwork-assistant/internal/projectwatch"
	"kwork-assistant/internal/proposal"
	"kwork-assistant/internal/reply"
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
	bot, err := telegram.NewBot(cfg, db, slog.Default())
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
	
	if cfg.IMAPHost != "" && cfg.IMAPUsername != "" {
		err := fetchEmails(ctx, cfg, db)
		if err != nil {
			fmt.Printf("Error fetching emails: %v\n", err)
			// We can continue to process already stored emails
		}
	} else {
		fmt.Println("Email intake disabled (IMAP config missing)")
	}

	bot, err := telegram.NewBot(cfg, db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}
	
	pipeCfg := pipeline.Config{
		Model:          cfg.OllamaModel,
		SystemPrompt:   "evaluation-v2",
		ProposalPrompt: proposal.CurrentPromptVersion,
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

	bot, err := telegram.NewBot(cfg, db, slog.Default())
	if err != nil {
		fmt.Printf("Failed to init telegram bot: %v\n", err)
		os.Exit(1)
	}

	// Start bot polling in background
	go bot.StartPolling(ctx)

	pipeCfg := pipeline.Config{
		Model:          cfg.OllamaModel,
		SystemPrompt:   "evaluation-v2",
		ProposalPrompt: proposal.CurrentPromptVersion,
	}
	
	svc := pipeline.NewService(db, aiClient, bot, pipeCfg, slog.Default())

	kworkSrc, err := getKworkSource(cfg)
	if err != nil {
		slog.Error("Failed to init Kwork source for chat sync", "error", err)
	}
	var chatOrch *chat.SyncOrchestrator
	if kworkSrc != nil {
		chatSvc := chat.NewService(db, kworkSrc)
		chatOrch = chat.NewSyncOrchestrator(cfg, db, chatSvc, bot)
		bot.SetChatOrchestrator(chatOrch)
		
		replyGen := reply.NewGenerator(aiClient, db, cfg.OllamaModel, slog.Default())
		adapter := &ReplyGeneratorAdapter{Gen: replyGen, DB: db}
		bot.SetReplyGenerator(adapter)

		propGen := proposal.NewGenerator(db, aiClient, cfg.OllamaModel)
		propAdapter := &ProposalGeneratorAdapter{
			Gen:           propGen,
			DB:            db,
			Model:         cfg.OllamaModel,
			PromptVersion: proposal.CurrentPromptVersion,
		}
		bot.SetProposalGenerator(propAdapter)
	}

	var pWatcher *projectwatch.Watcher
	if kworkSrc != nil && cfg.KworkProjectWatchEnabled {
		fmt.Println("Загружаю локальную AI-модель", cfg.OllamaModel, "...")
		fmt.Println("При первом запуске это может занять несколько минут.")
		
		prewarmDone := make(chan struct{})
		prewarmErr := make(chan error, 1)
		
		go func() {
			err := aiClient.Prewarm(ctx, cfg.OllamaModel, cfg.OllamaKeepAlive, cfg.AIPrewarmTimeoutSeconds)
			if err != nil {
				prewarmErr <- err
			}
			close(prewarmDone)
		}()
		
		prewarmTicker := time.NewTicker(25 * time.Second)
		defer prewarmTicker.Stop()
		
	prewarmLoop:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("Отмена запуска.")
				os.Exit(1)
			case err := <-prewarmErr:
				fmt.Println("Не удалось загрузить локальную AI-модель.")
				fmt.Println("Убедитесь, что Ollama запущен и модель", cfg.OllamaModel, "установлена.")
				fmt.Println("После этого запустите Kwork Assistant снова.")
				fmt.Printf("Ошибка: %v\n", err)
				os.Exit(1)
			case <-prewarmDone:
				fmt.Println("AI-модель готова.")
				break prewarmLoop
			case <-prewarmTicker.C:
				fmt.Println("Модель всё ещё загружается...")
				fmt.Println("Это нормально на первом запуске.")
			}
		}
		
		evaluator := evaluation.NewEvaluator(aiClient, cfg.OllamaModel, "evaluation-v2")
		pWatcher = projectwatch.NewWatcher(cfg, db, kworkSrc, evaluator, cfg.OllamaModel, "evaluation-v2", bot, slog.Default())
	}

	ticker := time.NewTicker(30 * time.Second) // Could be configured
	
	// Parse Watcher poll interval
	pollInterval := 60 * time.Second
	if d, err := time.ParseDuration(cfg.KworkProjectPollInterval); err == nil {
		pollInterval = d
	}
	projTicker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	defer projTicker.Stop()
	
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
			
			// 1. Fetch (if enabled)
			if cfg.IMAPHost != "" && cfg.IMAPUsername != "" {
				_ = fetchEmails(ctx, cfg, db) // Ignore errors, keep trying
			}
			
			// 2. Process pending emails
			_ = svc.ProcessPending(ctx, 10)

			// 3. Sync Kwork Chats (if enabled)
			if chatOrch != nil {
				_, err := chatOrch.Run(ctx)
				if err != nil {
					slog.Error("Chat sync failed", "error", err)
				}
			}
		case <-projTicker.C:
			status, err := db.GetAppMeta(ctx, "auto_processing_enabled")
			if err != nil || status != "true" {
				continue
			}
			if pWatcher != nil {
				_, err := pWatcher.Run(ctx)
				if err != nil {
					slog.Error("Project watch failed", "error", err)
				}
			}
		}
	}
}
