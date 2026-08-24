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
