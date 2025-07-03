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
Use -p to include code patches in the output.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# See what agent did
cu log fancy-mallard

# Include code changes
cu log fancy-mallard -p`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		patch, _ := app.Flags().GetBool("patch")

		return repo.Log(ctx, args[0], patch, os.Stdout)
	},
}

func init() {
	logCmd.Flags().BoolP("patch", "p", false, "Generate patch")
	rootCmd.AddCommand(logCmd)
}
