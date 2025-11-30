package model

import (
	"context"
	"encoding/json"
	"errors"
)

// ErrAlreadyExists is returned when an item with the same title already exists.
var ErrAlreadyExists = errors.New("item already exists")

// ErrItemNotFound is returned when an item does not exist.
var ErrItemNotFound = errors.New("item not found")

// ItemType represents the type of stored item.
type ItemType string

const (
	// ItemTypeLogin denotes a login/password item.
	ItemTypeLogin ItemType = "login"
	// ItemTypeNote denotes a text note item.
	ItemTypeNote ItemType = "note"
	// ItemTypeFile denotes a file item.
	ItemTypeFile ItemType = "file"
)

// Valid reports whether the item type is supported.
func (t ItemType) Valid() bool {
	switch t {
	case ItemTypeLogin, ItemTypeNote, ItemTypeFile:
		return true
	default:
		return false
	}
}

// ItemsRepository defines storage operations for items.
//
//go:generate mockgen -destination ../service/items/mocks/items_repo_gen.go -package mocks . ItemsRepository
type ItemsRepository interface {
	List(ctx context.Context, userID int, iType ItemType) ([]ItemMeta, error)
	GetItemByTitle(
		ctx context.Context,
		userID int,
		title string,
	) (*ItemEncrypted, error)
	CreateItem(
		ctx context.Context,
		userID int,
		item *ItemEncrypted,
	) (int, error)
	UpdateItem(
		ctx context.Context,
		userID int,
		title string,
		item *ItemEncrypted,
	) error
	DeleteItem(ctx context.Context, userID int, title string) error
}

// ItemMeta contains shared metadata for all item types.
type ItemMeta struct {
	Title string   `json:"title"`
	Type  ItemType `json:"type"`
}

// ItemDecrypted holds decrypted item data.
type ItemDecrypted struct {
	*ItemMeta

	Data json.RawMessage `json:"data"`
}

// ItemEncrypted stores encrypted item payloads and metadata.
type ItemEncrypted struct {
	*ItemMeta

	Ciphertext  []byte
	Nonce       []byte
	ObjectKey   string
	ChunkSize   int
	NoncePrefix []byte
}

// LoginData contains login item fields.
type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Notes    string `json:"notes"`
}

// NoteData contains note item data.
type NoteData struct {
	Data string `json:"data"`
}

// FileData describes stored file metadata.
type FileData struct {
	Filename    string `json:"filename"`
	Notes       string `json:"notes"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
