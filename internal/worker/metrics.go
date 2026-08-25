package worker

import "sync/atomic"

type Metrics struct {
	runs     atomic.Int64
	failures atomic.Int64
	due      atomic.Int64
	scanned  atomic.Int64
}

func (m *Metrics) RecordRun()          { m.runs.Add(1) }
func (m *Metrics) RecordFailure()      { m.failures.Add(1) }
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }

// RecordScanned tracks scanned compensation candidates separately from the
// due/success counter. Candidates are not execution results: until an attempt
// is confirmed, they must not inflate the success metric, otherwise cross-site
// aggregation overcounts consumed capacity.
func (m *Metrics) RecordScanned(count int) { m.scanned.Add(int64(count)) }

// RecordFailedDue0014 records an anomalous activation run. It counts the
// scanned candidates as candidates (not as successful/due items) and the run
// as a single failure. It deliberately does NOT add count to the due counter,
// so a delayed replica or conflicting activation cannot prematurely consume
// capacity in the aggregated success metric.
func (m *Metrics) RecordFailedDue0014(count int) {
	m.failures.Add(1)
	m.scanned.Add(int64(count))
}

func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}

// ScannedCount returns the number of scanned compensation candidates that have
// not been promoted to confirmed execution results.
func (m *Metrics) ScannedCount() int64 { return m.scanned.Load() }
