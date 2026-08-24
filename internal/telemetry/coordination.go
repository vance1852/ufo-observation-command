package telemetry

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type FenceLease struct {
	mu        sync.Mutex
	owner     string
	token     uint64
	expiresAt time.Time
}

func (l *FenceLease) Acquire(owner string, now time.Time, ttl time.Duration) (uint64, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if owner == "" || ttl <= 0 || l.owner != "" && now.Before(l.expiresAt) && l.owner != owner {
		return 0, false
	}
	l.token++
	l.owner = owner
	l.expiresAt = now.Add(ttl)
	return l.token, true
}

func (l *FenceLease) Commit(owner string, token uint64, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return owner != "" && owner == l.owner && token == l.token && now.Before(l.expiresAt)
}

type Capacity struct {
	mu   sync.Mutex
	used map[string]int
}

func NewCapacity() *Capacity { return &Capacity{used: make(map[string]int)} }

func (c *Capacity) Reserve(key string, limit int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if key == "" || limit <= 0 {
		return fmt.Errorf("capacity scope is invalid: %w", ErrInvalid)
	}
	if c.used[key] >= limit {
		return fmt.Errorf("capacity exhausted: %w", ErrConflict)
	}
	c.used[key]++
	return nil
}

func (c *Capacity) Used(key string) int { c.mu.Lock(); defer c.mu.Unlock(); return c.used[key] }

type Broadcaster struct {
	mu          sync.Mutex
	next        int
	subscribers map[int]chan []byte
}

func NewBroadcaster() *Broadcaster { return &Broadcaster{subscribers: make(map[int]chan []byte)} }

func (b *Broadcaster) Subscribe(buffer int) (<-chan []byte, func()) {
	b.mu.Lock()
	id := b.next
	b.next++
	channel := make(chan []byte, buffer)
	b.subscribers[id] = channel
	b.mu.Unlock()
	var once sync.Once
	return channel, func() {
		once.Do(func() {
			b.mu.Lock()
			delete(b.subscribers, id)
			close(channel)
			b.mu.Unlock()
		})
	}
}

func (b *Broadcaster) Publish(event []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, subscriber := range b.subscribers {
		subscriber <- append([]byte(nil), event...)
	}
}

type Sink interface {
	Persist(context.Context, []byte) error
	Acknowledge(context.Context, []byte) error
}

func Deliver(ctx context.Context, sink Sink, payload []byte, attempts int, wait time.Duration) error {
	if attempts <= 0 {
		return fmt.Errorf("attempt count is invalid: %w", ErrInvalid)
	}
	if err := sink.Persist(ctx, payload); err != nil {
		return err
	}
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := sink.Acknowledge(ctx, payload); err == nil {
			return nil
		} else {
			last = err
		}
		if attempt+1 < attempts {
			if err := WaitRetry(ctx, wait); err != nil {
				return err
			}
		}
	}
	return fmt.Errorf("acknowledge delivery: %w", last)
}

func RunWorker(ctx context.Context, source <-chan []byte, handle func([]byte) error) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case payload, ok := <-source:
			if !ok {
				return nil
			}
			if err := handle(payload); err != nil {
				return err
			}
		}
	}
}
