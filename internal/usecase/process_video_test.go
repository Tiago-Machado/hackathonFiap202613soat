package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"video-processor/internal/domain"
)

const retentionWindow = 7 * 24 * time.Hour

func seedPendingVideo(repo *fakeVideoRepo) *domain.Video {
	video := &domain.Video{
		ID:               "v1",
		UserID:           "u1",
		OriginalFilename: "ferias.mp4",
		StorageKey:       "inputs/u1/abc.mp4",
		Status:           domain.StatusPending,
		CreatedAt:        fixedTime,
	}
	repo.items[video.ID] = video
	return video
}

func TestProcessVideoHappyPath(t *testing.T) {
	videos := newFakeVideoRepo()
	seedPendingVideo(videos)
	extractor := &fakeExtractor{zipKey: "outputs/abc.zip", zipSize: 4096, frameCount: 5}
	notifier := &fakeNotifier{}
	users := newFakeUserRepo()
	uc := NewProcessVideo(videos, extractor, notifier, users, retentionWindow, fixedClock)

	if err := uc.Execute(context.Background(), "v1"); err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}

	final := videos.items["v1"]
	if final.Status != domain.StatusDone {
		t.Errorf("status final = %q, esperado DONE", final.Status)
	}
	if final.OutputKey != "outputs/abc.zip" {
		t.Errorf("OutputKey = %q", final.OutputKey)
	}
	if final.FrameCount != 5 {
		t.Errorf("FrameCount = %d, esperado 5", final.FrameCount)
	}
	if final.ExpiresAt == nil || !final.ExpiresAt.Equal(fixedTime.Add(retentionWindow)) {
		t.Error("ExpiresAt deveria ser CreatedAt + janela de retenção")
	}
	if extractor.calledWith != "inputs/u1/abc.mp4" {
		t.Errorf("extractor chamado com %q, esperado a StorageKey", extractor.calledWith)
	}
	if len(notifier.calls) != 0 {
		t.Error("não deveria notificar em caso de sucesso")
	}
}

func TestProcessVideoExtractionFailureMarksErrorAndNotifies(t *testing.T) {
	videos := newFakeVideoRepo()
	seedPendingVideo(videos)
	extractor := &fakeExtractor{err: errBoom}
	notifier := &fakeNotifier{}
	users := newFakeUserRepo()
	users.byID["u1"] = &domain.User{ID: "u1", Email: "dono@exemplo.com", IsActive: true}
	uc := NewProcessVideo(videos, extractor, notifier, users, retentionWindow, fixedClock)

	err := uc.Execute(context.Background(), "v1")
	if !errors.Is(err, errBoom) {
		t.Fatalf("esperava propagar a causa da falha, veio: %v", err)
	}

	final := videos.items["v1"]
	if final.Status != domain.StatusError {
		t.Errorf("status final = %q, esperado ERROR", final.Status)
	}
	if final.ErrorMessage == "" {
		t.Error("ErrorMessage deveria estar preenchida")
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("esperava 1 notificação, veio %d", len(notifier.calls))
	}
	if notifier.calls[0].to != "dono@exemplo.com" {
		t.Errorf("notificação enviada para %q", notifier.calls[0].to)
	}
}

func TestProcessVideoNotFound(t *testing.T) {
	videos := newFakeVideoRepo()
	uc := NewProcessVideo(videos, &fakeExtractor{}, &fakeNotifier{}, newFakeUserRepo(), retentionWindow, fixedClock)

	err := uc.Execute(context.Background(), "inexistente")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("esperava ErrNotFound, veio: %v", err)
	}
}
