package config

import "os"

type Config struct{ DatabaseURL, Address string }

func Load() Config {
	return Config{DatabaseURL: os.Getenv("DATABASE_URL"), Address: env("HTTP_ADDR", "127.0.0.1:8088")}
}
func env(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}
