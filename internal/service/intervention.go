package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) OpenIntegrityIncident(ctx context.Context, meta RequestMeta, in repository.IntegrityIncidentInput) (string, error) {
	if in.DueAt.IsZero() {
		in.DueAt = s.now().Add(72 * time.Hour)
	}
	safety_alert := domain.IntegrityIncident{RecoveryJobID: in.RecoveryJobID, Kind: in.Kind, Status: domain.IntegrityIncidentOpen, Reason: in.Reason, DueAt: in.DueAt}
	if err := safety_alert.Validate(s.now()); err != nil {
		return "", err
	}
	var id string
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		var err error
		id, err = tx.CreateIntegrityIncident(ctx, in)
		if err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "safety_alert", id, "open", "success", nil))
	})
	return id, err
}

func (s *Service) CloseIntegrityIncident(ctx context.Context, meta RequestMeta, id string) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			CloseIntegrityIncident(context.Context, string, time.Time) error
		})
		if !ok {
			return fmt.Errorf("safety_alert repository unavailable")
		}
		if err := repo.CloseIntegrityIncident(ctx, id, s.now()); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "safety_alert", id, "close", "success", nil))
	})
}

func (s *Service) MarkIntegrityIncidentInProgress(ctx context.Context, meta RequestMeta, id string) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			MarkIntegrityIncidentInProgress(context.Context, string) error
		})
		if !ok {
			return fmt.Errorf("safety_alert repository unavailable")
		}
		if err := repo.MarkIntegrityIncidentInProgress(ctx, id); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "safety_alert", id, "start", "success", nil))
	})
}
