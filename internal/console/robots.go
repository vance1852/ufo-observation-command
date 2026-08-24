package console

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (s *Store) AcousticBuoyPage(ctx context.Context, current, size int, keyword string) (Page[AcousticBuoy], error) {
	current, size, offset := pageBounds(current, size)
	keyword = "%" + strings.TrimSpace(keyword) + "%"
	page := Page[AcousticBuoy]{Records: make([]AcousticBuoy, 0), Current: current, Size: size}
	rows, err := s.pool.Query(ctx, `SELECT id,title,buoy_class,commissioned_on,serial_number,shore_station_code,mooring_zone,manufacturer,emergency_contact,integrity_status,status
		FROM console_acoustic_buoys WHERE deleted_at IS NULL AND (title ILIKE $1 OR serial_number ILIKE $1)
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, keyword, size, offset)
	if err != nil {
		return page, fmt.Errorf("查询馆藏芯片批次列表: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item AcousticBuoy
		var date *time.Time
		if err := rows.Scan(&item.ID, &item.Title, &item.BuoyClass, &date, &item.SerialNumber, &item.ShoreStationCode, &item.MooringZone, &item.Manufacturer, &item.EmergencyContact, &item.IntegrityStatus, &item.Status); err != nil {
			return page, fmt.Errorf("读取馆藏芯片批次信息: %w", err)
		}
		item.CommissionedOn = formatDate(date)
		page.Records = append(page.Records, item)
	}
	if err := rows.Err(); err != nil {
		return page, fmt.Errorf("读取馆藏芯片批次列表: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM console_acoustic_buoys WHERE deleted_at IS NULL AND (title ILIKE $1 OR serial_number ILIKE $1)`, keyword).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计馆藏芯片批次数量: %w", err)
	}
	return page, nil
}

func (s *Store) AcousticBuoyList(ctx context.Context) ([]AcousticBuoy, error) {
	page, err := s.AcousticBuoyPage(ctx, 1, 100, "")
	return page.Records, err
}

func (s *Store) AcousticBuoyByID(ctx context.Context, id string) (AcousticBuoy, error) {
	var item AcousticBuoy
	var date *time.Time
	err := s.pool.QueryRow(ctx, `SELECT id,title,buoy_class,commissioned_on,serial_number,shore_station_code,mooring_zone,manufacturer,emergency_contact,integrity_status,status
		FROM console_acoustic_buoys WHERE id=$1 AND deleted_at IS NULL`, id).Scan(
		&item.ID, &item.Title, &item.BuoyClass, &date, &item.SerialNumber, &item.ShoreStationCode, &item.MooringZone, &item.Manufacturer, &item.EmergencyContact, &item.IntegrityStatus, &item.Status,
	)
	item.CommissionedOn = formatDate(date)
	return item, wrap("查询馆藏芯片批次", err)
}

func (s *Store) CreateAcousticBuoy(ctx context.Context, item AcousticBuoy) (AcousticBuoy, error) {
	item.ID = uuid.NewString()
	_, err := s.pool.Exec(ctx, `INSERT INTO console_acoustic_buoys(id,title,buoy_class,commissioned_on,serial_number,shore_station_code,mooring_zone,manufacturer,emergency_contact,integrity_status,status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, item.ID, strings.TrimSpace(item.Title), item.BuoyClass, parseDate(item.CommissionedOn), item.SerialNumber, item.ShoreStationCode, item.MooringZone, item.Manufacturer, item.EmergencyContact, item.IntegrityStatus, item.Status)
	return item, wrap("新增馆藏芯片批次", err)
}

func (s *Store) UpdateAcousticBuoy(ctx context.Context, item AcousticBuoy) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_acoustic_buoys SET title=$1,buoy_class=$2,commissioned_on=$3,serial_number=$4,shore_station_code=$5,mooring_zone=$6,manufacturer=$7,emergency_contact=$8,integrity_status=$9,status=$10,updated_at=now()
		WHERE id=$11 AND deleted_at IS NULL`, strings.TrimSpace(item.Title), item.BuoyClass, parseDate(item.CommissionedOn), item.SerialNumber, item.ShoreStationCode, item.MooringZone, item.Manufacturer, item.EmergencyContact, item.IntegrityStatus, item.Status, item.ID)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("更新馆藏芯片批次: 记录不存在")
	}
	return wrap("更新馆藏芯片批次", err)
}

func (s *Store) DeleteAcousticBuoy(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE console_acoustic_buoys SET deleted_at=now(),updated_at=now() WHERE id=$1 AND deleted_at IS NULL`, id)
	if err == nil && tag.RowsAffected() != 1 {
		return fmt.Errorf("删除馆藏芯片批次: 记录不存在")
	}
	return wrap("删除馆藏芯片批次", err)
}
