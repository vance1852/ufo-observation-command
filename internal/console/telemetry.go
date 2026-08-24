package console

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) SignalRecoveryReportPage(ctx context.Context, current, size int, acoustic_buoyID string) (Page[SignalRecoveryReport], error) {
	current, size, offset := pageBounds(current, size)
	page := Page[SignalRecoveryReport]{Records: make([]SignalRecoveryReport, 0), Current: current, Size: size}
	where := ""
	args := []any{size, offset}
	if acoustic_buoyID != "" {
		where = "WHERE h.acoustic_buoy_id=$3"
		args = append(args, acoustic_buoyID)
	}
	rows, err := s.pool.Query(ctx, `SELECT h.id,h.acoustic_buoy_id,e.title,h.packet_loss_percent,h.water_temperature_c,h.signal_noise_db,h.clock_drift_ppm,h.corruption_index,h.remark,h.recorded_at
		FROM console_signal_recovery_reports h JOIN console_acoustic_buoys e ON e.id=h.acoustic_buoy_id `+where+` ORDER BY h.recorded_at DESC LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return page, fmt.Errorf("查询舱室环境记录: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item SignalRecoveryReport
		if err := rows.Scan(&item.ID, &item.AcousticBuoyID, &item.AcousticBuoyTitle, &item.PacketLossPercent, &item.WaterTemperatureC, &item.SignalNoiseDB, &item.ClockDriftPPM, &item.CorruptionIndex, &item.Remark, &item.RecordedAt); err != nil {
			return page, fmt.Errorf("读取舱室环境记录: %w", err)
		}
		page.Records = append(page.Records, item)
	}
	countQuery := `SELECT count(*) FROM console_signal_recovery_reports`
	countArgs := []any{}
	if acoustic_buoyID != "" {
		countQuery += ` WHERE acoustic_buoy_id=$1`
		countArgs = append(countArgs, acoustic_buoyID)
	}
	if err := s.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计舱室环境记录: %w", err)
	}
	return page, rows.Err()
}

func (s *Store) CreateSignalRecoveryReport(ctx context.Context, item SignalRecoveryReport) (SignalRecoveryReport, error) {
	item.ID = uuid.NewString()
	if item.RecordedAt.IsZero() {
		if err := s.pool.QueryRow(ctx, `INSERT INTO console_signal_recovery_reports(id,acoustic_buoy_id,packet_loss_percent,water_temperature_c,signal_noise_db,clock_drift_ppm,corruption_index,remark)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING recorded_at`, item.ID, item.AcousticBuoyID, item.PacketLossPercent, item.WaterTemperatureC, item.SignalNoiseDB, item.ClockDriftPPM, item.CorruptionIndex, item.Remark).Scan(&item.RecordedAt); err != nil {
			return SignalRecoveryReport{}, wrap("新增舱室环境记录", err)
		}
	} else {
		_, err := s.pool.Exec(ctx, `INSERT INTO console_signal_recovery_reports(id,acoustic_buoy_id,packet_loss_percent,water_temperature_c,signal_noise_db,clock_drift_ppm,corruption_index,remark,recorded_at)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, item.ID, item.AcousticBuoyID, item.PacketLossPercent, item.WaterTemperatureC, item.SignalNoiseDB, item.ClockDriftPPM, item.CorruptionIndex, item.Remark, item.RecordedAt)
		if err != nil {
			return SignalRecoveryReport{}, wrap("新增舱室环境记录", err)
		}
	}
	return item, nil
}

func (s *Store) LogPage(ctx context.Context, current, size int) (Page[Log], error) {
	current, size, offset := pageBounds(current, size)
	page := Page[Log]{Records: make([]Log, 0), Current: current, Size: size}
	rows, err := s.pool.Query(ctx, `SELECT id,username,operation,method,ip,created_at FROM console_logs ORDER BY created_at DESC LIMIT $1 OFFSET $2`, size, offset)
	if err != nil {
		return page, fmt.Errorf("查询日志: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item Log
		if err := rows.Scan(&item.ID, &item.Username, &item.Operation, &item.Method, &item.IP, &item.CreateTime); err != nil {
			return page, fmt.Errorf("读取日志: %w", err)
		}
		page.Records = append(page.Records, item)
	}
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM console_logs`).Scan(&page.Total); err != nil {
		return page, fmt.Errorf("统计日志: %w", err)
	}
	return page, rows.Err()
}
