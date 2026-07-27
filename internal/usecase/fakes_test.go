package usecase

import (
	"bytes"
	"context"
	"errors"
	"io"
	"time"

	"video-processor/internal/domain"
)

var (
	errBoom   = errors.New("falha simulada")
	fixedTime = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
)

func fixedClock() time.Time { return fixedTime }

type fakeVideoRepo struct {
	items      map[string]*domain.Video
	createErr  error
	updateErr  error
	findErr    error
	countToday int
	countErr   error
	created    []*domain.Video
	updated    []*domain.Video
}

func newFakeVideoRepo() *fakeVideoRepo {
	return &fakeVideoRepo{items: map[string]*domain.Video{}}
}

func (f *fakeVideoRepo) Create(ctx context.Context, v *domain.Video) error {
	if f.createErr != nil {
		return f.createErr
	}
	clone := *v
	f.items[v.ID] = &clone
	f.created = append(f.created, v)
	return nil
}

func (f *fakeVideoRepo) Update(ctx context.Context, v *domain.Video) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	clone := *v
	f.items[v.ID] = &clone
	f.updated = append(f.updated, &clone)
	return nil
}

func (f *fakeVideoRepo) FindByID(ctx context.Context, id string) (*domain.Video, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}
	v, ok := f.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	clone := *v
	return &clone, nil
}

func (f *fakeVideoRepo) ListByUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Video, error) {
	var out []*domain.Video
	for _, v := range f.items {
		if v.UserID == userID {
			clone := *v
			out = append(out, &clone)
		}
	}
	return out, nil
}

func (f *fakeVideoRepo) CountUploadsToday(ctx context.Context, userID string) (int, error) {
	return f.countToday, f.countErr
}

type fakeQuotaRepo struct {
	quota *domain.Quota
	err   error
}

func (f *fakeQuotaRepo) Get(ctx context.Context, userID string) (*domain.Quota, error) {
	return f.quota, f.err
}

type fakeObjectStore struct {
	putErr     error
	presignURL string
	presignErr error
	getContent []byte
	getErr     error
	deleteErr  error
	puts       map[string][]byte
}

func newFakeObjectStore() *fakeObjectStore {
	return &fakeObjectStore{puts: map[string][]byte{}}
}

func (f *fakeObjectStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	if f.putErr != nil {
		return f.putErr
	}
	data, _ := io.ReadAll(r)
	f.puts[key] = data
	return nil
}

func (f *fakeObjectStore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return io.NopCloser(bytes.NewReader(f.getContent)), nil
}

func (f *fakeObjectStore) Delete(ctx context.Context, key string) error {
	return f.deleteErr
}

func (f *fakeObjectStore) PresignedGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	return f.presignURL, f.presignErr
}

type fakePublisher struct {
	err       error
	published []VideoCreated
}

func (f *fakePublisher) Publish(ctx context.Context, event VideoCreated) error {
	if f.err != nil {
		return f.err
	}
	f.published = append(f.published, event)
	return nil
}

type notification struct {
	to     string
	name   string
	reason string
}

type fakeNotifier struct {
	err   error
	calls []notification
}

func (f *fakeNotifier) NotifyFailure(ctx context.Context, to, videoName, reason string) error {
	f.calls = append(f.calls, notification{to: to, name: videoName, reason: reason})
	return f.err
}

type fakeExtractor struct {
	zipKey     string
	zipSize    int64
	frameCount int
	err        error
	calledWith string
}

func (f *fakeExtractor) ExtractFrames(ctx context.Context, source string) (string, int64, int, error) {
	f.calledWith = source
	return f.zipKey, f.zipSize, f.frameCount, f.err
}

type fakeUserRepo struct {
	byID        map[string]*domain.User
	byEmail     map[string]*domain.User
	createErr   error
	findByIDErr error
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byID: map[string]*domain.User{}, byEmail: map[string]*domain.User{}}
}

func (f *fakeUserRepo) Create(ctx context.Context, u *domain.User) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.byID[u.ID] = u
	f.byEmail[u.Email] = u
	return nil
}

func (f *fakeUserRepo) FindByID(ctx context.Context, id string) (*domain.User, error) {
	if f.findByIDErr != nil {
		return nil, f.findByIDErr
	}
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

func (f *fakeUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := f.byEmail[email]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return u, nil
}

type fakeHasher struct{}

func (fakeHasher) Hash(plain string) (string, error) { return "hashed:" + plain, nil }

func (fakeHasher) Compare(hash, plain string) bool { return hash == "hashed:"+plain }

type fakeTokens struct {
	err error
}

func (f fakeTokens) Issue(userID string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	return "token:" + userID, nil
}
