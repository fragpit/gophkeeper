package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/fragpit/gophkeeper/internal/model"
	"github.com/spf13/cobra"
)

// ItemType describes an available item creation command.
type ItemType struct {
	Name        string
	Description string
	NewCmd      func() *cobra.Command
}

var itemTypes = []ItemType{
	{
		Name:        "login",
		Description: "Create login/password entry",
		NewCmd:      newCreateLoginCmd,
	},
	{
		Name:        "note",
		Description: "Create secure note",
		NewCmd:      newCreateNoteCmd,
	},
	{
		Name:        "file",
		Description: "Create file item",
		NewCmd:      newCreateFileCmd,
	},
}

// NewCreateItemCmd creates the command group for item creation by type.
func NewCreateItemCmd() *cobra.Command {
	itemCmd := &cobra.Command{
		Use:   "item",
		Short: "Create new item of specific type.",
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
		itemCmd.AddCommand(t.NewCmd())
	}

	return itemCmd
}

func newCreateLoginCmd() *cobra.Command {
	type options struct {
		Title    string
		Username string
		Password string
		URL      string
		Notes    string
	}

	var o options
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Create login/password item",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := json.Marshal(model.LoginData{
				Username: o.Username,
				Password: o.Password,
				URL:      o.URL,
				Notes:    o.Notes,
			})
			if err != nil {
				return fmt.Errorf("marshal login data: %w", err)
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: o.Title,
					Type:  "login",
				},
				Data: d,
			}

			if err := cli.Client.CreateItem(cmd.Context(), item); err != nil {
				return fmt.Errorf("cmd create item: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Item title")
	cmd.Flags().StringVar(&o.Username, "username", "", "Username")
	cmd.Flags().StringVar(&o.Password, "password", "", "Password")
	cmd.Flags().StringVar(&o.URL, "url", "", "Login URL")
	cmd.Flags().StringVar(&o.Notes, "notes", "", "Notes")

	return cmd
}

func newCreateNoteCmd() *cobra.Command {
	type options struct {
		Title string
		Data  string
	}

	var o options

	cmd := &cobra.Command{
		Use:   "note",
		Short: "Create secure note",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := json.Marshal(model.NoteData{Data: o.Data})
			if err != nil {
				return fmt.Errorf("marshal note data: %w", err)
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: o.Title,
					Type:  "login",
				},
				Data: d,
			}

			if err := cli.Client.CreateItem(cmd.Context(), item); err != nil {
				return fmt.Errorf("cmd create item: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Item title")
	cmd.Flags().StringVar(&o.Data, "data", "", "Note text")

	return cmd
}

func newCreateFileCmd() *cobra.Command {
	type options struct {
		Title    string
		FilePath string
		Notes    string
	}

	var o options

	cmd := &cobra.Command{
		Use:   "file",
		Short: "Create file item",
		RunE: func(cmd *cobra.Command, args []string) error {
			d, err := json.Marshal(model.FileData{
				Filename: filepath.Base(o.FilePath),
				Notes:    o.Notes,
			})
			if err != nil {
				return fmt.Errorf("marshal file data: %w", err)
			}

			item := &model.ItemDecrypted{
				ItemMeta: &model.ItemMeta{
					Title: o.Title,
					Type:  "file",
				},
				Data: d,
			}

			if err := cli.Client.CreateFile(cmd.Context(), item, o.FilePath); err != nil {
				return fmt.Errorf("cmd create item: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&o.Title, "title", "", "Item title")
	cmd.Flags().StringVar(&o.FilePath, "path", "", "Path to the file")
	cmd.Flags().StringVar(&o.Notes, "notes", "", "Notes")

	return cmd
}
