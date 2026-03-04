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

			existing, err := cli.Client.GetItemMetadata(cmd.Context(), o.Title)
			if err != nil {
				return fmt.Errorf("get existing item: %w", err)
			}

			var loginData model.LoginData
			if err := json.Unmarshal(existing.Data, &loginData); err != nil {
				return fmt.Errorf("unmarshal existing login data: %w", err)
			}

			if cmd.Flags().Changed("username") {
				loginData.Username = o.Username
			}
			if cmd.Flags().Changed("password") {
				loginData.Password = o.Password
			}
			if cmd.Flags().Changed("url") {
				loginData.URL = o.URL
			}
			if cmd.Flags().Changed("notes") {
				loginData.Notes = o.Notes
			}

			d, err := json.Marshal(loginData)
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

			if err := cli.Client.UpdateItem(
				cmd.Context(),
				o.Title,
				item,
			); err != nil {
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

			existing, err := cli.Client.GetItemMetadata(cmd.Context(), o.Title)
			if err != nil {
				return fmt.Errorf("get existing item: %w", err)
			}

			var noteData model.NoteData
			if err := json.Unmarshal(existing.Data, &noteData); err != nil {
				return fmt.Errorf("unmarshal existing note data: %w", err)
			}

			if cmd.Flags().Changed("data") {
				noteData.Data = o.Data
			}

			d, err := json.Marshal(noteData)
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

			if err := cli.Client.UpdateItem(
				cmd.Context(),
				o.Title,
				item,
			); err != nil {
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

			currentItem, err := cli.Client.GetItemMetadata(cmd.Context(), o.Title)
			if err != nil {
				return fmt.Errorf("get current item: %w", err)
			}

			if currentItem.Type != model.ItemTypeFile {
				return fmt.Errorf("item is not a file type")
			}

			var fileData model.FileData
			if len(currentItem.Data) > 0 {
				if err := json.Unmarshal(currentItem.Data, &fileData); err != nil {
					return fmt.Errorf("unmarshal current file data: %w", err)
				}
			}

			if cmd.Flags().Changed("notes") {
				fileData.Notes = o.Notes
			}

			if cmd.Flags().Changed("path") {
				fileData.Filename = filepath.Base(o.FilePath)
			}

			title := o.Title
			if o.NewTitle != "" {
				title = o.NewTitle
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

			if err := cli.Client.UpdateItem(
				cmd.Context(),
				o.Title,
				item,
			); err != nil {
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
