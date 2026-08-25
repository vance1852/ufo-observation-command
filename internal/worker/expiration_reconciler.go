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
		// Anomaly: scanned candidates are unconfirmed, so do not promote any
		// of them (not even result.Scanned) into the due/success metric.
		r.metrics.RecordFailure()
		return err
	}
	// Distinguish scanned candidates (unconfirmed inputs) from execution
	// results (confirmed marked tasks). Only confirmed results consume the
	// due/success metric; candidates are tracked separately so cross-site
	// aggregation cannot be distorted by unconfirmed scans.
	r.metrics.RecordScanned(result.Scanned)
	r.metrics.RecordDue(result.Marked)
	if result.Scanned > 0 {
		r.logger.Info("expired recovery_jobs reconciled", "scanned", result.Scanned, "marked", result.Marked, "skipped", result.Skipped, "failed", result.Failed)
	}
	return nil
}
