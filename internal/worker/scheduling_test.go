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

func TestMetricsAbnormalBatchDoesNotContaminateBaseline(t *testing.T) {
	var metrics Metrics
	// A prior round recorded the real submission volume.
	metrics.RecordDue(10)
	// The next round hit a downstream timeout: the abnormal batch must be
	// isolated from the statistics baseline, then the baseline reestablished
	// so the real submission volume can still be verified.
	metrics.RecordFailure()
	metrics.RecordFailedDue(7)
	metrics.ReestablishBaseline()
	runs, failures, due := metrics.Snapshot()
	if runs != 0 || failures != 1 || due != 10 {
		t.Fatalf("baseline contaminated: runs=%d failures=%d due=%d", runs, failures, due)
	}
	if got := metrics.FailedDue(); got != 0 {
		t.Fatalf("abnormal batch was not isolated: failedDue=%d", got)
	}
}
