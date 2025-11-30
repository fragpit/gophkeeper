package main

import (
	"github.com/spf13/cobra"
)

// NewGetCmd creates the parent command for retrieving items.
func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get server object",
	}

	cmd.AddCommand(NewGetItemCmd())

	return cmd
}
