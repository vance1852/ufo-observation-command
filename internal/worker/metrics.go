package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
}

func (m *Metrics) RecordRun()     { m.runs.Add(1) }
func (m *Metrics) RecordFailure() { m.failures.Add(1) }

// RecordDue advances the recovered-activations counter only. Callers must
// invoke it after a single confirmed success; failed or partial rounds must
// never advance it, or retries of the same round double-count and the daily
// report credits failed attempts as actual recoveries.
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}
