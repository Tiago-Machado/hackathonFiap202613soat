package domain

import (
	"errors"
	"time"
)

type Status string

const (
	StatusPending    Status = "PENDING"
	StatusProcessing Status = "PROCESSING"
	StatusDone       Status = "DONE"
	StatusError      Status = "ERROR"
)

var ErrInvalidTransition = errors.New("transição de status inválida")

var allowedTransitions = map[Status][]Status{
	StatusPending:    {StatusProcessing},
	StatusProcessing: {StatusDone, StatusError},
	StatusDone:       {},
	StatusError:      {},
}

type Video struct {
	ID               string
	UserID           string
	OriginalFilename string
	StorageKey       string
	ContentType      string
	SizeBytes        int64
	Status           Status
	Attempts         int
	ErrorMessage     string
	OutputKey        string
	OutputSizeBytes  int64
	FrameCount       int
	CreatedAt        time.Time
	StartedAt        *time.Time
	FinishedAt       *time.Time
	ExpiresAt        *time.Time
}

func (v *Video) canTransitionTo(next Status) bool {
	for _, allowed := range allowedTransitions[v.Status] {
		if allowed == next {
			return true
		}
	}
	return false
}

func (v *Video) MarkProcessing(at time.Time) error {
	if !v.canTransitionTo(StatusProcessing) {
		return ErrInvalidTransition
	}
	v.Status = StatusProcessing
	v.Attempts++
	v.StartedAt = &at
	return nil
}

func (v *Video) MarkDone(outputKey string, outputSize int64, frameCount int, at time.Time) error {
	if !v.canTransitionTo(StatusDone) {
		return ErrInvalidTransition
	}
	v.Status = StatusDone
	v.OutputKey = outputKey
	v.OutputSizeBytes = outputSize
	v.FrameCount = frameCount
	v.FinishedAt = &at
	return nil
}

func (v *Video) MarkError(message string, at time.Time) error {
	if !v.canTransitionTo(StatusError) {
		return ErrInvalidTransition
	}
	v.Status = StatusError
	v.ErrorMessage = message
	v.FinishedAt = &at
	return nil
}
