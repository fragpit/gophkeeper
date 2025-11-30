package model

import (
	"context"
	"encoding/json"
	"errors"
)

var ErrAlreadyExists = errors.New("item already exists")

type ItemType string

const (
	ItemTypeLogin ItemType = "login"
	ItemTypeNote  ItemType = "note"
	ItemTypeFile  ItemType = "file"
)

func (t ItemType) Valid() bool {
	switch t {
	case ItemTypeLogin, ItemTypeNote, ItemTypeFile:
		return true
	default:
		return false
	}
}

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
}

type ItemMeta struct {
	Title string   `json:"title"`
	Type  ItemType `json:"type"`
}

type ItemDecrypted struct {
	*ItemMeta

	Data json.RawMessage `json:"data"`
}

type ItemEncrypted struct {
	*ItemMeta

	Ciphertext  []byte
	Nonce       []byte
	ObjectKey   string
	ChunkSize   int
	NoncePrefix []byte
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	URL      string `json:"url"`
	Notes    string `json:"notes"`
}

type NoteData struct {
	Data string `json:"data"`
}

type FileData struct {
	Filename    string `json:"filename"`
	Notes       string `json:"notes"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}
