package service

import (
	"context"
	"encoding/json"
	"fmt"

	"ufo-observation-command/internal/idempotency"
)

type IdempotencyStore interface {
	Get(context.Context, string) (idempotency.Record, bool, error)
	Put(context.Context, idempotency.Record) error
}

func ReplayOr[T any](ctx context.Context, store IdempotencyStore, key string, body []byte, create func() (int, T, error)) (int, T, error) {
	var zero T
	if key == "" {
		return 0, zero, fmt.Errorf("idempotency key is required")
	}
	if record, ok, err := store.Get(ctx, key); err != nil {
		return 0, zero, err
	} else if ok {
		if !record.Matches(body) {
			return 0, zero, fmt.Errorf("request hash differs: idempotency conflict")
		}
		var replayed T
		if err := json.Unmarshal(record.ResponseBody, &replayed); err != nil {
			return 0, zero, fmt.Errorf("decode idempotent response: %w", err)
		}
		return record.ResponseCode, replayed, nil
	}
	code, value, err := create()
	if err != nil {
		return 0, zero, err
	}
	record, err := idempotency.NewRecord(key, body, code, value)
	if err != nil {
		return 0, zero, err
	}
	if err := store.Put(ctx, record); err != nil {
		return 0, zero, err
	}
	return code, value, nil
}
