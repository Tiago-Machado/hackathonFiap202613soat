package usecase

import (
	"context"
	"io"
	"time"

	"video-processor/internal/domain"
)

type VideoRepository interface {
	Create(ctx context.Context, v *domain.Video) error
	Update(ctx context.Context, v *domain.Video) error
	FindByID(ctx context.Context, id string) (*domain.Video, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Video, error)
	CountUploadsToday(ctx context.Context, userID string) (int, error)
}

type ObjectStore interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
	PresignedGet(ctx context.Context, key string, expiry time.Duration) (string, error)
}

type EventPublisher interface {
	Publish(ctx context.Context, event VideoCreated) error
}

type Notifier interface {
	NotifyFailure(ctx context.Context, to, videoName, reason string) error
}

type QuotaRepository interface {
	Get(ctx context.Context, userID string) (*domain.Quota, error)
}

type VideoCreated struct {
	VideoID    string
	UserID     string
	StorageKey string
}
