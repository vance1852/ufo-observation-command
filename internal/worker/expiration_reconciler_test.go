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
