package service

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

func (s *Service) MarkExpiredRecoveryJobs(ctx context.Context, now time.Time, limit int) (repository.ReconcileResult, error) {
	repo, ok := s.repo.(interface {
		MarkExpiredRecoveryJobs(context.Context, time.Time, int) (repository.ReconcileResult, error)
	})
	if !ok {
		return repository.ReconcileResult{}, fmt.Errorf("reconcile repository unavailable")
	}
	return repo.MarkExpiredRecoveryJobs(ctx, now, limit)
}

func (s *Service) SearchRecoveryJobsAdvanced(ctx context.Context, request domain.SearchRequest) (repository.Page, error) {
	repo, ok := s.repo.(interface {
		SearchRecoveryJobs(context.Context, domain.SearchRequest) (repository.Page, error)
	})
	if !ok {
		return repository.Page{}, fmt.Errorf("search repository unavailable")
	}
	return repo.SearchRecoveryJobs(ctx, request)
}
