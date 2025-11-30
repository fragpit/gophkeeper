package s3

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/assert"
)

func TestIsRetriableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "context canceled",
			err:      context.Canceled,
			expected: false,
		},
		{
			name:     "context deadline exceeded",
			err:      context.DeadlineExceeded,
			expected: true,
		},
		{
			name:     "wrapped context canceled",
			err:      errors.Join(context.Canceled, errors.New("extra")),
			expected: false,
		},
		{
			name:     "wrapped context deadline exceeded",
			err:      errors.Join(context.DeadlineExceeded, errors.New("extra")),
			expected: true,
		},
		{
			name:     "timeout network error",
			err:      &temporaryNetError{timeout: true},
			expected: true,
		},
		{
			name:     "non-temporary network error",
			err:      &temporaryNetError{},
			expected: false,
		},
		{
			name:     "DNS error",
			err:      &net.DNSError{Err: "lookup failed"},
			expected: true,
		},
		{
			name:     "op error",
			err:      &net.OpError{Op: "dial"},
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "connection reset",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "connection timed out",
			err:      errors.New("connection timed out"),
			expected: true,
		},
		{
			name:     "TLS handshake timeout",
			err:      errors.New("TLS handshake timeout"),
			expected: true,
		},
		{
			name:     "i/o timeout",
			err:      errors.New("i/o timeout"),
			expected: true,
		},
		{
			name:     "EOF error",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
		{
			name: "S3 408 Request Timeout",
			err: minio.ErrorResponse{
				StatusCode: 408,
				Code:       "RequestTimeout",
			},
			expected: true,
		},
		{
			name: "S3 429 Too Many Requests",
			err: minio.ErrorResponse{
				StatusCode: 429,
				Code:       "TooManyRequests",
			},
			expected: true,
		},
		{
			name: "S3 500 Internal Server Error",
			err: minio.ErrorResponse{
				StatusCode: 500,
				Code:       "InternalError",
			},
			expected: true,
		},
		{
			name: "S3 502 Bad Gateway",
			err: minio.ErrorResponse{
				StatusCode: 502,
				Code:       "BadGateway",
			},
			expected: true,
		},
		{
			name: "S3 503 Service Unavailable",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "ServiceUnavailable",
			},
			expected: true,
		},
		{
			name: "S3 504 Gateway Timeout",
			err: minio.ErrorResponse{
				StatusCode: 504,
				Code:       "GatewayTimeout",
			},
			expected: true,
		},
		{
			name: "S3 InternalError code",
			err: minio.ErrorResponse{
				StatusCode: 500,
				Code:       "InternalError",
			},
			expected: true,
		},
		{
			name: "S3 Throttling code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "Throttling",
			},
			expected: true,
		},
		{
			name: "S3 ThrottlingException code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "ThrottlingException",
			},
			expected: true,
		},
		{
			name: "S3 RequestLimitExceeded code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "RequestLimitExceeded",
			},
			expected: true,
		},
		{
			name: "S3 RequestThrottled code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "RequestThrottled",
			},
			expected: true,
		},
		{
			name: "S3 SlowDown code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "SlowDown",
			},
			expected: true,
		},
		{
			name: "S3 SlowDownWrite code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "SlowDownWrite",
			},
			expected: true,
		},
		{
			name: "S3 SlowDownRead code",
			err: minio.ErrorResponse{
				StatusCode: 503,
				Code:       "SlowDownRead",
			},
			expected: true,
		},
		{
			name: "S3 ExpiredToken code",
			err: minio.ErrorResponse{
				StatusCode: 401,
				Code:       "ExpiredToken",
			},
			expected: true,
		},
		{
			name: "S3 ExpiredTokenException code",
			err: minio.ErrorResponse{
				StatusCode: 401,
				Code:       "ExpiredTokenException",
			},
			expected: true,
		},
		{
			name: "S3 404 NoSuchKey",
			err: minio.ErrorResponse{
				StatusCode: 404,
				Code:       "NoSuchKey",
			},
			expected: false,
		},
		{
			name: "S3 404 NoSuchBucket",
			err: minio.ErrorResponse{
				StatusCode: 404,
				Code:       "NoSuchBucket",
			},
			expected: false,
		},
		{
			name: "S3 403 AccessDenied",
			err: minio.ErrorResponse{
				StatusCode: 403,
				Code:       "AccessDenied",
			},
			expected: false,
		},
		{
			name: "S3 401 InvalidAccessKeyID",
			err: minio.ErrorResponse{
				StatusCode: 401,
				Code:       "InvalidAccessKeyID",
			},
			expected: false,
		},
		{
			name: "S3 403 SignatureDoesNotMatch",
			err: minio.ErrorResponse{
				StatusCode: 403,
				Code:       "SignatureDoesNotMatch",
			},
			expected: false,
		},
		{
			name: "S3 400 InvalidArgument",
			err: minio.ErrorResponse{
				StatusCode: 400,
				Code:       "InvalidArgument",
			},
			expected: false,
		},
		{
			name: "S3 400 InvalidBucketName",
			err: minio.ErrorResponse{
				StatusCode: 400,
				Code:       "InvalidBucketName",
			},
			expected: false,
		},
		{
			name: "S3 400 InvalidDigest",
			err: minio.ErrorResponse{
				StatusCode: 400,
				Code:       "InvalidDigest",
			},
			expected: false,
		},
		{
			name: "S3 409 Conflict",
			err: minio.ErrorResponse{
				StatusCode: 409,
				Code:       "Conflict",
			},
			expected: false,
		},
		{
			name: "S3 409 BucketAlreadyExists",
			err: minio.ErrorResponse{
				StatusCode: 409,
				Code:       "BucketAlreadyExists",
			},
			expected: false,
		},
		{
			name: "S3 400 EntityTooLarge",
			err: minio.ErrorResponse{
				StatusCode: 400,
				Code:       "EntityTooLarge",
			},
			expected: false,
		},
		{
			name: "S3 501 NotImplemented",
			err: minio.ErrorResponse{
				StatusCode: 501,
				Code:       "NotImplemented",
			},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("some other error"),
			expected: false,
		},
		{
			name:     "wrapped generic error",
			err:      errors.Join(errors.New("some error"), errors.New("another")),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetriableError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type temporaryNetError struct {
	temporary bool
	timeout   bool
}

func (e *temporaryNetError) Error() string {
	return "temporary network error"
}

func (e *temporaryNetError) Temporary() bool {
	return e.temporary
}

func (e *temporaryNetError) Timeout() bool {
	return e.timeout
}

func (e *temporaryNetError) Network() string {
	return "tcp"
}
