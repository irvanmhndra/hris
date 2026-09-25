// Package migrations embeds the SQL migrations and applies them with
// golang-migrate, which records the applied version in schema_migrations so a
// database is never re-migrated or left half-upgraded.
package migrations

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed *.sql
var files embed.FS

// Open returns a migrator for db's current schema (its search_path). It borrows
// a single connection, so closing the migrator leaves the caller's pool open.
func Open(ctx context.Context, db *sql.DB) (*migrate.Migrate, error) {
	src, err := openSource()
	if err != nil {
		return nil, err
	}
	conn, err := db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	driver, err := postgres.WithConnection(ctx, conn, &postgres.Config{})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return migrate.NewWithInstance("iofs", src, "postgres", driver)
}

func openSource() (source.Driver, error) {
	src, err := iofs.New(files, ".")
	if err != nil {
		return nil, fmt.Errorf("load migrations: %w", err)
	}
	return src, nil
}

// ErrUntracked means the schema exists but was created outside the runner
// (the old docker-entrypoint-initdb.d setup), so `up` would re-run 000001.
var ErrUntracked = errors.New("schema sudah ada tetapi belum tercatat di schema_migrations; " +
	"jalankan `migrate force 4` sekali untuk menandai migrasi awal sebagai sudah diterapkan")

// Up applies every pending migration. An already up-to-date schema is not an error.
func Up(ctx context.Context, db *sql.DB) error {
	m, err := Open(ctx, db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	if _, _, err := m.Version(); errors.Is(err, migrate.ErrNilVersion) {
		var legacy bool
		if err := db.QueryRowContext(ctx, `SELECT to_regclass('companies') IS NOT NULL`).Scan(&legacy); err != nil {
			return err
		}
		if legacy {
			return ErrUntracked
		}
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// forwardOnly marks a down migration that must never run (it would destroy
// business data); such files start with this comment and raise an exception.
const forwardOnly = "-- Forward-only"

// Down rolls back n migrations. It checks every step first and refuses before
// touching the database if one is forward-only: golang-migrate would
// otherwise mark the schema dirty on the refused step.
func Down(ctx context.Context, db *sql.DB, n int) error {
	m, err := Open(ctx, db)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()
	v, _, err := m.Version()
	if err != nil {
		return err
	}
	src, err := openSource()
	if err != nil {
		return err
	}
	defer func() { _ = src.Close() }()
	for range n {
		r, _, err := src.ReadDown(v)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(r)
		_ = r.Close()
		if err != nil {
			return err
		}
		if strings.HasPrefix(string(body), forwardOnly) {
			return fmt.Errorf("migrasi %d bersifat forward-only; rollback membutuhkan rencana manual yang menjaga data", v)
		}
		if v, err = src.Prev(v); err != nil {
			break // reached the first migration; Steps reports if n is too large
		}
	}
	return m.Steps(-n)
}
