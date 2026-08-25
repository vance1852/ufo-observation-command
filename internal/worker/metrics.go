package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()          { m.runs.Add(1) }
func (m *Metrics) RecordFailure()      { m.failures.Add(1) }
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}

// Reset clears the run, failure and completion counters. It is invoked when a
// reconciler restarts or recovers so that residual error counts from a prior,
// conflict-disrupted scan do not leak into the freshly re-established metrics.
func (m *Metrics) Reset() {
	m.runs.Store(0)
	m.failures.Store(0)
	m.due.Store(0)
}
