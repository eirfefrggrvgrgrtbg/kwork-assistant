package main

import (
	"context"
	"encoding/json"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/evaluation"
	"kwork-assistant/internal/health"
	"kwork-assistant/internal/kwork"
	"kwork-assistant/internal/proposal"
)

func main() {
	_ = godotenv.Load()
	setupLogger()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	cfg, err := config.Load()
	if err != nil {
		slog.Error("Failed to load config", "error", err)
		os.Exit(1)
	}

	slog.Info("Application started", "event", "app_started", "env", cfg.AppEnv)

	db, err := database.InitDB(cfg.DatabasePath)
	if err != nil {
		slog.Error("Failed to initialize database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	aiClient := ai.NewOllamaClient(cfg.OllamaBaseURL, cfg.AITimeoutSeconds)

	ctx := context.Background()

	switch command {
	case "health":
		if err := health.Check(ctx, cfg, db, aiClient); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	case "ai-test":
		runAITest(ctx, cfg, db, aiClient)
	case "kwork-health":
		runKworkHealth(ctx, cfg)
	case "kwork-categories":
		runKworkCategories(ctx, cfg)
	case "kwork-fetch":
		runKworkFetch(ctx, cfg, db)
	case "projects":
		runProjects(ctx, db)
	case "project":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/app project <external_id> [--raw]")
			os.Exit(1)
		}
		extID := os.Args[2]
		showRaw := len(os.Args) >= 4 && os.Args[3] == "--raw"
		runProjectDetails(ctx, db, extID, showRaw)
	case "evaluate":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/app evaluate <external_id> [--force]")
			os.Exit(1)
		}
		extID := os.Args[2]
		force := len(os.Args) >= 4 && os.Args[3] == "--force"
		runEvaluate(ctx, cfg, db, aiClient, extID, force)
	case "evaluate-batch":
		limit := 20
		for i, arg := range os.Args {
			if arg == "--limit" && i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &limit)
			}
		}
		runEvaluateBatch(ctx, cfg, db, aiClient, limit)
	case "evaluations":
		runEvaluations(ctx, db, os.Args)
	case "evaluation":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/app evaluation <external_id> [--raw]")
			os.Exit(1)
		}
		extID := os.Args[2]
		showRaw := len(os.Args) >= 4 && os.Args[3] == "--raw"
		runEvaluation(ctx, db, extID, showRaw)
	case "proposal-draft":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/app proposal-draft <external_id> [--force]")
			os.Exit(1)
		}
		extID := os.Args[2]
		force := len(os.Args) >= 4 && os.Args[3] == "--force"
		runProposalDraft(ctx, cfg, db, aiClient, extID, force)
	case "proposal-batch":
		limit := 10
		for i, arg := range os.Args {
			if arg == "--limit" && i+1 < len(os.Args) {
				fmt.Sscanf(os.Args[i+1], "%d", &limit)
			}
		}
		runProposalBatch(ctx, cfg, db, aiClient, limit)
	case "proposals":
		runProposals(ctx, db, os.Args)
	case "proposal":
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run ./cmd/app proposal <external_id>")
			os.Exit(1)
		}
		extID := os.Args[2]
		runProposal(ctx, db, extID)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func setupLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)
}

func printUsage() {
	fmt.Println("Usage: go run ./cmd/app [command]")
	fmt.Println("Commands:")
	fmt.Println("  health           - Run health checks")
	fmt.Println("  ai-test          - Run a test against the local AI model")
	fmt.Println("  kwork-health     - Test Kwork auth and GetMe")
	fmt.Println("  kwork-categories - Print Kwork categories")
	fmt.Println("  kwork-fetch      - Fetch and normalize Kwork projects")
	fmt.Println("  projects         - List recent Kwork projects from local DB")
	fmt.Println("  project <id> [--raw] - View specific project details")
	fmt.Println("  proposal-draft <id> [--force] - Generate proposal draft for project")
	fmt.Println("  proposal-batch [--limit 10]   - Generate drafts for suitable projects")
	fmt.Println("  proposals [--min-score] [--category] - List proposal drafts")
	fmt.Println("  proposal <id>                 - View specific proposal draft")
}

func runAITest(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient) {
	prompt := `Evaluate the following freelance job based on the developer profile and return a JSON object with strictly these fields:
- "score": integer between 0 and 100
- "suitable": boolean
- "category": string, one of: "website", "telegram", "skip"
- "reasons": array of strings explaining the score
- "draft_response": a short greeting and proposal if suitable, otherwise empty string

TITLE:
"Нужен Telegram-бот для интернет-магазина"

DESCRIPTION:
"Нужно сделать каталог, корзину, оформление заявки, уведомления менеджеру и хранение заказов."

Developer profile:
Основные направления:
сайты;
web-приложения;
Telegram-боты;
Telegram Mini Apps.
Исполнитель не ограничивается одним языком программирования.
Если заказчик не требует конкретный стек, подходящую технологию можно выбрать самостоятельно.

Ensure the output is ONLY a valid JSON object. No markdown formatting, no code blocks.`

	slog.Info("Starting AI request", "event", "ai_request_started", "model", cfg.OllamaModel)
	startTime := time.Now()

	req := domain.GenerateRequest{
		Model:           cfg.OllamaModel,
		Prompt:          prompt,
		Format:          "json",
		KeepAlive:       cfg.OllamaKeepAlive,
		ContextTokens:   cfg.AIContextTokens,
		MaxOutputTokens: cfg.AIMaxOutputTokens,
		Temperature:     cfg.AITemperature,
	}

	resp, err := aiClient.Generate(ctx, req)
	duration := time.Since(startTime)
	durationMs := duration.Milliseconds()

	testRun := database.AITestRun{
		Model:      cfg.OllamaModel,
		Input:      prompt,
		DurationMs: durationMs,
	}

	if err != nil {
		slog.Error("AI request failed", "event", "ai_request_failed", "error", err, "duration_ms", durationMs)
		testRun.Success = false
		testRun.Error = err.Error()
		
		_ = db.SaveAITestRun(ctx, testRun)
		fmt.Printf("AI request failed: %v\n", err)
		os.Exit(1)
	}

	slog.Info("AI request completed", "event", "ai_request_completed", "duration_ms", durationMs)
	testRun.Output = resp.Response

	eval, err := ai.ParseJSONResponse(resp.Response)
	if err != nil {
		slog.Error("Failed to parse AI response", "error", err, "raw_response", resp.Response)
		testRun.Success = false
		testRun.Error = fmt.Sprintf("parse error: %v", err)
		_ = db.SaveAITestRun(ctx, testRun)
		fmt.Printf("Failed to parse AI response: %v\nRaw response: %s\n", err, resp.Response)
		os.Exit(1)
	}

	testRun.Success = true
	if err := db.SaveAITestRun(ctx, testRun); err != nil {
		slog.Error("Failed to save test run to database", "error", err)
	}

	// Output structured result
	fmt.Println("--- AI Evaluation Result ---")
	fmt.Printf("Score:          %d\n", eval.Score)
	fmt.Printf("Suitable:       %t\n", eval.Suitable)
	fmt.Printf("Category:       %s\n", eval.Category)
	
	fmt.Println("Reasons:")
	for _, r := range eval.Reasons {
		fmt.Printf("  - %s\n", r)
	}
	
	fmt.Printf("Draft response: %s\n", eval.DraftResponse)
	fmt.Printf("Duration:       %d ms\n", durationMs)
	
	rawJSON, _ := json.MarshalIndent(eval, "", "  ")
	fmt.Printf("\nRaw structured JSON:\n%s\n", string(rawJSON))
}

func getKworkSource(cfg *config.Config) (*kwork.KworkProjectSource, error) {
	if cfg.KworkLogin == "" || cfg.KworkPassword == "" {
		return nil, fmt.Errorf("KWORK_LOGIN and KWORK_PASSWORD must be set")
	}

	return kwork.NewKworkProjectSource(
		cfg.KworkLogin,
		cfg.KworkPassword,
		cfg.KworkPhoneLast,
	), nil
}

func runKworkHealth(ctx context.Context, cfg *config.Config) {
	fmt.Println("Testing Kwork API Health (Auth)...")
	src, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Error configuring Kwork source: %v\n", err)
		os.Exit(1)
	}

	id, username, err := src.GetMe(ctx)
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("KWORK AUTH OK")
	fmt.Printf("USER %s\n", username)
	fmt.Printf("USER_ID %d\n", id)
}

func runKworkCategories(ctx context.Context, cfg *config.Config) {
	fmt.Println("Fetching Kwork categories...")
	src, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Error configuring Kwork source: %v\n", err)
		os.Exit(1)
	}

	cats, err := src.FetchCategories(ctx)
	if err != nil {
		fmt.Printf("Failed to fetch categories: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Found %d categories/subcategories.\n", len(cats))
	// Just print first 10
	count := 0
	for id, name := range cats {
		fmt.Printf("[%d] %s\n", id, name)
		count++
		if count >= 10 {
			fmt.Println("... (truncating)")
			break
		}
	}
}

func runKworkFetch(ctx context.Context, cfg *config.Config, db *database.DB) {
	fmt.Println("Fetching recent Kwork projects...")
	src, err := getKworkSource(cfg)
	if err != nil {
		fmt.Printf("Error configuring Kwork source: %v\n", err)
		os.Exit(1)
	}

	limit := cfg.KworkPollLimit
	if limit == 0 {
		limit = 50
	}

	projects, err := src.FetchProjects(ctx, limit)
	if err != nil {
		fmt.Printf("Failed to fetch projects: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Fetched %d projects from API.\n", len(projects))

	newCount := 0
	knownCount := 0

	for _, p := range projects {
		isNew, err := db.IsProjectNew(ctx, p.Source, p.ExternalID)
		if err != nil {
			fmt.Printf("DB error checking project %d: %v\n", p.ExternalID, err)
			continue
		}

		if isNew {
			newCount++
		} else {
			knownCount++
		}

		_, err = db.UpsertProject(ctx, p)
		if err != nil {
			fmt.Printf("DB error upserting project %d: %v\n", p.ExternalID, err)
		}
	}

	fmt.Printf("Saved to DB. New: %d, Already known: %d\n", newCount, knownCount)
}

func runProjects(ctx context.Context, db *database.DB) {
	limit := 10
	projects, err := db.GetRecentProjects(ctx, limit)
	if err != nil {
		fmt.Printf("Failed to read projects from DB: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Last %d projects from DB:\n\n", len(projects))
	for _, p := range projects {
		budget := "Negotiable"
		if p.BudgetFrom.Valid {
			budget = fmt.Sprintf("%.2f %s", p.BudgetFrom.Float64, p.Currency.String)
			if p.BudgetTo.Valid && p.BudgetTo.Float64 > p.BudgetFrom.Float64 {
				budget = fmt.Sprintf("%.2f - %.2f %s", p.BudgetFrom.Float64, p.BudgetTo.Float64, p.Currency.String)
			}
		}

		fmt.Printf("[%d] ID: %d | %s\n", p.ID, p.ExternalID, p.Title)
		fmt.Printf("    Budget: %s | Offers: %d\n", budget, p.OffersCount)
		if p.CategoryID.Valid {
			fmt.Printf("    Category: %d\n", p.CategoryID.Int64)
		}
		fmt.Println("    " + p.URL)
		fmt.Println()
	}
}

func runProjectDetails(ctx context.Context, db *database.DB, extIDStr string, showRaw bool) {
	var extID int64
	_, err := fmt.Sscanf(extIDStr, "%d", &extID)
	if err != nil {
		fmt.Println("Invalid project ID")
		os.Exit(1)
	}

	p, err := db.GetProjectByExternalID(ctx, "kwork", extID)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Project not found in DB")
		} else {
			fmt.Printf("DB error: %v\n", err)
		}
		os.Exit(1)
	}

	fmt.Printf("Title:       %s\n", p.Title)
	fmt.Printf("URL:         %s\n", p.URL)
	if p.BudgetFrom.Valid {
		fmt.Printf("Budget From: %.2f\n", p.BudgetFrom.Float64)
	}
	if p.BudgetTo.Valid {
		fmt.Printf("Budget To:   %.2f\n", p.BudgetTo.Float64)
	}
		fmt.Printf("Offers:      %d\n", p.OffersCount)
	fmt.Printf("Buyer:       %s\n", p.BuyerName.String)
	fmt.Printf("\nDescription:\n%s\n", p.Description)

	if showRaw && p.RawJSON != "" {
		fmt.Printf("\nRaw JSON:\n%s\n", p.RawJSON)
	}
}

func runEvaluate(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient, extIDStr string, force bool) {
	extID, err := strconv.ParseInt(extIDStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid external ID: %s\n", extIDStr)
		os.Exit(1)
	}

	p, err := db.GetProjectByExternalID(ctx, "kwork", extID)
	if err != nil {
		fmt.Printf("Failed to get project: %v\n", err)
		os.Exit(1)
	}

	promptVersion := "evaluation-v2"
	
	if !force {
		hasEval, err := db.HasEvaluation(ctx, p.ID, cfg.OllamaModel, promptVersion)
		if err != nil {
			fmt.Printf("Error checking existing evaluation: %v\n", err)
			os.Exit(1)
		}
		if hasEval {
			fmt.Printf("Project already evaluated with %s / %s. Use --force to re-evaluate.\n", cfg.OllamaModel, promptVersion)
			os.Exit(0)
		}
	} else {
		db.DeleteEvaluation(ctx, p.ID, cfg.OllamaModel, promptVersion)
	}

	fmt.Printf("Evaluating Project %d: %s\n", p.ExternalID, p.Title)
	startTime := time.Now()

	evaluator := evaluation.NewEvaluator(aiClient, cfg.OllamaModel, promptVersion)
	eval, err := evaluator.EvaluateProject(ctx, p)
	if err != nil {
		fmt.Printf("Evaluation failed: %v\n", err)
		os.Exit(1)
	}
	
	duration := time.Since(startTime)

	err = db.SaveEvaluation(ctx, eval)
	if err != nil {
		fmt.Printf("Failed to save evaluation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- EVALUATION RESULT ---")
	fmt.Printf("PROJECT:    %d\n", p.ExternalID)
	fmt.Printf("CATEGORY:   %s\n", eval.Category)
	fmt.Printf("SCORE:      %d\n", eval.Score)
	fmt.Printf("SUITABLE:   %t\n", eval.Suitable)
	fmt.Printf("COMPLEXITY: %s\n", eval.Complexity)
	fmt.Printf("BUDGET:     %s\n", eval.BudgetAssessment)
	fmt.Printf("RISK:       %s\n", eval.RiskLevel)
	fmt.Printf("EFFORT:     %s\n", eval.EstimatedEffort)
	fmt.Printf("SUMMARY:    %s\n", eval.Summary)
	fmt.Println("REASONS:")
	for _, r := range eval.Reasons {
		fmt.Printf(" - %s\n", r)
	}
	if len(eval.Warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, w := range eval.Warnings {
			fmt.Printf(" - %s\n", w)
		}
	}
	fmt.Printf("AI DURATION: %v\n", duration)
}

func runEvaluateBatch(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient, limit int) {
	promptVersion := "evaluation-v2"
	projects, err := db.GetUnevaluatedProjects(ctx, cfg.OllamaModel, promptVersion, limit)
	if err != nil {
		fmt.Printf("Failed to get unevaluated projects: %v\n", err)
		os.Exit(1)
	}

	if len(projects) == 0 {
		fmt.Println("No unevaluated projects found.")
		return
	}

	fmt.Printf("Starting batch evaluation for %d projects...\n", len(projects))

	evaluator := evaluation.NewEvaluator(aiClient, cfg.OllamaModel, promptVersion)

	var website, telegram, skip, errors int
	var totalDuration time.Duration

	for i, p := range projects {
		fmt.Printf("[%d/%d] Project %d -> ", i+1, len(projects), p.ExternalID)
		start := time.Now()
		
		eval, err := evaluator.EvaluateProject(ctx, p)
		if err != nil {
			fmt.Printf("ERROR: %v\n", err)
			errors++
			continue
		}
		
		dur := time.Since(start)
		totalDuration += dur

		err = db.SaveEvaluation(ctx, eval)
		if err != nil {
			fmt.Printf("DB ERROR: %v\n", err)
			errors++
			continue
		}

		fmt.Printf("%s %d (%.1fs)\n", eval.Category, eval.Score, dur.Seconds())

		switch eval.Category {
		case "website":
			website++
		case "telegram":
			telegram++
		case "skip":
			skip++
		}

		// Throttle slightly
		if i < len(projects)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	fmt.Println("\nSummary:")
	fmt.Printf("Processed: %d\n", len(projects))
	fmt.Printf("Website:   %d\n", website)
	fmt.Printf("Telegram:  %d\n", telegram)
	fmt.Printf("Skip:      %d\n", skip)
	fmt.Printf("Errors:    %d\n", errors)
	if len(projects)-errors > 0 {
		avg := totalDuration / time.Duration(len(projects)-errors)
		fmt.Printf("Average duration: %.1fs\n", avg.Seconds())
	}
}

func runEvaluations(ctx context.Context, db *database.DB, args []string) {
	filter := domain.EvaluationFilter{Limit: 20}
	
	for i := 0; i < len(args); i++ {
		if args[i] == "--category" && i+1 < len(args) {
			filter.Category = args[i+1]
			i++
		} else if args[i] == "--min-score" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &filter.MinScore)
			i++
		} else if args[i] == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &filter.Limit)
			i++
		}
	}

	evals, err := db.GetEvaluations(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to get evaluations: %v\n", err)
		os.Exit(1)
	}

	for _, e := range evals {
		fmt.Printf("[%d] %s\n", e.Score, strings.ToUpper(e.Category))
		fmt.Printf("%s\n", e.Summary)
		fmt.Printf("Complexity: %s | Risk: %s\n", e.Complexity, e.RiskLevel)
		fmt.Println("--------------------------------")
	}
}

func runEvaluation(ctx context.Context, db *database.DB, extIDStr string, showRaw bool) {
	extID, err := strconv.ParseInt(extIDStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid external ID: %s\n", extIDStr)
		os.Exit(1)
	}

	e, err := db.GetEvaluationByExternalID(ctx, "kwork", extID)
	if err != nil {
		fmt.Printf("Failed to get evaluation: %v\n", err)
		os.Exit(1)
	}
	if e == nil {
		fmt.Printf("Evaluation not found for project %d\n", extID)
		os.Exit(1)
	}

	fmt.Printf("PROJECT ID: %d\n", e.ProjectID)
	fmt.Printf("CATEGORY:   %s\n", e.Category)
	fmt.Printf("SCORE:      %d\n", e.Score)
	fmt.Printf("SUITABLE:   %t\n", e.Suitable)
	fmt.Printf("COMPLEXITY: %s\n", e.Complexity)
	fmt.Printf("BUDGET:     %s\n", e.BudgetAssessment)
	fmt.Printf("RISK:       %s\n", e.RiskLevel)
	fmt.Printf("EFFORT:     %s\n", e.EstimatedEffort)
	fmt.Printf("SUMMARY:    %s\n", e.Summary)
	fmt.Println("REASONS:")
	for _, r := range e.Reasons {
		fmt.Printf(" - %s\n", r)
	}
	if len(e.Warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, w := range e.Warnings {
			fmt.Printf(" - %s\n", w)
		}
	}
	
	if showRaw {
		raw, _ := json.MarshalIndent(e, "", "  ")
		fmt.Printf("\nRAW EVALUATION:\n%s\n", string(raw))
	}
}

func runProposalDraft(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient, extIDStr string, force bool) {
	extID, err := strconv.ParseInt(extIDStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid external ID: %s\n", extIDStr)
		os.Exit(1)
	}

	p, err := db.GetProjectByExternalID(ctx, "kwork", extID)
	if err != nil {
		fmt.Printf("Failed to get project: %v\n", err)
		os.Exit(1)
	}

	eval, err := db.GetEvaluationByExternalID(ctx, "kwork", extID)
	if err != nil || eval == nil {
		fmt.Printf("Evaluation not found. Run evaluate first.\n")
		os.Exit(1)
	}

	promptVersion := "proposal-v3"

	if !force {
		hasDraft, err := db.HasProposalDraft(ctx, eval.ID, cfg.OllamaModel, promptVersion)
		if err != nil {
			fmt.Printf("Error checking draft: %v\n", err)
			os.Exit(1)
		}
		if hasDraft {
			fmt.Println("Already generated. Use --force to regenerate.")
			os.Exit(0)
		}
	} else {
		db.DeleteProposalDraft(ctx, eval.ID, cfg.OllamaModel, promptVersion)
	}

	fmt.Printf("Generating Proposal Draft for Project %d: %s\n", p.ExternalID, p.Title)
	startTime := time.Now()

	generator := proposal.NewGenerator(db, aiClient.(*ai.OllamaClient), cfg.OllamaModel)
	draft, err := generator.Generate(ctx, p, *eval, promptVersion)
	if err != nil {
		fmt.Printf("Proposal generation failed: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(startTime)

	err = db.SaveProposalDraft(ctx, draft)
	if err != nil {
		fmt.Printf("Failed to save proposal draft: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("--- PROPOSAL DRAFT ---")
	fmt.Printf("PROJECT:  %d\n", p.ExternalID)
	fmt.Printf("CATEGORY: %s\n", eval.Category)
	fmt.Printf("SCORE:    %d\n\n", eval.Score)
	fmt.Printf("PROPOSAL:\n%s\n\n", draft.Proposal)
	if draft.Question != "" {
		fmt.Printf("QUESTION:\n%s\n\n", draft.Question)
	}
	fmt.Printf("INTERNAL APPROACH:\n%s\n\n", draft.Approach)
	fmt.Printf("CONFIDENCE: %s\n", draft.Confidence)
	if len(draft.Warnings) > 0 {
		fmt.Println("WARNINGS:")
		for _, w := range draft.Warnings {
			fmt.Printf(" - %s\n", w)
		}
	}
	fmt.Printf("\nAI DURATION: %v\n", duration)
}

func runProposalBatch(ctx context.Context, cfg *config.Config, db *database.DB, aiClient ai.AIClient, limit int) {
	promptVersion := "proposal-v3"
	evalPromptVersion := "evaluation-v2"
	projects, err := db.GetProjectsForProposal(ctx, cfg.OllamaModel, evalPromptVersion, promptVersion, limit)
	if err != nil {
		fmt.Printf("Failed to get projects for proposal: %v\n", err)
		os.Exit(1)
	}

	if len(projects) == 0 {
		fmt.Println("No projects available for proposal generation.")
		return
	}

	fmt.Printf("Starting batch proposal generation for %d projects...\n", len(projects))

	generator := proposal.NewGenerator(db, aiClient.(*ai.OllamaClient), cfg.OllamaModel)

	var generated, skipped, errors int
	var totalDuration time.Duration

	for i, p := range projects {
		eval, err := db.GetEvaluationByExternalID(ctx, "kwork", p.ExternalID)
		if err != nil || eval == nil {
			fmt.Printf("[%d/%d] #%d -> ERROR: missing evaluation\n", i+1, len(projects), p.ExternalID)
			errors++
			continue
		}

		fmt.Printf("[%d/%d] #%d %s %d -> ", i+1, len(projects), p.ExternalID, strings.ToUpper(eval.Category), eval.Score)
		start := time.Now()

		draft, err := generator.Generate(ctx, p, *eval, promptVersion)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			errors++
			continue
		}
		
		dur := time.Since(start)
		totalDuration += dur

		err = db.SaveProposalDraft(ctx, draft)
		if err != nil {
			fmt.Printf("DB error: %v\n", err)
			errors++
			continue
		}

		fmt.Printf("draft generated (%.1fs)\n", dur.Seconds())
		generated++

		if i < len(projects)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	fmt.Println("\nSummary:")
	fmt.Printf("Generated: %d\n", generated)
	fmt.Printf("Skipped:   %d\n", skipped)
	fmt.Printf("Errors:    %d\n", errors)
	if generated > 0 {
		avg := totalDuration / time.Duration(generated)
		fmt.Printf("Average duration: %.1fs\n", avg.Seconds())
	}
}

func runProposals(ctx context.Context, db *database.DB, args []string) {
	filter := domain.EvaluationFilter{Limit: 20}
	
	for i := 0; i < len(args); i++ {
		if args[i] == "--category" && i+1 < len(args) {
			filter.Category = args[i+1]
			i++
		} else if args[i] == "--min-score" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &filter.MinScore)
			i++
		} else if args[i] == "--limit" && i+1 < len(args) {
			fmt.Sscanf(args[i+1], "%d", &filter.Limit)
			i++
		}
	}

	drafts, err := db.GetProposalDrafts(ctx, filter)
	if err != nil {
		fmt.Printf("Failed to get proposal drafts: %v\n", err)
		os.Exit(1)
	}

	for _, d := range drafts {
		_, _ = db.GetEvaluationByExternalID(ctx, "kwork", d.ProjectID)
		// Wait, db.GetProposalDrafts returns d.ProjectID. We should probably query project to get title.
		// Let's do it simply here.
		fmt.Printf("DRAFT ID: %d | PROJECT INTERNAL ID: %d | CONFIDENCE: %s\n", d.ID, d.ProjectID, d.Confidence)
		fmt.Printf("%s\n", d.Proposal)
		fmt.Println("--------------------------------")
	}
}

func runProposal(ctx context.Context, db *database.DB, extIDStr string) {
	extID, err := strconv.ParseInt(extIDStr, 10, 64)
	if err != nil {
		fmt.Printf("Invalid external ID: %s\n", extIDStr)
		os.Exit(1)
	}

	d, err := db.GetProposalDraftByExternalID(ctx, "kwork", extID)
	if err != nil {
		fmt.Printf("Failed to get proposal: %v\n", err)
		os.Exit(1)
	}
	if d == nil {
		fmt.Printf("Proposal not found for project %d\n", extID)
		os.Exit(1)
	}

	fmt.Printf("PROJECT ID:   %d\n", extID)
	fmt.Printf("CONFIDENCE:   %s\n", d.Confidence)
	fmt.Printf("\nPROPOSAL:\n%s\n", d.Proposal)
	if d.Question != "" {
		fmt.Printf("\nQUESTION:\n%s\n", d.Question)
	}
	fmt.Printf("\nAPPROACH:\n%s\n", d.Approach)
	if len(d.Warnings) > 0 {
		fmt.Println("\nWARNINGS:")
		for _, w := range d.Warnings {
			fmt.Printf(" - %s\n", w)
		}
	}
}
