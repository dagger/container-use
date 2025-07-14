package main

import (
	"context"
	"fmt"
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	mergeDelete bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge [<env>]",
	Short: "Accept an environment's work into your branch",
	Long: `Merge an environment's changes into your current git branch.
This makes the agent's work permanent in your repository.
Your working directory will be automatically stashed and restored.

If no environment is specified, automatically selects from environments 
that are descendants of the current HEAD.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Accept agent's work into current branch
container-use merge backend-api

# Merge and delete the environment after successful merge
container-use merge -d backend-api
container-use merge --delete backend-api

# Auto-select environment
container-use merge`,
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

		if err := repo.Merge(ctx, envID, os.Stdout); err != nil {
			return fmt.Errorf("failed to merge environment: %w", err)
		}

		return deleteAfterMerge(ctx, repo, envID, mergeDelete, "merged")
	},
}

func deleteAfterMerge(ctx context.Context, repo *repository.Repository, env string, delete bool, verb string) error {
	if !delete {
		fmt.Printf("Environment '%s' %s successfully.\n", env, verb)
		return nil
	}
	if err := repo.Delete(ctx, env); err != nil {
		return fmt.Errorf("environment '%s' %s but delete failed: %w", env, verb, err)
	}
	fmt.Printf("Environment '%s' %s and deleted successfully.\n", env, verb)
	return nil
}

func init() {
	mergeCmd.Flags().BoolVarP(&mergeDelete, "delete", "d", false, "Delete the environment after successful merge")

	rootCmd.AddCommand(mergeCmd)
}
