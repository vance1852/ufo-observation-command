package worker

import "sync/atomic"

type Metrics struct {
	runs      atomic.Int64
	failures  atomic.Int64
	due       atomic.Int64 // statistics baseline: successfully processed submissions only
	failedDue atomic.Int64 // abnormal batches, isolated from the baseline
}

func (m *Metrics) RecordRun() { m.runs.Add(1) }

func (m *Metrics) RecordFailure() { m.failures.Add(1) }

// RecordDue advances the statistics baseline by the number of submissions the
// scheduler successfully processed this round.
func (m *Metrics) RecordDue(count int) { m.due.Add(int64(count)) }

// RecordFailedDue records an abnormal batch (for example a downstream timeout
// received while the buoy group was stopping) without contaminating the
// statistics baseline. Abnormal batches are tracked separately so subsequent
// verification of the real submission volume stays accurate.
func (m *Metrics) RecordFailedDue(count int) {
	if count > 0 {
		m.failedDue.Add(int64(count))
	}
}

// Snapshot returns the statistics baseline used to verify the real submission
// volume: scheduler runs, failed batches, and successfully processed
// submissions. Abnormal batches recorded via RecordFailedDue are deliberately
// excluded so they cannot affect verification.
func (m *Metrics) Snapshot() (runs, failures, due int64) {
	return m.runs.Load(), m.failures.Load(), m.due.Load()
}

// FailedDue returns the isolated abnormal-batch count so it can be reported
// without affecting the statistics baseline.
func (m *Metrics) FailedDue() int64 { return m.failedDue.Load() }

// ReestablishBaseline clears the accumulated abnormal-batch records so
// verification of the next round starts from a clean baseline. Abnormal
// batches never re-enter the submission-volume baseline.
func (m *Metrics) ReestablishBaseline() {
	m.failedDue.Store(0)
}
