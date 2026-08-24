package worker

import ("context"; "errors"; "log/slog"; "testing"; "time")

type ufoFailSource0001 struct{}
func (ufoFailSource0001) ActivateDue(context.Context, time.Time, int) (int, error) { return 4, errors.New("database unavailable") }

func TestAssignmentWorkerDoesNotCountFailedActivationAsDue(t *testing.T) {
	metrics := &Metrics{}
	ctx, cancel := context.WithCancel(context.Background())
	source := ufoFailSource0001{}
	w := NewAssignmentWorker(source, time.Hour, slog.New(slog.NewTextHandler(testWriter0001{t}, nil)), metrics)
	go func(){ time.Sleep(20*time.Millisecond); cancel() }()
	if err := w.Run(ctx); !errors.Is(err, context.Canceled) { t.Fatalf("run err=%v", err) }
	_, failures, due := metrics.Snapshot()
	if failures != 1 || due != 0 { t.Fatalf("metrics failures=%d due=%d", failures, due) }
}
type testWriter0001 struct{ t *testing.T }
func (w testWriter0001) Write(p []byte)(int,error){ w.t.Helper(); return len(p),nil }
