package main

import (
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff [<env>]",
	Short: "Show what files an agent changed",
	Long: `Display the code changes made by an agent in an environment.
Shows a git diff between the environment's state and your current branch.

If no environment is specified, automatically selects from environments 
that are descendants of the current HEAD.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what changes the agent made
container-use diff fancy-mallard

# Quick assessment before merging
container-use diff backend-api

# Auto-select environment
container-use diff`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		envID, err := resolveEnvironmentID(ctx, repo, args)
		if err != nil {
			return err
		}

		return repo.Diff(ctx, envID, os.Stdout)
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
