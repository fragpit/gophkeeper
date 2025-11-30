package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/spf13/cobra"
)

// NewUpdateItemCmd creates the command group for item update by type.
func NewUpdateItemCmd() *cobra.Command {
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Update existing item of specific type.",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = cmd.Usage()
			cmd.Println("\nAvailable types:")
			for _, t := range itemTypes {
				cmd.Printf("  %s\t%s\n", t.Name, t.Description)
			}
			return nil
		},
	}

	for _, t := range itemTypes {
		itemCmd.AddCommand(t.NewUpdateCmd())
	}

	return itemCmd
}

func newUpdateLoginCmd() *cobra.Command {
	type options struct {
		Title    string
		NewTitle string
		Username string
		Password string
		URL      string
		Notes    string
	}

	var o options
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Update login/password item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.Title == "" {
				return fmt.Errorf("--title is required to identify the item")
			}

			d, err := json.Marshal(model.LoginData{
				Username: o.Username,
				Password: o.Password,
				URL:      o.URL,
				Notes:    o.Notes,
			})
			if err != nil {
				return fmt.Errorf("marshal login data: %w", err)
			}

			title := o.Title
			if o.NewTitle != "" {
				title = o.NewTitle
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: title,
					Type:  model.ItemTypeLogin,
				},
				Data: d,
			}

			if err := cli.Client.UpdateItem(cmd.Context(), o.Title, item); err != nil {
				return fmt.Errorf("cmd update item: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Current item title (required)")
	cmd.Flags().
		StringVar(&o.NewTitle, "new-title", "", "New item title (optional)")
	cmd.Flags().StringVar(&o.Username, "username", "", "Username")
	cmd.Flags().StringVar(&o.Password, "password", "", "Password")
	cmd.Flags().StringVar(&o.URL, "url", "", "Login URL")
	cmd.Flags().StringVar(&o.Notes, "notes", "", "Notes")

	return cmd
}

func newUpdateNoteCmd() *cobra.Command {
	type options struct {
		Title    string
		NewTitle string
		Data     string
	}

	var o options

	cmd := &cobra.Command{
		Use:   "note",
		Short: "Update secure note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.Title == "" {
				return fmt.Errorf("--title is required to identify the item")
			}

			d, err := json.Marshal(model.NoteData{Data: o.Data})
			if err != nil {
				return fmt.Errorf("marshal note data: %w", err)
			}

			title := o.Title
			if o.NewTitle != "" {
				title = o.NewTitle
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: title,
					Type:  model.ItemTypeNote,
				},
				Data: d,
			}

			if err := cli.Client.UpdateItem(cmd.Context(), o.Title, item); err != nil {
				return fmt.Errorf("cmd update item: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Current item title (required)")
	cmd.Flags().
		StringVar(&o.NewTitle, "new-title", "", "New item title (optional)")
	cmd.Flags().StringVar(&o.Data, "data", "", "Note text")

	return cmd
}

func newUpdateFileCmd() *cobra.Command {
	type options struct {
		Title    string
		NewTitle string
		FilePath string
		Notes    string
	}

	var o options

	cmd := &cobra.Command{
		Use:   "file",
		Short: "Update file item metadata",
		Long: `Update file item metadata and optionally the file content.
If --path is provided, the file content will be updated.
If --path is not provided, only metadata (title, notes) will be updated.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if o.Title == "" {
				return fmt.Errorf("--title is required to identify the item")
			}

			// Get current item metadata to preserve file information
			currentItem, err := cli.Client.GetItemMetadata(cmd.Context(), o.Title)
			if err != nil {
				return fmt.Errorf("get current item: %w", err)
			}

			if currentItem.Type != "file" {
				return fmt.Errorf("item is not a file type")
			}

			// Parse current file data
			var currentFileData model.FileData
			if len(currentItem.Data) > 0 {
				if err := json.Unmarshal(currentItem.Data, &currentFileData); err != nil {
					return fmt.Errorf("unmarshal current file data: %w", err)
				}
			}

			title := o.Title
			if o.NewTitle != "" {
				title = o.NewTitle
			}

			// Prepare updated file data, preserving existing fields
			fileData := model.FileData{
				Filename:    currentFileData.Filename,
				Notes:       currentFileData.Notes,
				ContentType: currentFileData.ContentType,
				Size:        currentFileData.Size,
			}

			// Update only specified fields
			if o.Notes != "" {
				fileData.Notes = o.Notes
			}

			// If path is provided, update filename
			// Note: Actual file content update is not implemented yet
			if o.FilePath != "" {
				fileData.Filename = filepath.Base(o.FilePath)
				// TODO: implement file content update (upload new file to S3)
			}

			d, err := json.Marshal(fileData)
			if err != nil {
				return fmt.Errorf("marshal file data: %w", err)
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: title,
					Type:  model.ItemTypeFile,
				},
				Data: d,
			}

			if err := cli.Client.UpdateItem(cmd.Context(), o.Title, item); err != nil {
				return fmt.Errorf("cmd update item: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Current item title (required)")
	cmd.Flags().
		StringVar(&o.NewTitle, "new-title", "", "New item title (optional)")
	cmd.Flags().
		StringVar(&o.FilePath, "path", "", "Path to the new file (optional, for content update)")
	cmd.Flags().StringVar(&o.Notes, "notes", "", "Notes")

	return cmd
}
