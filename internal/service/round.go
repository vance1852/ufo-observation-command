package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) StartDiveWindow(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.changeDiveWindow(ctx, meta, id, version, domain.DiveWindowRunning, "start")
}

func (s *Service) CompleteDiveWindow(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.changeDiveWindow(ctx, meta, id, version, domain.DiveWindowCompleted, "complete")
}

func (s *Service) CancelDiveWindow(ctx context.Context, meta RequestMeta, id string, version int64) error {
	return s.changeDiveWindow(ctx, meta, id, version, domain.DiveWindowCancelled, "cancel")
}

func (s *Service) changeDiveWindow(ctx context.Context, meta RequestMeta, id string, version int64, next domain.DiveWindowStatus, action string) error {
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			StartDiveWindow(context.Context, string, int64) error
			CompleteDiveWindow(context.Context, string, int64) error
			CancelDiveWindow(context.Context, string, int64) error
		})
		if !ok {
			return fmt.Errorf("dive_window repository unavailable")
		}
		var err error
		switch next {
		case domain.DiveWindowRunning:
			err = repo.StartDiveWindow(ctx, id, version)
		case domain.DiveWindowCompleted:
			err = repo.CompleteDiveWindow(ctx, id, version)
		case domain.DiveWindowCancelled:
			err = repo.CancelDiveWindow(ctx, id, version)
		}
		if err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "dive_window", id, action, "success", nil))
	})
}
