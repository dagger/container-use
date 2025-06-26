package main

import (
	"fmt"
	"os"
	"encoding/json"
	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure [agent]",
	Short: "Configure MCP server for different agents",
	Long:  `Setup the container-use MCP server according to the specified agent including Claude Code, Goose, Cursor, and others.`
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return interactiveConfiguration()
		}
		switch args[0] {
		case "claude":
			return configureClaude()
		case "goose":
			return configureGoose()
		case "cursor":
			return configureCursor()
		case "warp":
			return configureWarp()
		default:
			return fmt.Errorf("Unknown agent: %s", args[0])
		}
	},
}

func interactiveConfiguration() error {
	// Implement interactive agent selection
	return nil
}

func configureClaude() error {
	// Implement Claude configuration
	cmd := "npm install -g @anthropic-ai/claude-code; cd /path/to/repository; claude mcp add container-use -- <full path to cu command> stdio; curl https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md >> CLAUDE.md"
	// Execute commands as needed
	return nil
}

func configureGoose() error {
	// Implement Goose configuration
	configPath := os.Getenv("HOME") + "/.config/goose/config.yaml"
	// ... existing implementation ...
	return nil
}

func configureCursor() error {
	// Implement Cursor configuration
	curlCmd := "curl --create-dirs -o .cursor/rules/container-use.mdc 'https://raw.githubusercontent.com/dagger/container-use/main/rules/cursor.mdc'"
	// Execute commands as needed
	return nil
}

func configureWarp() error {
	// Implement Warp configuration
	// ... existing implementation ...
	return nil
}

func init() {
	rootCmd.AddCommand(configureCmd)
}