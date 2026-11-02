package config

import (
	"testing"
	"time"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("DATABASE_URL", "postgres://localhost/db")
	t.Setenv("JWT_SECRET", "12345678901234567890123456789012")
}

func TestLoadReminderDefaults(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.ReminderEnabled {
		t.Error("expected reminder enabled by default")
	}
	if cfg.ReminderInterval != 5*time.Minute {
		t.Errorf("expected default interval 5m, got %s", cfg.ReminderInterval)
	}
}

func TestLoadReminderCustom(t *testing.T) {
	setValidEnv(t)
	t.Setenv("REMINDER_ENABLED", "false")
	t.Setenv("REMINDER_INTERVAL", "30s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ReminderEnabled {
		t.Error("expected reminder disabled")
	}
	if cfg.ReminderInterval != 30*time.Second {
		t.Errorf("expected 30s, got %s", cfg.ReminderInterval)
	}
}

func TestLoadReminderInvalid(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
	}{
		{"bad interval", map[string]string{"REMINDER_INTERVAL": "lima menit"}},
		{"negative interval", map[string]string{"REMINDER_INTERVAL": "-1m"}},
		{"bad enabled", map[string]string{"REMINDER_ENABLED": "iya"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setValidEnv(t)
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			if _, err := Load(); err == nil {
				t.Error("expected error")
			}
		})
	}
}
