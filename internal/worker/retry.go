package worker

import (
	"context"
	"fmt"
	"time"
)

type RetryPolicy struct {
	Attempts  int
	BaseDelay time.Duration
	MaxDelay  time.Duration
}

func (p RetryPolicy) Delay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	delay := p.BaseDelay
	for i := 1; i < attempt; i++ {
		if delay >= p.MaxDelay/2 {
			return p.MaxDelay
		}
		delay *= 2
	}
	if delay > p.MaxDelay {
		return p.MaxDelay
	}
	return delay
}

func RunWithRetry(ctx context.Context, policy RetryPolicy, operation func(context.Context) error) error {
	if policy.Attempts < 1 {
		return fmt.Errorf("retry attempts must be positive")
	}
	var last error
	for attempt := 1; attempt <= policy.Attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := operation(ctx); err == nil {
			return nil
		} else {
			last = err
		}
		if attempt == policy.Attempts {
			break
		}
		timer := time.NewTimer(policy.Delay(attempt))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", policy.Attempts, last)
}
