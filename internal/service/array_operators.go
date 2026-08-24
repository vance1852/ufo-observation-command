package service

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"github.com/google/uuid"
)

func (s *Service) RegisterArrayOperator(ctx context.Context, meta RequestMeta, name string, role domain.ArrayOperatorRole) (domain.ArrayOperator, error) {
	array_operator := domain.ArrayOperator{ID: uuid.NewString(), Name: name, Role: role}
	if err := array_operator.Validate(); err != nil {
		return domain.ArrayOperator{}, err
	}
	err := s.repo.InTx(ctx, func(tx repository.Repository) error {
		repo, ok := tx.(interface {
			CreateArrayOperator(context.Context, domain.ArrayOperator) error
		})
		if !ok {
			return fmt.Errorf("array_operator repository unavailable")
		}
		if err := repo.CreateArrayOperator(ctx, array_operator); err != nil {
			return fmt.Errorf("register array_operator: %w", err)
		}
		return tx.WriteAudit(ctx, audit(meta, "array_operator", array_operator.ID, "create", "success", nil))
	})
	if err != nil {
		return domain.ArrayOperator{}, err
	}
	return array_operator, nil
}

func (s *Service) LoadArrayOperator(ctx context.Context, id string) (domain.ArrayOperator, error) {
	if repo, ok := s.repo.(interface {
		GetArrayOperator(context.Context, string) (domain.ArrayOperator, error)
	}); ok {
		return repo.GetArrayOperator(ctx, id)
	}
	return domain.ArrayOperator{}, fmt.Errorf("array_operator repository unavailable")
}

var _ repository.Repository = (*repository.Postgres)(nil)
