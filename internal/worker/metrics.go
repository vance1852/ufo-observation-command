package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()     { m.runs.Add(1) }
func (m *Metrics) RecordFailure() { m.failures.Add(1) }

// RecordFailedDue records a failed reconciliation pass without inflating the
// completed/due counter. A failed background query must not pollute the
// handover report's completion count, which should only reflect audit-backed
// successes.
func (m *Metrics) RecordFailedDue(count int) {
	m.failures.Add(int64(count))
}
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}
