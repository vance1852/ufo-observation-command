package console

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) TreatmentPage(ctx context.Context, current, size int, keyword string) (Page[Treatment], error) {
	current, size, offset := pageBounds(current, size)
	keyword = "%" + strings.TrimSpace(keyword) + "%"
	page := Page[Treatment]{Records: make([]Treatment, 0), Current: current, Size: size}
	rows, err := s.pool.Query(ctx, `SELECT id,name,description,risk_budget,duration_minutes,status FROM console_arrayops_profiles
		WHERE deleted_at IS NULL AND name ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, keyword, size, offset)
	if err != nil {
		return page, fmt.Errorf("查询服务列表: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Treatment
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.RiskBudget, &item.DurationMinutes, &item.Status); err != nil {
			return page, fmt.Errorf("读取服务信息: %w", err)
		}
		page.Records = append(page.Records, item)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM console_arrayops_profiles WHERE deleted_at IS NULL AND name ILIKE $1`, keyword).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计服务数量: %w", err)
	}
	return page, rows.Err()
}

func (s *Store) TreatmentList(ctx context.Context) ([]Treatment, error) {
	page, err := s.TreatmentPage(ctx, 1, 100, "")
	return page.Records, err
}

func (s *Store) CreateTreatment(ctx context.Context, item Treatment) (Treatment, error) {
	item.ID = uuid.NewString()
	_, err := s.pool.Exec(ctx, `INSERT INTO console_arrayops_profiles(id,name,description,risk_budget,duration_minutes,status) VALUES($1,$2,$3,$4,$5,$6)`, item.ID, strings.TrimSpace(item.Name), item.Description, item.RiskBudget, item.DurationMinutes, item.Status)
	return item, wrap("新增服务", err)
}

func (s *Store) UpdateTreatment(ctx context.Context, item Treatment) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_arrayops_profiles SET name=$1,description=$2,risk_budget=$3,duration_minutes=$4,status=$5,updated_at=now() WHERE id=$6 AND deleted_at IS NULL`, strings.TrimSpace(item.Name), item.Description, item.RiskBudget, item.DurationMinutes, item.Status, item.ID)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("更新服务: 记录不存在")
	}
	return wrap("更新服务", err)
}

func (s *Store) DeleteTreatment(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_arrayops_profiles SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("删除服务: 记录不存在")
	}
	return wrap("删除服务", err)
}
