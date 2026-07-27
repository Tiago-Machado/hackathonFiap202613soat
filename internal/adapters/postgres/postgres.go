package postgres

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

const (
	maxOpenConns    = 20
	maxIdleConns    = 10
	connMaxLifetime = 30 * time.Minute
)

func Open(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(connMaxLifetime)
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
