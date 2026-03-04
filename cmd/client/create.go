package main

import (
	"github.com/spf13/cobra"
)

// NewCreateCmd creates the parent command for item creation.
func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create server object",
	}

	cmd.AddCommand(NewCreateItemCmd())

	return cmd
}
