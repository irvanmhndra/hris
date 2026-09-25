package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	DatabaseURL string
	Address     string
	// AllowedOrigins enables CORS for a cross-origin frontend. Empty (default)
	// means same-origin only: the SPA reaches /api through a reverse proxy.
	AllowedOrigins []string
	DB             DBConfig
}

type DBConfig struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

func Load() Config {
	return Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		Address:        env("HTTP_ADDR", "127.0.0.1:8088"),
		AllowedOrigins: list(os.Getenv("CORS_ALLOWED_ORIGINS")),
		DB: DBConfig{
			MaxOpenConns:    envInt("DB_MAX_OPEN_CONNS", 20),
			MaxIdleConns:    envInt("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: envDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
		},
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil && n > 0 {
		return n
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil && d > 0 {
		return d
	}
	return fallback
}

func list(raw string) []string {
	var out []string
	for v := range strings.SplitSeq(raw, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}
