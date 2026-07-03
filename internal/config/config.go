package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort    string
	DatabaseURL   string
	MigrationsDir string
	SMTPHost      string
	SMTPPort      int
	SMTPFrom      string
	LogLevel      string
}

func Load() (*Config, error) {
	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "1025"))
	if err != nil {
		return nil, fmt.Errorf("invalid SMTP_PORT: %w", err)
	}

	return &Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://nanojira:nanojira@localhost:5432/nanojira?sslmode=disable"),
		MigrationsDir: getEnv("MIGRATIONS_DIR", "migrations"),
		SMTPHost:      getEnv("SMTP_HOST", "localhost"),
		SMTPPort:      smtpPort,
		SMTPFrom:      getEnv("SMTP_FROM", "nanojira@localhost"),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
