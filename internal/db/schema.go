package db

import (
	"context"
	"fmt"
)

func RequiredTables(ctx context.Context, pool *Pool) ([]string, error) {
	rows, err := pool.Query(ctx, `SELECT table_name FROM information_schema.tables WHERE table_schema='public' ORDER BY table_name`)
	if err != nil {
		return nil, fmt.Errorf("list tables: %w", err)
	}
	defer rows.Close()
	items := make([]string, 0)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		items = append(items, name)
	}
	return items, rows.Err()
}

func HasTable(tables []string, wanted string) bool {
	for _, table := range tables {
		if table == wanted {
			return true
		}
	}
	return false
}
