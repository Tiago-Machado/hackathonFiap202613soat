package usecase

import (
	"context"
	"errors"
	"io"
	"path"
	"time"

	"video-processor/internal/domain"

	"github.com/google/uuid"
)

var (
	ErrQuotaExceeded     = errors.New("cota diária de uploads excedida")
	ErrUnsupportedFormat = errors.New("formato de vídeo não suportado")
)

var supportedExtensions = map[string]bool{
	".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
	".wmv": true, ".flv": true, ".webm": true,
}

type Clock func() time.Time

type UploadVideo struct {
	videos    VideoRepository
	quotas    QuotaRepository
	store     ObjectStore
	publisher EventPublisher
	now       Clock
}

func NewUploadVideo(videos VideoRepository, quotas QuotaRepository, store ObjectStore, publisher EventPublisher, now Clock) *UploadVideo {
	return &UploadVideo{videos: videos, quotas: quotas, store: store, publisher: publisher, now: now}
}

type UploadVideoInput struct {
	UserID      string
	Filename    string
	ContentType string
	Size        int64
	Content     io.Reader
}

func (uc *UploadVideo) Execute(ctx context.Context, in UploadVideoInput) (*domain.Video, error) {
	if !supportedExtensions[path.Ext(in.Filename)] {
		return nil, ErrUnsupportedFormat
	}

	if err := uc.assertWithinQuota(ctx, in.UserID); err != nil {
		return nil, err
	}

	video := &domain.Video{
		ID:               uuid.NewString(),
		UserID:           in.UserID,
		OriginalFilename: in.Filename,
		StorageKey:       path.Join("inputs", in.UserID, uuid.NewString()+path.Ext(in.Filename)),
		ContentType:      in.ContentType,
		SizeBytes:        in.Size,
		Status:           domain.StatusPending,
		CreatedAt:        uc.now(),
	}

	if err := uc.store.Put(ctx, video.StorageKey, in.Content, in.Size, in.ContentType); err != nil {
		return nil, err
	}

	if err := uc.videos.Create(ctx, video); err != nil {
		return nil, err
	}

	event := VideoCreated{VideoID: video.ID, UserID: video.UserID, StorageKey: video.StorageKey}
	if err := uc.publisher.Publish(ctx, event); err != nil {
		return nil, err
	}

	return video, nil
}

func (uc *UploadVideo) assertWithinQuota(ctx context.Context, userID string) error {
	quota, err := uc.quotas.Get(ctx, userID)
	if err != nil {
		return err
	}
	used, err := uc.videos.CountUploadsToday(ctx, userID)
	if err != nil {
		return err
	}
	if used >= quota.MaxUploadsPerDay {
		return ErrQuotaExceeded
	}
	return nil
}
