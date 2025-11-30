package main

import (
	"github.com/spf13/cobra"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create server object",
	}

	cmd.AddCommand(NewCreateItemCmd())

	return cmd
}
