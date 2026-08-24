package worker

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type perJobExecutor struct {
	mu    sync.Mutex
	calls map[string]int
}

func (e *perJobExecutor) Execute(_ context.Context, job DiveWindowJob) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.calls[job.ID]++
	if job.ID == "long" && e.calls[job.ID] >= 3 {
		return nil
	}
	return errors.New("temporary")
}

func TestDiveWindowAttemptLimitDoesNotAffectTheNextJob(t *testing.T) {
	executor := &perJobExecutor{calls: map[string]int{}}
	processor := NewDiveWindowProcessor(executor, RetryPolicy{Attempts: 3, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond}, nil, nil)
	if err := processor.Process(t.Context(), DiveWindowJob{ID: "short", MaxAttempts: 1}); err == nil {
		t.Fatal("one-attempt job unexpectedly succeeded")
	}
	if err := processor.Process(t.Context(), DiveWindowJob{ID: "long"}); err != nil {
		t.Fatal(err)
	}
	if executor.calls["short"] != 1 || executor.calls["long"] != 3 {
		t.Fatalf("calls=%v", executor.calls)
	}
}

type countingReconciler struct{ calls int }

func (r *countingReconciler) Reconcile(context.Context, time.Time) error {
	r.calls++
	return nil
}

func TestCancelledPeriodicDoesNotStartWork(t *testing.T) {
	reconciler := &countingReconciler{}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	err := NewPeriodic(time.Hour, reconciler, nil).Run(ctx)
	if !errors.Is(err, context.Canceled) || reconciler.calls != 0 {
		t.Fatalf("err=%v calls=%d", err, reconciler.calls)
	}
}
