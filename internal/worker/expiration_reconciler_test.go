package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/repository"
)

type expirationSourceFunc func(context.Context, time.Time, int) (repository.ReconcileResult, error)

func (f expirationSourceFunc) MarkExpiredRecoveryJobs(ctx context.Context, now time.Time, limit int) (repository.ReconcileResult, error) {
	return f(ctx, now, limit)
}

func TestExpirationReconcilerRunsSource(t *testing.T) {
	called := false
	source := expirationSourceFunc(func(_ context.Context, _ time.Time, limit int) (repository.ReconcileResult, error) {
		called = true
		if limit != 100 {
			t.Fatalf("limit=%d", limit)
		}
		return repository.ReconcileResult{Scanned: 2, Marked: 2}, nil
	})
	if err := NewExpirationReconciler(source, nil, nil).Reconcile(t.Context(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expiration source was not called")
	}
}

func TestExpirationReconcilerRejectsMissingSourceAndCancelledContext(t *testing.T) {
	if err := NewExpirationReconciler(nil, nil, nil).Reconcile(t.Context(), time.Now().UTC()); err == nil {
		t.Fatal("nil source was accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := NewExpirationReconciler(nil, nil, nil).Reconcile(ctx, time.Now().UTC()); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
}

// TestExpirationReconcilerRecordsRolledBackNotDueOnWriteTimeout locks in the
// fix for the sea-state surge recovery path: when the write to the database
// times out the transaction rolls back, so none of the scanned recovery jobs
// were persisted. Monitoring must report them as rolled back and must NOT
// count them under due (which the console renders as "all processed").
func TestExpirationReconcilerRecordsRolledBackNotDueOnWriteTimeout(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		// MarkExpiredRecoveryJobs returns the scanned scope even when the
		// commit times out, so the reconciler can attribute the rollback.
		return repository.ReconcileResult{Scanned: 3, Marked: 0}, context.DeadlineExceeded
	})
	reconciler := NewExpirationReconciler(source, nil, metrics)
	if err := reconciler.Reconcile(t.Context(), time.Now().UTC()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("error=%v", err)
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 {
		t.Fatalf("runs=%d failures=%d", runs, failures)
	}
	if due != 0 {
		t.Fatalf("rolled-back write must not be reported as due (processed): due=%d", due)
	}
	if rolled := metrics.RolledBackSnapshot(); rolled != 3 {
		t.Fatalf("rolled-back count must mirror the post-rollback state: rolled=%d", rolled)
	}
}

// TestExpirationReconcilerRecordsDueOnCommittedWrite confirms a successful
// write keeps reporting the persisted count under due, so the two failure
// and success paths stay distinguishable in monitoring.
func TestExpirationReconcilerRecordsDueOnCommittedWrite(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 3, Marked: 3}, nil
	})
	if err := NewExpirationReconciler(source, nil, metrics).Reconcile(t.Context(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	_, failures, due := metrics.Snapshot()
	if failures != 0 || due != 3 {
		t.Fatalf("failures=%d due=%d", failures, due)
	}
	if rolled := metrics.RolledBackSnapshot(); rolled != 0 {
		t.Fatalf("committed write must not report rollbacks: rolled=%d", rolled)
	}
}
