package usecase

import (
	"context"
	"testing"
	"time"

	"video-processor/internal/domain"
)

const presignTTL = 15 * time.Minute

func TestListVideosBuildsDownloadURLForDone(t *testing.T) {
	videos := newFakeVideoRepo()
	videos.items["v1"] = &domain.Video{ID: "v1", UserID: "u1", Status: domain.StatusDone, OutputKey: "outputs/abc.zip"}
	store := newFakeObjectStore()
	store.presignURL = "https://minio.local/outputs/abc.zip?assinatura"
	uc := NewListVideos(videos, store, presignTTL)

	views, err := uc.Execute(context.Background(), "u1", 1)
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("esperava 1 view, veio %d", len(views))
	}
	if views[0].DownloadURL != store.presignURL {
		t.Errorf("DownloadURL = %q", views[0].DownloadURL)
	}
}

func TestListVideosNoURLWhenPending(t *testing.T) {
	videos := newFakeVideoRepo()
	videos.items["v1"] = &domain.Video{ID: "v1", UserID: "u1", Status: domain.StatusPending}
	store := newFakeObjectStore()
	store.presignURL = "nao-deveria-aparecer"
	uc := NewListVideos(videos, store, presignTTL)

	views, _ := uc.Execute(context.Background(), "u1", 1)
	if len(views) != 1 || views[0].DownloadURL != "" {
		t.Error("vídeo PENDING não deveria ter DownloadURL")
	}
}

func TestListVideosSwallowsPresignError(t *testing.T) {
	videos := newFakeVideoRepo()
	videos.items["v1"] = &domain.Video{ID: "v1", UserID: "u1", Status: domain.StatusDone, OutputKey: "outputs/abc.zip"}
	store := newFakeObjectStore()
	store.presignErr = errBoom
	uc := NewListVideos(videos, store, presignTTL)

	views, err := uc.Execute(context.Background(), "u1", 1)
	if err != nil {
		t.Fatalf("erro de presign não deveria derrubar a listagem: %v", err)
	}
	if views[0].DownloadURL != "" {
		t.Error("DownloadURL deveria ficar vazia quando o presign falha")
	}
}
