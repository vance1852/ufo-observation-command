package worker

import (
	"context"
	"log/slog"
	"time"
)

type AssignmentSource interface {
	ActivateDue(context.Context, time.Time, int) (int, error)
}

type AssignmentWorker struct {
	source   AssignmentSource
	interval time.Duration
	logger   *slog.Logger
	metrics  *Metrics
}

func NewAssignmentWorker(source AssignmentSource, interval time.Duration, logger *slog.Logger, metrics *Metrics) *AssignmentWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	if metrics == nil {
		metrics = &Metrics{}
	}
	return &AssignmentWorker{source: source, interval: interval, logger: logger, metrics: metrics}
}

func (w *AssignmentWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if w.source != nil {
			count, err := w.source.ActivateDue(ctx, time.Now().UTC(), 100)
			w.metrics.RecordRun()
			if err != nil && ctx.Err() == nil {
				w.metrics.RecordFailure()
				w.logger.Error("assignment activation failed", "error", err)
			} else {
				w.metrics.RecordDue(count)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
