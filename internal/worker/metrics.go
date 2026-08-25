package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()     { m.runs.Add(1) }
func (m *Metrics) RecordFailure() { m.failures.Add(1) }
func (m *Metrics) RecordDue(count int) {
	// due aggregates only effective (committed) work so cross-region
	// capacity accounting never counts rolled-back scan candidates.
	m.due.Add(int64(count))
}
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}
