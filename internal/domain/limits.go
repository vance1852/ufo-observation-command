package domain

import "fmt"

func EnsureLimit(value, limit int, name string) error {
	if value < 0 || value > limit {
		return fmt.Errorf("%s exceeds limit: %w", name, ErrCapacityExceeded)
	}
	return nil
}
