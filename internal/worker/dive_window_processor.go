package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

type DiveWindowJob struct {
	ID          string
	Attempts    int
	MaxAttempts int
}
type DiveWindowExecutor interface {
	Execute(context.Context, DiveWindowJob) error
}

type DiveWindowProcessor struct {
	executor DiveWindowExecutor
	policy   RetryPolicy
	logger   *slog.Logger
	metrics  *Metrics
}

func NewDiveWindowProcessor(executor DiveWindowExecutor, policy RetryPolicy, logger *slog.Logger, metrics *Metrics) *DiveWindowProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &DiveWindowProcessor{executor: executor, policy: policy, logger: logger, metrics: metrics}
}

func (p *DiveWindowProcessor) Process(ctx context.Context, job DiveWindowJob) error {
	if job.ID == "" {
		return fmt.Errorf("dive_window job id is required")
	}
	if p.executor == nil {
		return fmt.Errorf("dive_window executor is nil")
	}
	policy := p.policy
	if job.MaxAttempts > 0 && policy.Attempts > job.MaxAttempts {
		policy.Attempts = job.MaxAttempts
	}
	start := time.Now()
	err := RunWithRetry(ctx, policy, func(callCtx context.Context) error { job.Attempts++; return p.executor.Execute(callCtx, job) })
	p.metrics.RecordRun()
	if err != nil {
		p.metrics.RecordFailure()
		p.logger.Error("dive_window job failed", "dive_window_id", job.ID, "attempts", job.Attempts, "duration", time.Since(start), "error", err)
		return err
	}
	p.logger.Info("dive_window job completed", "dive_window_id", job.ID, "attempts", job.Attempts, "duration", time.Since(start))
	return nil
}
