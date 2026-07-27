package usecase

import (
	"context"
	"time"

	"video-processor/internal/domain"
)

type FrameExtractor interface {
	ExtractFrames(ctx context.Context, source string) (zipKey string, zipSize int64, frameCount int, err error)
}

type ProcessVideo struct {
	videos    VideoRepository
	extractor FrameExtractor
	notifier  Notifier
	users     UserRepository
	retention time.Duration
	now       Clock
}

type UserRepository interface {
	FindByID(ctx context.Context, id string) (*domain.User, error)
}

func NewProcessVideo(videos VideoRepository, extractor FrameExtractor, notifier Notifier, users UserRepository, retention time.Duration, now Clock) *ProcessVideo {
	return &ProcessVideo{videos: videos, extractor: extractor, notifier: notifier, users: users, retention: retention, now: now}
}

func (uc *ProcessVideo) Execute(ctx context.Context, videoID string) error {
	video, err := uc.videos.FindByID(ctx, videoID)
	if err != nil {
		return err
	}

	if err := video.MarkProcessing(uc.now()); err != nil {
		return err
	}
	if err := uc.videos.Update(ctx, video); err != nil {
		return err
	}

	zipKey, zipSize, frameCount, extractErr := uc.extractor.ExtractFrames(ctx, video.StorageKey)
	if extractErr != nil {
		return uc.fail(ctx, video, extractErr)
	}

	expiresAt := uc.now().Add(uc.retention)
	video.ExpiresAt = &expiresAt
	if err := video.MarkDone(zipKey, zipSize, frameCount, uc.now()); err != nil {
		return err
	}
	return uc.videos.Update(ctx, video)
}

func (uc *ProcessVideo) fail(ctx context.Context, video *domain.Video, cause error) error {
	if err := video.MarkError(cause.Error(), uc.now()); err != nil {
		return err
	}
	if err := uc.videos.Update(ctx, video); err != nil {
		return err
	}
	if user, err := uc.users.FindByID(ctx, video.UserID); err == nil {
		_ = uc.notifier.NotifyFailure(ctx, user.Email, video.OriginalFilename, cause.Error())
	}
	return cause
}
