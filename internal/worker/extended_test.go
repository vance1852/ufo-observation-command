package worker

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeExecutor struct {
	attempts int
	fail     int
}

func (f *fakeExecutor) Execute(context.Context, DiveWindowJob) error {
	f.attempts++
	if f.attempts <= f.fail {
		return errors.New("temporary")
	}
	return nil
}

func TestDiveWindowProcessorRecordsSuccessAfterRetry(t *testing.T) {
	executor := &fakeExecutor{fail: 1}
	metrics := &Metrics{}
	processor := NewDiveWindowProcessor(executor, RetryPolicy{Attempts: 3, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}, nil, metrics)
	if err := processor.Process(context.Background(), DiveWindowJob{ID: "dive_window-1"}); err != nil {
		t.Fatal(err)
	}
	if executor.attempts != 2 {
		t.Fatalf("attempts=%d", executor.attempts)
	}
	runs, failures, _ := metrics.Snapshot()
	if runs != 1 || failures != 0 {
		t.Fatalf("metrics=%d,%d", runs, failures)
	}
}

func TestDiveWindowProcessorRecordsPermanentFailure(t *testing.T) {
	executor := &fakeExecutor{fail: 10}
	metrics := &Metrics{}
	processor := NewDiveWindowProcessor(executor, RetryPolicy{Attempts: 2, BaseDelay: time.Nanosecond, MaxDelay: time.Nanosecond}, nil, metrics)
	if err := processor.Process(context.Background(), DiveWindowJob{ID: "dive_window-2"}); err == nil {
		t.Fatal("failure was swallowed")
	}
	_, failures, _ := metrics.Snapshot()
	if failures != 1 {
		t.Fatalf("failures=%d", failures)
	}
}

func TestOutcomeLogCopiesRecords(t *testing.T) {
	var log OutcomeLog
	log.Append(OutcomeRecord{JobID: "j1", Outcome: OutcomeSuccess})
	items := log.List()
	items[0].JobID = "changed"
	if log.List()[0].JobID != "j1" {
		t.Fatal("outcome log leaked mutable slice")
	}
	if log.Count(OutcomeSuccess) != 1 {
		t.Fatal("success count mismatch")
	}
}

type failingSource struct {
	count int
	err   error
	done  chan struct{}
}

func (f *failingSource) ActivateDue(context.Context, time.Time, int) (int, error) {
	defer close(f.done)
	return f.count, f.err
}

// TestAssignmentFailureDoesNotCountAsDue guards the recovery-task processing
// chain: when the backend fails to persist (for example a brief database
// disconnect during array recovery), the unpersisted results must not be
// folded into the "due/completed" counter that the monitoring surface reads,
// or on-call staff will believe the backlog has cleared.
func TestAssignmentFailureDoesNotCountAsDue(t *testing.T) {
	metrics := &Metrics{}
	source := &failingSource{count: 3, err: errors.New("database unavailable"), done: make(chan struct{})}
	worker := NewAssignmentWorker(source, time.Hour, nil, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	<-source.done
	cancel()

	runs, failures, due := metrics.Snapshot()
	if due != 0 {
		t.Fatalf("unpersisted results counted as completed: due=%d", due)
	}
	if runs != 1 || failures != 1 {
		t.Fatalf("runs=%d failures=%d", runs, failures)
	}
	if metrics.Failed() != 3 {
		t.Fatalf("failed=%d want 3", metrics.Failed())
	}
}

func TestHealthNeedsRecentRun(t *testing.T) {
	var health Health
	health.Start()
	now := time.Now().UTC()
	if err := health.Check(context.Background(), time.Minute, now); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("missing run error=%v", err)
	}
	health.RecordRun(now)
	if err := health.Check(context.Background(), time.Minute, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	health.Stop()
	if err := health.Check(context.Background(), time.Minute, now); !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped health error=%v", err)
	}
}
