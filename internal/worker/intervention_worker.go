package worker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ufo-observation-command/internal/service"
)

type IntegrityIncidentWorker struct {
	service  *service.Service
	interval time.Duration
	log      *slog.Logger
}

func NewIntegrityIncidentWorker(svc *service.Service, interval time.Duration, logger *slog.Logger) *IntegrityIncidentWorker {
	if interval <= 0 {
		interval = time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &IntegrityIncidentWorker{service: svc, interval: interval, log: logger}
}

func (w *IntegrityIncidentWorker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if w.service == nil {
			return fmt.Errorf("safety_alert service is nil")
		}
		if err := w.reconcile(ctx); err != nil && ctx.Err() == nil {
			w.log.Error("safety_alert reconciliation failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *IntegrityIncidentWorker) reconcile(ctx context.Context) error {
	items, err := w.service.DueIntegrityIncidents(ctx, time.Now().UTC(), 100)
	if err != nil {
		return err
	}
	for _, item := range items {
		w.log.Warn("safety_alert is due", "recovery_job_id", item.RecoveryJobID, "kind", item.Kind, "due_at", item.DueAt)
	}
	return nil
}
