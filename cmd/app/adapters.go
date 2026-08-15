package main

import (
	"context"
	"database/sql"
	"fmt"

	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/proposal"
	"kwork-assistant/internal/reply"
)

type ReplyGeneratorAdapter struct {
	Gen *reply.Generator
	DB  *database.DB
}

func (a *ReplyGeneratorAdapter) GenerateDraft(ctx context.Context, conversationID int64, latestMessageID int64) error {
	history, err := a.DB.GetUnprocessedKworkMessages(ctx, conversationID)
	if err != nil {
		return err
	}

	var proj *domain.Project
	var projID sql.NullInt64
	if err := a.DB.QueryRowContext(ctx, "SELECT project_id FROM kwork_conversations WHERE id = ?", conversationID).Scan(&projID); err == nil && projID.Valid {
		proj, _ = a.DB.GetProject(ctx, projID.Int64)
	}

	draft, err := a.Gen.GenerateDraft(ctx, conversationID, latestMessageID, proj, history)
	if err != nil {
		return err
	}

	if draft != nil {
		return a.DB.SaveReplyDraft(ctx, draft)
	}
	return nil
}

type ProposalGeneratorAdapter struct {
	Gen           *proposal.Generator
	DB            *database.DB
	Model         string
	PromptVersion string
}

func (a *ProposalGeneratorAdapter) GenerateDraft(ctx context.Context, projectID int64, forceRegenerate bool) (string, error) {
	proj, err := a.DB.GetProject(ctx, projectID)
	if err != nil {
		return "", err
	}

	eval, err := a.DB.GetEvaluationByExternalID(ctx, proj.Source, proj.ExternalID)
	if err != nil {
		return "", fmt.Errorf("evaluation not found for project %d: %v", projectID, err)
	}

	// Check for existing draft if not forcing regenerate
	if !forceRegenerate {
		draft, err := a.DB.GetProposalDraftByExternalID(ctx, proj.Source, proj.ExternalID)
		if err == nil && draft != nil && draft.EvaluationID == eval.ID && draft.PromptVersion == a.PromptVersion {
			res := draft.Proposal
			if draft.Question != "" {
				res += "\n\n❓ " + draft.Question
			}
			return res, nil
		}
	}

	draft, err := a.Gen.Generate(ctx, *proj, *eval, a.PromptVersion)
	if err != nil {
		return "", err
	}

	err = a.DB.SaveProposalDraft(ctx, draft)
	if err != nil {
		return "", err
	}

	res := draft.Proposal
	if draft.Question != "" {
		res += "\n\n❓ " + draft.Question
	}
	return res, nil
}
