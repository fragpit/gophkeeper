package main

import (
	"github.com/spf13/cobra"
)

// NewUpdateCmd creates the update command group.
func NewUpdateCmd() *cobra.Command {
	updateCmd := &cobra.Command{
		Use:   "update",
		Short: "Update existing items",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	updateCmd.AddCommand(NewUpdateItemCmd())

	return updateCmd
}
