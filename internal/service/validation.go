package service

import (
	"fmt"
	"strings"

	"ufo-observation-command/internal/domain"
)

func validateCode(value, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required: %w", field, domain.ErrConflict)
	}
	if len(value) > 80 {
		return fmt.Errorf("%s is too long: %w", field, domain.ErrConflict)
	}
	return nil
}

func validateIDs(ids []string) error {
	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if err := validateCode(id, "id"); err != nil {
			return err
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("duplicate id %s: %w", id, domain.ErrConflict)
		}
		seen[id] = struct{}{}
	}
	return nil
}
