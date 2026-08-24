package arrayops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func RunSteps(ctx context.Context, attempts int, backoff time.Duration, step func(context.Context, int) error) error {
	if attempts <= 0 || step == nil {
		return fmt.Errorf("worker retry policy is invalid: %w", ErrInvalid)
	}
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		last = step(ctx, attempt)
		if last == nil {
			return nil
		}
		if attempt < attempts {
			if err := WaitBackoff(ctx, backoff); err != nil {
				return err
			}
		}
	}
	return last
}

func CollectResults(ctx context.Context, jobs []string, run func(context.Context, string) error) []error {
	results := make(chan error, len(jobs))
	var workers sync.WaitGroup
	for _, job := range jobs {
		job := job
		workers.Add(1)
		go func() {
			defer workers.Done()
			results <- run(ctx, job)
		}()
	}
	go func() {
		workers.Wait()
		close(results)
	}()
	collected := make([]error, 0, len(jobs))
	for result := range results {
		collected = append(collected, result)
	}
	return collected
}

type OnceStep struct {
	mu      sync.Mutex
	seen    map[string]struct{}
	execute func(context.Context, string) error
}

func NewOnceStep(execute func(context.Context, string) error) *OnceStep {
	return &OnceStep{seen: make(map[string]struct{}), execute: execute}
}

func (s *OnceStep) Run(ctx context.Context, key string) error {
	s.mu.Lock()
	if _, exists := s.seen[key]; exists {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()
	if err := s.execute(ctx, key); err != nil {
		return err
	}
	s.mu.Lock()
	s.seen[key] = struct{}{}
	s.mu.Unlock()
	return nil
}
