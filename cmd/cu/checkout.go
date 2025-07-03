package main

import (
	"fmt"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var checkoutCmd = &cobra.Command{
	Use:   "checkout <env>",
	Short: "Switch to an environment's branch locally",
	Long: `Bring an environment's work into your local git workspace.
This creates a local branch from the environment's state so you can
explore files in your IDE, make changes, or continue development.`,
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	Example: `# Switch to environment's branch locally
cu checkout fancy-mallard

# Create custom branch name
cu checkout fancy-mallard -b my-review-branch`,
	RunE: func(app *cobra.Command, args []string) error {
		ctx := app.Context()
		envID := args[0]

		// Ensure we're in a git repository
		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return err
		}

		branchName, err := app.Flags().GetString("branch")
		if err != nil {
			return err
		}

		branch, err := repo.Checkout(ctx, envID, branchName)
		if err != nil {
			return err
		}

		fmt.Printf("Switched to branch '%s'\n", branch)
		return nil
	},
}

func init() {
	checkoutCmd.Flags().StringP("branch", "b", "", "Local branch name to use")
	rootCmd.AddCommand(checkoutCmd)
}
