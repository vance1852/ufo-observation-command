package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
	rolled   atomic.Int64
}

func (m *Metrics) RecordRun()     { m.runs.Add(1) }
func (m *Metrics) RecordFailure() { m.failures.Add(1) }

// RecordRolledBack records recovery jobs whose write transaction timed out and
// rolled back. These are neither processed nor due: counting them under
// RecordDue would make the console claim "all processed" while the database
// still holds the pending rows. The rolled-back counter keeps monitoring in
// sync with the post-rollback state so a failed write cannot masquerade as a
// completed one.
func (m *Metrics) RecordRolledBack(count int) {
	if count > 0 {
		m.rolled.Add(int64(count))
	}
}

func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}

// RolledBackSnapshot returns the count of recovery jobs whose write
// transaction timed out and rolled back since the metrics were created.
func (m *Metrics) RolledBackSnapshot() int64 { return m.rolled.Load() }
