package config

import (
	"errors"
	"os"
)

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
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
	return &Config{
		Port:        port,
		DatabaseURL: dbURL,
		JWTSecret:   jwtSecret,
	}, nil

}
