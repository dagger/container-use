package main_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MCPServerProcess represents a running container-use MCP server
type MCPServerProcess struct {
	cmd        *exec.Cmd
	client     *client.Client
	repoDir    string
	configDir  string
	serverInfo *mcp.InitializeResult
	t          *testing.T
}

// NewMCPServerProcess starts a new container-use MCP server process
func NewMCPServerProcess(t *testing.T, testName string) *MCPServerProcess {
	ctx := context.Background()

	repoDir, err := os.MkdirTemp("", fmt.Sprintf("cu-e2e-%s-repo-*", testName))
	require.NoError(t, err, "Failed to create repo dir")

	configDir, err := os.MkdirTemp("", fmt.Sprintf("cu-e2e-%s-config-*", testName))
	require.NoError(t, err, "Failed to create config dir")

	setupGitRepo(t, repoDir)

	containerUseBinary := getContainerUseBinary(t)
	cmd := exec.CommandContext(ctx, containerUseBinary, "stdio")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("CONTAINER_USE_CONFIG_DIR=%s", configDir))

	mcpClient, err := client.NewStdioMCPClient(containerUseBinary, cmd.Env, "stdio")
	require.NoError(t, err, "Failed to create MCP client")
	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    fmt.Sprintf("E2E Test Client - %s", testName),
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := mcpClient.Initialize(ctx, initRequest)
	require.NoError(t, err, "Failed to initialize MCP client")

	server := &MCPServerProcess{
		cmd:        cmd,
		client:     mcpClient,
		repoDir:    repoDir,
		configDir:  configDir,
		serverInfo: serverInfo,
		t:          t,
	}

	t.Cleanup(func() {
		server.Close()
	})

	return server
}

// Close shuts down the MCP server process and cleans up resources
func (s *MCPServerProcess) Close() {
	if s.client != nil {
		s.client.Close()
	}
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}
	os.RemoveAll(s.repoDir)
	os.RemoveAll(s.configDir)
}

// CreateEnvironment creates a new environment via MCP
func (s *MCPServerProcess) CreateEnvironment(title, explanation string) (string, error) {
	ctx := context.Background()

	request := mcp.CallToolRequest{}
	request.Params.Name = "environment_create"
	request.Params.Arguments = map[string]any{
		"environment_source": s.repoDir,
		"title":              title,
		"explanation":        explanation,
	}

	result, err := s.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Check if the tool call resulted in an error
	if result.IsError && len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return "", fmt.Errorf("environment creation failed: %s", textContent.Text)
		}
		return "", fmt.Errorf("environment creation failed with unknown error format")
	}

	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			var envResponse struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(textContent.Text), &envResponse); err != nil {
				return "", fmt.Errorf("failed to parse environment response (content: %q): %w", textContent.Text, err)
			}
			return envResponse.ID, nil
		}
	}

	return "", fmt.Errorf("no valid response content found")
}

// FileRead reads a file from an environment via MCP
func (s *MCPServerProcess) FileRead(envID, targetFile string) (string, error) {
	ctx := context.Background()

	request := mcp.CallToolRequest{}
	request.Params.Name = "environment_file_read"
	request.Params.Arguments = map[string]any{
		"environment_source":      s.repoDir,
		"environment_id":          envID,
		"target_file":             targetFile,
		"should_read_entire_file": true,
		"explanation":             "Reading file for verification",
	}

	result, err := s.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "", nil
}

// FileWrite writes a file to an environment via MCP
func (s *MCPServerProcess) FileWrite(envID, targetFile, contents, explanation string) error {
	ctx := context.Background()

	request := mcp.CallToolRequest{}
	request.Params.Name = "environment_file_write"
	request.Params.Arguments = map[string]any{
		"environment_source": s.repoDir,
		"environment_id":     envID,
		"target_file":        targetFile,
		"contents":           contents,
		"explanation":        explanation,
	}

	_, err := s.client.CallTool(ctx, request)
	return err
}

// RunCommand executes a command in an environment via MCP
func (s *MCPServerProcess) RunCommand(envID, command, explanation string) (string, error) {
	ctx := context.Background()

	request := mcp.CallToolRequest{}
	request.Params.Name = "environment_run_cmd"
	request.Params.Arguments = map[string]any{
		"environment_source": s.repoDir,
		"environment_id":     envID,
		"command":            command,
		"explanation":        explanation,
	}

	result, err := s.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "", nil
}

// createMCPServerForRepositoryTest creates an MCP server process for repository contention testing
func createMCPServerForRepositoryTest(t *testing.T, i int, repoDir, configDir string, singleTenant bool) *MCPServerProcess {
	ctx := context.Background()
	containerUseBinary := getContainerUseBinary(t)

	var cmd *exec.Cmd
	var clientArgs []string
	if singleTenant {
		cmd = exec.CommandContext(ctx, containerUseBinary, "stdio", "--single-tenant")
		clientArgs = []string{"stdio", "--single-tenant"}
	} else {
		cmd = exec.CommandContext(ctx, containerUseBinary, "stdio")
		clientArgs = []string{"stdio"}
	}

	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("CONTAINER_USE_CONFIG_DIR=%s", configDir))

	mcpClient, err := client.NewStdioMCPClient(containerUseBinary, cmd.Env, clientArgs...)
	require.NoError(t, err)

	initRequest := mcp.InitializeRequest{}
	initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initRequest.Params.ClientInfo = mcp.Implementation{
		Name:    fmt.Sprintf("Repository Test Client %d", i),
		Version: "1.0.0",
	}
	initRequest.Params.Capabilities = mcp.ClientCapabilities{}

	serverInfo, err := mcpClient.Initialize(ctx, initRequest)
	require.NoError(t, err)

	server := &MCPServerProcess{
		cmd:        cmd,
		client:     mcpClient,
		repoDir:    repoDir,
		configDir:  configDir,
		serverInfo: serverInfo,
		t:          t,
	}

	return server
}

// runRepositoryContentionTest runs the core repository contention test logic
func runRepositoryContentionTest(t *testing.T, servers []*MCPServerProcess, testPrefix string) {
	numServers := len(servers)
	var wg sync.WaitGroup
	envIDs := make([]string, numServers)
	errors := make([]error, numServers)

	for i := range numServers {
		wg.Add(1)
		go func(serverIdx int) {
			defer wg.Done()

			envID, err := servers[serverIdx].CreateEnvironment(
				fmt.Sprintf("%s Test %d", testPrefix, serverIdx),
				fmt.Sprintf("Testing %s repository contention %d", testPrefix, serverIdx),
			)
			if err != nil {
				errors[serverIdx] = fmt.Errorf("environment creation failed: %w", err)
				return
			}
			envIDs[serverIdx] = envID

			for j := range 3 {
				err := servers[serverIdx].FileWrite(
					envID,
					fmt.Sprintf("server%d_file%d.txt", serverIdx, j),
					fmt.Sprintf("Content from server %d, file %d\nTimestamp: concurrent test", serverIdx, j),
					fmt.Sprintf("Writing file %d from server %d", j, serverIdx),
				)
				if err != nil {
					errors[serverIdx] = fmt.Errorf("file write failed: %w", err)
					return
				}
			}

			content, err := servers[serverIdx].FileRead(envID, fmt.Sprintf("server%d_file0.txt", serverIdx))
			if err != nil {
				errors[serverIdx] = fmt.Errorf("file read failed: %w", err)
				return
			}
			if content == "" {
				errors[serverIdx] = fmt.Errorf("file read returned empty content")
				return
			}

			listOutput, err := servers[serverIdx].RunCommand(
				envID,
				fmt.Sprintf("ls -la server%d_*.txt | wc -l", serverIdx),
				"Count files created by this server",
			)
			if err != nil {
				errors[serverIdx] = fmt.Errorf("command execution failed: %w", err)
				return
			}

			lines := strings.Split(strings.TrimSpace(listOutput), "\n")
			if len(lines) == 0 {
				errors[serverIdx] = fmt.Errorf("command returned empty output")
				return
			}
			firstLine := lines[0]
			if firstLine != "3" {
				errors[serverIdx] = fmt.Errorf("expected 3 files, got output: %q (first line: %q)", listOutput, firstLine)
				return
			}

			_, err = servers[serverIdx].RunCommand(
				envID,
				fmt.Sprintf("echo 'Server %d completed successfully' > completion_%d.txt", serverIdx, serverIdx),
				"Mark completion",
			)
			if err != nil {
				errors[serverIdx] = fmt.Errorf("completion command failed: %w", err)
				return
			}
		}(i)
	}

	wg.Wait()

	for i := range numServers {
		assert.NoError(t, errors[i], "Server %d should handle %s repository contention successfully", i, testPrefix)
		assert.NotEmpty(t, envIDs[i], "Server %d should have environment ID", i)
	}

	envIDSet := make(map[string]bool)
	for _, envID := range envIDs {
		if envID != "" {
			assert.False(t, envIDSet[envID], "Environment ID %s should be unique", envID)
			envIDSet[envID] = true
		}
	}

	t.Logf("All %d %s servers completed successfully", numServers, testPrefix)
}

func TestRepositoryContention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	const numServers = 10
	sharedRepoDir, err := os.MkdirTemp("", "cu-e2e-shared-repo-*")
	require.NoError(t, err)
	defer os.RemoveAll(sharedRepoDir)

	setupGitRepo(t, sharedRepoDir)

	sharedConfigDir, err := os.MkdirTemp("", "cu-e2e-shared-config-*")
	require.NoError(t, err)
	defer os.RemoveAll(sharedConfigDir)

	servers := make([]*MCPServerProcess, numServers)

	for i := range numServers {
		servers[i] = createMCPServerForRepositoryTest(t, i, sharedRepoDir, sharedConfigDir, false)
		t.Cleanup(func() {
			servers[i].Close()
		})
	}

	runRepositoryContentionTest(t, servers, "Multi-Tenant")
}

func TestSingleTenantRepositoryContention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	const numServers = 10
	sharedRepoDir, err := os.MkdirTemp("", "cu-e2e-single-tenant-repo-*")
	require.NoError(t, err)
	defer os.RemoveAll(sharedRepoDir)

	setupGitRepo(t, sharedRepoDir)

	sharedConfigDir, err := os.MkdirTemp("", "cu-e2e-single-tenant-config-*")
	require.NoError(t, err)
	defer os.RemoveAll(sharedConfigDir)

	servers := make([]*MCPServerProcess, numServers)

	for i := range numServers {
		servers[i] = createMCPServerForRepositoryTest(t, i, sharedRepoDir, sharedConfigDir, true)
		t.Cleanup(func() {
			servers[i].Close()
		})
	}

	runRepositoryContentionTest(t, servers, "Single-Tenant")
}
