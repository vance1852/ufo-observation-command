package worker

import (
	"context"
	"sync/atomic"
	"time"
)

type Health struct {
	running   atomic.Bool
	lastRun   atomic.Int64
	lastError atomic.Int64
}

func (h *Health) Start()                    { h.running.Store(true) }
func (h *Health) Stop()                     { h.running.Store(false) }
func (h *Health) RecordRun(now time.Time)   { h.lastRun.Store(now.UnixNano()) }
func (h *Health) RecordError(now time.Time) { h.lastError.Store(now.UnixNano()) }
func (h *Health) Check(ctx context.Context, staleAfter time.Duration, now time.Time) error {
	if !h.running.Load() {
		return context.Canceled
	}
	last := time.Unix(0, h.lastRun.Load())
	if last.IsZero() || now.Sub(last) > staleAfter {
		return context.DeadlineExceeded
	}
	return nil
}
