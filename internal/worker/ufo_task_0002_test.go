package worker

import ("context"; "errors"; "log/slog"; "testing"; "time")

type ufoFailSource0002 struct{}
func (ufoFailSource0002) ActivateDue(context.Context, time.Time, int) (int, error) { return 5, errors.New("activation write failed") }

func TestWorkerFailureDoesNotCountCompletedWork0002(t *testing.T) {
	metrics := &Metrics{}
	ctx, cancel := context.WithCancel(context.Background())
	w := NewAssignmentWorker(ufoFailSource0002{}, time.Hour, slog.Default(), metrics)
	go func(){ time.Sleep(15*time.Millisecond); cancel() }()
	if err := w.Run(ctx); !errors.Is(err, context.Canceled) { t.Fatalf("run err=%v", err) }
	_, failures, due := metrics.Snapshot()
	if failures != 1 || due != 0 { t.Fatalf("failures=%d due=%d", failures, due) }
}
