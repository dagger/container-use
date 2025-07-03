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
Shows a git diff of all changes made since the environment was created.
Use -b to compare against a specific branch instead of showing full diff.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what changes the agent made (full diff)
cu diff fancy-mallard

# Compare against main branch
cu diff fancy-mallard -b main

# Quick assessment before merging
cu diff backend-api -b main`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		branch, _ := app.Flags().GetString("branch")

		return repo.Diff(ctx, args[0], branch, os.Stdout)
	},
}

func init() {
	diffCmd.Flags().StringP("branch", "b", "", "Compare against specified branch (uses merge-base)")
	rootCmd.AddCommand(diffCmd)
}