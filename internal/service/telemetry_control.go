package service

import (
	"context"
	"time"

	"ufo-observation-command/internal/telemetry"
)

type TelemetryControl struct {
	manager  *telemetry.Manager
	capacity *telemetry.Capacity
	lease    telemetry.FenceLease
	bus      *telemetry.Broadcaster
}

func NewTelemetryControl(manager *telemetry.Manager) *TelemetryControl {
	if manager == nil {
		manager = telemetry.NewManager()
	}
	return &TelemetryControl{manager: manager, capacity: telemetry.NewCapacity(), bus: telemetry.NewBroadcaster()}
}

func (c *TelemetryControl) Open(scope telemetry.Scope, expected int, labels map[string]string) (telemetry.Stream, error) {
	return c.manager.Open(scope, expected, labels)
}

func (c *TelemetryControl) Append(ctx context.Context, key string, index int, digest string, payload []byte) (bool, error) {
	return c.manager.Append(ctx, key, index, digest, payload)
}

func (c *TelemetryControl) Snapshot(key string) (telemetry.Stream, bool) {
	return c.manager.Snapshot(key)
}

func (c *TelemetryControl) Finalize(ctx context.Context, key string, version int64, checksum string, audit func(telemetry.Record) error) (telemetry.Record, error) {
	return c.manager.Finalize(ctx, key, version, checksum, audit)
}

func (c *TelemetryControl) AwaitComplete(ctx context.Context, key string) error {
	return c.manager.AwaitComplete(ctx, key)
}

func (c *TelemetryControl) Decode(ctx context.Context, timeout time.Duration, decoder telemetry.Decoder, codec string, payload []byte) ([]byte, error) {
	decodeCtx, cancel := telemetry.DerivedContext(ctx, timeout)
	defer cancel()
	if err := decodeCtx.Err(); err != nil {
		return nil, err
	}
	return telemetry.Decode(decoder, codec, payload)
}

func (c *TelemetryControl) ReserveChannel(scope string, limit int) error {
	return c.capacity.Reserve(scope, limit)
}

func (c *TelemetryControl) ChannelUsage(scope string) int { return c.capacity.Used(scope) }

func (c *TelemetryControl) AcquireWorker(owner string, now time.Time, ttl time.Duration) (uint64, bool) {
	return c.lease.Acquire(owner, now, ttl)
}

func (c *TelemetryControl) CommitWorker(owner string, token uint64, now time.Time) bool {
	return c.lease.Commit(owner, token, now)
}

func (c *TelemetryControl) WaitRetry(ctx context.Context, delay time.Duration) error {
	return telemetry.WaitRetry(ctx, delay)
}

func (c *TelemetryControl) WrapAndClassify(operation string, err error) (int, string) {
	return telemetry.Classify(telemetry.Wrap(operation, err))
}

func (c *TelemetryControl) RunLimited(ctx context.Context, limiter *telemetry.Limiter, operation func() error) error {
	return limiter.Run(ctx, operation)
}

func (c *TelemetryControl) DecodeBatch(ctx context.Context, tracker *telemetry.ResourceTracker, inputs [][]byte, decode func([]byte) error) error {
	return telemetry.DecodeBatch(ctx, tracker, inputs, decode)
}

func (c *TelemetryControl) Subscribe(buffer int) (<-chan []byte, func()) {
	return c.bus.Subscribe(buffer)
}

func (c *TelemetryControl) Publish(payload []byte) { c.bus.Publish(payload) }

func (c *TelemetryControl) Deliver(ctx context.Context, sink telemetry.Sink, payload []byte, attempts int, wait time.Duration) error {
	return telemetry.Deliver(ctx, sink, payload, attempts, wait)
}

func (c *TelemetryControl) RunWorker(ctx context.Context, source <-chan []byte, handle func([]byte) error) error {
	return telemetry.RunWorker(ctx, source, handle)
}
