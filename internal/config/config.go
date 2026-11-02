package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Port             string
	DatabaseURL      string
	JWTSecret        string
	ReminderEnabled  bool
	ReminderInterval time.Duration
	AllowedOrigins   []string
}

func Load() (*Config, error) {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}
	if len(jwtSecret) < 32 {
		return nil, errors.New("JWT_SECRET must be at least 32 characters long")
	}

	reminderEnabled := true
	if raw := os.Getenv("REMINDER_ENABLED"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return nil, errors.New("REMINDER_ENABLED must be true or false")
		}
		reminderEnabled = parsed
	}

	reminderInterval := 5 * time.Minute
	if raw := os.Getenv("REMINDER_INTERVAL"); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return nil, errors.New("REMINDER_INTERVAL must be a valid duration, e.g. 5m")
		}
		if parsed <= 0 {
			return nil, errors.New("REMINDER_INTERVAL must be greater than zero")
		}
		reminderInterval = parsed
	}

	return &Config{
		Port:             port,
		DatabaseURL:      dbURL,
		JWTSecret:        jwtSecret,
		ReminderEnabled:  reminderEnabled,
		ReminderInterval: reminderInterval,
		AllowedOrigins:   parseOrigins(os.Getenv("ALLOWED_ORIGINS")),
	}, nil
}

func parseOrigins(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	origins := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			origins = append(origins, trimmed)
		}
	}
	return origins
}
