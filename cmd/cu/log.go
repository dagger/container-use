package main

import (
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log <env>",
	Short: "View what an agent did step-by-step",
	Long: `Display the complete development history for an environment.
Shows all commits made by the agent plus command execution notes.
Use -p to include code patches in the output.
Use -b to compare against a specific branch instead of showing full history.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what agent did (full history)
cu log fancy-mallard

# Include code changes
cu log fancy-mallard -p

# Compare against main branch
cu log fancy-mallard -b main

# Compare against main with patches
cu log fancy-mallard -b main -p`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		patch, _ := app.Flags().GetBool("patch")
		branch, _ := app.Flags().GetString("branch")

		return repo.Log(ctx, args[0], patch, branch, os.Stdout)
	},
}

func init() {
	logCmd.Flags().BoolP("patch", "p", false, "Generate patch")
	logCmd.Flags().StringP("branch", "b", "", "Compare against specified branch (uses merge-base)")
	rootCmd.AddCommand(logCmd)
}