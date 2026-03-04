package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewListItemCmd creates the command that lists items with optional filters.
func NewListItemCmd() *cobra.Command {
	var iType string
	itemCmd := &cobra.Command{
		Use:   "items",
		Short: "List items using specified filters.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cli.Client.List(cmd.Context(), iType); err != nil {
				return fmt.Errorf("cmd list items: %w", err)
			}

			return nil
		},
	}

	itemCmd.Flags().StringVar(&iType, "type", "", "filter item types")

	return itemCmd
}
