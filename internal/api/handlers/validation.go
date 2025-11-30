package handlers

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/fragpit/gophkeeper/internal/model"
)

const (
	// MaxItemTitleLength максимальная длина заголовка item.
	MaxItemTitleLength = 255

	// MaxItemDataSize максимальный размер данных item (в байтах).
	// Ограничение для login/note items.
	MaxItemDataSize = 1024 * 1024 // 1 MB

	// MaxFileSize максимальный размер файла (в байтах).
	MaxFileSize = 1024 * 1024 * 1024 // 1 GB
)

var (
	// ErrItemTitleTooLong возвращается когда заголовок item превышает максимальную длину.
	ErrItemTitleTooLong = errors.New("item title too long")

	// ErrItemDataTooLarge возвращается когда данные item превышают максимальный размер.
	ErrItemDataTooLarge = errors.New("item data too large")

	// ErrFileTooLarge возвращается когда файл превышает максимальный размер.
	ErrFileTooLarge = errors.New("file size exceeds maximum limit")
)

// ValidateItemSize проверяет размеры данных item.
func ValidateItemSize(item *model.ItemDecrypted) error {
	if item == nil || item.ItemMeta == nil {
		return fmt.Errorf("item is nil")
	}

	// Проверка длины заголовка
	if len(item.Title) > MaxItemTitleLength {
		return fmt.Errorf(
			"%w: max %d characters",
			ErrItemTitleTooLong,
			MaxItemTitleLength,
		)
	}

	// Проверка размера данных
	if len(item.Data) > MaxItemDataSize {
		return fmt.Errorf("%w: max %d bytes", ErrItemDataTooLarge, MaxItemDataSize)
	}

	// Дополнительная проверка для типов login и note
	switch item.Type {
	case model.ItemTypeLogin:
		var loginData model.LoginData
		if err := json.Unmarshal(item.Data, &loginData); err == nil {
			totalSize := len(loginData.Username) + len(loginData.Password) +
				len(loginData.URL) + len(loginData.Notes)
			if totalSize > MaxItemDataSize {
				return fmt.Errorf(
					"%w: total fields size %d bytes",
					ErrItemDataTooLarge,
					totalSize,
				)
			}
		}
	case model.ItemTypeNote:
		var noteData model.NoteData
		if err := json.Unmarshal(item.Data, &noteData); err == nil {
			if len(noteData.Data) > MaxItemDataSize {
				return fmt.Errorf(
					"%w: note data size %d bytes",
					ErrItemDataTooLarge,
					len(noteData.Data),
				)
			}
		}
	}

	return nil
}

// ValidateFileSize проверяет размер файла.
func ValidateFileSize(fileSize int64) error {
	if fileSize > MaxFileSize {
		return fmt.Errorf("%w: file size %d bytes, max %d bytes",
			ErrFileTooLarge, fileSize, MaxFileSize)
	}
	return nil
}
