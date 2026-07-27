package postgres

import (
	"context"
	"database/sql"
	"errors"

	"video-processor/internal/domain"
)

type VideoRepository struct {
	db *sql.DB
}

func NewVideoRepository(db *sql.DB) *VideoRepository {
	return &VideoRepository{db: db}
}

const insertVideo = `
INSERT INTO videos
	(id, user_id, original_filename, storage_key, content_type, size_bytes, status, attempts, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

func (r *VideoRepository) Create(ctx context.Context, v *domain.Video) error {
	_, err := r.db.ExecContext(ctx, insertVideo,
		v.ID, v.UserID, v.OriginalFilename, v.StorageKey,
		nullString(v.ContentType), v.SizeBytes, string(v.Status), v.Attempts, v.CreatedAt)
	return err
}

const updateVideo = `
UPDATE videos SET
	status = $2, attempts = $3, error_message = $4,
	output_key = $5, output_size_bytes = $6, frame_count = $7,
	started_at = $8, finished_at = $9, expires_at = $10
WHERE id = $1`

func (r *VideoRepository) Update(ctx context.Context, v *domain.Video) error {
	_, err := r.db.ExecContext(ctx, updateVideo,
		v.ID, string(v.Status), v.Attempts, nullString(v.ErrorMessage),
		nullString(v.OutputKey), v.OutputSizeBytes, v.FrameCount,
		v.StartedAt, v.FinishedAt, v.ExpiresAt)
	return err
}

const selectVideoColumns = `
	id, user_id, original_filename, storage_key,
	COALESCE(content_type, ''), COALESCE(size_bytes, 0), status, attempts,
	COALESCE(error_message, ''), COALESCE(output_key, ''), COALESCE(output_size_bytes, 0),
	COALESCE(frame_count, 0), created_at, started_at, finished_at, expires_at`

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanVideo(s rowScanner) (*domain.Video, error) {
	var v domain.Video
	var status string
	err := s.Scan(
		&v.ID, &v.UserID, &v.OriginalFilename, &v.StorageKey,
		&v.ContentType, &v.SizeBytes, &status, &v.Attempts,
		&v.ErrorMessage, &v.OutputKey, &v.OutputSizeBytes,
		&v.FrameCount, &v.CreatedAt, &v.StartedAt, &v.FinishedAt, &v.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	v.Status = domain.Status(status)
	return &v, nil
}

func (r *VideoRepository) FindByID(ctx context.Context, id string) (*domain.Video, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+selectVideoColumns+` FROM videos WHERE id = $1`, id)
	v, err := scanVideo(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return v, err
}

const listByUser = `
SELECT ` + selectVideoColumns + `
FROM videos
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

func (r *VideoRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Video, error) {
	rows, err := r.db.QueryContext(ctx, listByUser, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		v, err := scanVideo(rows)
		if err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}

const countUploadsToday = `
SELECT count(*) FROM videos
WHERE user_id = $1 AND created_at >= date_trunc('day', now())`

func (r *VideoRepository) CountUploadsToday(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, countUploadsToday, userID).Scan(&count)
	return count, err
}
