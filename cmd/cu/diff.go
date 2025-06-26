package main

import (
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:               "diff <env>",
	Short:             "Show changes between the environment and the local branch",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		return repo.Diff(ctx, args[0], os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
