package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Configuration structures for different agents
type GooseExtension struct {
	Name    string            `yaml:"name"`
	Type    string            `yaml:"type"`
	Enabled bool              `yaml:"enabled"`
	Cmd     string            `yaml:"cmd"`
	Args    []string          `yaml:"args"`
	Envs    map[string]string `yaml:"envs"`
}

type MCPServersConfig struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

type MCPServer struct {
	Command       string            `json:"command"`
	Args          []string          `json:"args"`
	Env           map[string]string `json:"env,omitempty"`
	Timeout       *int              `json:"timeout,omitempty"`
	Disabled      *bool             `json:"disabled,omitempty"`
	AutoApprove   []string          `json:"autoApprove,omitempty"`
	AlwaysAllow   []string          `json:"alwaysAllow,omitempty"`
	WorkingDir    *string           `json:"working_directory,omitempty"`
	StartOnLaunch *bool             `json:"start_on_launch,omitempty"`
}

type VSCodeSettings struct {
	MCP *VSCodeMCP `json:"mcp,omitempty"`
}

type VSCodeMCP struct {
	Servers map[string]VSCodeMCPServer `json:"servers"`
}

type VSCodeMCPServer struct {
	Type    string   `json:"type"`
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// TOML structures for Codex configuration
type CodexConfig struct {
	MCPServers map[string]CodexMCPServer `toml:"mcp_servers"`
}

type CodexMCPServer struct {
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
}

var configureCmd = &cobra.Command{
	Use:   "configure [agent]",
	Short: "Configure MCP server for different agents",
	Long:  `Setup the container-use MCP server according to the specified agent including Claude Code, Goose, Cursor, and others.`,
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
		case "codex":
			return configureCodex()
		case "amazonq":
			return configureAmazonQ()
		default:
			return fmt.Errorf("unknown agent: %s. Supported agents: claude, goose, cursor, codex, amazonq", args[0])
		}
	},
}

func interactiveConfiguration() error {
	fmt.Println("Select an agent to configure:")
	fmt.Println("1. Claude Code")
	fmt.Println("2. Goose")
	fmt.Println("3. Cursor")
	fmt.Println("4. OpenAI Codex")
	fmt.Println("5. Amazon Q Developer")
	fmt.Println("Enter number (1-5):")

	var choice string
	fmt.Scanln(&choice)

	switch choice {
	case "1":
		return configureClaude()
	case "2":
		return configureGoose()
	case "3":
		return configureCursor()
	case "4":
		return configureCodex()
	case "5":
		return configureAmazonQ()
	default:
		return fmt.Errorf("invalid selection: %s", choice)
	}
}

func configureClaude() error {
	fmt.Println("Configuring Claude Code...")

	// Check if claude is installed
	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("Claude Code not found. Please install it first:")
		fmt.Println("npm install -g @anthropic-ai/claude-code")
		return fmt.Errorf("claude command not found")
	}

	// Get the path to cu command
	cuPath, err := exec.LookPath("cu")
	if err != nil {
		return fmt.Errorf("cu command not found in PATH: %w", err)
	}

	// Add MCP server
	cmd := exec.Command("claude", "mcp", "add", "container-use", "--", cuPath, "stdio")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Could not automatically add MCP server: %v\n", err)
		fmt.Printf("Please run manually: claude mcp add container-use -- %s stdio\n", cuPath)
	} else {
		fmt.Println("✓ Added container-use MCP server to Claude")
	}

	// Download and append agent rules
	if err := downloadAgentRules("CLAUDE.md"); err != nil {
		fmt.Printf("Warning: Could not download agent rules: %v\n", err)
		fmt.Println("Please run manually: curl https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md >> CLAUDE.md")
	} else {
		fmt.Println("✓ Added agent rules to CLAUDE.md")
	}

	fmt.Println("\nClaude Code configuration complete!")
	fmt.Println("To use with trusted tools only:")
	fmt.Println("claude --allowedTools mcp__container-use__environment_checkpoint,mcp__container-use__environment_file_delete,mcp__container-use__environment_file_list,mcp__container-use__environment_file_read,mcp__container-use__environment_file_write,mcp__container-use__environment_open,mcp__container-use__environment_run_cmd,mcp__container-use__environment_update")
	return nil
}

func configureGoose() error {
	fmt.Println("Configuring Goose...")

	configPath := filepath.Join(os.Getenv("HOME"), ".config", "goose", "config.yaml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Get extensions map
	var extensions map[string]interface{}
	if ext, ok := config["extensions"]; ok {
		extensions = ext.(map[string]interface{})
	} else {
		extensions = make(map[string]interface{})
		config["extensions"] = extensions
	}

	// Check if container-use already exists
	if _, exists := extensions["container-use"]; exists {
		fmt.Println("✓ container-use already configured in Goose")
		return nil
	}

	// Add container-use extension
	extensions["container-use"] = map[string]interface{}{
		"name":    "container-use",
		"type":    "stdio",
		"enabled": true,
		"cmd":     "cu",
		"args":    []interface{}{"stdio"},
		"envs":    map[string]interface{}{},
	}

	// Write config back
	data, err := yaml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✓ Added container-use extension to %s\n", configPath)
	fmt.Println("Goose configuration complete!")
	return nil
}

func configureCursor() error {
	fmt.Println("Configuring Cursor...")

	// Download cursor rules
	if err := downloadFile(".cursor/rules/container-use.mdc", "https://raw.githubusercontent.com/dagger/container-use/main/rules/cursor.mdc"); err != nil {
		fmt.Printf("Warning: Could not download cursor rules: %v\n", err)
		fmt.Println("Please run manually: curl --create-dirs -o .cursor/rules/container-use.mdc https://raw.githubusercontent.com/dagger/container-use/main/rules/cursor.mdc")
	} else {
		fmt.Println("✓ Downloaded cursor rules to .cursor/rules/container-use.mdc")
	}

	fmt.Println("\nCursor configuration complete!")
	fmt.Println("Please also install the MCP server using the deeplink in the README.md") // TODO
	return nil
}

func configureCodex() error {
	fmt.Println("Configuring OpenAI Codex...")

	configPath := filepath.Join(os.Getenv("HOME"), ".codex", "config.toml")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config map[string]interface{}
	if data, err := os.ReadFile(configPath); err == nil {
		if err := toml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]interface{})
	}

	// Get mcp_servers map
	var mcpServers map[string]interface{}
	if servers, ok := config["mcp_servers"]; ok {
		mcpServers = servers.(map[string]interface{})
	} else {
		mcpServers = make(map[string]interface{})
		config["mcp_servers"] = mcpServers
	}

	// Check if container-use already exists
	if _, exists := mcpServers["container-use"]; exists {
		fmt.Println("✓ container-use already configured in Codex")
		return nil
	}

	// Add container-use server
	mcpServers["container-use"] = map[string]interface{}{
		"command": "cu",
		"args":    []interface{}{"stdio"},
		"auto_approve": []interface{}{
			"environment_open",
			"environment_update",
			"environment_checkpoint",
			"environment_run_cmd",
			"environment_file_read",
			"environment_file_write",
			"environment_file_list",
			"environment_file_delete",
		},
	}

	// Write config back
	data, err := toml.Marshal(&config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✓ Added container-use server to %s\n", configPath)
	fmt.Println("OpenAI Codex configuration complete!")
	return nil
}

func configureAmazonQ() error {
	fmt.Println("Configuring Amazon Q Developer CLI chat...")

	configPath := filepath.Join(os.Getenv("HOME"), ".aws", "amazonq", "mcp.json")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config MCPServersConfig
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	}

	// Initialize mcpServers map if nil
	if config.MCPServers == nil {
		config.MCPServers = make(map[string]MCPServer)
	}

	// Check if container-use already exists
	if _, exists := config.MCPServers["container-use"]; exists {
		fmt.Println("✓ container-use already configured in Amazon Q")
		return nil
	}

	// Add container-use server
	config.MCPServers["container-use"] = MCPServer{
		Command: "cu",
		Args:    []string{"stdio"},
		Env:     map[string]string{},
		Timeout: &[]int{60000}[0],
	}

	// Write config back
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	fmt.Printf("✓ Added container-use server to %s\n", configPath)

	// Download agent rules
	if err := downloadFile(".amazonq/rules/container-use.md", "https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md"); err != nil {
		fmt.Printf("Warning: Could not download agent rules: %v\n", err)
		fmt.Println("Please run manually: mkdir -p ./.amazonq/rules && curl https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md > .amazonq/rules/container-use.md")
	} else {
		fmt.Println("✓ Downloaded agent rules to .amazonq/rules/container-use.md")
	}

	fmt.Println("\nAmazon Q configuration complete!")
	fmt.Println("To use with trusted tools only:")
	fmt.Println("q chat --trust-tools=container_use___environment_checkpoint,container_use___environment_file_delete,container_use___environment_file_list,container_use___environment_file_read,container_use___environment_file_write,container_use___environment_open,container_use___environment_run_cmd,container_use___environment_update")
	return nil
}

// Helper functions
func downloadFile(localPath, url string) error {
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cmd := exec.Command("curl", "--create-dirs", "-o", localPath, url)
	return cmd.Run()
}

func downloadAgentRules(filename string) error {
	cmd := exec.Command("curl", "https://raw.githubusercontent.com/dagger/container-use/main/rules/agent.md")
	output, err := cmd.Output()
	if err != nil {
		return err
	}

	// Append to file if it exists, create if it doesn't
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(output)
	return err
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
