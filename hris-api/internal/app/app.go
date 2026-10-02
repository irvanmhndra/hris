// Package app wires configuration, the database, and the HTTP server.
package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/irvanmhndra/hris-api/config"
	"github.com/irvanmhndra/hris-api/internal/router"
	"github.com/irvanmhndra/hris-api/pkg/apperror"
	"github.com/irvanmhndra/hris-api/pkg/httputil"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v5"
	em "github.com/labstack/echo/v5/middleware"
	_ "github.com/lib/pq"
)

const maxBodyBytes = 1 << 20 // 1 MB

// New connects to PostgreSQL and builds the server. The caller owns the DB.
func New(cfg config.Config) (*echo.Echo, *sqlx.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, nil, errors.New("DATABASE_URL wajib diisi")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := sqlx.ConnectContext(ctx, "postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(cfg.DB.MaxOpenConns)
	db.SetMaxIdleConns(cfg.DB.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.DB.ConnMaxLifetime)
	return NewServer(db, cfg), db, nil
}

// NewServer builds the HTTP server on an existing connection (used by tests).
func NewServer(db *sqlx.DB, cfg config.Config) *echo.Echo {
	e := echo.New()
	e.HTTPErrorHandler = errorHandler

	e.Use(em.RequestID())
	e.Use(requestLogger())
	e.Use(em.Recover())
	if len(cfg.AllowedOrigins) > 0 {
		e.Use(em.CORSWithConfig(em.CORSConfig{
			AllowOrigins: cfg.AllowedOrigins,
			AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete, http.MethodOptions},
			AllowHeaders: []string{"Accept", "Authorization", "Content-Type", echo.HeaderXRequestID},
		}))
	}
	// Uploads carry their own 10 MB limit; every other request is capped at 1 MB.
	e.Use(em.BodyLimitWithConfig(em.BodyLimitConfig{
		LimitBytes: maxBodyBytes,
		Skipper:    func(c *echo.Context) bool { return c.Request().URL.Path == "/api/v1/files" },
	}))

	e.GET("/health", health(db))

	svcs := newServices(newRepositories(db), cfg)
	router.Setup(e, newHandlers(svcs), svcs.Auth)
	return e
}

// health reports readiness, including database reachability.
func health(db *sqlx.DB) echo.HandlerFunc {
	return func(c *echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			return httputil.Error(c, apperror.New(http.StatusServiceUnavailable, "UNAVAILABLE", "Database tidak dapat dihubungi"))
		}
		return httputil.Success(c, map[string]string{"status": "ok", "database": "connected"})
	}
}

func requestLogger() echo.MiddlewareFunc {
	return em.RequestLoggerWithConfig(em.RequestLoggerConfig{
		LogLatency:   true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogRequestID: true,
		LogStatus:    true,
		HandleError:  false,
		LogValuesFunc: func(c *echo.Context, v em.RequestLoggerValues) error {
			attrs := []any{
				"method", v.Method, "uri", v.URI, "status", v.Status,
				"latency", v.Latency, "remote_ip", v.RemoteIP, "request_id", v.RequestID,
			}
			if v.Error != nil {
				slog.Error("request error", append(attrs, "error", v.Error)...)
				return nil
			}
			slog.Info("request", attrs...)
			return nil
		},
	})
}

// errorHandler renders errors raised by Echo itself (unknown route, wrong
// method, oversized body, rate limit, recovered panic) in the API envelope.
func errorHandler(c *echo.Context, err error) {
	if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
		return
	}
	var sc echo.HTTPStatusCoder
	if errors.As(err, &sc) {
		_ = httputil.Error(c, statusError(sc.StatusCode()))
		return
	}
	_ = httputil.Error(c, err)
}

func statusError(status int) *apperror.Error {
	switch status {
	case http.StatusBadRequest:
		return apperror.New(status, "BAD_REQUEST", "Permintaan tidak valid")
	case http.StatusUnauthorized:
		return apperror.Unauthorized("Silakan masuk terlebih dahulu")
	case http.StatusForbidden:
		return apperror.Forbidden("Akses tidak diizinkan")
	case http.StatusNotFound:
		return apperror.NotFound("Endpoint tidak ditemukan")
	case http.StatusMethodNotAllowed:
		return apperror.New(status, "METHOD_NOT_ALLOWED", "Metode tidak diizinkan")
	case http.StatusRequestEntityTooLarge:
		return apperror.New(status, "PAYLOAD_TOO_LARGE", "Permintaan terlalu besar (maksimal 1 MB)")
	case http.StatusTooManyRequests:
		return apperror.New(status, "TOO_MANY_REQUESTS", "Terlalu banyak percobaan; coba lagi sebentar")
	}
	text := http.StatusText(status)
	return apperror.New(status, strings.ToUpper(strings.ReplaceAll(text, " ", "_")), text)
}
