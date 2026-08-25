package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ufo-observation-command/internal/repository"
)

type ExpirationSource interface {
	MarkExpiredRecoveryJobs(context.Context, time.Time, int) (repository.ReconcileResult, error)
}

type ExpirationReconciler struct {
	source  ExpirationSource
	logger  *slog.Logger
	metrics *Metrics
}

func NewExpirationReconciler(source ExpirationSource, logger *slog.Logger, metrics *Metrics) *ExpirationReconciler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &ExpirationReconciler{source: source, logger: logger, metrics: metrics}
}

func (r *ExpirationReconciler) Reconcile(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.source == nil {
		return fmt.Errorf("expiration source is nil")
	}
	r.metrics.RecordRun()
	result, err := r.source.MarkExpiredRecoveryJobs(ctx, now, 100)
	if err != nil {
		r.metrics.RecordFailure()
		// A timed-out write rolled the transaction back, so none of the
		// scanned recovery jobs were persisted. Record them as rolled back so
		// monitoring stays in sync with the post-rollback database state
		// instead of reporting a stale "processed" count.
		rolledBack := result.Scanned - result.Marked
		r.metrics.RecordRolledBack(rolledBack)
		r.logger.Error("expired recovery_jobs write rolled back", "scanned", result.Scanned, "marked", result.Marked, "rolled_back", rolledBack, "error", err)
		return err
	}
	r.metrics.RecordDue(result.Marked)
	if result.Marked > 0 {
		r.logger.Info("expired recovery_jobs reconciled", "scanned", result.Scanned, "marked", result.Marked)
	}
	return nil
}
