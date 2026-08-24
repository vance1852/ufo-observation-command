package worker

import (
	"context"
	"log/slog"
	"time"
)

type ReportGenerator interface {
	Generate(context.Context, time.Time) error
}

type ReportWorker struct {
	generator ReportGenerator
	interval  time.Duration
	logger    *slog.Logger
}

func NewReportWorker(generator ReportGenerator, interval time.Duration, logger *slog.Logger) *ReportWorker {
	if interval <= 0 {
		interval = time.Hour
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ReportWorker{generator: generator, interval: interval, logger: logger}
}

func (w *ReportWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if w.generator != nil {
			if err := w.generator.Generate(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
				w.logger.Error("report generation failed", "error", err)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
