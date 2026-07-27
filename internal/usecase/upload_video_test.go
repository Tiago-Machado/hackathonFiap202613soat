package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"

	"video-processor/internal/domain"
)

func validQuota() *domain.Quota {
	return &domain.Quota{UserID: "u1", MaxUploadsPerDay: 10, MaxStorageBytes: 1 << 30}
}

func TestUploadVideoHappyPath(t *testing.T) {
	videos := newFakeVideoRepo()
	quotas := &fakeQuotaRepo{quota: validQuota()}
	store := newFakeObjectStore()
	publisher := &fakePublisher{}
	uc := NewUploadVideo(videos, quotas, store, publisher, fixedClock)

	video, err := uc.Execute(context.Background(), UploadVideoInput{
		UserID:      "u1",
		Filename:    "ferias.mp4",
		ContentType: "video/mp4",
		Size:        1234,
		Content:     strings.NewReader("conteudo-do-video"),
	})
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if video.Status != domain.StatusPending {
		t.Errorf("status = %q, esperado PENDING", video.Status)
	}
	if len(videos.created) != 1 {
		t.Errorf("esperava 1 vídeo persistido, veio %d", len(videos.created))
	}
	if len(store.puts) != 1 {
		t.Errorf("esperava 1 objeto no storage, veio %d", len(store.puts))
	}
	if len(publisher.published) != 1 {
		t.Fatalf("esperava 1 evento publicado, veio %d", len(publisher.published))
	}
	if publisher.published[0].VideoID != video.ID {
		t.Error("evento publicado com VideoID divergente")
	}
	if publisher.published[0].StorageKey != video.StorageKey {
		t.Error("evento publicado com StorageKey divergente")
	}
}

func TestUploadVideoUnsupportedFormat(t *testing.T) {
	videos := newFakeVideoRepo()
	store := newFakeObjectStore()
	publisher := &fakePublisher{}
	uc := NewUploadVideo(videos, &fakeQuotaRepo{quota: validQuota()}, store, publisher, fixedClock)

	_, err := uc.Execute(context.Background(), UploadVideoInput{
		UserID:   "u1",
		Filename: "documento.pdf",
		Content:  strings.NewReader("x"),
	})
	if !errors.Is(err, ErrUnsupportedFormat) {
		t.Fatalf("esperava ErrUnsupportedFormat, veio: %v", err)
	}
	if len(store.puts) != 0 || len(publisher.published) != 0 {
		t.Error("nada deveria ser armazenado ou publicado em formato inválido")
	}
}

func TestUploadVideoQuotaExceeded(t *testing.T) {
	videos := newFakeVideoRepo()
	videos.countToday = 10
	quotas := &fakeQuotaRepo{quota: validQuota()}
	store := newFakeObjectStore()
	publisher := &fakePublisher{}
	uc := NewUploadVideo(videos, quotas, store, publisher, fixedClock)

	_, err := uc.Execute(context.Background(), UploadVideoInput{
		UserID:   "u1",
		Filename: "ferias.mp4",
		Content:  strings.NewReader("x"),
	})
	if !errors.Is(err, ErrQuotaExceeded) {
		t.Fatalf("esperava ErrQuotaExceeded, veio: %v", err)
	}
	if len(store.puts) != 0 {
		t.Error("não deveria armazenar quando a cota está estourada")
	}
}

func TestUploadVideoStoreFailureStopsFlow(t *testing.T) {
	videos := newFakeVideoRepo()
	store := newFakeObjectStore()
	store.putErr = errBoom
	publisher := &fakePublisher{}
	uc := NewUploadVideo(videos, &fakeQuotaRepo{quota: validQuota()}, store, publisher, fixedClock)

	_, err := uc.Execute(context.Background(), UploadVideoInput{
		UserID:   "u1",
		Filename: "ferias.mp4",
		Content:  strings.NewReader("x"),
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("esperava propagar erro do storage, veio: %v", err)
	}
	if len(videos.created) != 0 || len(publisher.published) != 0 {
		t.Error("não deveria persistir nem publicar se o storage falhou")
	}
}

func TestUploadVideoPublishFailurePropagates(t *testing.T) {
	videos := newFakeVideoRepo()
	store := newFakeObjectStore()
	publisher := &fakePublisher{err: errBoom}
	uc := NewUploadVideo(videos, &fakeQuotaRepo{quota: validQuota()}, store, publisher, fixedClock)

	_, err := uc.Execute(context.Background(), UploadVideoInput{
		UserID:   "u1",
		Filename: "ferias.mp4",
		Content:  strings.NewReader("x"),
	})
	if !errors.Is(err, errBoom) {
		t.Fatalf("esperava propagar erro do publisher, veio: %v", err)
	}
}
