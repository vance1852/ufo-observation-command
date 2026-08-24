package arrayops

import (
	"errors"
	"fmt"
)

type SignatureError struct {
	Digest string
	Cause  error
}

func (e *SignatureError) Error() string {
	return fmt.Sprintf("signature verification failed for %s: %v", e.Digest, e.Cause)
}

func (e *SignatureError) Unwrap() error { return e.Cause }

func ClassifyError(err error) (int, string) {
	var signature *SignatureError
	switch {
	case errors.As(err, &signature):
		return 422, "signature_invalid"
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

func WrapOperation(operation string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s failed: %w", operation, err)
}
