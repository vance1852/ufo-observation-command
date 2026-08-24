package repository

import (
	"context"
	"fmt"
	"time"

	"ufo-observation-command/internal/domain"
	"github.com/google/uuid"
)

func (p *Postgres) CreateAssignment(ctx context.Context, assignment domain.Assignment) error {
	return createAssignment(ctx, p.pool, assignment)
}

func (t *transaction) CreateAssignment(ctx context.Context, assignment domain.Assignment) error {
	return createAssignment(ctx, t.tx, assignment)
}

func createAssignment(ctx context.Context, q sqler, assignment domain.Assignment) error {
	if err := assignment.Validate(); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `INSERT INTO assignments(id,survey_mission_id,acoustic_buoy_id,array_operator_id,status,starts_at,ends_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, assignment.ID, assignment.SurveyMissionID, assignment.AcousticBuoyID, assignment.ArrayOperatorID, assignment.Status, assignment.StartsAt, assignment.EndsAt)
	return wrapWrite(err)
}

func (p *Postgres) AdvanceAssignment(ctx context.Context, id, next string, version int64) error {
	return advanceAssignment(ctx, p.pool, id, next, version)
}

func (t *transaction) AdvanceAssignment(ctx context.Context, id, next string, version int64) error {
	return advanceAssignment(ctx, t.tx, id, next, version)
}

func advanceAssignment(ctx context.Context, q sqler, id, next string, version int64) error {
	allowedFrom, ok := domain.AssignmentSourcesFor(next)
	if !ok || len(allowedFrom) == 0 {
		return domain.ErrInvalidTransition
	}
	result, err := q.Exec(ctx, `UPDATE assignments SET status=$1,version=version+1 WHERE id=$2 AND version=$3 AND status=ANY($4)`, next, id, version, allowedFrom)
	if err != nil {
		return fmt.Errorf("advance assignment: %w", err)
	}
	if result.RowsAffected() != 1 {
		return domain.ErrConflict
	}
	return nil
}

func NewAssignment(survey_missionID, acoustic_buoyID, array_operatorID string, startsAt, endsAt time.Time) domain.Assignment {
	return domain.Assignment{ID: uuid.NewString(), SurveyMissionID: survey_missionID, AcousticBuoyID: acoustic_buoyID, ArrayOperatorID: array_operatorID, StartsAt: startsAt, EndsAt: endsAt, Status: "queued"}
}
