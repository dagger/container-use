package main

import (
	"os"
	"os/exec"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var mergeCmd = &cobra.Command{
	Use:               "merge <env>",
	Short:             "Accept an environment's work into your branch",
	Long: `Merge an environment's changes into your current git branch.
This makes the agent's work permanent in your repository.
Your working directory will be automatically stashed and restored.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Accept agent's work into current branch
cu merge fancy-mallard

# Merge after reviewing with diff/log
cu merge backend-api`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		if _, err := repository.Open(ctx, "."); err != nil {
			return err
		}

		env := args[0]
		err := exec.Command("git", "stash", "--include-untracked", "-q").Run()
		if err == nil {
			defer exec.Command("git", "stash", "pop", "-q").Run()
		}
		cmd := exec.CommandContext(ctx, "git", "merge", "-m", "Merge environment "+env, "--", "container-use/"+env)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		return cmd.Run()
	},
}

func init() {
	rootCmd.AddCommand(mergeCmd)
}