package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
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
	client     *minio.Client
	retrier    *retry.Retrier
	bucketName string
}

func isRetriableError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, context.Canceled) {
		return false
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return true
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return netErr.Timeout()
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "connection refused") ||
		strings.Contains(errMsg, "connection reset") ||
		strings.Contains(errMsg, "connection timed out") ||
		strings.Contains(errMsg, "TLS handshake timeout") ||
		strings.Contains(errMsg, "i/o timeout") ||
		strings.Contains(errMsg, "EOF") {
		return true
	}

	errResp := minio.ToErrorResponse(err)
	if errResp.Code != "" {
		switch errResp.StatusCode {
		case 408, 429, 500, 502, 503, 504:
			return true
		}

		switch errResp.Code {
		case "InternalError",
			"RequestTimeout",
			"Throttling",
			"ThrottlingException",
			"RequestLimitExceeded",
			"RequestThrottled",
			"SlowDown",
			"SlowDownWrite",
			"SlowDownRead",
			"ExpiredToken",
			"ExpiredTokenException":
			return true
		default:
			return false
		}
	}

	return false
}

// NewS3Storage initializes an S3Storage client with retries and bucket setup.
func NewS3Storage(
	ctx context.Context,
	endpoint, accessKeyID, secretAccessKey, bucketName string,
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

	retrier := retry.New(isRetriableError)

	if err := ensureBucket(ctx, s3client, bucketName); err != nil {
		return nil, err
	}

	return &S3Storage{
		client:     s3client,
		retrier:    retrier,
		bucketName: bucketName,
	}, nil
}

// Put uploads an object to the configured bucket.
// Note: This method does NOT retry on errors because io.Reader may not be replayable
// (e.g., io.Pipe, http.Request.Body). Retrying with a partially consumed reader
// would result in corrupted or empty uploads.
func (s *S3Storage) Put(ctx context.Context, name string, r io.Reader) error {
	_, err := s.client.PutObject(
		ctx,
		s.bucketName,
		name,
		r,
		-1,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return fmt.Errorf("s3 put object: %w", err)
	}

	return nil
}

// Get downloads an object from the specified bucket.
func (s *S3Storage) Get(
	ctx context.Context,
	bucket, name string,
) (io.ReadCloser, error) {
	obj, err := retry.DoWithResult(
		ctx,
		s.retrier,
		func(ctx context.Context) (*minio.Object, error) {
			return s.client.GetObject(ctx, bucket, name, minio.GetObjectOptions{})
		},
	)
	if err != nil {
		return nil, fmt.Errorf("s3 get object: %w", err)
	}

	return obj, nil
}

func (s *S3Storage) Delete(ctx context.Context, bucket, name string) error {
	return s.retrier.Do(ctx, func(ctx context.Context) error {
		err := s.client.RemoveObject(ctx, bucket, name, minio.RemoveObjectOptions{})
		if err != nil {
			return fmt.Errorf("s3 delete object: %w", err)
		}
		return nil
	})
}

func ensureBucket(
	ctx context.Context,
	client *minio.Client,
	bucketName string,
) error {
	exists, err := client.BucketExists(ctx, bucketName)
	if err != nil {
		return fmt.Errorf("check bucket exists: %w", err)
	}
	if exists {
		return nil
	}

	if err := client.MakeBucket(
		ctx,
		bucketName,
		minio.MakeBucketOptions{},
	); err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	return nil
}
