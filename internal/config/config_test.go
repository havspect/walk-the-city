package config

import (
	"os"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Clear any relevant environment variables.
	os.Unsetenv("PORT")
	os.Unsetenv("DB_PATH")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("ENV")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Port != "8080" {
		t.Errorf("expected default Port '8080', got %q", cfg.Port)
	}
	if cfg.DBPath != "data/walkthecity.db" {
		t.Errorf("expected default DBPath 'data/walkthecity.db', got %q", cfg.DBPath)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected default LogLevel 'info', got %q", cfg.LogLevel)
	}
	if cfg.Env != "development" {
		t.Errorf("expected default Env 'development', got %q", cfg.Env)
	}
}

func TestLoadCustomEnv(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_PATH", "/tmp/custom.db")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("ENV", "production")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.Port != "9090" {
		t.Errorf("expected Port '9090', got %q", cfg.Port)
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("expected DBPath '/tmp/custom.db', got %q", cfg.DBPath)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got %q", cfg.LogLevel)
	}
	if cfg.Env != "production" {
		t.Errorf("expected Env 'production', got %q", cfg.Env)
	}
}
