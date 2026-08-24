package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Addr             string
	DatabaseURL      string
	DatabaseMaxConns int32
	DatabaseMinConns int32
	WorkerInterval   time.Duration
	ShutdownTimeout  time.Duration
}

func Load() (Config, error) {
	c := Config{
		Addr:             env("APP_ADDR", ":8080"),
		DatabaseURL:      env("DATABASE_URL", "postgres://acoustic:acoustic@localhost:5432/ufo_observation_command?sslmode=disable"),
		DatabaseMaxConns: int32(envInt("DATABASE_MAX_CONNS", 12)),
		DatabaseMinConns: int32(envInt("DATABASE_MIN_CONNS", 2)),
		WorkerInterval:   envDuration("WORKER_INTERVAL", 5*time.Second),
		ShutdownTimeout:  envDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
	}
	if c.DatabaseMaxConns < 1 || c.DatabaseMinConns < 0 || c.DatabaseMinConns > c.DatabaseMaxConns {
		return Config{}, fmt.Errorf("invalid database pool limits")
	}
	if c.WorkerInterval <= 0 || c.ShutdownTimeout <= 0 {
		return Config{}, fmt.Errorf("durations must be positive")
	}
	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("DATABASE_URL is required")
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return n
}

func envDuration(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}
