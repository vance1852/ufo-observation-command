package service

import (
	"context"
	"fmt"

	auditpkg "ufo-observation-command/internal/audit"
	"ufo-observation-command/internal/repository"
)

func ValidateAuditEvent(event auditpkg.Event) error {
	if err := event.Validate(); err != nil {
		return fmt.Errorf("audit event: %w", err)
	}
	_, err := event.JSON()
	return err
}

func (s *Service) WriteAuditEvent(ctx context.Context, event auditpkg.Event) error {
	detail, err := event.JSON()
	if err != nil {
		return err
	}
	return s.repo.WriteAudit(ctx, auditInput(event, detail))
}

func auditInput(event auditpkg.Event, detail []byte) repository.AuditInput {
	return repository.AuditInput{RequestID: event.RequestID, ArrayOperatorID: event.ArrayOperatorID, ObjectType: event.ObjectType, ObjectID: event.ObjectID, Action: event.Action, Outcome: event.Outcome, Detail: detail}
}
