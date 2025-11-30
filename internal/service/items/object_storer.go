package items

import (
	"context"
	"io"
)

type ObjectStorer interface {
	Get(ctx context.Context, bucket, name string) (io.ReadCloser, error)
	Put(ctx context.Context, name string, r io.Reader) error
}
