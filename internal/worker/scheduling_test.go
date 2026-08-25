package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
)

type fakeReconciler struct {
	calls int
	err   error
}

func (f *fakeReconciler) Reconcile(context.Context, time.Time) error { f.calls++; return f.err }

func TestPeriodicStopsOnContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := &fakeReconciler{}
	periodic := NewPeriodic(time.Hour, runner, nil)
	cancel()
	if err := periodic.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("error=%v", err)
	}
	if runner.calls > 1 {
		t.Fatalf("calls=%d", runner.calls)
	}
}

type fakeExpiryRepo struct{ items int }

func (f fakeExpiryRepo) ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error) {
	return make([]domain.RecoveryJob, f.items), nil
}

type failingExpiryRepo struct{ items int }

func (f failingExpiryRepo) ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error) {
	return make([]domain.RecoveryJob, f.items), errors.New("unavailable")
}

func TestMetricsRecordCounters(t *testing.T) {
	var metrics Metrics
	metrics.RecordRun()
	metrics.RecordFailure()
	metrics.RecordDue(4)
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 || due != 4 {
		t.Fatalf("metrics=%d,%d,%d", runs, failures, due)
	}
}

// TestExpiryReconcilerFailureDoesNotCarryStaleDueIntoNextRound guards the
// long-range patrol retry path: when surfacing expiring tasks fails, the
// partial count must not be folded into the pending-work counter, otherwise
// the stale quantity resurfaces every round, hiding the real backlog of
// unclaimed tasks. The failure is still recorded so the count is honest.
func TestExpiryReconcilerFailureDoesNotCarryStaleDueIntoNextRound(t *testing.T) {
	metrics := &Metrics{}
	reconciler := NewRecoveryJobExpiryReconciler(failingExpiryRepo{items: 3}, nil, metrics)

	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err == nil {
		t.Fatal("reconciler swallowed failure")
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 {
		t.Fatalf("after failed pass metrics=runs=%d failures=%d", runs, failures)
	}
	if due != 0 {
		t.Fatalf("failed pass carried stale quantity into due counter: due=%d", due)
	}

	// A subsequent successful pass surfaces the real backlog and lets the
	// operator reclaim the still-expiring tasks.
	success := NewRecoveryJobExpiryReconciler(fakeExpiryRepo{items: 2}, nil, metrics)
	if err := success.Reconcile(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, failures, due = metrics.Snapshot()
	if runs != 2 || failures != 1 || due != 2 {
		t.Fatalf("after success metrics=runs=%d failures=%d due=%d", runs, failures, due)
	}
}
