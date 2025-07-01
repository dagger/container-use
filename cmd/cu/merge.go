package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var (
	mergeDelete bool
)

var mergeCmd = &cobra.Command{
	Use:   "merge <env>",
	Short: "Accept an environment's work into your branch",
	Long: `Merge an environment's changes into your current git branch.
This makes the agent's work permanent in your repository.
Your working directory will be automatically stashed and restored.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Accept agent's work into current branch
cu merge backend-api

# Merge and delete the environment after successful merge
cu merge -d backend-api
cu merge --delete backend-api`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		env := args[0]
		err = exec.Command("git", "stash", "--include-untracked", "-q").Run()
		if err == nil {
			defer exec.Command("git", "stash", "pop", "-q").Run()
		}
		cmd := exec.CommandContext(ctx, "git", "merge", "-m", "Merge environment "+env, "--", "container-use/"+env)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("failed to merge environment: %w", err)
		}

		// Delete the environment if the flag is set and merge was successful
		if mergeDelete {
			if err := repo.Delete(ctx, env); err != nil {
				return fmt.Errorf("merge succeeded but failed to delete environment: %w", err)
			}
			fmt.Printf("Environment '%s' merged and deleted successfully.\n", env)
		} else {
			fmt.Printf("Environment '%s' merged successfully.\n", env)
		}

		return nil
	},
}

func init() {
	mergeCmd.Flags().BoolVarP(&mergeDelete, "delete", "d", false, "Delete the environment after successful merge")
	rootCmd.AddCommand(mergeCmd)
}
