package service

import (
	"context"
	"fmt"
	"time"

	auditpkg "ufo-observation-command/internal/audit"
	"ufo-observation-command/internal/domain"
)

type AuditPage struct {
	Items  []auditpkg.Event `json:"items"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
}

func (s *Service) QueryAudit(ctx context.Context, filter auditpkg.Filter, from, to time.Time, limit, offset int) (AuditPage, error) {
	if from.IsZero() || to.IsZero() || !to.After(from) {
		return AuditPage{}, fmt.Errorf("audit time range is invalid: %w", domain.ErrConflict)
	}
	if limit < 1 || limit > 200 || offset < 0 {
		return AuditPage{}, fmt.Errorf("audit page bounds are invalid: %w", domain.ErrConflict)
	}
	repo, ok := s.repo.(interface {
		QueryAudit(context.Context, auditpkg.Filter, time.Time, time.Time, int, int) ([]auditpkg.Event, int, error)
	})
	if !ok {
		return AuditPage{}, fmt.Errorf("audit query repository unavailable")
	}
	items, total, err := repo.QueryAudit(ctx, filter, from.UTC(), to.UTC(), limit, offset)
	if err != nil {
		return AuditPage{}, err
	}
	copied := make([]auditpkg.Event, len(items))
	copy(copied, items)
	return AuditPage{
		Items:  copied,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}
