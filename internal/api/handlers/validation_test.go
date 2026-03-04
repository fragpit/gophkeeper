package handlers

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fragpit/gophkeeper/internal/model"
)

func TestValidateItemSize(t *testing.T) {
	tests := []struct {
		name    string
		item    *model.ItemDecrypted
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil item",
			item:    nil,
			wantErr: true,
		},
		{
			name: "valid login item",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: json.RawMessage(`{"username":"user","password":"pass"}`),
			},
			wantErr: false,
		},
		{
			name: "title too long",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: strings.Repeat("a", MaxItemTitleLength+1),
					Type:  model.ItemTypeLogin,
				},
				Data: json.RawMessage(`{"username":"user"}`),
			},
			wantErr: true,
			errMsg:  "item title too long",
		},
		{
			name: "data too large - raw json",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: json.RawMessage(strings.Repeat("a", MaxItemDataSize+1)),
			},
			wantErr: true,
			errMsg:  "item data too large",
		},
		{
			name: "login fields too large",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeLogin,
				},
				Data: func() json.RawMessage {
					login := model.LoginData{
						Username: strings.Repeat("u", MaxItemDataSize/2),
						Password: strings.Repeat("p", MaxItemDataSize/2+1),
						URL:      "http://example.com",
						Notes:    "notes",
					}
					data, _ := json.Marshal(login)
					return data
				}(),
			},
			wantErr: true,
			errMsg:  "item data too large",
		},
		{
			name: "note data too large",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeNote,
				},
				Data: func() json.RawMessage {
					note := model.NoteData{
						Data: strings.Repeat("n", MaxItemDataSize+1),
					}
					data, _ := json.Marshal(note)
					return data
				}(),
			},
			wantErr: true,
			errMsg:  "item data too large",
		},
		{
			name: "valid note at boundary",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "test",
					Type:  model.ItemTypeNote,
				},
				Data: func() json.RawMessage {
					note := model.NoteData{
						Data: strings.Repeat(
							"n",
							MaxItemDataSize-100,
						), // оставляем место для JSON
					}
					data, _ := json.Marshal(note)
					return data
				}(),
			},
			wantErr: false,
		},
		{
			name: "valid title at max length",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: strings.Repeat("a", MaxItemTitleLength),
					Type:  model.ItemTypeLogin,
				},
				Data: json.RawMessage(`{"username":"user"}`),
			},
			wantErr: false,
		},
		{
			name: "file type item",
			item: &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: "myfile",
					Type:  model.ItemTypeFile,
				},
				Data: json.RawMessage(`{"filename":"test.txt","size":1000}`),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateItemSize(tt.item)
			if tt.wantErr {
				if err == nil {
					t.Errorf(
						"ValidateItemSize() expected error containing %q, got nil",
						tt.errMsg,
					)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidateItemSize() error = %v, want error containing %q",
						err,
						tt.errMsg,
					)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateItemSize() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "valid small file",
			fileSize: 1024,
			wantErr:  false,
		},
		{
			name:     "valid file at boundary",
			fileSize: MaxFileSize,
			wantErr:  false,
		},
		{
			name:     "file too large",
			fileSize: MaxFileSize + 1,
			wantErr:  true,
			errMsg:   "file size exceeds maximum limit",
		},
		{
			name:     "very large file",
			fileSize: MaxFileSize * 2,
			wantErr:  true,
			errMsg:   "file size exceeds maximum limit",
		},
		{
			name:     "zero size file",
			fileSize: 0,
			wantErr:  false,
		},
		{
			name:     "1 MB file",
			fileSize: 1024 * 1024,
			wantErr:  false,
		},
		{
			name:     "500 MB file",
			fileSize: 500 * 1024 * 1024,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.fileSize)
			if tt.wantErr {
				if err == nil {
					t.Errorf(
						"ValidateFileSize() expected error containing %q, got nil",
						tt.errMsg,
					)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf(
						"ValidateFileSize() error = %v, want error containing %q",
						err,
						tt.errMsg,
					)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFileSize() unexpected error = %v", err)
				}
			}
		})
	}
}
