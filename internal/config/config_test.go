package config

import (
	"os"
	"testing"
)

func TestLoadUsesEnvironmentAndDefaults(t *testing.T) {
	for _, key := range []string{"APP_ADDR", "DATABASE_URL", "DATABASE_MAX_CONNS", "DATABASE_MIN_CONNS", "WORKER_INTERVAL", "SHUTDOWN_TIMEOUT"} {
		t.Setenv(key, "")
	}
	c, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr != ":8080" || c.DatabaseMaxConns != 12 || c.DatabaseMinConns != 2 {
		t.Fatalf("defaults = %+v", c)
	}
}

func TestLoadRejectsInvalidPoolBounds(t *testing.T) {
	t.Setenv("DATABASE_MAX_CONNS", "1")
	t.Setenv("DATABASE_MIN_CONNS", "2")
	if _, err := Load(); err == nil {
		t.Fatal("invalid pool bounds accepted")
	}
}

func TestConfigValidateRequiresHostPort(t *testing.T) {
	c := Config{Addr: "invalid", DatabaseMaxConns: 2, DatabaseMinConns: 1}
	if err := c.Validate(); err == nil {
		t.Fatal("invalid arrayops_lane accepted")
	}
	_ = os.Getenv("PATH")
}
