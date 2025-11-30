package main

import (
	"github.com/spf13/cobra"
)

func NewGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get server object",
	}

	cmd.AddCommand(NewGetItemCmd())

	return cmd
}
