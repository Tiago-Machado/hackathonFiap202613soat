package domain

import "time"

type User struct {
	ID           string
	Email        string
	PasswordHash string
	IsActive     bool
	CreatedAt    time.Time
}

type Quota struct {
	UserID           string
	MaxUploadsPerDay int
	MaxStorageBytes  int64
}
