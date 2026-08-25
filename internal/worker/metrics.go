package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()          { m.runs.Add(1) }
func (m *Metrics) RecordFailure()      { m.failures.Add(1) }

// RecordDue records a completion quantity observed in the daily report.
// Only quantities backed by a traceable persistent record (a persisted
// recovery_jobs row or audit_events entry) may be recorded here, so the
// report can always be explained by the database. A no-op reconcile pass
// or a failed transaction that rolled back must not contribute a count,
// since nothing durable explains it.
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}
