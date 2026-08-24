package worker

import (
	"sync"
	"time"
)

type Outcome string

const (
	OutcomeSuccess          Outcome = "success"
	OutcomeRetry            Outcome = "retry"
	OutcomePermanentFailure Outcome = "permanent_failure"
	OutcomeCancelled        Outcome = "cancelled"
)

type OutcomeRecord struct {
	JobID    string
	Outcome  Outcome
	Attempts int
	Error    string
	At       time.Time
}

type OutcomeLog struct {
	mu      sync.RWMutex
	records []OutcomeRecord
}

func (l *OutcomeLog) Append(record OutcomeRecord) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.records = append(l.records, record)
}
func (l *OutcomeLog) List() []OutcomeRecord {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return append([]OutcomeRecord(nil), l.records...)
}
func (l *OutcomeLog) Count(outcome Outcome) int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	count := 0
	for _, record := range l.records {
		if record.Outcome == outcome {
			count++
		}
	}
	return count
}
