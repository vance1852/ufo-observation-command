package integration

import (
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"ufo-observation-command/internal/audit"
	"ufo-observation-command/internal/db"
	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
	"ufo-observation-command/internal/service"
)

func TestAuditQueryReturnsFilteredTotalBeyondPage(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "acoustic_buoy_operator")
	repo := repository.NewPostgres(pool)
	_, task := createWorkflow(t, ctx, service.New(repo), array_operator)
	items, total, err := repo.QueryAudit(ctx, audit.Filter{ObjectID: task.ID}, time.Now().Add(-time.Hour), time.Now().Add(time.Hour), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || total != 4 {
		t.Fatalf("items=%d total=%d", len(items), total)
	}
}

func TestDatabaseConstraintErrorsAreBusinessConflicts(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	array_operator := insertArrayOperator(t, ctx, pool, "safety_supervisor")
	svc := service.New(repository.NewPostgres(pool))
	now := time.Now().UTC()
	request := service.CreateSurveyMissionRequest{
		Code: "UNIQUE-PLAN", Name: "Unique", Timezone: "UTC", StartsAt: now, EndsAt: now.Add(time.Hour), CreatedBy: array_operator,
		AcousticBuoys: []repository.AcousticBuoyInput{{Code: "UNIQUE-MANAGED_DEVICE", ArrayOpsLane: "A", RequiredSuccesses: 1}},
	}
	if _, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "first"}, request); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateSurveyMission(ctx, service.RequestMeta{RequestID: "second"}, request); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate error=%v", err)
	}
}

func TestAdvisoryLockUsesOneDatabaseSession(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	verifier, err := db.Open(ctx, os.Getenv("DATABASE_URL"), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer verifier.Close()
	verificationConn, err := verifier.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer verificationConn.Release()
	const key int64 = 987654321
	if err := db.AdvisoryLock(ctx, pool, key, func() error {
		var acquired bool
		if err := verificationConn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&acquired); err != nil {
			return err
		}
		if acquired {
			_, _ = verificationConn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, key)
			t.Fatal("another database session acquired the held lock")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var acquired bool
	if err := verificationConn.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, key).Scan(&acquired); err != nil {
		t.Fatal(err)
	}
	if !acquired {
		t.Fatal("advisory lock remained held after callback")
	}
	if _, err := verificationConn.Exec(ctx, `SELECT pg_advisory_unlock($1)`, key); err != nil {
		t.Fatal(err)
	}
}

func TestConcurrentMigrationChecksSucceed(t *testing.T) {
	pool, ctx := openDatabase(t)
	defer pool.Close()
	const workers = 6
	errorsCh := make(chan error, workers)
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errorsCh <- db.Migrate(ctx, pool)
		}()
	}
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		if err != nil {
			t.Fatal(err)
		}
	}
}
