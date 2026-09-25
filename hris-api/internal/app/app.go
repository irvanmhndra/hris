package app

import (
	"context"
	"fmt"
	"github.com/irvanmhndra/hris-api/config"
	"github.com/irvanmhndra/hris-api/internal/repository/postgres"
	"github.com/irvanmhndra/hris-api/internal/router"
	"github.com/irvanmhndra/hris-api/internal/service"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	_ "github.com/lib/pq"
	"time"
)

func New(cfg config.Config) (*echo.Echo, *sqlx.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, fmt.Errorf("DATABASE_URL wajib diisi")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sqlx.ConnectContext(ctx, "postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	return router.New(&service.Service{Repo: &postgres.Repository{DB: db}}), db, nil
}
