package worker

import ("context"; "errors"; "log/slog"; "testing"; "time"; "ufo-observation-command/internal/repository")

type ufoFailSource0012 struct{}
func (ufoFailSource0012) MarkExpiredRecoveryJobs(context.Context, time.Time, int) (repository.ReconcileResult, error) { return repository.ReconcileResult{Marked: 3}, errors.New("lease store unavailable") }

func TestWorkerFailureDoesNotCountCompletedWork0012(t *testing.T) {
	metrics := &Metrics{}
	r := NewExpirationReconciler(ufoFailSource0012{}, slog.Default(), metrics)
	if err := r.Reconcile(context.Background(), time.Now()); err == nil { t.Fatal("reconcile unexpectedly succeeded") }
	_, failures, due := metrics.Snapshot()
	if failures != 1 || due != 0 { t.Fatalf("failures=%d due=%d", failures, due) }
}
