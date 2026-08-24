package integration

import (
	"testing"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
	"github.com/google/uuid"
)

func TestArrayOperatorCreationRollsBackWhenAuditWriteFails(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	svc := service.New(repository.NewPostgres(pool))
	missingArrayOperator := uuid.NewString()
	_, err := svc.RegisterArrayOperator(ctx, service.RequestMeta{RequestID: "rollback", ArrayOperatorID: &missingArrayOperator}, "Rollback ArrayOperator", domain.RoleSafetySupervisor)
	if err == nil {
		t.Fatal("array_operator creation succeeded despite rejected audit event")
	}
	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM array_operators WHERE name='Rollback ArrayOperator'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("array_operator survived audit rollback: %d", count)
	}
}

func TestDiveWindowTransitionRollsBackWhenAuditWriteFails(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	repo := repository.NewPostgres(pool)
	dive_windowID, err := repo.CreateDiveWindow(ctx, repository.DiveWindowInput{Code: "ROLLBACK-ROUND", Method: "daily-acoustic_buoy", Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	missingArrayOperator := uuid.NewString()
	err = service.New(repo).StartDiveWindow(ctx, service.RequestMeta{RequestID: "rollback", ArrayOperatorID: &missingArrayOperator}, dive_windowID, 1)
	if err == nil {
		t.Fatal("dive_window transition succeeded despite rejected audit event")
	}
	var status string
	var version int64
	if err := pool.QueryRow(ctx, `SELECT status,version FROM dive_windows WHERE id=$1`, dive_windowID).Scan(&status, &version); err != nil {
		t.Fatal(err)
	}
	if status != string(domain.DiveWindowQueued) || version != 1 {
		t.Fatalf("dive_window survived audit rollback: status=%s version=%d", status, version)
	}
}
