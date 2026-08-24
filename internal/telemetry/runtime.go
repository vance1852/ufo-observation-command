package telemetry

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

func DerivedContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, timeout)
}

func WaitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func Decode(decoder Decoder, codec string, payload []byte) (output []byte, err error) {
	if decoder == nil {
		return nil, fmt.Errorf("decoder is unavailable: %w", ErrUnavailable)
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = &DecodeError{Codec: codec, Cause: fmt.Errorf("decoder panic: %v", recovered)}
			output = nil
		}
	}()
	output, err = decoder.Decode(append([]byte(nil), payload...))
	if err != nil {
		return nil, &DecodeError{Codec: codec, Cause: err}
	}
	return append([]byte(nil), output...), nil
}

func Classify(err error) (int, string) {
	var decode *DecodeError
	switch {
	case errors.As(err, &decode):
		return 422, "decode_failed"
	case errors.Is(err, ErrUnauthorized):
		return 403, "forbidden"
	case errors.Is(err, ErrConflict):
		return 409, "conflict"
	case errors.Is(err, ErrInvalid):
		return 400, "invalid_request"
	case errors.Is(err, ErrUnavailable):
		return 503, "unavailable"
	default:
		return 500, "internal_error"
	}
}

func Wrap(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

type Limiter struct {
	tokens chan struct{}
}

func NewLimiter(capacity int) *Limiter { return &Limiter{tokens: make(chan struct{}, capacity)} }

func (l *Limiter) Run(ctx context.Context, operation func() error) error {
	select {
	case l.tokens <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-l.tokens }()
	return operation()
}

func (l *Limiter) InUse() int { return len(l.tokens) }

type ResourceTracker struct {
	mu   sync.Mutex
	open int
}

func (r *ResourceTracker) Acquire() func() {
	r.mu.Lock()
	r.open++
	r.mu.Unlock()
	var once sync.Once
	return func() { once.Do(func() { r.mu.Lock(); r.open--; r.mu.Unlock() }) }
}

func (r *ResourceTracker) Open() int { r.mu.Lock(); defer r.mu.Unlock(); return r.open }

func DecodeBatch(ctx context.Context, tracker *ResourceTracker, inputs [][]byte, decode func([]byte) error) error {
	for _, input := range inputs {
		if err := ctx.Err(); err != nil {
			return err
		}
		release := tracker.Acquire()
		err := decode(input)
		release()
		if err != nil {
			return err
		}
	}
	return nil
}
