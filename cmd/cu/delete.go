package main

import (
	"github.com/dagger/container-use/cmd/cli"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:               "delete <env>...",
	Short:             "Delete environments and start fresh",
	Long: `Delete one or more environments and their associated resources.
This permanently removes the environment's branch and container state.
Use this when starting over with a different approach.`,
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Delete a single environment
cu delete fancy-mallard

# Delete multiple environments at once
cu delete env1 env2 env3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cli.DeleteEnvironments(cmd.Context(), ".", args)
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}