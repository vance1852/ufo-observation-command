package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

func (p *Postgres) SearchRecoveryJobs(ctx context.Context, request domain.SearchRequest) (Page, error) {
	request = request.Normalize()
	where := "WHERE TRUE"
	args := []any{request.Limit, request.Offset}
	countArgs := make([]any, 0, 3)
	countWhere := "WHERE TRUE"
	if request.Filter.SurveyMissionID != "" {
		args = append(args, request.Filter.SurveyMissionID)
		where += fmt.Sprintf(" AND survey_mission_id=$%d", len(args))
		countArgs = append(countArgs, request.Filter.SurveyMissionID)
		countWhere += fmt.Sprintf(" AND survey_mission_id=$%d", len(countArgs))
	}
	if request.Filter.Status != "" {
		args = append(args, request.Filter.Status)
		where += fmt.Sprintf(" AND status=$%d", len(args))
		countArgs = append(countArgs, request.Filter.Status)
		countWhere += fmt.Sprintf(" AND status=$%d", len(countArgs))
	}
	if request.Filter.Search != "" {
		args = append(args, "%"+request.Filter.Search+"%")
		where += fmt.Sprintf(" AND lower(task_code) LIKE lower($%d)", len(args))
		countArgs = append(countArgs, "%"+request.Filter.Search+"%")
		countWhere += fmt.Sprintf(" AND lower(task_code) LIKE lower($%d)", len(countArgs))
	}
	order := "created_at"
	switch request.Sort {
	case domain.SortExpiry:
		order = "expires_at"
	case domain.SortCode:
		order = "task_code"
	}
	direction := "ASC"
	if request.Desc {
		direction = "DESC"
	}
	query := fmt.Sprintf(`SELECT id,survey_mission_id,acoustic_buoy_id,task_code,status,completed_at,accepted_at,expires_at,version FROM recovery_jobs %s ORDER BY %s %s LIMIT $1 OFFSET $2`, where, order, direction)
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return Page{}, fmt.Errorf("search recovery_jobs: %w", err)
	}
	defer rows.Close()
	page := Page{Items: make([]domain.RecoveryJob, 0), Offset: request.Offset, Limit: request.Limit}
	for rows.Next() {
		var item domain.RecoveryJob
		if err := rows.Scan(&item.ID, &item.SurveyMissionID, &item.AcousticBuoyID, &item.TaskCode, &item.Status, &item.CompletedAt, &item.AcceptedAt, &item.ExpiresAt, &item.Version); err != nil {
			return Page{}, err
		}
		page.Items = append(page.Items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	countQuery := fmt.Sprintf("SELECT count(*) FROM recovery_jobs %s", countWhere)
	if err := p.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&page.Total); err != nil {
		return Page{}, fmt.Errorf("count recovery_jobs: %w", err)
	}
	return page, nil
}
