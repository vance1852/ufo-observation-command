package worker

import ("context"; "errors"; "log/slog"; "testing"; "time"; "ufo-observation-command/internal/domain")

type ufoFailSource0022 struct{}
func (ufoFailSource0022) ExpiringRecoveryJobs(context.Context, time.Time, int) ([]domain.RecoveryJob, error) { return []domain.RecoveryJob{{ID: "job-expiring"}}, errors.New("query timeout") }

func TestWorkerFailureDoesNotCountCompletedWork0022(t *testing.T) {
	metrics := &Metrics{}
	r := NewRecoveryJobExpiryReconciler(ufoFailSource0022{}, slog.Default(), metrics)
	if err := r.Reconcile(context.Background(), time.Now()); err == nil { t.Fatal("reconcile unexpectedly succeeded") }
	_, failures, due := metrics.Snapshot()
	if failures != 1 || due != 0 { t.Fatalf("failures=%d due=%d", failures, due) }
}
