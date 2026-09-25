// Command migrate applies the embedded SQL migrations.
//
//	migrate up          apply all pending migrations (default)
//	migrate down N      roll back N migrations (000003+ are forward-only and refuse)
//	migrate version     print the current version
//	migrate force V     mark version V as applied without running it — for
//	                    databases created before this runner existed
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/irvanmhndra/hris-api/migrations"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL wajib diisi")
	}
	ctx := context.Background()
	db, err := sqlx.ConnectContext(ctx, "postgres", dsn)
	if err != nil {
		return err
	}
	defer func() { _ = db.Close() }()

	cmd := "up"
	if len(args) > 0 {
		cmd = args[0]
	}
	switch cmd {
	case "up":
		if err := migrations.Up(ctx, db.DB); err != nil {
			return err
		}
		slog.Info("migrations applied")
		return nil
	case "down":
		n, err := intArg(args, 1)
		if err != nil {
			return err
		}
		return migrations.Down(ctx, db.DB, n)
	}

	m, err := migrations.Open(ctx, db.DB)
	if err != nil {
		return err
	}
	defer func() { _, _ = m.Close() }()

	switch cmd {
	case "version":
		v, dirty, err := m.Version()
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("no migrations applied")
			return nil
		}
		if err != nil {
			return err
		}
		fmt.Printf("version %d (dirty: %t)\n", v, dirty)
		return nil
	case "force":
		v, err := intArg(args, 1)
		if err != nil {
			return err
		}
		return m.Force(v)
	}
	return fmt.Errorf("perintah tidak dikenal %q (up | down N | version | force V)", cmd)
}

func intArg(args []string, i int) (int, error) {
	if len(args) <= i {
		return 0, fmt.Errorf("%s membutuhkan argumen angka", args[0])
	}
	n, err := strconv.Atoi(args[i])
	if err != nil || n < 1 {
		return 0, fmt.Errorf("argumen %q harus angka positif", args[i])
	}
	return n, nil
}
