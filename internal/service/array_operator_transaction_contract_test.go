package service

import (
	"context"
	"errors"
	"testing"

	"ufo-observation-command/internal/domain"
	"ufo-observation-command/internal/repository"
)

type array_operatorTransactionRepository struct {
	repository.Repository
	array_operators map[string]domain.ArrayOperator
	auditErr        error
}

func (r *array_operatorTransactionRepository) InTx(ctx context.Context, fn func(repository.Repository) error) error {
	pending := make(map[string]domain.ArrayOperator, len(r.array_operators))
	for id, array_operator := range r.array_operators {
		pending[id] = array_operator
	}
	tx := &array_operatorTransactionRepository{array_operators: pending, auditErr: r.auditErr}
	if err := fn(tx); err != nil {
		return err
	}
	r.array_operators = pending
	return nil
}

func (r *array_operatorTransactionRepository) CreateArrayOperatorOutsideTransaction(_ context.Context, array_operator domain.ArrayOperator) error {
	r.array_operators[array_operator.ID] = array_operator
	return nil
}

func (r *array_operatorTransactionRepository) CreateArrayOperator(_ context.Context, array_operator domain.ArrayOperator) error {
	r.array_operators[array_operator.ID] = array_operator
	return nil
}

func (r *array_operatorTransactionRepository) WriteAudit(context.Context, repository.AuditInput) error {
	return r.auditErr
}

func TestArrayOperatorRegistrationRollsBackWhenAuditFails(t *testing.T) {
	repo := &array_operatorTransactionRepository{array_operators: map[string]domain.ArrayOperator{}, auditErr: errors.New("audit rejected")}
	_, err := New(repo).RegisterArrayOperator(t.Context(), RequestMeta{RequestID: "array_operator-create"}, "Rollback ArrayOperator", domain.RoleSafetySupervisor)
	if err == nil {
		t.Fatal("array_operator registration succeeded despite audit failure")
	}
	if len(repo.array_operators) != 0 {
		t.Fatalf("persisted array_operators=%d", len(repo.array_operators))
	}
}
