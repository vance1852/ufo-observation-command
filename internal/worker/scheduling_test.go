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

type assignmentSourceFunc func(context.Context, time.Time, int) (int, error)

func (f assignmentSourceFunc) ActivateDue(ctx context.Context, now time.Time, limit int) (int, error) {
	return f(ctx, now, limit)
}

func TestAssignmentWorkerFailedRoundDoesNotAdvanceDueCounter(t *testing.T) {
	calls := 0
	source := assignmentSourceFunc(func(context.Context, time.Time, int) (int, error) {
		calls++
		// A cold-water lease renewal hits a lock wait: the round reports a
		// partial count alongside an error and must not count as recovered.
		return 5, errors.New("lock wait timeout exceeded")
	})
	metrics := &Metrics{}
	worker := NewAssignmentWorker(source, time.Millisecond, nil, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_ = worker.Run(ctx)
	if calls == 0 {
		t.Fatal("source was never invoked")
	}
	runs, failures, due := metrics.Snapshot()
	if failures != int64(calls) {
		t.Fatalf("failures=%d calls=%d", failures, calls)
	}
	if due != 0 {
		t.Fatalf("failed round advanced due counter: runs=%d failures=%d due=%d", runs, failures, due)
	}
}

func TestAssignmentWorkerConfirmedSuccessAdvancesDueCounterOnce(t *testing.T) {
	source := assignmentSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 3, nil
	})
	metrics := &Metrics{}
	worker := NewAssignmentWorker(source, time.Millisecond, nil, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	_ = worker.Run(ctx)
	runs, failures, due := metrics.Snapshot()
	if runs < 1 || failures != 0 || due != 3*runs {
		t.Fatalf("runs=%d failures=%d due=%d", runs, failures, due)
	}
}
