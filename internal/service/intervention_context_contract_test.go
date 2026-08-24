package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/repository"
)

type cancellationRepository struct {
	repository.Repository
}

func (cancellationRepository) DueIntegrityIncidents(ctx context.Context, _ time.Time, _ int) ([]repository.IntegrityIncidentInput, error) {
	return nil, ctx.Err()
}

func (cancellationRepository) DueIntegrityIncidentsDetached(context.Context, time.Time, int) ([]repository.IntegrityIncidentInput, error) {
	return nil, nil
}

func TestDueIntegrityIncidentQueryPreservesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := New(cancellationRepository{}).DueIntegrityIncidents(ctx, time.Now().UTC(), 10)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}
