package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubSource struct {
	count int
	err   error
}

func (s *stubSource) ActivateDue(context.Context, time.Time, int) (int, error) {
	return s.count, s.err
}

// TestAssignmentWorkerNoSpuriousDueOnActivationError guards the capacity board:
// when the refresh write fails (e.g. deadline exceeded), the returned count is
// uncommitted and must never be reflected in the due metric.
func TestAssignmentWorkerNoSpuriousDueOnActivationError(t *testing.T) {
	metrics := &Metrics{}
	source := &stubSource{count: 7, err: errors.New("context deadline exceeded")}
	worker := NewAssignmentWorker(source, time.Hour, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := worker.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	runs, failures, due := metrics.Snapshot()
	if due != 0 {
		t.Fatalf("uncommitted count leaked into due metric: runs=%d failures=%d due=%d", runs, failures, due)
	}
}

// TestAssignmentWorkerRecordsDueOnlyOnCommittedActivation confirms the due
// metric reflects only successfully committed assignments.
func TestAssignmentWorkerRecordsDueOnlyOnCommittedActivation(t *testing.T) {
	metrics := &Metrics{}
	source := &signalSource{count: 5, done: make(chan struct{})}
	worker := NewAssignmentWorker(source, time.Hour, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-source.done
		cancel()
	}()
	if err := worker.Run(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	_, _, due := metrics.Snapshot()
	if due != 5 {
		t.Fatalf("due=%d want 5", due)
	}
}

type signalSource struct {
	count int
	done  chan struct{}
}

func (s *signalSource) ActivateDue(context.Context, time.Time, int) (int, error) {
	close(s.done)
	return s.count, nil
}
