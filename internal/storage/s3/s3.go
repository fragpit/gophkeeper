package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/fragpit/gophkeeper/internal/service/items"
	"github.com/fragpit/gophkeeper/pkg/utils/retry"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var _ items.ObjectStorer = (*S3Storage)(nil)

// ErrS3ServerNotAvailable is returned when the S3 server cannot be reached.
var ErrS3ServerNotAvailable error = errors.New("s3 server not available")

// S3Storage implements ObjectStorer using an S3-compatible backend.
type S3Storage struct {
	client  *minio.Client
	retrier *retry.Retrier
}

const objectBucket = "gophkeeper"

// NewS3Storage initializes an S3Storage client with retries and bucket setup.
func NewS3Storage(
	ctx context.Context,
	endpoint, accessKeyID, secretAccessKey string,
) (*S3Storage, error) {
	s3client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: false,
	})
	if err != nil {
		return nil, fmt.Errorf("init s3 client: %w", err)
	}

	cancelFn, err := s3client.HealthCheck(5 * time.Second)
	if err == nil {
		defer cancelFn()
	}

	if s3client.IsOffline() {
		return nil, ErrS3ServerNotAvailable
	}

	isRetriable := func(err error) bool {
		return true
	}
	retrier := retry.New(isRetriable)

	if err := ensureBucket(ctx, s3client); err != nil {
		return nil, err
	}

	return &S3Storage{
		client:  s3client,
		retrier: retrier,
	}, nil
}

// Put uploads an object to the configured bucket.
func (s *S3Storage) Put(ctx context.Context, name string, r io.Reader) error {
	return s.retrier.Do(ctx, func(ctx context.Context) error {
		_, err := s.client.PutObject(
			ctx,
			objectBucket,
			name,
			r,
			-1,
			minio.PutObjectOptions{},
		)
		if err != nil {
			return fmt.Errorf("s3 put object: %w", err)
		}

		return nil
	})
}

// Get downloads an object from the specified bucket.
func (s *S3Storage) Get(
	ctx context.Context,
	bucket, name string,
) (io.ReadCloser, error) {
	var obj *minio.Object
	err := s.retrier.Do(ctx, func(ctx context.Context) error {
		var err error
		obj, err = s.client.GetObject(ctx, bucket, name, minio.GetObjectOptions{})
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return obj, nil
}

func ensureBucket(ctx context.Context, client *minio.Client) error {
	exists, err := client.BucketExists(ctx, objectBucket)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}
	if exists {
		return nil
	}

	if err := client.MakeBucket(ctx, objectBucket, minio.MakeBucketOptions{}); err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return nil
}
