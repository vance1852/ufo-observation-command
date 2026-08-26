package service

import (
	"context"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/pagination"
)

func (s *Service) SearchRecoveryJobs(ctx context.Context, query pagination.Query, survey_missionID string, status domain.RecoveryJobStatus) (pagination.Page[domain.RecoveryJob], error) {
	query = pagination.Normalize(query.Offset, query.Limit)
	page, err := s.repo.ListRecoveryJobs(ctx, query.Offset, query.Limit, survey_missionID, status)
	if err != nil {
		return pagination.Page[domain.RecoveryJob]{}, err
	}
	return pagination.From(page.Items, page.Total, query), nil
}
