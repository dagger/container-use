package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/dagger/container-use/cmd/container-use/agent"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/spf13/cobra"
)

// Helper function for read-only config operations
func withConfig(cmd *cobra.Command, fn func(*environment.EnvironmentConfig) error) error {
	ctx := cmd.Context()
	repo, err := repository.Open(ctx, ".")
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	config := environment.DefaultConfig()
	if err := config.Load(repo.SourcePath()); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return fn(config)
}

// Helper function for config update operations
func updateConfig(cmd *cobra.Command, fn func(*environment.EnvironmentConfig) error) error {
	ctx := cmd.Context()
	repo, err := repository.Open(ctx, ".")
	if err != nil {
		return fmt.Errorf("failed to open repository: %w", err)
	}

	config := environment.DefaultConfig()
	if err := config.Load(repo.SourcePath()); err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if err := fn(config); err != nil {
		return err
	}

	if err := config.Save(repo.SourcePath()); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	return nil
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage environment configuration",
	Long: `Configure the development environment settings such as base image and setup commands.
These settings are stored in .container-use/environment.json and apply to all new environments.`,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show all environment configuration",
	Long:  `Display all current environment configuration including base image and setup commands.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfig(cmd, func(config *environment.EnvironmentConfig) error {
			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			defer tw.Flush()

			fmt.Fprintf(tw, "Base Image:\t%s\n", config.BaseImage)
			fmt.Fprintf(tw, "Workdir:\t%s\n", config.Workdir)

			if len(config.SetupCommands) > 0 {
				fmt.Fprintf(tw, "Setup Commands:\t\n")
				for i, cmd := range config.SetupCommands {
					fmt.Fprintf(tw, "  %d.\t%s\n", i+1, cmd)
				}
			} else {
				fmt.Fprintf(tw, "Setup Commands:\t(none)\n")
			}

			envKeys := config.Env.Keys()
			if len(envKeys) > 0 {
				fmt.Fprintf(tw, "Environment Variables:\t\n")
				for i, key := range envKeys {
					value := config.Env.Get(key)
					fmt.Fprintf(tw, "  %d.\t%s=%s\n", i+1, key, value)
				}
			} else {
				fmt.Fprintf(tw, "Environment Variables:\t(none)\n")
			}

			secretKeys := config.Secrets.Keys()
			if len(secretKeys) > 0 {
				fmt.Fprintf(tw, "Secrets:\t\n")
				for i, key := range secretKeys {
					value := config.Secrets.Get(key)
					fmt.Fprintf(tw, "  %d.\t%s=%s\n", i+1, key, value)
				}
			} else {
				fmt.Fprintf(tw, "Secrets:\t(none)\n")
			}

			return nil
		})
	},
}

// Base image object commands
var configBaseImageCmd = &cobra.Command{
	Use:   "base-image",
	Short: "Manage base container image",
	Long:  `Manage the base container image for new environments.`,
}

var configBaseImageSetCmd = &cobra.Command{
	Use:   "set <image>",
	Short: "Set the base container image",
	Long:  `Set the base container image for new environments (e.g., python:3.11, node:18, ubuntu:22.04).`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		baseImage := args[0]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.BaseImage = baseImage
			fmt.Printf("Base image set to: %s\n", baseImage)
			return nil
		})
	},
}

var configBaseImageGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get the current base container image",
	Long:  `Display the current base container image.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfig(cmd, func(config *environment.EnvironmentConfig) error {
			fmt.Println(config.BaseImage)
			return nil
		})
	},
}

var configBaseImageResetCmd = &cobra.Command{
	Use:   "reset",
	Short: "Reset base image to default",
	Long:  `Reset the base container image to the default (ubuntu:24.04).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			defaultConfig := environment.DefaultConfig()
			config.BaseImage = defaultConfig.BaseImage
			fmt.Printf("Base image reset to default: %s\n", defaultConfig.BaseImage)
			return nil
		})
	},
}

// Setup command object commands
var configSetupCommandCmd = &cobra.Command{
	Use:   "setup-command",
	Short: "Manage setup commands",
	Long:  `Manage setup commands that are run when creating environments.`,
}

var configSetupCommandAddCmd = &cobra.Command{
	Use:   "add <command>",
	Short: "Add a setup command",
	Long:  `Add a command to be run when creating new environments (e.g., "apt update && apt install -y python3").`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := args[0]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.SetupCommands = append(config.SetupCommands, command)
			fmt.Printf("Setup command added: %s\n", command)
			return nil
		})
	},
}

var configSetupCommandRemoveCmd = &cobra.Command{
	Use:   "remove <command>",
	Short: "Remove a setup command",
	Long:  `Remove a setup command from the environment configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		command := args[0]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			found := false
			newCommands := make([]string, 0, len(config.SetupCommands))
			for _, existing := range config.SetupCommands {
				if existing != command {
					newCommands = append(newCommands, existing)
				} else {
					found = true
				}
			}

			if !found {
				return fmt.Errorf("setup command not found: %s", command)
			}

			config.SetupCommands = newCommands
			fmt.Printf("Setup command removed: %s\n", command)
			return nil
		})
	},
}

var configSetupCommandListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all setup commands",
	Long:  `List all setup commands that will be run when creating environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfig(cmd, func(config *environment.EnvironmentConfig) error {
			if len(config.SetupCommands) == 0 {
				fmt.Println("No setup commands configured")
				return nil
			}

			for i, command := range config.SetupCommands {
				fmt.Printf("%d. %s\n", i+1, command)
			}
			return nil
		})
	},
}

var configSetupCommandClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all setup commands",
	Long:  `Remove all setup commands from the environment configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.SetupCommands = []string{}
			fmt.Println("All setup commands cleared")
			return nil
		})
	},
}

// Environment variable object commands
var configEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage environment variables",
	Long:  `Manage environment variables that are set when creating environments.`,
}

var configEnvSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set an environment variable",
	Long:  `Set an environment variable to be used when creating new environments (e.g., "PATH" "/usr/local/bin:$PATH").`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.Env.Set(key, value)
			fmt.Printf("Environment variable set: %s=%s\n", key, value)
			return nil
		})
	},
}

var configEnvUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Unset an environment variable",
	Long:  `Unset an environment variable from the environment configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			if !config.Env.Unset(key) {
				return fmt.Errorf("environment variable not found: %s", key)
			}
			fmt.Printf("Environment variable unset: %s\n", key)
			return nil
		})
	},
}

var configEnvListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all environment variables",
	Long:  `List all environment variables that will be set when creating environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfig(cmd, func(config *environment.EnvironmentConfig) error {
			keys := config.Env.Keys()
			if len(keys) == 0 {
				fmt.Println("No environment variables configured")
				return nil
			}

			for i, key := range keys {
				value := config.Env.Get(key)
				fmt.Printf("%d. %s=%s\n", i+1, key, value)
			}
			return nil
		})
	},
}

var configEnvClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all environment variables",
	Long:  `Remove all environment variables from the environment configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.Env.Clear()
			fmt.Println("All environment variables cleared")
			return nil
		})
	},
}

// Secret object commands
var configSecretCmd = &cobra.Command{
	Use:   "secret",
	Short: "Manage secrets",
	Long:  `Manage secrets that are set when creating environments.`,
}

var configSecretSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a secret",
	Long:  `Set a secret to be used when creating new environments (e.g., "API_KEY" "op://vault/item/field").`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.Secrets.Set(key, value)
			fmt.Printf("Secret set: %s=%s\n", key, value)
			return nil
		})
	},
}

var configSecretUnsetCmd = &cobra.Command{
	Use:   "unset <key>",
	Short: "Unset a secret",
	Long:  `Unset a secret from the environment configuration.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			if !config.Secrets.Unset(key) {
				return fmt.Errorf("secret not found: %s", key)
			}
			fmt.Printf("Secret unset: %s\n", key)
			return nil
		})
	},
}

var configSecretListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all secrets",
	Long:  `List all secrets that will be set when creating environments.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return withConfig(cmd, func(config *environment.EnvironmentConfig) error {
			keys := config.Secrets.Keys()
			if len(keys) == 0 {
				fmt.Println("No secrets configured")
				return nil
			}

			for i, key := range keys {
				value := config.Secrets.Get(key)
				fmt.Printf("%d. %s=%s\n", i+1, key, value)
			}
			return nil
		})
	},
}

var configSecretClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all secrets",
	Long:  `Remove all secrets from the environment configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateConfig(cmd, func(config *environment.EnvironmentConfig) error {
			config.Secrets.Clear()
			fmt.Println("All secrets cleared")
			return nil
		})
	},
}

func init() {
	// Add base-image commands
	configBaseImageCmd.AddCommand(configBaseImageSetCmd)
	configBaseImageCmd.AddCommand(configBaseImageGetCmd)
	configBaseImageCmd.AddCommand(configBaseImageResetCmd)

	// Add setup-command commands
	configSetupCommandCmd.AddCommand(configSetupCommandAddCmd)
	configSetupCommandCmd.AddCommand(configSetupCommandRemoveCmd)
	configSetupCommandCmd.AddCommand(configSetupCommandListCmd)
	configSetupCommandCmd.AddCommand(configSetupCommandClearCmd)

	// Add env commands
	configEnvCmd.AddCommand(configEnvSetCmd)
	configEnvCmd.AddCommand(configEnvUnsetCmd)
	configEnvCmd.AddCommand(configEnvListCmd)
	configEnvCmd.AddCommand(configEnvClearCmd)

	// Add secret commands
	configSecretCmd.AddCommand(configSecretSetCmd)
	configSecretCmd.AddCommand(configSecretUnsetCmd)
	configSecretCmd.AddCommand(configSecretListCmd)
	configSecretCmd.AddCommand(configSecretClearCmd)

	// Add object commands to config
	configCmd.AddCommand(configBaseImageCmd)
	configCmd.AddCommand(configSetupCommandCmd)
	configCmd.AddCommand(configEnvCmd)
	configCmd.AddCommand(configSecretCmd)
	configCmd.AddCommand(configShowCmd)

	// Add agent command
	configCmd.AddCommand(agent.AgentCmd)

	// Add config command to root
	rootCmd.AddCommand(configCmd)
}
