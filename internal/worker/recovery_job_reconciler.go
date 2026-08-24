package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ufo-observation-command/internal/domain"
)

type RecoveryJobExpiryRepository interface {
	ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error)
}

type RecoveryJobExpiryReconciler struct {
	repo    RecoveryJobExpiryRepository
	log     *slog.Logger
	metrics *Metrics
}

func NewRecoveryJobExpiryReconciler(repo RecoveryJobExpiryRepository, logger *slog.Logger, metrics *Metrics) *RecoveryJobExpiryReconciler {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &RecoveryJobExpiryReconciler{repo: repo, log: logger, metrics: metrics}
}

func (r *RecoveryJobExpiryReconciler) Reconcile(ctx context.Context, now time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r.repo == nil {
		return fmt.Errorf("task expiry repository is nil")
	}
	r.metrics.RecordRun()
	items, err := r.repo.ExpiringRecoveryJobs(ctx, now, 100)
	if err != nil {
		r.metrics.RecordFailure()
		r.metrics.RecordFailedDue0019(len(items))
		return err
	}
	r.metrics.RecordDue(len(items))
	for _, item := range items {
		r.log.Warn("task is near expiry", "recovery_job_id", item.ID, "task_code", item.TaskCode, "expires_at", item.ExpiresAt)
	}
	return nil
}
