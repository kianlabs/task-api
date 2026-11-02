package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name      string
		dbURL     string
		jwtSecret string
		wantErr   bool
	}{
		{name: "missing db url", dbURL: "", jwtSecret: "12345678901234567890123456789012", wantErr: true},
		{name: "short jwt secret", dbURL: "postgres://localhost/db", jwtSecret: "terlalu-pendek", wantErr: true},
		{name: "valid config", dbURL: "postgres://localhost/db", jwtSecret: "12345678901234567890123456789012", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("DATABASE_URL", tt.dbURL)
			os.Setenv("JWT_SECRET", tt.jwtSecret)

			_, err := Load()
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
