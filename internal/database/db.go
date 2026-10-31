package database

import (
	"database/sql"
	"fmt"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func Connect(connString string) (*sql.DB, error) {
	if connString == "" {
		return nil, fmt.Errorf("database connection string is empty")
	}

	db, err := sql.Open("pgx", connString)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
