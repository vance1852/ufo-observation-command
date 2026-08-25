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

func TestMetricsRolledBackIsSeparateFromDue(t *testing.T) {
	var metrics Metrics
	// A timed-out write rolled back three recovery jobs. They must not bleed
	// into the due (processed) counter that the console renders as progress.
	metrics.RecordRolledBack(3)
	metrics.RecordDue(3)
	_, _, due := metrics.Snapshot()
	if due != 3 {
		t.Fatalf("rolled-back count leaked into due: due=%d", due)
	}
	if metrics.RolledBackSnapshot() != 3 {
		t.Fatalf("rolled-back snapshot mismatch")
	}
	metrics.RecordRolledBack(0)
	if metrics.RolledBackSnapshot() != 3 {
		t.Fatalf("zero count must not change the rolled-back counter")
	}
}
