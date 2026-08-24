package arrayops

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func WaitBackoff(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func DerivedOperationContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

type Lease struct {
	mu        sync.Mutex
	owner     string
	expiresAt time.Time
}

func (l *Lease) Acquire(owner string, now time.Time, ttl time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if owner == "" || ttl <= 0 {
		return false
	}
	if l.owner != "" && now.Before(l.expiresAt) && l.owner != owner {
		return false
	}
	l.owner = owner
	l.expiresAt = now.Add(ttl)
	return true
}

func (l *Lease) Renew(owner string, now time.Time, ttl time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if owner == "" || owner != l.owner || !now.Before(l.expiresAt) || ttl <= 0 {
		return false
	}
	l.expiresAt = now.Add(ttl)
	return true
}

func (l *Lease) Release(owner string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if owner == "" || owner != l.owner {
		return false
	}
	l.owner = ""
	l.expiresAt = time.Time{}
	return true
}

type Capacity struct {
	mu   sync.Mutex
	used map[string]int
}

func NewCapacity() *Capacity {
	return &Capacity{used: make(map[string]int)}
}

func (c *Capacity) Reserve(key string, limit int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if key == "" || limit <= 0 {
		return fmt.Errorf("capacity input is invalid: %w", ErrInvalid)
	}
	if c.used[key] >= limit {
		return fmt.Errorf("capacity exhausted: %w", ErrConflict)
	}
	c.used[key]++
	return nil
}

func (c *Capacity) Used(key string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.used[key]
}
