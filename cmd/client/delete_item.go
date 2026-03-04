package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewDeleteItemCmd() *cobra.Command {
	var (
		title   string
		confirm bool
	)

	deleteCmd := &cobra.Command{
		Use:   "item",
		Short: "Delete item",
		RunE: func(cmd *cobra.Command, args []string) error {
			if title == "" {
				return fmt.Errorf("title is required")
			}

			if !confirm {
				fmt.Printf(
					"Are you sure you want to delete item '%s'? (yes/no): ",
					title,
				)
				reader := bufio.NewReader(os.Stdin)
				response, err := reader.ReadString('\n')
				if err != nil {
					return fmt.Errorf("read confirmation: %w", err)
				}

				response = strings.TrimSpace(strings.ToLower(response))
				if response != "yes" && response != "y" {
					fmt.Println("Deletion cancelled")
					return nil
				}
			}

			if err := cli.Client.DeleteItem(cmd.Context(), title); err != nil {
				return err
			}

			return nil
		},
	}

	deleteCmd.Flags().StringVar(&title, "title", "", "Item title to delete")
	deleteCmd.Flags().BoolVar(&confirm, "yes", false, "Skip confirmation prompt")

	return deleteCmd
}
