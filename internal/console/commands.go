package console

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

func (s *Store) WorkOrderPage(ctx context.Context, current, size int) (Page[WorkOrder], error) {
	current, size, offset := pageBounds(current, size)
	page := Page[WorkOrder]{Records: make([]WorkOrder, 0), Current: current, Size: size}
	rows, err := s.pool.Query(ctx, `SELECT o.id,o.work_order_no,o.acoustic_buoy_id,e.title,o.arrayops_profile_id,svc.name,o.array_operator_id,coalesce(w.name,''),o.scheduled_at,o.status,o.remark,o.version
		FROM console_command_orders o JOIN console_acoustic_buoys e ON e.id=o.acoustic_buoy_id JOIN console_arrayops_profiles svc ON svc.id=o.arrayops_profile_id
		LEFT JOIN console_array_operators w ON w.id=o.array_operator_id ORDER BY o.created_at DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return page, fmt.Errorf("查询修复工单列表: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item WorkOrder
		if err := rows.Scan(&item.ID, &item.WorkOrderNo, &item.AcousticBuoyID, &item.AcousticBuoyTitle, &item.ArrayOpsProfileID, &item.TreatmentName, &item.ArrayOperatorID, &item.ArrayOperatorName, &item.ScheduledAt, &item.Status, &item.Remark, &item.Version); err != nil {
			return page, fmt.Errorf("读取修复工单信息: %w", err)
		}
		page.Records = append(page.Records, item)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM console_command_orders`).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计修复工单数量: %w", err)
	}
	return page, rows.Err()
}

func (s *Store) CreateWorkOrder(ctx context.Context, item WorkOrder) (WorkOrder, error) {
	item.ID = uuid.NewString()
	item.WorkOrderNo = fmt.Sprintf("RESTORE-%s", time.Now().UTC().Format("20060102150405.000000"))
	item.Status = 0
	item.Version = 1
	_, err := s.pool.Exec(ctx, `INSERT INTO console_command_orders(id,work_order_no,acoustic_buoy_id,arrayops_profile_id,array_operator_id,scheduled_at,status,remark,version)
		VALUES($1,$2,$3,$4,$5,$6,0,$7,1)`, item.ID, item.WorkOrderNo, item.AcousticBuoyID, item.ArrayOpsProfileID, item.ArrayOperatorID, item.ScheduledAt, item.Remark)
	return item, wrap("创建修复工单", err)
}

func (s *Store) UpdateWorkOrderStatus(ctx context.Context, id string, next int) error {
	allowed := map[int][]int{1: {0}, 2: {1}, 3: {0}}
	from, ok := allowed[next]
	if !ok {
		return fmt.Errorf("不支持的修复工单状态")
	}
	tag, err := s.pool.Exec(ctx, `UPDATE console_command_orders SET status=$1,version=version+1,updated_at=now() WHERE id=$2 AND status=ANY($3)`, next, id, from)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("修复工单状态已变化，请刷新后重试")
	}
	return wrap("更新修复工单状态", err)
}
