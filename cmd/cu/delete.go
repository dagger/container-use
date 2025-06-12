package main

import (
	"fmt"
	"io"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <env>",
	Short: "Delete an environment",
	Long:  `Delete an environment and its associated resources.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		envID := args[0]

		dag, err := dagger.Connect(ctx, dagger.WithLogOutput(io.Discard))
		if err != nil {
			return fmt.Errorf("failed to connect to dagger: %w", err)
		}
		defer dag.Close()
		environment.Initialize(dag)

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}
		if err := repo.Delete(ctx, envID); err != nil {
			return fmt.Errorf("failed to delete environment: %w", err)
		}

		fmt.Printf("Environment '%s' deleted successfully.\n", envID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
