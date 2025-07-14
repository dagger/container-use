package main

import (
	"fmt"
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	applyDelete bool
)

var applyCmd = &cobra.Command{
	Use:   "apply [<env>]",
	Short: "Apply an environment's work as staged changes to your branch",
	Long: `Apply an environment's changes to your current git branch as staged modifications.
Unlike 'merge' which preserves the original commit history, 'apply' stages all changes
for you to commit manually, discarding the original commit sequence. This lets you
review and customize the final commit before making the agent's work permanent.
Your working directory will be automatically stashed and restored.

If no environment is specified, automatically selects from environments 
that are descendants of the current HEAD.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Apply agent's work as staged changes to current branch
cu apply backend-api

# Apply and delete the environment after successful application
cu apply -d backend-api
cu apply --delete backend-api

# After applying, you can review and commit the changes
git status
git commit -m "Add backend API implementation"

# Auto-select environment
cu apply`,
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

		if err := repo.Apply(ctx, envID, os.Stdout); err != nil {
			return fmt.Errorf("failed to apply environment: %w", err)
		}

		return deleteAfterMerge(ctx, repo, envID, applyDelete, "applied")
	},
}

func init() {
	applyCmd.Flags().BoolVarP(&applyDelete, "delete", "d", false, "Delete the environment after successful application")

	rootCmd.AddCommand(applyCmd)
}
