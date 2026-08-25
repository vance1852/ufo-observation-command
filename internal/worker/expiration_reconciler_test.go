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

// A renewal that returns an internal error (e.g. under severe sea-state
// pressure) has its reconciliation transaction rolled back, so the scanned
// candidates never become effective work. They must not be aggregated into
// the due counter, otherwise cross-region capacity accounting is skewed.
func TestExpirationReconcilerFailedRenewalDoesNotCountScannedAsEffective(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 5, Marked: 5}, errors.New("renewal internal error")
	})
	err := NewExpirationReconciler(source, nil, metrics).Reconcile(t.Context(), time.Now().UTC())
	if err == nil {
		t.Fatal("renewal error was swallowed")
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 {
		t.Fatalf("runs=%d failures=%d", runs, failures)
	}
	if due != 0 {
		t.Fatalf("rolled-back scan candidates counted as effective work: due=%d", due)
	}
}
