package repository

import (
	"context"
	"fmt"

	"ufo-observation-command/internal/domain"
)

type SurveyMissionFilter struct {
	Status domain.SurveyMissionStatus
	Search string
	Limit  int
	Offset int
}

func (p *Postgres) ListSurveyMissions(ctx context.Context, filter SurveyMissionFilter) ([]domain.SurveyMission, int, error) {
	if filter.Limit < 1 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	where := "WHERE TRUE"
	args := []any{filter.Limit, filter.Offset}
	if filter.Status != "" {
		args = append(args, filter.Status)
		where += fmt.Sprintf(" AND status=$%d", len(args))
	}
	if filter.Search != "" {
		args = append(args, "%"+filter.Search+"%")
		where += fmt.Sprintf(" AND (code ILIKE $%d OR name ILIKE $%d)", len(args), len(args))
	}
	rows, err := p.pool.Query(ctx, fmt.Sprintf(`SELECT id,code,name,status,timezone,starts_at,ends_at,version,created_by FROM survey_missions %s ORDER BY starts_at DESC LIMIT $1 OFFSET $2`, where), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list cases: %w", err)
	}
	defer rows.Close()
	items := make([]domain.SurveyMission, 0)
	for rows.Next() {
		var item domain.SurveyMission
		if err := rows.Scan(&item.ID, &item.Code, &item.Name, &item.Status, &item.Timezone, &item.StartsAt, &item.EndsAt, &item.Version, &item.CreatedBy); err != nil {
			return nil, 0, fmt.Errorf("scan survey_mission: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	countArgs := args[2:]
	countWhere := "WHERE TRUE"
	if filter.Status != "" {
		countWhere += " AND status=$1"
	}
	if filter.Search != "" {
		index := 1
		if filter.Status != "" {
			index = 2
		}
		countWhere += fmt.Sprintf(" AND (code ILIKE $%d OR name ILIKE $%d)", index, index)
	}
	var total int
	if err := p.pool.QueryRow(ctx, "SELECT count(*) FROM survey_missions "+countWhere, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cases: %w", err)
	}
	return items, total, nil
}

func (p *Postgres) ListSurveyMissionAcousticBuoys(ctx context.Context, survey_missionID string) ([]domain.AcousticBuoy, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,survey_mission_id,code,arrayops_lane,required_successes,completed_installs FROM acoustic_buoys WHERE survey_mission_id=$1 ORDER BY code`, survey_missionID)
	if err != nil {
		return nil, fmt.Errorf("list survey_mission acoustic_buoys: %w", err)
	}
	defer rows.Close()
	items := make([]domain.AcousticBuoy, 0)
	for rows.Next() {
		var item domain.AcousticBuoy
		if err := rows.Scan(&item.ID, &item.SurveyMissionID, &item.Code, &item.ArrayOpsLane, &item.RequiredSuccesses, &item.Completed); err != nil {
			return nil, fmt.Errorf("scan acoustic_buoy: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
