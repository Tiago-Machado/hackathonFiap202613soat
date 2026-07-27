package domain

import (
	"errors"
	"testing"
	"time"
)

func newPendingVideo() *Video {
	return &Video{ID: "v1", UserID: "u1", Status: StatusPending}
}

func TestMarkProcessingFromPending(t *testing.T) {
	video := newPendingVideo()
	at := time.Now()

	if err := video.MarkProcessing(at); err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if video.Status != StatusProcessing {
		t.Errorf("status = %q, esperado PROCESSING", video.Status)
	}
	if video.Attempts != 1 {
		t.Errorf("attempts = %d, esperado 1", video.Attempts)
	}
	if video.StartedAt == nil {
		t.Error("StartedAt não deveria ser nil")
	}
}

func TestMarkDoneSetsOutput(t *testing.T) {
	video := newPendingVideo()
	_ = video.MarkProcessing(time.Now())

	err := video.MarkDone("outputs/v1.zip", 2048, 5, time.Now())
	if err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if video.Status != StatusDone {
		t.Errorf("status = %q, esperado DONE", video.Status)
	}
	if video.OutputKey != "outputs/v1.zip" {
		t.Errorf("OutputKey = %q", video.OutputKey)
	}
	if video.OutputSizeBytes != 2048 || video.FrameCount != 5 {
		t.Errorf("output size/frames incorretos: %d / %d", video.OutputSizeBytes, video.FrameCount)
	}
	if video.FinishedAt == nil {
		t.Error("FinishedAt não deveria ser nil")
	}
}

func TestMarkErrorSetsMessage(t *testing.T) {
	video := newPendingVideo()
	_ = video.MarkProcessing(time.Now())

	if err := video.MarkError("ffmpeg falhou", time.Now()); err != nil {
		t.Fatalf("esperava sucesso, veio erro: %v", err)
	}
	if video.Status != StatusError {
		t.Errorf("status = %q, esperado ERROR", video.Status)
	}
	if video.ErrorMessage != "ffmpeg falhou" {
		t.Errorf("ErrorMessage = %q", video.ErrorMessage)
	}
}

func TestInvalidTransitions(t *testing.T) {
	cases := []struct {
		name string
		call func(*Video) error
		from Status
	}{
		{"pending_direto_para_done", func(v *Video) error { return v.MarkDone("k", 1, 1, time.Now()) }, StatusPending},
		{"done_nao_reprocessa", func(v *Video) error { return v.MarkProcessing(time.Now()) }, StatusDone},
		{"error_nao_reprocessa", func(v *Video) error { return v.MarkProcessing(time.Now()) }, StatusError},
		{"processing_novamente", func(v *Video) error { return v.MarkProcessing(time.Now()) }, StatusProcessing},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			video := &Video{Status: tc.from}
			if err := tc.call(video); !errors.Is(err, ErrInvalidTransition) {
				t.Errorf("esperava ErrInvalidTransition, veio: %v", err)
			}
		})
	}
}
