package model

import "testing"

func TestItemType_Valid(t *testing.T) {
	tests := []struct {
		name     string
		itemType ItemType
		want     bool
	}{
		{"valid login", ItemTypeLogin, true},
		{"valid note", ItemTypeNote, true},
		{"valid file", ItemTypeFile, true},
		{"invalid empty", "", false},
		{"invalid custom", "custom", false},
		{"invalid unknown", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.itemType.Valid(); got != tt.want {
				t.Errorf("ItemType.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}
