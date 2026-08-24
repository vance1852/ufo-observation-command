package service

import (
	"context"
	"testing"
	"time"

	auditpkg "ufo-observation-command/internal/audit"
	"ufo-observation-command/internal/repository"
)

type auditPageRepository struct {
	repository.Repository
}

func (auditPageRepository) QueryAuditPage(context.Context, auditpkg.Filter, time.Time, time.Time, int, int) ([]auditpkg.Event, error) {
	return []auditpkg.Event{{RequestID: "first"}}, nil
}

func (auditPageRepository) QueryAudit(context.Context, auditpkg.Filter, time.Time, time.Time, int, int) ([]auditpkg.Event, int, error) {
	return []auditpkg.Event{{RequestID: "first"}}, 2, nil
}

func TestAuditPagePreservesFilteredTotal(t *testing.T) {
	now := time.Now().UTC()
	page, err := New(auditPageRepository{}).QueryAudit(t.Context(), auditpkg.Filter{}, now.Add(-time.Hour), now.Add(time.Hour), 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Total != 2 {
		t.Fatalf("page=%+v", page)
	}
}
