package console

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

func (s *Store) ArrayOperatorPage(ctx context.Context, current, size int, keyword string) (Page[ArrayOperator], error) {
	current, size, offset := pageBounds(current, size)
	keyword = "%" + strings.TrimSpace(keyword) + "%"
	page := Page[ArrayOperator]{Records: make([]ArrayOperator, 0), Current: current, Size: size}
	rows, err := s.pool.Query(ctx, `SELECT id,name,specialty_level,phone,skills,status,created_at FROM console_array_operators
		WHERE deleted_at IS NULL AND (name ILIKE $1 OR phone ILIKE $1) ORDER BY created_at DESC LIMIT $2 OFFSET $3`, keyword, size, offset)
	if err != nil {
		return page, fmt.Errorf("查询校准操作员列表: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item ArrayOperator
		if err := rows.Scan(&item.ID, &item.Name, &item.SpecialtyLevel, &item.Phone, &item.Skills, &item.Status, &item.CreateTime); err != nil {
			return page, fmt.Errorf("读取校准操作员信息: %w", err)
		}
		page.Records = append(page.Records, item)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM console_array_operators WHERE deleted_at IS NULL AND (name ILIKE $1 OR phone ILIKE $1)`, keyword).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计校准操作员数量: %w", err)
	}
	return page, rows.Err()
}

func (s *Store) ArrayOperatorList(ctx context.Context) ([]ArrayOperator, error) {
	page, err := s.ArrayOperatorPage(ctx, 1, 100, "")
	return page.Records, err
}

func (s *Store) CreateArrayOperator(ctx context.Context, item ArrayOperator) (ArrayOperator, error) {
	item.ID = uuid.NewString()
	err := s.pool.QueryRow(ctx, `INSERT INTO console_array_operators(id,name,specialty_level,phone,skills,status) VALUES($1,$2,$3,$4,$5,$6) RETURNING created_at`, item.ID, strings.TrimSpace(item.Name), item.SpecialtyLevel, item.Phone, item.Skills, item.Status).Scan(&item.CreateTime)
	return item, wrap("新增校准操作员", err)
}

func (s *Store) UpdateArrayOperator(ctx context.Context, item ArrayOperator) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_array_operators SET name=$1,specialty_level=$2,phone=$3,skills=$4,status=$5,updated_at=now() WHERE id=$6 AND deleted_at IS NULL`, strings.TrimSpace(item.Name), item.SpecialtyLevel, item.Phone, item.Skills, item.Status, item.ID)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("更新校准操作员: 记录不存在")
	}
	return wrap("更新校准操作员", err)
}

func (s *Store) DeleteArrayOperator(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_array_operators SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("删除校准操作员: 记录不存在")
	}
	return wrap("删除校准操作员", err)
}
