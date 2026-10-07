package config

import (
	"fmt"
	"os"
	"postfeed/internal/storage/psql"
	"strconv"
	"time"
)

const (
	StorageTypePSQL     = "psql"
	StorageTypeInMemory = "inmemory"
)

type Config struct {
	ServerPort  string
	StorageType string
	Storage     psql.Config
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:  getEnv("APP_PORT", "8080"),
		StorageType: getEnv("STORAGE_TYPE", StorageTypeInMemory),
		Storage:     loadStorageConfig(),
	}

	if cfg.Storage.DSN == "" {
		return nil, fmt.Errorf("DATABASE_DSN is required")
	}

	return cfg, nil
}

func loadStorageConfig() psql.Config {
	return psql.Config{
		DSN:             getEnv("DATABASE_URL", "postgres://app:secret@localhost:5433/posts?sslmode=disable"),
		MaxOpenConns:    getEnvInt("DATABASE_MAX_OPEN_CONNS", 25),
		MaxIdleConns:    getEnvInt("DATABASE_MAX_IDLE_CONNS", 25),
		ConnMaxLifetime: getEnvDuration("DATABASE_CONN_MAX_LIFETIME", 5*time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
