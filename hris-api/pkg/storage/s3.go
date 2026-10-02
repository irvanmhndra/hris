package storage

import (
	"context"
	"io"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3Config targets any S3-compatible service: AWS S3, Cloudflare R2
// (endpoint <account>.r2.cloudflarestorage.com, region "auto"), or MinIO.
type S3Config struct {
	Endpoint        string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	Region          string
	UseSSL          bool
}

// S3 stores objects under their key in one private bucket.
type S3 struct {
	client *minio.Client
	bucket string
}

func NewS3(cfg S3Config) (*S3, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &S3{client: client, bucket: cfg.Bucket}, nil
}

func (s *S3) Put(ctx context.Context, key string, r io.Reader) (int64, error) {
	if !validKey.MatchString(key) {
		return 0, errInvalidKey
	}
	info, err := s.client.PutObject(ctx, s.bucket, key, r, -1, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
		// Unknown size streams in 16 MB parts; uploads are capped at 10 MB.
		PartSize: 16 << 20,
	})
	return info.Size, err
}

func (s *S3) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	if !validKey.MatchString(key) {
		return nil, errInvalidKey
	}
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	// GetObject is lazy; Stat surfaces a missing object before streaming.
	if _, err = obj.Stat(); err != nil {
		_ = obj.Close()
		return nil, err
	}
	return obj, nil
}

func (s *S3) Delete(ctx context.Context, key string) error {
	if !validKey.MatchString(key) {
		return errInvalidKey
	}
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}
