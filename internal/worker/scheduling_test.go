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

type failingExpiryRepo struct{}

func (failingExpiryRepo) ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error) {
	return nil, errors.New("backend query failed")
}

func TestRecoveryJobExpiryReconcilerFailureKeepsCompletionCountClean(t *testing.T) {
	metrics := &Metrics{}
	reconciler := NewRecoveryJobExpiryReconciler(failingExpiryRepo{}, nil, metrics)
	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err == nil {
		t.Fatal("reconciler swallowed backend query failure")
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 || due != 0 {
		t.Fatalf("failed reconcile polluted handover counters: runs=%d failures=%d due=%d", runs, failures, due)
	}
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

func TestFailedReconcileDoesNotInflateCompletionCount(t *testing.T) {
	var metrics Metrics
	metrics.RecordFailedDue(3)
	_, failures, due := metrics.Snapshot()
	if failures != 3 || due != 0 {
		t.Fatalf("failed pass polluted handover metrics: failures=%d due=%d", failures, due)
	}
}
