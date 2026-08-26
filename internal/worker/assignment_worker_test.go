package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type assignmentSourceFunc func(context.Context, time.Time, int) (int, error)

func (f assignmentSourceFunc) ActivateDue(ctx context.Context, now time.Time, limit int) (int, error) {
	return f(ctx, now, limit)
}

// runAssignmentWorkerOnce drives the real Run loop through exactly one
// activation. The optional cancelFromSource hook lets a cancelled source
// cancel the run context from within ActivateDue (mirroring the real
// cancellation boundary). Otherwise the loop is stopped by cancelling after
// the source returns, so the switch observes ctx.Err()==nil for the committed
// and failure cases.
func runAssignmentWorkerOnce(t *testing.T, source AssignmentSource, metrics *Metrics, cancelFromSource bool) error {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	called := make(chan struct{}, 1)
	wrapped := assignmentSourceFunc(func(c context.Context, now time.Time, limit int) (int, error) {
		select {
		case called <- struct{}{}:
		default:
		}
		if cancelFromSource {
			cancel()
		}
		return source.ActivateDue(c, now, limit)
	})
	w := NewAssignmentWorker(wrapped, time.Hour, nil, metrics)
	done := make(chan error, 1)
	go func() { done <- w.Run(ctx) }()
	if !cancelFromSource {
		select {
		case <-called:
		case <-time.After(time.Second):
			t.Fatal("source was never invoked")
		}
		cancel()
	}
	select {
	case err := <-done:
		return err
	case <-time.After(time.Second):
		t.Fatal("worker did not stop")
		return nil
	}
}

func TestAssignmentWorkerDoesNotReportCancelledCountAsComplete(t *testing.T) {
	metrics := &Metrics{}
	// The pool was briefly unavailable and the request was cancelled while the
	// source still returned a count. That count was never committed, so it must
	// not be reported as completed progress.
	source := assignmentSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 5, context.Canceled
	})
	if err := runAssignmentWorkerOnce(t, source, metrics, true); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	runs, failures, due := metrics.Snapshot()
	if due != 0 {
		t.Fatalf("cancelled count was reported as complete: runs=%d failures=%d due=%d", runs, failures, due)
	}
}

func TestAssignmentWorkerRecordsCommittedCountAsComplete(t *testing.T) {
	metrics := &Metrics{}
	source := assignmentSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 3, nil
	})
	_ = runAssignmentWorkerOnce(t, source, metrics, false)
	_, _, due := metrics.Snapshot()
	if due != 3 {
		t.Fatalf("committed count was not reported: due=%d", due)
	}
}

func TestAssignmentWorkerRecordsFailureWithoutReportingDue(t *testing.T) {
	metrics := &Metrics{}
	source := assignmentSourceFunc(func(context.Context, time.Time, int) (int, error) {
		return 7, errors.New("pool unavailable")
	})
	_ = runAssignmentWorkerOnce(t, source, metrics, false)
	runs, failures, due := metrics.Snapshot()
	if failures != 1 || due != 0 {
		t.Fatalf("failed count was reported as complete: runs=%d failures=%d due=%d", runs, failures, due)
	}
}
