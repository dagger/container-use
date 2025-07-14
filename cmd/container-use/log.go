package main

import (
	"os"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var logCmd = &cobra.Command{
	Use:   "log [<env>]",
	Short: "View what an agent did step-by-step",
	Long: `Display the complete development history for an environment.
Shows all commits made by the agent plus command execution notes.
Use -p to include code patches in the output.

If no environment is specified, automatically selects from environments 
that are descendants of the current HEAD.`,
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what agent did
container-use log fancy-mallard

# Include code changes
container-use log fancy-mallard -p

# Auto-select environment
container-use log`,
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

		patch, _ := app.Flags().GetBool("patch")

		return repo.Log(ctx, envID, patch, os.Stdout)
	},
}

func init() {
	logCmd.Flags().BoolP("patch", "p", false, "Generate patch")
	rootCmd.AddCommand(logCmd)
}
