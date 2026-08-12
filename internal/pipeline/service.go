package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"kwork-assistant/internal/ai"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/evaluation"
	"kwork-assistant/internal/intake"
	"kwork-assistant/internal/proposal"
	"kwork-assistant/internal/telegram"
)

type Config struct {
	Model         string
	SystemPrompt  string
	ProposalPrompt string
}

type TelegramNotifier interface {
	SendMessageWithButton(chatID int64, text string, buttonText, url string) (int64, error)
	GetOwnerChatID() int64
}

type Service struct {
	db        *database.DB
	aiClient  ai.AIClient
	evaluator *evaluation.Evaluator
	generator *proposal.Generator
	bot       TelegramNotifier
	cfg       Config
	logger    *slog.Logger
}

func NewService(db *database.DB, aiClient ai.AIClient, bot TelegramNotifier, cfg Config, logger *slog.Logger) *Service {
	evaluator := evaluation.NewEvaluator(aiClient, cfg.Model, cfg.SystemPrompt)
	generator := proposal.NewGenerator(db, aiClient, cfg.Model)
	return &Service{
		db:        db,
		evaluator: evaluator,
		generator: generator,
		bot:       bot,
		logger:    logger.With("component", "pipeline"),
		cfg:       cfg,
	}
}

func (s *Service) ProcessPending(ctx context.Context, limit int) error {
	emails, err := s.db.GetPendingInboundEmails(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to get pending emails: %w", err)
	}
	
	if len(emails) == 0 {
		return nil
	}

	for _, e := range emails {
		err := s.processSingleEmail(ctx, e)
		if err != nil {
			s.logger.Error("Failed to process email", "id", e.ID, "error", err)
			// Generally we treat errors as retryable (AI temp failure, network error, telegram API error)
			s.db.UpdateInboundEmailStatus(ctx, e.ID, domain.ProcessingStatusRetryableError, err.Error())
		}
	}
	return nil
}

func (s *Service) processSingleEmail(ctx context.Context, e domain.InboundEmail) error {
	s.logger.Info("Processing email", "id", e.ID, "subject", e.Subject)

	// 1. Parse Email
	candidate, isKwork := intake.ParseKworkEmail(e)
	if !isKwork {
		s.logger.Info("Ignoring non-kwork email", "id", e.ID)
		return s.db.UpdateInboundEmailStatus(ctx, e.ID, domain.ProcessingStatusIgnored, "not a kwork project email")
	}

	// 2. Project Upsert
	if candidate.ParseConfidence == domain.ParseConfidenceLow {
		s.logger.Warn("Low confidence parsing", "id", e.ID, "missing", candidate.MissingFields)
		return s.db.UpdateInboundEmailStatus(ctx, e.ID, domain.ProcessingStatusTerminalError, "low confidence parse")
	}

	p := domain.Project{
		ExternalID:  candidate.ExternalID,
		Source:      "kwork_email",
		URL:         candidate.URL,
		Title:       candidate.Title,
		Description: candidate.Description,
		BudgetFrom:  sql.NullFloat64{Float64: candidate.BudgetFrom, Valid: candidate.BudgetFrom > 0},
		BudgetTo:    sql.NullFloat64{Float64: candidate.BudgetTo, Valid: candidate.BudgetTo > 0},
		Currency:    sql.NullString{String: candidate.Currency, Valid: candidate.Currency != ""},
		PublishedAt: time.Now(), // We use email received time or parse time
		FetchedAt:   time.Now(),
	}

	isNew, err := s.db.IsProjectNew(ctx, p.Source, p.ExternalID)
	if err != nil {
		return fmt.Errorf("failed to check project existence: %w", err)
	}

	_, err = s.db.UpsertProject(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to upsert project: %w", err)
	}

	// Retrieve actual inserted project with ID
	p, err = s.db.GetProjectByExternalID(ctx, p.Source, p.ExternalID)
	if err != nil {
		return fmt.Errorf("failed to reload project: %w", err)
	}

	s.db.UpdateInboundEmailStatus(ctx, e.ID, domain.ProcessingStatusParsed, "")

	if !isNew {
		s.logger.Info("Project already known", "ext_id", p.ExternalID)
	}

	// 3. Evaluation
	hasEval, err := s.db.HasEvaluation(ctx, p.ID, s.cfg.Model, s.cfg.SystemPrompt)
	if err != nil {
		return fmt.Errorf("failed to check evaluation: %w", err)
	}

	var eval domain.ProjectEvaluation
	if !hasEval {
		s.logger.Info("Evaluating project", "project_id", p.ID)
		e, err := s.evaluator.EvaluateProject(ctx, p)
		if err != nil {
			return fmt.Errorf("evaluation failed: %w", err)
		}
		
		err = s.db.SaveEvaluation(ctx, e)
		if err != nil {
			return fmt.Errorf("failed to save evaluation: %w", err)
		}
		
		// Reload evaluation
		evalPtr, err := s.db.GetEvaluationByExternalID(ctx, p.Source, p.ExternalID)
		if err != nil || evalPtr == nil {
			return fmt.Errorf("failed to reload evaluation")
		}
		eval = *evalPtr
	} else {
		s.logger.Info("Evaluation already exists", "project_id", p.ID)
		evalPtr, err := s.db.GetEvaluationByExternalID(ctx, p.Source, p.ExternalID)
		if err != nil || evalPtr == nil {
			return fmt.Errorf("failed to load existing evaluation: %w", err)
		}
		eval = *evalPtr
	}

	// 4. Generate Proposal
	if eval.Suitable {
		hasDraft, err := s.db.HasProposalDraft(ctx, eval.ID, s.cfg.Model, s.cfg.ProposalPrompt)
		if err != nil {
			return fmt.Errorf("failed to check proposal draft: %w", err)
		}

		var draft domain.ProposalDraft
		if !hasDraft {
			s.logger.Info("Generating proposal draft", "evaluation_id", eval.ID)
			d, err := s.generator.Generate(ctx, p, eval, s.cfg.ProposalPrompt)
			if err != nil {
				// Proposal generation failed (e.g. validator rejected)
				return fmt.Errorf("proposal generation failed: %w", err)
			}
			err = s.db.SaveProposalDraft(ctx, d)
			if err != nil {
				return fmt.Errorf("failed to save proposal draft: %w", err)
			}

			draftPtr, err := s.db.GetProposalDraftByExternalID(ctx, p.Source, p.ExternalID)
			if err != nil || draftPtr == nil {
				return fmt.Errorf("failed to reload draft: %w", err)
			}
			draft = *draftPtr
		} else {
			s.logger.Info("Proposal draft already exists", "evaluation_id", eval.ID)
			draftPtr, err := s.db.GetProposalDraftByExternalID(ctx, p.Source, p.ExternalID)
			if err != nil || draftPtr == nil {
				return fmt.Errorf("failed to load existing draft: %w", err)
			}
			draft = *draftPtr
		}

		// 5. Send Telegram Notification
		hasNotif, err := s.db.HasTelegramNotification(ctx, p.ID, domain.NotificationTypeSuitable)
		if err != nil {
			return fmt.Errorf("failed to check telegram notification: %w", err)
		}

		if !hasNotif {
			text := telegram.FormatSuitableProject(p, eval, draft)
			
			s.logger.Info("Sending telegram notification", "project_id", p.ID)
			msgID, err := s.bot.SendMessageWithButton(s.bot.GetOwnerChatID(), text, "Открыть Kwork", p.URL)
			
			notif := domain.TelegramNotification{
				ProjectID:         p.ID,
				EvaluationID:      eval.ID,
				ProposalDraftID:   draft.ID,
				NotificationType:  domain.NotificationTypeSuitable,
				ChatID:            s.bot.GetOwnerChatID(),
				TelegramMessageID: msgID,
				Status:            domain.NotificationStatusSent,
			}
			
			if err != nil {
				s.logger.Error("Failed to send telegram notification", "error", err)
				notif.Status = domain.NotificationStatusFailed
				notif.Error = err.Error()
			} else {
				notif.SentAt = time.Now()
			}
			
			s.db.SaveTelegramNotification(ctx, &notif)
		}
	}

	return s.db.UpdateInboundEmailStatus(ctx, e.ID, domain.ProcessingStatusProcessed, "")
}
