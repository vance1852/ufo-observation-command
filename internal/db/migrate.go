package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func Migrate(ctx context.Context, pool *Pool) error {
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("create migration table: %w", err)
	}
	migrations, err := discoverMigrations()
	if err != nil {
		return err
	}
	for _, migration := range migrations {
		if err := applyMigration(ctx, pool, migration); err != nil {
			return err
		}
	}
	return nil
}

type migrationFile struct {
	version int
	path    string
}

func discoverMigrations() ([]migrationFile, error) {
	var directory string
	for _, candidate := range []string{"migrations", filepath.Join("..", "..", "migrations")} {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			directory = candidate
			break
		}
	}
	if directory == "" {
		return nil, fmt.Errorf("migration directory not found")
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return nil, fmt.Errorf("read migrations: %w", err)
	}
	items := make([]migrationFile, 0, len(entries))
	seen := make(map[int]string)
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		prefix, _, ok := strings.Cut(entry.Name(), "_")
		if !ok {
			return nil, fmt.Errorf("migration %q has no numeric prefix", entry.Name())
		}
		version, err := strconv.Atoi(prefix)
		if err != nil || version < 1 {
			return nil, fmt.Errorf("migration %q has invalid version", entry.Name())
		}
		if previous, exists := seen[version]; exists {
			return nil, fmt.Errorf("migration version %d is duplicated by %q and %q", version, previous, entry.Name())
		}
		seen[version] = entry.Name()
		items = append(items, migrationFile{version: version, path: filepath.Join(directory, entry.Name())})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].version < items[j].version })
	if len(items) == 0 {
		return nil, fmt.Errorf("no migrations found")
	}
	return items, nil
}

func applyMigration(ctx context.Context, pool *Pool, migration migrationFile) error {
	sql, err := os.ReadFile(migration.path)
	if err != nil {
		return fmt.Errorf("read migration %d: %w", migration.version, err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %d: %w", migration.version, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, int64(0x45535001)); err != nil {
		return fmt.Errorf("lock migration %d: %w", migration.version, err)
	}
	var applied bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version=$1)`, migration.version).Scan(&applied); err != nil {
		return fmt.Errorf("read migration %d state: %w", migration.version, err)
	}
	if !applied {
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("apply migration %d: %w", migration.version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES ($1)`, migration.version); err != nil {
			return fmt.Errorf("record migration %d: %w", migration.version, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %d: %w", migration.version, err)
	}
	return nil
}
