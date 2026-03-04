package main

import (
	"github.com/spf13/cobra"
)

func NewDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete server object",
	}

	cmd.AddCommand(NewDeleteItemCmd())

	return cmd
}
