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

// A transaction conflict during the expiration scan must increment the failure
// counter and must not leak the failed items into the completion counter.
func TestExpirationReconcilerConflictDoesNotInflateCompletionCount(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 3, Marked: 0}, errors.New("serialization failure")
	})
	reconciler := NewExpirationReconciler(source, nil, metrics)
	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err == nil {
		t.Fatal("conflict was swallowed")
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 || due != 0 {
		t.Fatalf("metrics=%d,%d,%d", runs, failures, due)
	}
}

// After a restart, residual error counts from a prior conflict-disrupted scan
// are cleared so the recovered statistics become trustworthy again.
func TestExpirationReconcilerClearsResidualErrorCountOnRestart(t *testing.T) {
	metrics := &Metrics{}
	metrics.RecordRun()
	metrics.RecordFailure()
	metrics.RecordDue(5)
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 1, Marked: 1}, nil
	})
	reconciler := NewExpirationReconciler(source, nil, metrics)
	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 0 || due != 1 {
		t.Fatalf("metrics=%d,%d,%d", runs, failures, due)
	}
}
