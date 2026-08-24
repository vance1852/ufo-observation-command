package telemetry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

type Manager struct {
	mu      sync.Mutex
	streams map[string]Stream
	now     func() time.Time
	changed chan struct{}
}

func NewManager() *Manager {
	return &Manager{streams: make(map[string]Stream), now: time.Now, changed: make(chan struct{})}
}

func StreamKey(scope Scope) (string, error) {
	parts := []string{scope.TenantID, scope.MissionID, scope.BuoyID, scope.CaptureID}
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			return "", fmt.Errorf("stream scope is incomplete: %w", ErrInvalid)
		}
	}
	return strings.Join(parts, "\x00"), nil
}

func (m *Manager) Open(scope Scope, expected int, labels map[string]string) (Stream, error) {
	key, err := StreamKey(scope)
	if err != nil {
		return Stream{}, err
	}
	if expected <= 0 {
		return Stream{}, fmt.Errorf("expected segment count must be positive: %w", ErrInvalid)
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.streams[key]; exists {
		return Stream{}, fmt.Errorf("stream already exists: %w", ErrConflict)
	}
	stream := Stream{Key: key, Scope: scope, ExpectedSegments: expected, Segments: make(map[int]Segment), Labels: cloneLabels(labels), State: "collecting", Version: 1}
	m.streams[key] = cloneStream(stream)
	m.signalLocked()
	return cloneStream(stream), nil
}

func (m *Manager) Append(ctx context.Context, key string, index int, digest string, payload []byte) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stream, ok := m.streams[key]
	if !ok {
		return false, fmt.Errorf("stream not found: %w", ErrConflict)
	}
	if stream.State != "collecting" {
		return false, fmt.Errorf("sealed stream cannot accept segments: %w", ErrConflict)
	}
	if index < 0 || index >= stream.ExpectedSegments || digest == "" || len(payload) == 0 {
		return false, fmt.Errorf("segment input is invalid: %w", ErrInvalid)
	}
	if prior, exists := stream.Segments[index]; exists {
		if prior.Digest == digest {
			return false, nil
		}
		return false, fmt.Errorf("segment index has different content: %w", ErrConflict)
	}
	stream.Segments[index] = Segment{Index: index, Digest: digest, Payload: append([]byte(nil), payload...)}
	stream.Version++
	m.streams[key] = stream
	m.signalLocked()
	return true, nil
}

func (m *Manager) Snapshot(key string) (Stream, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	stream, ok := m.streams[key]
	return cloneStream(stream), ok
}

func (m *Manager) Finalize(ctx context.Context, key string, expectedVersion int64, checksum string, audit func(Record) error) (Record, error) {
	if err := ctx.Err(); err != nil {
		return Record{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	stream, ok := m.streams[key]
	if !ok || stream.State != "collecting" {
		return Record{}, fmt.Errorf("stream cannot be finalized: %w", ErrConflict)
	}
	if stream.Version != expectedVersion {
		return Record{}, fmt.Errorf("stream version changed: %w", ErrConflict)
	}
	if len(stream.Segments) != stream.ExpectedSegments {
		return Record{}, fmt.Errorf("stream still has missing segments: %w", ErrConflict)
	}
	payload := assemble(stream)
	digest := sha256.Sum256(payload)
	actual := hex.EncodeToString(digest[:])
	if checksum == "" || checksum != actual {
		return Record{}, fmt.Errorf("stream checksum mismatch: %w", ErrConflict)
	}
	record := Record{StreamKey: key, Checksum: actual, Payload: append([]byte(nil), payload...), SealedAt: m.now().UTC()}
	if audit != nil {
		if err := audit(record); err != nil {
			return Record{}, fmt.Errorf("write seal audit: %w", err)
		}
	}
	stream.State = "sealed"
	stream.Version++
	m.streams[key] = stream
	m.signalLocked()
	return record, nil
}

func (m *Manager) AwaitComplete(ctx context.Context, key string) error {
	for {
		m.mu.Lock()
		stream, ok := m.streams[key]
		if ok && len(stream.Segments) == stream.ExpectedSegments {
			m.mu.Unlock()
			return nil
		}
		changed := m.changed
		m.mu.Unlock()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-changed:
		}
	}
}

func (m *Manager) signalLocked() {
	close(m.changed)
	m.changed = make(chan struct{})
}

func assemble(stream Stream) []byte {
	indexes := make([]int, 0, len(stream.Segments))
	for index := range stream.Segments {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)
	var output []byte
	for _, index := range indexes {
		output = append(output, stream.Segments[index].Payload...)
	}
	return output
}

func cloneLabels(value map[string]string) map[string]string {
	output := make(map[string]string, len(value))
	for key, item := range value {
		output[key] = item
	}
	return output
}

func cloneStream(value Stream) Stream {
	clone := value
	clone.Labels = cloneLabels(value.Labels)
	clone.Segments = make(map[int]Segment, len(value.Segments))
	for index, segment := range value.Segments {
		segment.Payload = append([]byte(nil), segment.Payload...)
		clone.Segments[index] = segment
	}
	return clone
}
