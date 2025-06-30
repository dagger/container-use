package main

import (
	"github.com/dagger/container-use/cmd/cli"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:               "delete <env>...",
	Short:             "Delete environments",
	Long:              `Delete one or more environments and their associated resources.`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cli.DeleteEnvironments(cmd.Context(), ".", args)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
