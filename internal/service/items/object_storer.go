package items

import (
	"context"
	"io"
)

// ObjectStorer abstracts object storage interactions.
//
//go:generate mockgen -destination ./mocks/object_storer_gen.go -package mocks . ObjectStorer
type ObjectStorer interface {
	Get(ctx context.Context, bucket, name string) (io.ReadCloser, error)
	Put(ctx context.Context, name string, r io.Reader) error
	Delete(ctx context.Context, bucket, name string) error
}
