package main

import (
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff <env>",
	Short: "Show what files an agent changed",
	Long: `Display the code changes made by an agent in an environment.
Shows a git diff between the environment's state and your current branch.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what changes the agent made
cu diff fancy-mallard

# Quick assessment before merging
cu diff backend-api`,
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
