package worker

import (
	"context"
	"log/slog"
	"time"
)

type Reconciler interface {
	Reconcile(context.Context, time.Time) error
}

type Periodic struct {
	interval time.Duration
	run      Reconciler
	log      *slog.Logger
}

func NewPeriodic(interval time.Duration, run Reconciler, logger *slog.Logger) *Periodic {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Periodic{interval: interval, run: run, log: logger}
}

func (p *Periodic) Run(ctx context.Context) error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if p.run != nil {
			if err := p.run.Reconcile(ctx, time.Now().UTC()); err != nil && ctx.Err() == nil {
				p.log.Error("periodic reconcile failed", "error", err)
			}
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
