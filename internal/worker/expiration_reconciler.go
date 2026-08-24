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
		// The batch failed (e.g. database refused access); nothing is committed, so
		// the affected recovery_jobs remain retryable. Do not account the failed,
		// uncommitted work against the quota here — quota must only change on truly
		// successful work, otherwise retried requests cannot obtain a slot.
		r.metrics.RecordFailure()
		return err
	}
	r.metrics.RecordDue(result.Marked)
	if result.Marked > 0 {
		r.logger.Info("expired recovery_jobs reconciled", "scanned", result.Scanned, "marked", result.Marked)
	}
	return nil
}
