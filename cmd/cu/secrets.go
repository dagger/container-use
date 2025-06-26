package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage environment secrets",
	Long:  `Add, remove, and list secrets for container-use environments`,
}

var secretAddCmd = &cobra.Command{
	Use:   "add <SECRET_NAME> <schema://value>",
	Short: "Add a secret to an environment",
	Long: `Add a secret to a container-use environment.
Supported schemas:
- file://PATH: local file path
- env://NAME: environment variable
- op://<vault-name>/<item-name>/[section-name/]<field-name>: 1Password secret`,
	Args:              cobra.ExactArgs(2),
	ValidArgsFunction: suggestEnvironments,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		secretName := args[0]
		secretSpec := args[1]

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		var cfg environment.EnvironmentConfig
		if err := cfg.Load(repo.SourcePath()); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to load environment config: %w", err)
			}
		}

		if err := cfg.Secrets.AddSecret(secretName, secretSpec); err != nil {
			return fmt.Errorf("failed to add secret: %w", err)
		}

		if err := cfg.Save(repo.SourcePath()); err != nil {
			return fmt.Errorf("failed to update environment config: %w", err)
		}

		fmt.Printf("Secret %s successfully added to environment\n", secretName)
		return nil
	},
}

var secretDeleteCmd = &cobra.Command{
	Use:               "delete [SECRET_NAME]",
	Short:             "Delete a secret from an environment",
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: suggestEnvironments,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		secretName := args[0]

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		var cfg environment.EnvironmentConfig
		if err := cfg.Load(repo.SourcePath()); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to load environment config: %w", err)
			}
		}

		if err := cfg.Secrets.DeleteSecret(secretName); err != nil {
			return fmt.Errorf("failed to delete secret: %w", err)
		}

		if err := cfg.Save(repo.SourcePath()); err != nil {
			return fmt.Errorf("failed to update environment config: %w", err)
		}

		fmt.Printf("Secret %s successfully deleted\n", secretName)
		return nil
	},
}

var secretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets in an environment",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		repo, err := repository.Open(ctx, ".")
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}

		var cfg environment.EnvironmentConfig
		if err := cfg.Load(repo.SourcePath()); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("failed to load environment config: %w", err)
			}
		}

		secretNames := cfg.Secrets.List()
		if len(secretNames) == 0 {
			fmt.Printf("No secrets found\n")
			return nil
		}

		fmt.Printf("Secrets:\n")
		for _, name := range secretNames {
			fmt.Printf("- %s\n", name)
		}
		return nil
	},
}

func init() {
	secretsCmd.AddCommand(secretAddCmd)
	secretsCmd.AddCommand(secretDeleteCmd)
	secretsCmd.AddCommand(secretListCmd)
	rootCmd.AddCommand(secretsCmd)
}
