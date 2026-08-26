package domain

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrConflict          = errors.New("conflict")
	ErrInvalidTransition = errors.New("invalid state transition")
	ErrCapacityExceeded  = errors.New("capacity exceeded")
	ErrExpired           = errors.New("task expired")
	ErrIdempotency       = errors.New("idempotency key conflict")
)
