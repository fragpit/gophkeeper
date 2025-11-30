package main

import (
	"github.com/spf13/cobra"
)

// NewGetItemCmd creates the command for fetching a single item.
func NewGetItemCmd() *cobra.Command {
	var (
		title    string
		filePath string
	)
	getCmd := &cobra.Command{
		Use:   "item",
		Short: "Get item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.Client.GetItem(cmd.Context(), title, filePath); err != nil {
				return err
			}

			return nil
		},
	}

	getCmd.Flags().StringVar(&title, "title", "", "Item title")
	getCmd.Flags().
		StringVar(&filePath, "file-path", "", "Save file to path, dir must exist")

	return getCmd
}
