package service

import (
	"context"
	"testing"

	"ufo-observation-command/internal/idempotency"
)

type replayStore struct {
	record idempotency.Record
	found  bool
}

func (s *replayStore) Get(context.Context, string) (idempotency.Record, bool, error) {
	return s.record, s.found, nil
}

func (s *replayStore) Put(_ context.Context, record idempotency.Record) error {
	s.record, s.found = record, true
	return nil
}

func TestReplayReturnsTheOriginalResponse(t *testing.T) {
	type response struct {
		ID string `json:"id"`
	}
	store := &replayStore{}
	createCalls := 0
	create := func() (int, response, error) {
		createCalls++
		return 201, response{ID: "task-1"}, nil
	}
	if _, _, err := ReplayOr(t.Context(), store, "key-1", []byte(`{"code":"S-1"}`), create); err != nil {
		t.Fatal(err)
	}
	code, replayed, err := ReplayOr(t.Context(), store, "key-1", []byte(`{"code":"S-1"}`), create)
	if err != nil {
		t.Fatal(err)
	}
	if createCalls != 1 || code != 201 || replayed.ID != "task-1" {
		t.Fatalf("calls=%d code=%d response=%+v", createCalls, code, replayed)
	}
}
