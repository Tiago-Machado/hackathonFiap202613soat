package usecase

import (
	"context"
	"time"

	"video-processor/internal/domain"
)

const defaultPageSize = 20

type ListVideos struct {
	videos      VideoRepository
	store       ObjectStore
	downloadTTL time.Duration
}

func NewListVideos(videos VideoRepository, store ObjectStore, downloadTTL time.Duration) *ListVideos {
	return &ListVideos{videos: videos, store: store, downloadTTL: downloadTTL}
}

type VideoView struct {
	Video       *domain.Video
	DownloadURL string
}

func (uc *ListVideos) Execute(ctx context.Context, userID string, page int) ([]VideoView, error) {
	if page < 1 {
		page = 1
	}
	offset := (page - 1) * defaultPageSize

	videos, err := uc.videos.ListByUser(ctx, userID, defaultPageSize, offset)
	if err != nil {
		return nil, err
	}

	views := make([]VideoView, 0, len(videos))
	for _, v := range videos {
		view := VideoView{Video: v}
		if v.Status == domain.StatusDone && v.OutputKey != "" {
			if url, err := uc.store.PresignedGet(ctx, v.OutputKey, uc.downloadTTL); err == nil {
				view.DownloadURL = url
			}
		}
		views = append(views, view)
	}
	return views, nil
}
