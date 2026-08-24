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

// A failed reclaim sweep must not produce any completion credential: the
// database transaction rolled back, so no recovery job was durably marked.
// The duty-summary "due" counter must therefore stay at zero while the
// failure is recorded.
func TestExpirationReconcilerFailureProducesNoCompletionCredential(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(_ context.Context, _ time.Time, _ int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 3, Marked: 3}, errors.New("pool exhausted")
	})
	err := NewExpirationReconciler(source, nil, metrics).Reconcile(t.Context(), time.Now().UTC())
	if err == nil {
		t.Fatal("reclaim error swallowed")
	}
	runs, failures, due := metrics.Snapshot()
	if due != 0 {
		t.Fatalf("failed reclaim advanced due=%d, expected 0", due)
	}
	if runs != 1 || failures != 1 {
		t.Fatalf("runs=%d failures=%d, expected 1/1", runs, failures)
	}
}

// Even when the failed sweep reports a partial Marked count, no completion
// credential is produced: due stays at zero because the transaction rolled
// back, and the failed run is counted exactly once.
func TestExpirationReconcilerPartialFailureProducesNoCompletionCredential(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(_ context.Context, _ time.Time, _ int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 3, Marked: 3}, errors.New("pool exhausted")
	})
	if err := NewExpirationReconciler(source, nil, metrics).Reconcile(t.Context(), time.Now().UTC()); err == nil {
		t.Fatal("reclaim error swallowed")
	}
	_, failures, due := metrics.Snapshot()
	if due != 0 {
		t.Fatalf("failed reclaim advanced due=%d, expected 0", due)
	}
	if failures != 1 {
		t.Fatalf("failures=%d, expected 1", failures)
	}
}

// A successful reclaim still advances the completion counter once, so the
// successful path is unaffected by the fix.
func TestExpirationReconcilerSuccessAdvancesDueOnce(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(_ context.Context, _ time.Time, _ int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 2, Marked: 2}, nil
	})
	if err := NewExpirationReconciler(source, nil, metrics).Reconcile(t.Context(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 0 || due != 2 {
		t.Fatalf("runs=%d failures=%d due=%d, expected 1/0/2", runs, failures, due)
	}
}
