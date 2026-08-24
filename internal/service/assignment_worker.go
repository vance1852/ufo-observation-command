package service

import (
	"context"
	"fmt"
	"time"
)

func (s *Service) ActivateDueAssignments(ctx context.Context, now time.Time, limit int) (int, error) {
	repo, ok := s.repo.(interface {
		ActivateDue(context.Context, time.Time, int) (int, error)
	})
	if !ok {
		return 0, fmt.Errorf("assignment worker repository unavailable")
	}
	return repo.ActivateDue(ctx, now, limit)
}
