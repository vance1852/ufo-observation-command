package service_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ufo-observation-command/internal/service"
	"ufo-observation-command/internal/telemetry"
)

func newTelemetryControl() *service.TelemetryControl {
	return service.NewTelemetryControl(telemetry.NewManager())
}

func scope(capture string) telemetry.Scope {
	return telemetry.Scope{TenantID: "institute-a", MissionID: "mission-1", BuoyID: "buoy-1", CaptureID: capture}
}

func checksum(parts ...[]byte) string {
	hash := sha256.New()
	for _, part := range parts {
		_, _ = hash.Write(part)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func openComplete(t *testing.T, control *service.TelemetryControl, capture string, parts ...[]byte) telemetry.Stream {
	t.Helper()
	stream, err := control.Open(scope(capture), len(parts), map[string]string{"depth": "6000m"})
	if err != nil {
		t.Fatal(err)
	}
	for index, part := range parts {
		if inserted, err := control.Append(t.Context(), stream.Key, index, fmt.Sprintf("digest-%d", index), part); err != nil || !inserted {
			t.Fatalf("append %d inserted=%v err=%v", index, inserted, err)
		}
	}
	stream, _ = control.Snapshot(stream.Key)
	return stream
}

func TestStreamIdentitySeparatesCaptureWindows(t *testing.T) {
	control := newTelemetryControl()
	first, err := control.Open(scope("capture-a"), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	second, err := control.Open(scope("capture-b"), 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	if first.Key == second.Key {
		t.Fatalf("capture windows share key %q", first.Key)
	}
}

func TestOpenRejectsNonPositiveSegmentPlan(t *testing.T) {
	control := newTelemetryControl()
	if _, err := control.Open(scope("invalid-plan"), 0, nil); !errors.Is(err, telemetry.ErrInvalid) {
		t.Fatalf("open err=%v", err)
	}
}

func TestAppendRejectsIndexOutsideDeclaredPlan(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("range"), 2, nil)
	if _, err := control.Append(t.Context(), stream.Key, 2, "digest", []byte("late")); !errors.Is(err, telemetry.ErrInvalid) {
		t.Fatalf("append err=%v", err)
	}
	snapshot, _ := control.Snapshot(stream.Key)
	if len(snapshot.Segments) != 0 {
		t.Fatalf("segments=%v", snapshot.Segments)
	}
}

func TestDuplicateSegmentWithDifferentDigestConflicts(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("duplicate-conflict"), 1, nil)
	_, _ = control.Append(t.Context(), stream.Key, 0, "digest-a", []byte("a"))
	if _, err := control.Append(t.Context(), stream.Key, 0, "digest-b", []byte("b")); !errors.Is(err, telemetry.ErrConflict) {
		t.Fatalf("duplicate err=%v", err)
	}
	snapshot, _ := control.Snapshot(stream.Key)
	if string(snapshot.Segments[0].Payload) != "a" {
		t.Fatalf("payload=%q", snapshot.Segments[0].Payload)
	}
}

func TestSameSegmentReplayIsIdempotent(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("duplicate-replay"), 1, nil)
	inserted, _ := control.Append(t.Context(), stream.Key, 0, "digest-a", []byte("a"))
	replayed, err := control.Append(t.Context(), stream.Key, 0, "digest-a", []byte("a"))
	if !inserted || replayed || err != nil {
		t.Fatalf("inserted=%v replayed=%v err=%v", inserted, replayed, err)
	}
}

func TestAppendCopiesCallerPayload(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("payload-owner"), 1, nil)
	payload := []byte("echo")
	_, _ = control.Append(t.Context(), stream.Key, 0, "digest", payload)
	payload[0] = 'X'
	snapshot, _ := control.Snapshot(stream.Key)
	if string(snapshot.Segments[0].Payload) != "echo" {
		t.Fatalf("stored=%q", snapshot.Segments[0].Payload)
	}
}

func TestSnapshotDoesNotExposeStoredSegmentsOrLabels(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("snapshot-owner"), 1, map[string]string{"zone": "hadal"})
	_, _ = control.Append(t.Context(), stream.Key, 0, "digest", []byte("echo"))
	first, _ := control.Snapshot(stream.Key)
	first.Labels["zone"] = "slope"
	segment := first.Segments[0]
	segment.Payload[0] = 'X'
	first.Segments[0] = segment
	again, _ := control.Snapshot(stream.Key)
	if again.Labels["zone"] != "hadal" || string(again.Segments[0].Payload) != "echo" {
		t.Fatalf("snapshot leaked=%+v", again)
	}
}

func TestFinalizeRejectsMissingSegmentsWithoutSealing(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("missing"), 2, nil)
	_, _ = control.Append(t.Context(), stream.Key, 0, "digest", []byte("a"))
	stream, _ = control.Snapshot(stream.Key)
	if _, err := control.Finalize(t.Context(), stream.Key, stream.Version, checksum([]byte("a")), nil); !errors.Is(err, telemetry.ErrConflict) {
		t.Fatalf("finalize err=%v", err)
	}
	after, _ := control.Snapshot(stream.Key)
	if after.State != "collecting" || len(after.Segments) != 1 {
		t.Fatalf("after=%+v", after)
	}
}

func TestChecksumFailurePreservesCollectedSegments(t *testing.T) {
	control := newTelemetryControl()
	stream := openComplete(t, control, "checksum", []byte("a"), []byte("b"))
	if _, err := control.Finalize(t.Context(), stream.Key, stream.Version, "wrong", nil); !errors.Is(err, telemetry.ErrConflict) {
		t.Fatalf("finalize err=%v", err)
	}
	after, _ := control.Snapshot(stream.Key)
	if after.State != "collecting" || len(after.Segments) != 2 {
		t.Fatalf("after=%+v", after)
	}
}

func TestFinalizeRejectsStaleStreamVersion(t *testing.T) {
	control := newTelemetryControl()
	initial, _ := control.Open(scope("version"), 1, nil)
	_, _ = control.Append(t.Context(), initial.Key, 0, "digest", []byte("a"))
	if _, err := control.Finalize(t.Context(), initial.Key, initial.Version, checksum([]byte("a")), nil); !errors.Is(err, telemetry.ErrConflict) {
		t.Fatalf("finalize err=%v", err)
	}
	after, _ := control.Snapshot(initial.Key)
	if after.State != "collecting" {
		t.Fatalf("state=%s", after.State)
	}
}

func TestAuditFailureRollsBackSealState(t *testing.T) {
	control := newTelemetryControl()
	stream := openComplete(t, control, "audit", []byte("a"))
	_, err := control.Finalize(t.Context(), stream.Key, stream.Version, checksum([]byte("a")), func(telemetry.Record) error { return errors.New("audit offline") })
	if err == nil {
		t.Fatal("audit failure accepted")
	}
	after, _ := control.Snapshot(stream.Key)
	if after.State != "collecting" || after.Version != stream.Version {
		t.Fatalf("after=%+v", after)
	}
}

func TestCanceledFinalizeLeavesStreamUntouched(t *testing.T) {
	control := newTelemetryControl()
	stream := openComplete(t, control, "cancel-finalize", []byte("a"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := control.Finalize(ctx, stream.Key, stream.Version, checksum([]byte("a")), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
	after, _ := control.Snapshot(stream.Key)
	if after.State != "collecting" {
		t.Fatalf("state=%s", after.State)
	}
}

func TestAwaitCompleteObservesLaterSegmentArrival(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("late-arrival"), 1, nil)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	waited := make(chan error, 1)
	go func() { waited <- control.AwaitComplete(ctx, stream.Key) }()
	time.Sleep(25 * time.Millisecond)
	if _, err := control.Append(t.Context(), stream.Key, 0, "digest", []byte("echo")); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-waited:
		if err != nil {
			t.Fatalf("await err=%v", err)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waiter did not observe the completed stream")
	}
}

func TestSealedStreamRejectsLateSegment(t *testing.T) {
	control := newTelemetryControl()
	stream := openComplete(t, control, "sealed", []byte("a"))
	_, err := control.Finalize(t.Context(), stream.Key, stream.Version, checksum([]byte("a")), nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := control.Append(t.Context(), stream.Key, 0, "late", []byte("late")); !errors.Is(err, telemetry.ErrConflict) {
		t.Fatalf("append err=%v", err)
	}
}

func TestAwaitCompleteStopsWhenCallerCancels(t *testing.T) {
	control := newTelemetryControl()
	stream, _ := control.Open(scope("await"), 1, nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := control.AwaitComplete(ctx, stream.Key)
	if !errors.Is(err, context.Canceled) || time.Since(start) > 200*time.Millisecond {
		t.Fatalf("err=%v elapsed=%s", err, time.Since(start))
	}
}

func TestDecodeContextInheritsParentCancellation(t *testing.T) {
	control := newTelemetryControl()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	_, err := control.Decode(ctx, time.Minute, telemetry.DecoderFunc(func(input []byte) ([]byte, error) { called = true; return input, nil }), "pcm", []byte("x"))
	if !errors.Is(err, context.Canceled) || called {
		t.Fatalf("err=%v called=%v", err, called)
	}
}

func TestRetryWaitReturnsOnCancellation(t *testing.T) {
	control := newTelemetryControl()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	err := control.WaitRetry(ctx, time.Second)
	if !errors.Is(err, context.Canceled) || time.Since(start) > 200*time.Millisecond {
		t.Fatalf("err=%v elapsed=%s", err, time.Since(start))
	}
}

func TestNilDecoderReturnsUnavailableInsteadOfPanicking(t *testing.T) {
	control := newTelemetryControl()
	_, err := control.Decode(t.Context(), time.Second, nil, "pcm", []byte("x"))
	if !errors.Is(err, telemetry.ErrUnavailable) {
		t.Fatalf("err=%v", err)
	}
}

func TestDecoderPanicBecomesTypedFailure(t *testing.T) {
	control := newTelemetryControl()
	_, err := control.Decode(t.Context(), time.Second, telemetry.DecoderFunc(func([]byte) ([]byte, error) { panic("corrupt table") }), "flac", []byte("x"))
	var typed *telemetry.DecodeError
	if !errors.As(err, &typed) || typed.Codec != "flac" {
		t.Fatalf("err=%T %v", err, err)
	}
}

func TestWrappedDecodeFailureKeepsHTTPClassification(t *testing.T) {
	control := newTelemetryControl()
	status, code := control.WrapAndClassify("recover segment", &telemetry.DecodeError{Codec: "flac", Cause: errors.New("bad frame")})
	if status != 422 || code != "decode_failed" {
		t.Fatalf("status=%d code=%s", status, code)
	}
}

func TestLimiterReleasesSlotAfterOperationError(t *testing.T) {
	control := newTelemetryControl()
	limiter := telemetry.NewLimiter(1)
	expected := errors.New("decode failed")
	err := control.RunLimited(t.Context(), limiter, func() error { return expected })
	if !errors.Is(err, expected) || limiter.InUse() != 0 {
		t.Fatalf("err=%v in_use=%d", err, limiter.InUse())
	}
	if err := control.RunLimited(t.Context(), limiter, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
}

func TestBatchDecoderReleasesEachResourceBeforeReturning(t *testing.T) {
	control := newTelemetryControl()
	tracker := &telemetry.ResourceTracker{}
	err := control.DecodeBatch(t.Context(), tracker, [][]byte{{1}, {2}}, func(input []byte) error {
		if input[0] == 2 {
			return errors.New("bad")
		}
		return nil
	})
	if err == nil || tracker.Open() != 0 {
		t.Fatalf("err=%v open=%d", err, tracker.Open())
	}
}

func TestStaleLeaseHolderCannotCommitAfterTakeover(t *testing.T) {
	control := newTelemetryControl()
	now := time.Now()
	tokenA, ok := control.AcquireWorker("worker-a", now, time.Second)
	if !ok {
		t.Fatal("first acquire failed")
	}
	tokenB, ok := control.AcquireWorker("worker-b", now.Add(time.Second), time.Minute)
	if !ok {
		t.Fatal("takeover failed")
	}
	if control.CommitWorker("worker-a", tokenA, now.Add(time.Second)) {
		t.Fatalf("stale owner committed with tokens a=%d b=%d", tokenA, tokenB)
	}
}

func TestLeaseTakeoverIssuesMonotonicFenceToken(t *testing.T) {
	control := newTelemetryControl()
	now := time.Now()
	first, _ := control.AcquireWorker("worker-a", now, time.Second)
	second, _ := control.AcquireWorker("worker-b", now.Add(time.Second), time.Second)
	third, _ := control.AcquireWorker("worker-c", now.Add(2*time.Second), time.Second)
	if !(first < second && second < third) {
		t.Fatalf("tokens=%d,%d,%d", first, second, third)
	}
}

func TestConcurrentChannelReservationDoesNotOversubscribe(t *testing.T) {
	control := newTelemetryControl()
	start := make(chan struct{})
	results := make(chan error, 2)
	var ready sync.WaitGroup
	ready.Add(2)
	for range 2 {
		go func() { ready.Done(); <-start; results <- control.ReserveChannel("north", 1) }()
	}
	ready.Wait()
	close(start)
	success := 0
	for range 2 {
		if <-results == nil {
			success++
		}
	}
	if success != 1 || control.ChannelUsage("north") != 1 {
		t.Fatalf("success=%d usage=%d", success, control.ChannelUsage("north"))
	}
}

func TestUnsubscribeClosesOnlyItsOwnDeliveryChannel(t *testing.T) {
	control := newTelemetryControl()
	first, stopFirst := control.Subscribe(1)
	second, stopSecond := control.Subscribe(1)
	defer stopSecond()
	stopFirst()
	if _, ok := <-first; ok {
		t.Fatal("first channel remains open")
	}
	control.Publish([]byte("echo"))
	if string(<-second) != "echo" {
		t.Fatal("second subscriber lost")
	}
}

func TestBroadcastCopiesPayloadForEverySubscriber(t *testing.T) {
	control := newTelemetryControl()
	first, stopFirst := control.Subscribe(1)
	defer stopFirst()
	second, stopSecond := control.Subscribe(1)
	defer stopSecond()
	payload := []byte("echo")
	control.Publish(payload)
	payload[0] = 'X'
	one, two := <-first, <-second
	one[1] = 'Y'
	if string(two) != "echo" {
		t.Fatalf("subscribers share payload one=%q two=%q", one, two)
	}
}

type countingSink struct {
	persists atomic.Int32
	acks     atomic.Int32
}

func (s *countingSink) Persist(context.Context, []byte) error { s.persists.Add(1); return nil }
func (s *countingSink) Acknowledge(context.Context, []byte) error {
	if s.acks.Add(1) < 2 {
		return errors.New("temporary ack")
	}
	return nil
}

func TestAckRetryDoesNotPersistRecoveredPayloadTwice(t *testing.T) {
	control := newTelemetryControl()
	sink := &countingSink{}
	if err := control.Deliver(t.Context(), sink, []byte("echo"), 2, time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if sink.persists.Load() != 1 || sink.acks.Load() != 2 {
		t.Fatalf("persists=%d acks=%d", sink.persists.Load(), sink.acks.Load())
	}
}

func TestDeliveryRetryHonorsOverallCancellationBudget(t *testing.T) {
	control := newTelemetryControl()
	sink := &countingSink{}
	sink.acks.Store(-100)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	start := time.Now()
	err := control.Deliver(ctx, sink, []byte("echo"), 5, time.Second)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 300*time.Millisecond {
		t.Fatalf("err=%v elapsed=%s", err, time.Since(start))
	}
}

func TestWorkerStopsBeforeConsumingAfterCancellation(t *testing.T) {
	control := newTelemetryControl()
	source := make(chan []byte, 1)
	source <- []byte("echo")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var handled atomic.Int32
	err := control.RunWorker(ctx, source, func([]byte) error { handled.Add(1); return nil })
	if !errors.Is(err, context.Canceled) || handled.Load() != 0 {
		t.Fatalf("err=%v handled=%d", err, handled.Load())
	}
}
