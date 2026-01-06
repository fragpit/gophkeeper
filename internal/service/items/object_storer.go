package items

import (
	"context"
	"io"
)

// ObjectStorer abstracts object storage interactions.
type ObjectStorer interface {
	Get(ctx context.Context, bucket, name string) (io.ReadCloser, error)
	Put(ctx context.Context, name string, r io.Reader) error
}
