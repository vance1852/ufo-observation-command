package arrayops

import (
	"fmt"
	"time"
)

func AuthorizeSession(session Session, tenantID, requiredRole string, now time.Time) error {
	if session.Token == "" || session.TenantID != tenantID {
		return fmt.Errorf("session tenant does not match: %w", ErrUnauthorized)
	}
	if session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
		return fmt.Errorf("session is no longer active: %w", ErrUnauthorized)
	}
	if session.Role != requiredRole {
		return fmt.Errorf("session role cannot perform operation: %w", ErrUnauthorized)
	}
	return nil
}

func RotateRole(session Session, nextRole string) (Session, error) {
	if nextRole == "" || nextRole == session.Role {
		return Session{}, fmt.Errorf("new role is invalid: %w", ErrConflict)
	}
	session.Role = nextRole
	return session, nil
}
