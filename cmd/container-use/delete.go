package main

import (
	"fmt"

	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [<env>...]",
	Short: "Delete environments and start fresh",
	Long: `Delete one or more environments and their associated resources.
This permanently removes the environment's branch and container state.
Use this when starting over with a different approach.

Use --all to delete all environments at once.`,
	Args: func(cmd *cobra.Command, args []string) error {
		all, _ := cmd.Flags().GetBool("all")
		if all && len(args) > 0 {
			return fmt.Errorf("cannot specify environment names when using --all flag")
		}
		if !all && len(args) == 0 {
			return fmt.Errorf("must specify at least one environment name or use --all flag")
		}
		return nil
	},
	ValidArgsFunction: suggestEnvironments,
	Example: `# Delete a single environment
container-use delete fancy-mallard

# Delete multiple environments at once
container-use delete env1 env2 env3

# Delete all environments
container-use delete --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		all, _ := cmd.Flags().GetBool("all")

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		var envIDs []string
		if all {
			// Get all environment IDs
			envs, err := repo.List(ctx)
			if err != nil {
				return fmt.Errorf("failed to list environments: %w", err)
			}
			if len(envs) == 0 {
				fmt.Println("No environments found to delete.")
				return nil
			}
			for _, env := range envs {
				envIDs = append(envIDs, env.ID)
			}
			fmt.Printf("Deleting %d environment(s)...\n", len(envIDs))
		} else {
			envIDs = args
		}

		for _, envID := range envIDs {
			if err := repo.Delete(ctx, envID); err != nil {
				return fmt.Errorf("failed to delete environment '%s': %w", envID, err)
			}
			fmt.Printf("Environment '%s' deleted successfully.\n", envID)
		}

		if all {
			fmt.Printf("Successfully deleted %d environment(s).\n", len(envIDs))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().Bool("all", false, "Delete all environments")
}
