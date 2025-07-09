package main

import (
	"github.com/dagger/container-use/cmd/container-use/agent"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management for container-use",
	Long:  `Manage configuration for container-use including agent setup and other settings.`,
}

func init() {
	configCmd.AddCommand(agent.AgentCmd)

	rootCmd.AddCommand(configCmd)
}
