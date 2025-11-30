package main

import (
	"github.com/spf13/cobra"
)

func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List server objects",
	}

	cmd.AddCommand(NewListItemCmd())

	return cmd
}
