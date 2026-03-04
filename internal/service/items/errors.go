package items

import "errors"

var (
	ErrInvalidItemData  = errors.New("invalid item data")
	ErrInvalidItemType  = errors.New("invalid item type")
	ErrEmptyObjectKey   = errors.New("empty object key")
	ErrEmptyNoncePrefix = errors.New("empty nonce prefix")
	ErrEmptyChunkSize   = errors.New("empty chunk size")
	ErrInvalidFileData  = errors.New("invalid file data")
)
