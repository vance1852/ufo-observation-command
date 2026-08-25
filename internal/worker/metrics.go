package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()     { m.runs.Add(1) }
func (m *Metrics) RecordFailure() { m.failures.Add(1) }

// RecordDue records the count of pending work observed during a successful
// pass. It is deliberately additive per pass, so callers must only invoke it
// when work was actually surfaced and not merely observed in a failed round:
// on failure no work has been delivered, so feeding the partial count back
// here would carry stale quantity into the next round.
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}
