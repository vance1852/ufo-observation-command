package config

import (
	"testing"
	"time"
)

func TestRedactDatabaseURLKeepsHostAndHidesCredentials(t *testing.T) {
	got := RedactDatabaseURL("postgres://alice:secret@db.example:5432/task")
	if got != "postgres://***:***@db.example:5432/task" {
		t.Fatalf("got %s", got)
	}
}

func TestPublicSettingsDoNotExposePassword(t *testing.T) {
	c := Config{Addr: ":8080", DatabaseURL: "postgres://alice:secret@db/task", DatabaseMaxConns: 4, DatabaseMinConns: 1, WorkerInterval: time.Second}
	public := c.Public()
	if public["database"] == c.DatabaseURL || public["database"] == nil {
		t.Fatalf("public config leaked URL: %+v", public)
	}
	s := NewSettings(c, "dev", time.Now())
	if s.Public()["version"] != "dev" {
		t.Fatal("version missing")
	}
}
