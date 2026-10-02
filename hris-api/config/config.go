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
	// SignupEnabled allows anyone to register a new company (SIGNUP_ENABLED).
	SignupEnabled bool
	// Portal URLs used in emailed links, e.g. password resets.
	AdminURL    string
	EmployeeURL string
	// Storage is "local" (UploadDir) or "s3" (S3, also Cloudflare R2/MinIO).
	Storage   string
	UploadDir string
	S3        S3Config
	SMTP      SMTPConfig
}

type S3Config struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	UseSSL          bool
}

// SMTPConfig sends email. Without a host, emails are written to the log
// (development only: the log then contains reset links).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
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
		SignupEnabled: os.Getenv("SIGNUP_ENABLED") == "true",
		AdminURL:      strings.TrimRight(env("ADMIN_URL", "http://127.0.0.1:5180"), "/"),
		EmployeeURL:   strings.TrimRight(env("EMPLOYEE_URL", "http://127.0.0.1:5181"), "/"),
		Storage:       env("STORAGE_DRIVER", "local"),
		UploadDir:     env("UPLOAD_DIR", "data/uploads"),
		S3: S3Config{
			Endpoint:        os.Getenv("S3_ENDPOINT"),
			Bucket:          os.Getenv("S3_BUCKET"),
			AccessKeyID:     os.Getenv("S3_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("S3_SECRET_ACCESS_KEY"),
			Region:          env("S3_REGION", "auto"),
			UseSSL:          os.Getenv("S3_USE_SSL") != "false",
		},
		SMTP: SMTPConfig{
			Host:     os.Getenv("SMTP_HOST"),
			Port:     envInt("SMTP_PORT", 587),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     env("MAIL_FROM", "People HRIS <no-reply@localhost>"),
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
