package projectwatch

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"kwork-assistant/internal/config"
	"kwork-assistant/internal/database"
	"kwork-assistant/internal/domain"
	"kwork-assistant/internal/kwork"
)

type RunStats struct {
	ProjectsFetched      int
	NewProjects          int
	AlreadyKnown         int
	Evaluated            int
	SkippedEvaluation    int
	Website              int
	Telegram             int
	Skip                 int
	SuitableAbove80      int
	TelegramNotifsSent   int
	TelegramDupesSkipped int
	ProposalGenerations  int
}

type Notifier interface {
	SendSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation) (int, error)
	UpdateSuitableProjectNotification(project *domain.Project, eval *domain.ProjectEvaluation, messageID int) error
}

type Evaluator interface {
	EvaluateProject(ctx context.Context, p domain.Project) (domain.ProjectEvaluation, error)
}

type ProjectSource interface {
	FetchProjects(ctx context.Context, limit int) ([]domain.Project, error)
}

type Watcher struct {
	cfg                 *config.Config
	db                  *database.DB
	source              ProjectSource
	evaluator           Evaluator
	model               string
	promptVer           string
	notifier            Notifier
	logger              *slog.Logger
	urlResolveCooldowns map[int64]time.Time
}

func NewWatcher(cfg *config.Config, db *database.DB, source ProjectSource, evaluator Evaluator, model, promptVer string, notifier Notifier, logger *slog.Logger) *Watcher {
	return &Watcher{
		cfg:                 cfg,
		db:                  db,
		source:              source,
		evaluator:           evaluator,
		model:               model,
		promptVer:           promptVer,
		notifier:            notifier,
		logger:              logger.With("component", "projectwatch"),
		urlResolveCooldowns: make(map[int64]time.Time),
	}
}

func (w *Watcher) Run(ctx context.Context) (RunStats, error) {
	var stats RunStats
	if !w.cfg.KworkProjectWatchEnabled {
		return stats, nil
	}

	projects, err := w.source.FetchProjects(ctx, w.cfg.KworkPollLimit)
	if err != nil {
		return stats, fmt.Errorf("failed to fetch projects: %w", err)
	}

	w.logger.Info("Fetched projects for watch", "count", len(projects))
	stats.ProjectsFetched = len(projects)

	for _, p := range projects {
		isNew, err := w.db.UpsertProject(ctx, p)
		if err != nil {
			w.logger.Error("Failed to upsert project", "id", p.ExternalID, "error", err)
			continue
		}

		if isNew {
			stats.NewProjects++
		} else {
			stats.AlreadyKnown++
		}
		
		// We need the ID of the newly saved (or existing) project for evaluation
		savedProj, err := w.db.GetProjectByExternalID(ctx, p.Source, p.ExternalID)
		if err != nil {
			w.logger.Error("Failed to get saved project", "id", p.ExternalID, "error", err)
			continue
		}
		
		if savedProj.URL == "" {
			shouldResolve := false
			if isNew {
				shouldResolve = true
			} else {
				lastAttempt, ok := w.urlResolveCooldowns[savedProj.ID]
				if !ok || time.Since(lastAttempt) > 30*time.Minute {
					shouldResolve = true
				}
			}

			if shouldResolve {
				w.urlResolveCooldowns[savedProj.ID] = time.Now()
				if resolvedURL := kwork.ResolveProjectURL(ctx, savedProj); resolvedURL != "" {
					savedProj.URL = resolvedURL
					w.db.UpsertProject(ctx, savedProj)
				}
			}
		}
		
		w.evaluateProject(ctx, &savedProj, &stats)
	}

	return stats, nil
}

func (w *Watcher) evaluateProject(ctx context.Context, p *domain.Project, stats *RunStats) {
	// Check if already evaluated with the current model/prompt AND matching input hash
	eval, err := w.db.GetEvaluationByExternalID(ctx, p.Source, p.ExternalID)
	
	hasEval := err == nil && eval != nil && eval.Model == w.model && eval.PromptVersion == w.promptVer && eval.InputHash == p.InputHash()

	var newEvalGenerated bool
	if hasEval {
		if stats != nil {
			stats.SkippedEvaluation++
		}
	} else {
		newEvalGenerated = true
		newEval, err := w.evaluator.EvaluateProject(ctx, *p)
		if err != nil {
			w.logger.Error("Failed to evaluate project", "id", p.ExternalID, "error", err)
			return
		}
		eval = &newEval
		
		if stats != nil {
			stats.Evaluated++
			if eval.Category == "website" {
				stats.Website++
			} else if eval.Category == "telegram" {
				stats.Telegram++
			} else {
				stats.Skip++
			}
		}

		err = w.db.SaveEvaluation(ctx, *eval)
		if err != nil {
			w.logger.Error("Failed to save evaluation", "id", p.ExternalID, "error", err)
			return
		}
	}

	hasNotif, msgID := w.db.GetTelegramNotificationMessageID(ctx, p.ID, domain.NotificationTypeSuitable)
	isSuitable := eval.Score >= w.cfg.KworkSuitableScore && eval.Suitable && (eval.Category == "website" || eval.Category == "telegram")

	if hasNotif && newEvalGenerated {
		if w.notifier != nil {
			if err := w.notifier.UpdateSuitableProjectNotification(p, eval, msgID); err != nil {
				w.logger.Error("Failed to update notification", "projectID", p.ID, "error", err)
			}
		}
	} else if isSuitable && !hasNotif {
		if stats != nil {
			stats.SuitableAbove80++
		}
		if w.notifier != nil {
			msgID, err := w.notifier.SendSuitableProjectNotification(p, eval)
			if err != nil {
				w.logger.Error("Failed to send notification", "projectID", p.ID, "error", err)
			} else {
				if stats != nil {
					stats.TelegramNotifsSent++
				}
				// Record notification
				_ = w.db.SaveTelegramNotification(ctx, &domain.TelegramNotification{
					ProjectID:        p.ID,
					TelegramMessageID: int64(msgID),
					NotificationType: domain.NotificationTypeSuitable,
					SentAt:           time.Now(),
				})
			}
		}
	} else if isSuitable && hasNotif {
		if stats != nil {
			stats.TelegramDupesSkipped++
		}
	}
}
