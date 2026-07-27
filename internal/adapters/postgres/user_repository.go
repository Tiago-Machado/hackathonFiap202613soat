package postgres

import (
	"context"
	"database/sql"
	"errors"

	"video-processor/internal/domain"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

const insertUser = `
INSERT INTO users (id, email, password_hash, is_active, created_at)
VALUES ($1, $2, $3, $4, $5)`

func (r *UserRepository) Create(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, insertUser,
		u.ID, u.Email, u.PasswordHash, u.IsActive, u.CreatedAt)
	return err
}

const selectUserByID = `
SELECT id, email, password_hash, is_active, created_at FROM users WHERE id = $1`

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanOne(ctx, selectUserByID, id)
}

const selectUserByEmail = `
SELECT id, email, password_hash, is_active, created_at FROM users WHERE email = $1`

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanOne(ctx, selectUserByEmail, email)
}

func (r *UserRepository) scanOne(ctx context.Context, query, arg string) (*domain.User, error) {
	var u domain.User
	err := r.db.QueryRowContext(ctx, query, arg).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.IsActive, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
