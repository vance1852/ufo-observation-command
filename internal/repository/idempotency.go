package repository

import (
	"context"
	"errors"
	"fmt"

	"ufo-observation-command/internal/idempotency"
	"github.com/jackc/pgx/v5"
)

func (p *Postgres) GetIdempotency(ctx context.Context, key string) (idempotency.Record, bool, error) {
	var record idempotency.Record
	err := p.pool.QueryRow(ctx, `SELECT key,request_hash,response_code,response_body FROM idempotency_keys WHERE key=$1`, key).Scan(&record.Key, &record.RequestHash, &record.ResponseCode, &record.ResponseBody)
	if errors.Is(err, pgx.ErrNoRows) {
		return idempotency.Record{}, false, nil
	}
	if err != nil {
		return idempotency.Record{}, false, fmt.Errorf("get idempotency record: %w", err)
	}
	return record, true, nil
}

func (p *Postgres) PutIdempotency(ctx context.Context, record idempotency.Record) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO idempotency_keys(key,request_hash,response_code,response_body) VALUES ($1,$2,$3,$4) ON CONFLICT (key) DO NOTHING`, record.Key, record.RequestHash, record.ResponseCode, record.ResponseBody)
	return wrapWrite(err)
}
