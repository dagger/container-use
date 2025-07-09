package main

import (
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management for container-use",
	Long:  `Manage configuration for container-use including agent setup and other settings.`,
}

func init() {
	rootCmd.AddCommand(configCmd)
}
