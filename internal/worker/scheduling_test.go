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

type activationSourceFunc func(context.Context, time.Time, int) (int, error)

func (f activationSourceFunc) ActivateDue(ctx context.Context, now time.Time, limit int) (int, error) {
	return f(ctx, now, limit)
}

// TestAssignmentWorkerFailureDoesNotRecordDue locks down the metric semantics: when the
// activation call fails (e.g. the database is briefly unavailable), the count it
// returns reflects unprocessed tasks, not completed ones. Recording it into the
// "due" counter would let an operator misread the failed tasks as executed coverage.
func TestAssignmentWorkerFailureDoesNotRecordDue(t *testing.T) {
	metrics := &Metrics{}
	source := activationSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 12, errors.New("database unavailable")
	})
	worker := NewAssignmentWorker(source, time.Nanosecond, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = worker.Run(ctx)
		close(done)
	}()
	// allow one iteration to land, then stop the worker
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	runs, failures, due := metrics.Snapshot()
	if runs < 1 {
		t.Fatalf("runs=%d", runs)
	}
	if failures != runs {
		t.Fatalf("failures=%d want=%d", failures, runs)
	}
	if due != 0 {
		t.Fatalf("due=%d, failed activation must not be recorded as executed coverage", due)
	}
}

func TestAssignmentWorkerSuccessRecordsDue(t *testing.T) {
	metrics := &Metrics{}
	source := activationSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 3, nil
	})
	worker := NewAssignmentWorker(source, time.Nanosecond, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = worker.Run(ctx)
		close(done)
	}()
	time.Sleep(20 * time.Millisecond)
	cancel()
	<-done

	_, failures, due := metrics.Snapshot()
	if failures != 0 {
		t.Fatalf("failures=%d", failures)
	}
	if due < 1 {
		t.Fatalf("due=%d, successful activation must be recorded", due)
	}
}
