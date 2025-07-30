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

	// Create isolated temp directories
	repoDir, err := os.MkdirTemp("", fmt.Sprintf("cu-e2e-%s-repo-*", testName))
	require.NoError(t, err, "Failed to create repo dir")

	configDir, err := os.MkdirTemp("", fmt.Sprintf("cu-e2e-%s-config-*", testName))
	require.NoError(t, err, "Failed to create config dir")

	// Initialize git repo
	setupGitRepo(t, repoDir)

	// Start container-use stdio process
	containerUseBinary := getContainerUseBinary(t)
	cmd := exec.CommandContext(ctx, containerUseBinary, "stdio")
	cmd.Dir = repoDir
	cmd.Env = append(os.Environ(), fmt.Sprintf("CONTAINER_USE_CONFIG_DIR=%s", configDir))

	// Create MCP client that communicates via stdio
	mcpClient, err := client.NewStdioMCPClient(containerUseBinary, cmd.Env, "stdio")
	require.NoError(t, err, "Failed to create MCP client")

	// Initialize the MCP connection
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

	// Setup cleanup
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
		"title":             title,
		"explanation":       explanation,
	}

	result, err := s.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Parse the environment ID from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			var envResponse struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(textContent.Text), &envResponse); err != nil {
				return "", fmt.Errorf("failed to parse environment response: %w", err)
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
		"environment_source":        s.repoDir,
		"environment_id":            envID,
		"target_file":               targetFile,
		"should_read_entire_file":   true,
		"explanation":               "Reading file for verification",
	}

	result, err := s.client.CallTool(ctx, request)
	if err != nil {
		return "", err
	}

	// Extract file content from result
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

	// Extract command output from result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text, nil
		}
	}

	return "", nil
}

// Test Cases



// Test Cases

func TestParallelEnvironmentCreation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	const numServers = 3
	const envsPerServer = 2

	// Start multiple MCP servers
	servers := make([]*MCPServerProcess, numServers)
	for i := range numServers {
		servers[i] = NewMCPServerProcess(t, fmt.Sprintf("parallel-create-%d", i))
	}

	// Create environments in parallel
	var wg sync.WaitGroup
	envIDs := make([][]string, numServers)
	errors := make([][]error, numServers)

	for i := range numServers {
		envIDs[i] = make([]string, envsPerServer)
		errors[i] = make([]error, envsPerServer)
		
		wg.Add(1)
		go func(serverIdx int) {
			defer wg.Done()
			server := servers[serverIdx]
			
			for j := range envsPerServer {
				envID, err := server.CreateEnvironment(
					fmt.Sprintf("Server %d Env %d", serverIdx, j),
					fmt.Sprintf("Creating environment %d on server %d", j, serverIdx),
				)
				envIDs[serverIdx][j] = envID
				errors[serverIdx][j] = err
			}
		}(i)
	}

	wg.Wait()

	// Verify all environments were created successfully
	allEnvIDs := make(map[string]bool)
	for i := range numServers {
		for j := range envsPerServer {
			assert.NoError(t, errors[i][j], "Server %d should create environment %d", i, j)
			assert.NotEmpty(t, envIDs[i][j], "Environment ID should not be empty")
			
			// Verify environment IDs are unique across all servers
			assert.False(t, allEnvIDs[envIDs[i][j]], "Environment ID %s should be unique", envIDs[i][j])
			allEnvIDs[envIDs[i][j]] = true
		}
	}
}

func TestParallelWorkflowWithLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	const numServers = 2
	servers := make([]*MCPServerProcess, numServers)
	envIDs := make([]string, numServers)

	// Start servers and create environments
	for i := range numServers {
		servers[i] = NewMCPServerProcess(t, fmt.Sprintf("parallel-workflow-%d", i))
		
		envID, err := servers[i].CreateEnvironment(
			fmt.Sprintf("Workflow Test %d", i),
			fmt.Sprintf("Environment for workflow testing %d", i),
		)
		require.NoError(t, err)
		envIDs[i] = envID
		t.Logf("Server %d created environment: %s", i, envID)
	}

	// Perform parallel workflow: A writes files, B reads and processes them
	var wg sync.WaitGroup
	results := make([]map[string]string, numServers)

	for i := range numServers {
		results[i] = make(map[string]string)
		wg.Add(1)
		
		go func(serverIdx int) {
			defer wg.Done()
			server := servers[serverIdx]
			envID := envIDs[serverIdx]
			
			// Step A: Write initial files
			for fileIdx := range 3 {
				fileName := fmt.Sprintf("data_%d.txt", fileIdx)
				content := fmt.Sprintf("Server %d - File %d - Initial content", serverIdx, fileIdx)
				
				err := server.FileWrite(
					envID,
					fileName,
					content,
					fmt.Sprintf("Writing initial data file %d", fileIdx),
				)
				if err != nil {
					results[serverIdx]["error"] = fmt.Sprintf("FileWrite failed: %v", err)
					return
				}
			}
			
			// Step B: Process files (read, modify, write back)
			for fileIdx := range 3 {
				fileName := fmt.Sprintf("data_%d.txt", fileIdx)
				
				// Read the file
				readResult, err := server.FileRead(envID, fileName)
				if err != nil {
					results[serverIdx]["error"] = fmt.Sprintf("FileRead failed: %v", err)
					return
				}
				
				// Process the content
				processedContent := readResult + " - PROCESSED"
				
				// Write back processed content
				err = server.FileWrite(
					envID,
					fileName,
					processedContent,
					fmt.Sprintf("Processing and updating file %d", fileIdx),
				)
				if err != nil {
					results[serverIdx]["error"] = fmt.Sprintf("FileWrite (process) failed: %v", err)
					return
				}
			}
			
			// Step C: Run commands to verify the work
			listOutput, err := server.RunCommand(
				envID,
				"ls -la *.txt",
				"List all text files to verify creation",
			)
			if err != nil {
				results[serverIdx]["error"] = fmt.Sprintf("RunCommand (ls) failed: %v", err)
				return
			}
			results[serverIdx]["ls_output"] = listOutput
			
			// Count processed files
			countOutput, err := server.RunCommand(
				envID,
				"grep -c PROCESSED *.txt",
				"Count how many files were processed",
			)
			if err != nil {
				results[serverIdx]["error"] = fmt.Sprintf("RunCommand (grep) failed: %v", err)
				return
			}
			results[serverIdx]["count_output"] = countOutput
			
			// Final verification - read one file to confirm processing
			finalContent, err := server.FileRead(envID, "data_0.txt")
			if err != nil {
				results[serverIdx]["error"] = fmt.Sprintf("Final FileRead failed: %v", err)
				return
			}
			results[serverIdx]["final_content"] = finalContent
			results[serverIdx]["success"] = "true"
		}(i)
	}

	wg.Wait()

	// Verify all workflows completed successfully
	for i := range numServers {
		result := results[i]
		
		// Check for errors
		if errorMsg, hasError := result["error"]; hasError {
			t.Errorf("Server %d workflow failed: %s", i, errorMsg)
			continue
		}
		
		// Verify success
		assert.Equal(t, "true", result["success"], "Server %d should complete workflow successfully", i)
		
		// Verify file listing shows our files
		lsOutput := result["ls_output"]
		assert.Contains(t, lsOutput, "data_0.txt", "Server %d should have data_0.txt", i)
		assert.Contains(t, lsOutput, "data_1.txt", "Server %d should have data_1.txt", i)
		assert.Contains(t, lsOutput, "data_2.txt", "Server %d should have data_2.txt", i)
		
		// Verify processing count (should be 3 files processed)
		countOutput := result["count_output"]
		// grep -c outputs "filename:count" for each file, so we should see 3 lines with ":1"
		lines := strings.Split(strings.TrimSpace(countOutput), "\n")
		processedLines := 0
		for _, line := range lines {
			if strings.Contains(line, ":1") {
				processedLines++
			}
		}
		assert.Equal(t, 3, processedLines, "Server %d should have processed 3 files", i)
		
		// Verify final content shows processing
		finalContent := result["final_content"]
		assert.Contains(t, finalContent, "PROCESSED", "Server %d final content should show processing", i)
		assert.Contains(t, finalContent, fmt.Sprintf("Server %d", i), "Server %d final content should contain server ID", i)
		
		t.Logf("Server %d workflow completed successfully:", i)
		t.Logf("  Environment: %s", envIDs[i])
		t.Logf("  Files created: %s", lsOutput)
		t.Logf("  Files processed: %s", countOutput)
		t.Logf("  Final content sample: %s", finalContent)
	}

	// Verify environments are isolated (each should have different content)
	if len(results) >= 2 && results[0]["success"] == "true" && results[1]["success"] == "true" {
		content0 := results[0]["final_content"]
		content1 := results[1]["final_content"]
		assert.NotEqual(t, content0, content1, "Different servers should have different content (isolation test)")
	}
}

func TestParallelFileOperations(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	const numServers = 2
	servers := make([]*MCPServerProcess, numServers)
	envIDs := make([]string, numServers)

	// Start servers and create one environment each
	for i := range numServers {
		servers[i] = NewMCPServerProcess(t, fmt.Sprintf("parallel-files-%d", i))
		
		envID, err := servers[i].CreateEnvironment(
			fmt.Sprintf("File Test Env %d", i),
			fmt.Sprintf("Environment for file operations test %d", i),
		)
		require.NoError(t, err)
		envIDs[i] = envID
	}

	// Perform file operations in parallel
	var wg sync.WaitGroup
	results := make([]error, numServers)

	for i := range numServers {
		wg.Add(1)
		go func(serverIdx int) {
			defer wg.Done()
			server := servers[serverIdx]
			envID := envIDs[serverIdx]
			
			// Write multiple files
			for j := range 3 {
				err := server.FileWrite(
					envID,
					fmt.Sprintf("file%d.txt", j),
					fmt.Sprintf("Content from server %d, file %d", serverIdx, j),
					fmt.Sprintf("Writing file %d", j),
				)
				if err != nil {
					results[serverIdx] = err
					return
				}
			}
			
			// Run a command
			_, err := server.RunCommand(
				envID,
				"ls -la",
				"List files",
			)
			results[serverIdx] = err
		}(i)
	}

	wg.Wait()

	// Verify all operations succeeded
	for i := range numServers {
		assert.NoError(t, results[i], "Server %d operations should succeed", i)
	}
}

func TestResourceContention(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test")
	}

	// This test uses the same repository directory for multiple servers
	// to test resource contention scenarios
	
	// Create shared repository
	sharedRepoDir, err := os.MkdirTemp("", "cu-e2e-shared-repo-*")
	require.NoError(t, err)
	defer os.RemoveAll(sharedRepoDir)
	
	setupGitRepo(t, sharedRepoDir)

	// Start multiple servers pointing to the same repo
	const numServers = 2
	servers := make([]*MCPServerProcess, numServers)
	
	for i := range numServers {
		// Create separate config dirs but use same repo
		configDir, err := os.MkdirTemp("", fmt.Sprintf("cu-e2e-shared-config-%d-*", i))
		require.NoError(t, err)
		defer os.RemoveAll(configDir)
		
		ctx := context.Background()
		containerUseBinary := getContainerUseBinary(t)
		cmd := exec.CommandContext(ctx, containerUseBinary, "stdio")
		cmd.Dir = sharedRepoDir
		cmd.Env = append(os.Environ(), fmt.Sprintf("CONTAINER_USE_CONFIG_DIR=%s", configDir))

		mcpClient, err := client.NewStdioMCPClient(containerUseBinary, cmd.Env, "stdio")
		require.NoError(t, err)

		initRequest := mcp.InitializeRequest{}
		initRequest.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
		initRequest.Params.ClientInfo = mcp.Implementation{
			Name:    fmt.Sprintf("Contention Test Client %d", i),
			Version: "1.0.0",
		}
		initRequest.Params.Capabilities = mcp.ClientCapabilities{}

		serverInfo, err := mcpClient.Initialize(ctx, initRequest)
		require.NoError(t, err)

		servers[i] = &MCPServerProcess{
			cmd:        cmd,
			client:     mcpClient,
			repoDir:    sharedRepoDir,
			configDir:  configDir,
			serverInfo: serverInfo,
			t:          t,
		}
		
		t.Cleanup(func() {
			servers[i].Close()
		})
	}

	// Try to create environments simultaneously on the same repo
	var wg sync.WaitGroup
	envIDs := make([]string, numServers)
	errors := make([]error, numServers)

	for i := range numServers {
		wg.Add(1)
		go func(serverIdx int) {
			defer wg.Done()
			
			envID, err := servers[serverIdx].CreateEnvironment(
				fmt.Sprintf("Contention Test %d", serverIdx),
				fmt.Sprintf("Testing resource contention %d", serverIdx),
			)
			envIDs[serverIdx] = envID
			errors[serverIdx] = err
		}(i)
	}

	wg.Wait()

	// Both should succeed (or at least handle contention gracefully)
	successCount := 0
	for i := range numServers {
		if errors[i] == nil {
			successCount++
			assert.NotEmpty(t, envIDs[i], "Successful environment should have ID")
		} else {
			t.Logf("Server %d failed (expected in contention): %v", i, errors[i])
		}
	}

	// At least one should succeed
	assert.Greater(t, successCount, 0, "At least one server should handle contention successfully")
}