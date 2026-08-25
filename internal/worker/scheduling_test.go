package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
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

// TestMetricsSeparateScannedCandidatesFromDue guards the bug where
// unconfirmed scanned candidates were miscounted into the due/success metric.
// On the anomaly path the scanned candidates must inflate scanned, not due.
func TestMetricsSeparateScannedCandidatesFromDue(t *testing.T) {
	var metrics Metrics
	metrics.RecordRun()
	metrics.RecordFailedDue0014(5)
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 || due != 0 {
		t.Fatalf("anomaly metrics=%d,%d,%d", runs, failures, due)
	}
	if metrics.ScannedCount() != 5 {
		t.Fatalf("scanned=%d want 5", metrics.ScannedCount())
	}
}

// TestExpirationReconcilerDoesNotPromoteCandidatesOnAnomaly ensures that when
// the compensation scheduler fails (e.g. a delayed replica), scanned candidates
// do not prematurely consume the due/success capacity metric.
func TestExpirationReconcilerDoesNotPromoteCandidatesOnAnomaly(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 3, Marked: 2}, errors.New("replica lag")
	})
	reconciler := NewExpirationReconciler(source, nil, metrics)
	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err == nil {
		t.Fatal("anomaly was swallowed")
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 1 || due != 0 {
		t.Fatalf("anomaly metrics=%d,%d,%d", runs, failures, due)
	}
	if metrics.ScannedCount() != 0 {
		t.Fatalf("scanned candidates leaked into scanned metric on anomaly: %d", metrics.ScannedCount())
	}
}

// TestExpirationReconcilerDistinguishesScannedFromConfirmed ensures scanned
// candidates are tracked separately from confirmed execution results.
func TestExpirationReconcilerDistinguishesScannedFromConfirmed(t *testing.T) {
	metrics := &Metrics{}
	source := expirationSourceFunc(func(context.Context, time.Time, int) (repository.ReconcileResult, error) {
		return repository.ReconcileResult{Scanned: 4, Marked: 2, Skipped: 1}, nil
	})
	reconciler := NewExpirationReconciler(source, nil, metrics)
	if err := reconciler.Reconcile(context.Background(), time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	runs, failures, due := metrics.Snapshot()
	if runs != 1 || failures != 0 || due != 2 {
		t.Fatalf("metrics=%d,%d,%d", runs, failures, due)
	}
	if metrics.ScannedCount() != 4 {
		t.Fatalf("scanned=%d want 4", metrics.ScannedCount())
	}
}
