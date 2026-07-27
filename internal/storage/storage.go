package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store struct {
	internal *minio.Client
	public   *minio.Client
	bucket   string
}

func New(internalEndpoint, publicEndpoint, accessKey, secretKey, bucket string, useSSL bool) (*Store, error) {
	internal, err := minio.New(internalEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}

	publicURL, err := url.Parse(publicEndpoint)
	if err != nil {
		return nil, err
	}
	public, err := minio.New(publicURL.Host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: publicURL.Scheme == "https",
	})
	if err != nil {
		return nil, err
	}

	return &Store{internal: internal, public: public, bucket: bucket}, nil
}

func (s *Store) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.internal.PutObject(ctx, s.bucket, key, r, size,
		minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *Store) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.internal.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
}

func (s *Store) Delete(ctx context.Context, key string) error {
	return s.internal.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *Store) PresignedGet(ctx context.Context, key string, expiry time.Duration) (string, error) {
	presigned, err := s.public.PresignedGetObject(ctx, s.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return presigned.String(), nil
}
