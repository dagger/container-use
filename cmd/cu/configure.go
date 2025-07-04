package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dagger/container-use/mcpserver"
	"github.com/dagger/container-use/rules"
	"github.com/mitchellh/go-homedir"
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
		default: // TODO: auto configure based on existing local configs
			return fmt.Errorf("unknown agent: %s. Supported agents: claude, goose, cursor, codex, amazonq", args[0])
		}
	},
}

func interactiveConfiguration() error {
	selectedAgent, err := RunAgentSelector()
	if err != nil {
		// If the user quits, it's not an error, just exit gracefully.
		if err.Error() == "no agent selected" {
			return nil
		}
		return fmt.Errorf("failed to select agent: %w", err)
	}

	switch selectedAgent {
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
		return fmt.Errorf("unknown agent: %s", selectedAgent)
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

	// Add MCP server TODO add to .claude/settings.local.json
	cmd := exec.Command("claude", "mcp", "add", "container-use", "--", cuPath, "stdio")
	if err := cmd.Run(); err != nil {
		fmt.Printf("Warning: Could not automatically add MCP server: %v\n", err)
		fmt.Printf("Please run manually: claude mcp add container-use -- %s stdio\n", cuPath)
	} else {
		fmt.Println("✓ Added container-use MCP server to Claude")
	}

	// Download and append agent rules
	if err := saveFile("CLAUDE.md", rules.AgentRules); err != nil {
		return fmt.Errorf("failed to save agent rules: %v\n", err)
	} else {
		fmt.Println("✓ Added agent rules to CLAUDE.md")
	}

	fmt.Println("\nClaude Code configuration complete!")
	fmt.Println("To use with trusted tools only:") // TODO generate from the real list of tools
	fmt.Println("claude --allowedTools mcp__container-use__environment_checkpoint,mcp__container-use__environment_file_delete,mcp__container-use__environment_file_list,mcp__container-use__environment_file_read,mcp__container-use__environment_file_write,mcp__container-use__environment_open,mcp__container-use__environment_run_cmd,mcp__container-use__environment_update")
	return nil
}

func configureGoose() error {
	fmt.Println("Configuring Goose...")

	configPath, err := homedir.Expand(filepath.Join(".config", "goose", "config.yaml"))
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Get extensions map
	var extensions map[string]any
	if ext, ok := config["extensions"]; ok {
		extensions = ext.(map[string]any)
	} else {
		extensions = make(map[string]any)
		config["extensions"] = extensions
	}

	// Check if container-use already exists
	if _, exists := extensions["container-use"]; exists {
		fmt.Println("✓ container-use already configured in Goose")
		return nil
	}

	// Add container-use extension TODO make it re-entrant
	extensions["container-use"] = map[string]any{
		"name":    "container-use",
		"type":    "stdio",
		"enabled": true,
		"cmd":     "cu",
		"args":    []any{"stdio"},
		"envs":    map[string]any{},
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

	// Download cursor rules TODO embed cursor.mdc
	if err := saveFile(".cursor/rules/container-use.mdc", rules.CursorRules); err != nil {
		return fmt.Errorf("failed to save cursor rules: %v\n", err)
	} else {
		fmt.Println("✓ Downloaded cursor rules to .cursor/rules/container-use.mdc")
	}

	fmt.Println("\nCursor configuration complete!")
	fmt.Println("Please also install the MCP server using the deeplink in the README.md") // TODO generate deeplink or config
	return nil
}

func configureCodex() error {
	fmt.Println("Configuring OpenAI Codex...")

	configPath, err := homedir.Expand(filepath.Join(".codex", "config.toml"))
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Read existing config or create new
	var config map[string]any
	if data, err := os.ReadFile(configPath); err == nil {
		if err := toml.Unmarshal(data, &config); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		config = make(map[string]any)
	}

	// Get mcp_servers map
	var mcpServers map[string]any
	if servers, ok := config["mcp_servers"]; ok {
		mcpServers = servers.(map[string]any)
	} else {
		mcpServers = make(map[string]any)
		config["mcp_servers"] = mcpServers
	}

	// Check if container-use already exists
	if _, exists := mcpServers["container-use"]; exists {
		fmt.Println("✓ container-use already configured in Codex")
		return nil
	}

	// Add container-use server TODO make it re-entrant
	mcpServers["container-use"] = map[string]any{
		"command":      "cu",
		"args":         []any{"stdio"},
		"auto_approve": tools(""),
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

	configPath, err := homedir.Expand(filepath.Join(".aws", "amazonq", "mcp.json"))
	if err != nil {
		return err
	}

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

	// Add container-use server TODO make it re-entrant
	config.MCPServers["container-use"] = MCPServer{
		Command: "cu",
		Args:    []string{"stdio"},
		Env:     map[string]string{},
		Timeout: &[]int{60000}[0], // TODO: configure trusted tools
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
	if err := saveFile(".amazonq/rules/container-use.md", rules.AgentRules); err != nil {
		return fmt.Errorf("failed to save agent rules: %v\n", err)
	} else {
		fmt.Println("✓ Downloaded agent rules to .amazonq/rules/container-use.md")
	}

	// TODO: configure trusted tools
	fmt.Println("\nAmazon Q configuration complete!")
	fmt.Println("To use with trusted tools only:")
	fmt.Println("q chat --trust-tools=container_use___environment_checkpoint,container_use___environment_file_delete,container_use___environment_file_list,container_use___environment_file_read,container_use___environment_file_write,container_use___environment_open,container_use___environment_run_cmd,container_use___environment_update")
	return nil
}

// Helper functions
func saveFile(localPath, content string) error {
	dir := filepath.Dir(localPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Append to file if it exists, create if it doesn't TODO make it re-entrant
	file, err := os.OpenFile(localPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write([]byte(content))
	return err
}

func tools(prefix string) []string {
	tools := []string{}
	for _, t := range mcpserver.Tools() {
		tools = append(tools, fmt.Sprintf("%s%s", prefix, t.Definition.Name))
	}
	return tools
}

func init() {
	rootCmd.AddCommand(configureCmd)
}
