package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/proposal"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.InitDB("data/kwork-assistant.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	aiClient := ai.NewOllamaClient(cfg.OllamaBaseURL, cfg.AITimeoutSeconds)
	gen := proposal.NewGenerator(db, aiClient, cfg.OllamaModel)

	for _, id := range []int64{3233614, 3233615} {
		proj, err := db.GetProjectByExternalID(context.Background(), "kwork", id)
		if err != nil {
			log.Printf("ERROR getting project %d: %v", id, err)
			continue
		}
		eval, err := db.GetEvaluationByExternalID(context.Background(), proj.Source, proj.ExternalID)
		if err != nil {
			log.Printf("ERROR getting eval %d: %v", id, err)
			continue
		}

		fmt.Printf("=== PROJECT %d: %s ===\n", id, proj.Title)
		draft, err := gen.Generate(context.Background(), proj, *eval, "proposal-v4")
		if err != nil {
			log.Printf("ERROR generating: %v", err)
			continue
		}
		fmt.Printf("PROPOSAL:\n%s\n", strings.TrimSpace(draft.Proposal))
		fmt.Printf("QUESTION: %s\n\n", strings.TrimSpace(draft.Question))
	}
}
