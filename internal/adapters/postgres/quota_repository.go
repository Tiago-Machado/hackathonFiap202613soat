package postgres

import (
	"context"
	"database/sql"
	"errors"

	"video-processor/internal/domain"
)

type QuotaRepository struct {
	db *sql.DB
}

func NewQuotaRepository(db *sql.DB) *QuotaRepository {
	return &QuotaRepository{db: db}
}

const selectQuota = `
SELECT user_id, max_uploads_per_day, max_storage_bytes
FROM user_quotas WHERE user_id = $1`

func (r *QuotaRepository) Get(ctx context.Context, userID string) (*domain.Quota, error) {
	var q domain.Quota
	err := r.db.QueryRowContext(ctx, selectQuota, userID).
		Scan(&q.UserID, &q.MaxUploadsPerDay, &q.MaxStorageBytes)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return &q, err
}
