package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) ListArrayOperators(ctx context.Context, role domain.ArrayOperatorRole, limit, offset int) ([]domain.ArrayOperator, int, error) {
	repo, ok := s.repo.(interface {
		ListArrayOperators(context.Context, domain.ArrayOperatorRole, int, int) ([]domain.ArrayOperator, int, error)
	})
	if !ok {
		return nil, 0, fmt.Errorf("array_operator query repository unavailable")
	}
	items, total, err := repo.ListArrayOperators(ctx, role, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return append([]domain.ArrayOperator(nil), items...), total, nil
}

func (s *Service) RenameArrayOperator(ctx context.Context, meta RequestMeta, id, name string) error {
	if err := validateCode(name, "array_operator name"); err != nil {
		return err
	}
	return s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			RenameArrayOperator(context.Context, string, string) error
		})
		if !ok {
			return fmt.Errorf("array_operator mutation repository unavailable")
		}
		if err := repo.RenameArrayOperator(ctx, id, name); err != nil {
			return err
		}
		return tx.WriteAudit(ctx, audit(meta, "array_operator", id, "rename", "success", nil))
	})
}
