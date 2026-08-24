package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/idempotency"
)

type PersistentIdempotency struct {
	store interface {
		GetIdempotency(context.Context, string) (idempotency.Record, bool, error)
		PutIdempotency(context.Context, idempotency.Record) error
	}
}

func NewPersistentIdempotency(store interface {
	GetIdempotency(context.Context, string) (idempotency.Record, bool, error)
	PutIdempotency(context.Context, idempotency.Record) error
}) *PersistentIdempotency {
	return &PersistentIdempotency{store: store}
}

func (p *PersistentIdempotency) Get(ctx context.Context, key string) (idempotency.Record, bool, error) {
	return p.store.GetIdempotency(ctx, key)
}
func (p *PersistentIdempotency) Put(ctx context.Context, record idempotency.Record) error {
	return p.store.PutIdempotency(ctx, record)
}

func (s *Service) IdempotentCreate(ctx context.Context, store IdempotencyStore, key string, body []byte, create func() (int, any, error)) (int, []byte, error) {
	record, found, err := store.Get(ctx, key)
	if err != nil {
		return 0, nil, err
	}
	if found {
		if !record.Matches(body) {
			return 0, nil, fmt.Errorf("idempotency key reused with different request: %w", domain.ErrIdempotency)
		}
		return record.ResponseCode, append([]byte(nil), record.ResponseBody...), nil
	}
	code, value, err := create()
	if err != nil {
		return 0, nil, err
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return 0, nil, fmt.Errorf("marshal idempotent response: %w", err)
	}
	newRecord, err := idempotency.NewRecord(key, body, code, value)
	if err != nil {
		return 0, nil, err
	}
	if err := store.Put(ctx, newRecord); err != nil {
		return 0, nil, err
	}
	return code, payload, nil
}
