package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/mcpserver"
	"github.com/dagger/container-use/repository"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testDaggerClient *dagger.Client
	daggerOnce       sync.Once
	daggerErr        error
)

// init sets up logging for tests
func init() {
	// Only show warnings and errors in tests unless TEST_VERBOSE is set
	level := slog.LevelWarn
	if os.Getenv("TEST_VERBOSE") != "" {
		level = slog.LevelInfo
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	})))
}

// WithRepository runs a test function with an isolated repository and UserActions
func WithRepository(t *testing.T, name string, setup RepositorySetup, fn func(t *testing.T, repo *repository.Repository, user *UserActions)) {
	// Initialize Dagger (needed for environment operations)
	initializeDaggerOnce(t)

	ctx := context.Background()

	// Create isolated temp directories
	repoDir, err := os.MkdirTemp("", "cu-test-"+name+"-*")
	require.NoError(t, err, "Failed to create repo dir")

	configDir, err := os.MkdirTemp("", "cu-test-config-"+name+"-*")
	require.NoError(t, err, "Failed to create config dir")

	// Set the base path in context for all operations in this test
	ctx = context.WithValue(ctx, "container_use_base_path", configDir)

	// Initialize git repo
	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"config", "commit.gpgsign", "false"},
	}

	for _, cmd := range cmds {
		_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
		require.NoError(t, err, "Failed to run git %v", cmd)
	}

	// Run setup to populate repo
	if setup != nil {
		setup(t, repoDir)
	}

	// Open repository - it will use the isolated base path from context
	repo, err := repository.Open(ctx, repoDir)
	require.NoError(t, err, "Failed to open repository")

	// Create UserActions with extended capabilities
	user := NewUserActions(ctx, t, repo, testDaggerClient).WithDirectAccess(repoDir, configDir)

	// Cleanup
	t.Cleanup(func() {
		// Clean up any environments created during the test
		envs, _ := repo.List(ctx)
		for _, env := range envs {
			repo.Delete(ctx, env.ID)
		}

		// Remove directories
		os.RemoveAll(repoDir)
		os.RemoveAll(configDir)
	})

	// Run the test function
	fn(t, repo, user)
}

// RepositorySetup is a function that prepares a test repository
type RepositorySetup func(t *testing.T, repoDir string)

// Common repository setups
var (
	SetupPythonRepo = func(t *testing.T, repoDir string) {
		writeFile(t, repoDir, "main.py", "def main():\n    print('Hello World')\n\nif __name__ == '__main__':\n    main()\n")
		writeFile(t, repoDir, "requirements.txt", "requests==2.31.0\nnumpy==1.24.0\n")
		writeFile(t, repoDir, ".gitignore", "__pycache__/\n*.pyc\n.env\nvenv/\n")
		gitCommit(t, repoDir, "Initial Python project")
	}

	SetupPythonRepoNoGitignore = func(t *testing.T, repoDir string) {
		writeFile(t, repoDir, "main.py", "def main():\n    print('Hello World')\n\nif __name__ == '__main__':\n    main()\n")
		writeFile(t, repoDir, "requirements.txt", "requests==2.31.0\nnumpy==1.24.0\n")
		gitCommit(t, repoDir, "Initial Python project")
	}

	SetupNodeRepo = func(t *testing.T, repoDir string) {
		packageJSON := `{
  "name": "test-project",
  "version": "1.0.0",
  "main": "index.js",
  "scripts": {
    "start": "node index.js",
    "test": "jest"
  },
  "dependencies": {
    "express": "^4.18.0"
  }
}`
		writeFile(t, repoDir, "package.json", packageJSON)
		writeFile(t, repoDir, "index.js", "console.log('Hello from Node.js');\n")
		writeFile(t, repoDir, ".gitignore", "node_modules/\n.env\n")
		gitCommit(t, repoDir, "Initial Node project")
	}

	SetupEmptyRepo = func(t *testing.T, repoDir string) {
		writeFile(t, repoDir, "README.md", "# Test Project\n")
		gitCommit(t, repoDir, "Initial commit")
	}
)

// Helper functions for repository setup
func writeFile(t *testing.T, repoDir, path, content string) {
	fullPath := filepath.Join(repoDir, path)
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err, "Failed to create dir")
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(t, err, "Failed to write file")
}

func gitCommit(t *testing.T, repoDir, message string) {
	ctx := context.Background()
	_, err := repository.RunGitCommand(ctx, repoDir, "add", ".")
	require.NoError(t, err, "Failed to stage files")
	_, err = repository.RunGitCommand(ctx, repoDir, "commit", "-m", message)
	require.NoError(t, err, "Failed to commit")
}

// initializeDaggerOnce initializes Dagger client once for all tests
func initializeDaggerOnce(t *testing.T) {
	daggerOnce.Do(func() {
		if testDaggerClient != nil {
			return
		}

		ctx := context.Background()
		client, err := dagger.Connect(ctx, dagger.WithLogOutput(os.Stderr))
		if err != nil {
			daggerErr = err
			return
		}

		testDaggerClient = client
	})

	if daggerErr != nil {
		t.Skipf("Skipping test - Dagger not available: %v", daggerErr)
	}
}

// UserActions provides test helpers that mirror MCP tool behavior exactly
// These represent what a user would experience when using the MCP tools
type UserActions struct {
	t         *testing.T
	ctx       context.Context
	repo      *repository.Repository
	dag       *dagger.Client
	repoDir   string // Source directory (for direct manipulation)
	configDir string // Container-use config directory
	mcp       *MCPToolInvoker
}

func NewUserActions(ctx context.Context, t *testing.T, repo *repository.Repository, dag *dagger.Client) *UserActions {
	ua := &UserActions{
		t:    t,
		ctx:  ctx,
		repo: repo,
		dag:  dag,
	}
	ua.mcp = &MCPToolInvoker{
		t:         t,
		ctx:       ua.ctx,
		dag:       dag,
		repoDir:   "",
		configDir: "",
	}
	return ua
}

// WithDirectAccess adds direct filesystem access for edge case testing
func (u *UserActions) WithDirectAccess(repoDir, configDir string) *UserActions {
	u.repoDir = repoDir
	u.configDir = configDir
	// Update MCP invoker with paths
	u.mcp.repoDir = repoDir
	u.mcp.configDir = configDir
	return u
}

// MCPToolInvoker provides direct access to MCP tool handlers for testing
type MCPToolInvoker struct {
	t         *testing.T
	ctx       context.Context
	dag       *dagger.Client
	repoDir   string
	configDir string
}

// createMCPRequest creates an MCP CallToolRequest from a map of parameters
func createMCPRequest(toolName string, params map[string]interface{}) mcp.CallToolRequest {
	req := mcp.CallToolRequest{}
	req.Params.Name = toolName
	req.Params.Arguments = params
	return req
}

// CallTool invokes an MCP tool by name with the given parameters
func (m *MCPToolInvoker) CallTool(toolName string, params map[string]interface{}) (*mcp.CallToolResult, error) {
	// Add common environment_source parameter if not provided
	if _, ok := params["environment_source"]; !ok && m.repoDir != "" {
		params["environment_source"] = m.repoDir
	}

	// Find the tool
	tool := mcpserver.GetToolByName(toolName)
	if tool == nil {
		return nil, fmt.Errorf("tool not found: %s", toolName)
	}

	// Create request
	request := createMCPRequest(toolName, params)

	// Set up context with dagger client
	ctx := context.WithValue(m.ctx, "dagger_client", m.dag)

	// Call the handler
	return tool.Handler(ctx, request)
}


// FileWrite mirrors environment_file_write MCP tool behavior
func (u *UserActions) FileWrite(envID, targetFile, contents, explanation string) {
	result, err := u.mcp.CallTool("environment_file_write", map[string]interface{}{
		"environment_id": envID,
		"target_file":    targetFile,
		"contents":       contents,
		"explanation":    explanation,
	})
	require.NoError(u.t, err, "FileWrite should succeed")
	require.NotNil(u.t, result, "FileWrite should return a result")
}

// RunCommand mirrors environment_run_cmd MCP tool behavior
func (u *UserActions) RunCommand(envID, command, explanation string) string {
	result, err := u.mcp.CallTool("environment_run_cmd", map[string]interface{}{
		"environment_id": envID,
		"command":        command,
		"explanation":    explanation,
		"shell":          "/bin/sh",
		"background":     false,
	})
	require.NoError(u.t, err, "Run command should succeed")
	require.NotNil(u.t, result, "Run command should return a result")
	
	// Extract the output from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			// The tool appends additional info, but we want just the output
			// Split on the newline and return the first part
			output := textContent.Text
			if idx := strings.Index(output, "\n\nAny changes to the container workdir"); idx > 0 {
				output = output[:idx]
			}
			return output
		}
	}
	
	return ""
}

// CreateEnvironment mirrors environment_create MCP tool behavior
func (u *UserActions) CreateEnvironment(title, explanation string) *environment.Environment {
	result, err := u.mcp.CallTool("environment_create", map[string]interface{}{
		"title":       title,
		"explanation": explanation,
	})
	require.NoError(u.t, err, "Create environment should succeed")
	require.NotNil(u.t, result, "Create environment should return a result")
	
	// Since we need to return an actual Environment object for the tests to work,
	// we still need to get it from the repository after creation
	// The MCP tool returns JSON with the environment ID
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			var resp struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal([]byte(textContent.Text), &resp); err == nil && resp.ID != "" {
				env, err := u.repo.Get(u.ctx, u.dag, resp.ID)
				require.NoError(u.t, err, "Should be able to get created environment")
				return env
			}
		}
	}
	
	u.t.Fatal("Failed to extract environment ID from create response")
	return nil
}

// UpdateEnvironment mirrors environment_update MCP tool behavior
func (u *UserActions) UpdateEnvironment(envID, title, explanation string, config *environment.EnvironmentConfig) {
	params := map[string]interface{}{
		"environment_id":  envID,
		"title":          title,
		"explanation":    explanation,
		"instructions":   config.Instructions,
		"base_image":     config.BaseImage,
		"setup_commands": config.SetupCommands,
		"envs":           config.Env,
		"secrets":        config.Secrets,
		"workdir":        config.Workdir,
	}
	
	result, err := u.mcp.CallTool("environment_update", params)
	require.NoError(u.t, err, "UpdateEnvironment should succeed")
	require.NotNil(u.t, result, "UpdateEnvironment should return a result")
}

// FileDelete mirrors environment_file_delete MCP tool behavior
func (u *UserActions) FileDelete(envID, targetFile, explanation string) {
	result, err := u.mcp.CallTool("environment_file_delete", map[string]interface{}{
		"environment_id": envID,
		"target_file":    targetFile,
		"explanation":    explanation,
	})
	require.NoError(u.t, err, "FileDelete should succeed")
	require.NotNil(u.t, result, "FileDelete should return a result")
}

// FileRead mirrors environment_file_read MCP tool behavior (read-only, no update)
func (u *UserActions) FileRead(envID, targetFile string) string {
	result, err := u.mcp.CallTool("environment_file_read", map[string]interface{}{
		"environment_id":           envID,
		"target_file":             targetFile,
		"should_read_entire_file": true,
		"explanation":             "Reading file for test",
	})
	require.NoError(u.t, err, "FileRead should succeed")
	require.NotNil(u.t, result, "FileRead should return a result")
	
	// Extract the content from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	
	return ""
}

// FileList mirrors environment_file_list MCP tool behavior
func (u *UserActions) FileList(envID, path, explanation string) string {
	result, err := u.mcp.CallTool("environment_file_list", map[string]interface{}{
		"environment_id": envID,
		"path":           path,
		"explanation":    explanation,
	})
	require.NoError(u.t, err, "FileList should succeed")
	require.NotNil(u.t, result, "FileList should return a result")
	
	// Extract the content from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	
	return ""
}

// AddService mirrors environment_add_service MCP tool behavior
func (u *UserActions) AddService(envID, name, image, command, explanation string, ports []int, envs []string, secrets []string) *environment.Service {
	params := map[string]interface{}{
		"environment_id": envID,
		"name":           name,
		"image":          image,
		"explanation":    explanation,
	}
	
	if command != "" {
		params["command"] = command
	}
	if len(ports) > 0 {
		params["ports"] = ports
	}
	if len(envs) > 0 {
		params["envs"] = envs
	}
	if len(secrets) > 0 {
		params["secrets"] = secrets
	}
	
	result, err := u.mcp.CallTool("environment_add_service", params)
	require.NoError(u.t, err, "AddService should succeed")
	require.NotNil(u.t, result, "AddService should return a result")
	
	// Extract service information from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			// Parse the JSON response to get service info
			// The response format is: "Service added and started successfully: <json>"
			jsonStart := strings.Index(textContent.Text, "{")
			if jsonStart >= 0 {
				jsonStr := textContent.Text[jsonStart:]
				var service environment.Service
				if err := json.Unmarshal([]byte(jsonStr), &service); err == nil {
					return &service
				}
			}
		}
	}
	
	return nil
}

// Checkpoint mirrors environment_checkpoint MCP tool behavior
func (u *UserActions) Checkpoint(envID, destination, explanation string) string {
	result, err := u.mcp.CallTool("environment_checkpoint", map[string]interface{}{
		"environment_id": envID,
		"destination":    destination,
		"explanation":    explanation,
	})
	require.NoError(u.t, err, "Checkpoint should succeed")
	require.NotNil(u.t, result, "Checkpoint should return a result")
	
	// Extract the checkpoint info from the result
	if len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(mcp.TextContent); ok {
			return textContent.Text
		}
	}
	
	return ""
}

// OpenEnvironment mirrors environment_open MCP tool behavior
func (u *UserActions) OpenEnvironment(envID, explanation string) *environment.Environment {
	result, err := u.mcp.CallTool("environment_open", map[string]interface{}{
		"environment_id": envID,
		"explanation":    explanation,
	})
	require.NoError(u.t, err, "OpenEnvironment should succeed")
	require.NotNil(u.t, result, "OpenEnvironment should return a result")
	
	// Since we need to return an actual Environment object for the tests to work,
	// we get it from the repository
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Should be able to get opened environment")
	return env
}

// FileReadExpectError is for testing expected failures
func (u *UserActions) FileReadExpectError(envID, targetFile string) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	_, err = env.FileRead(u.ctx, targetFile, true, 0, 0)
	assert.Error(u.t, err, "FileRead should fail for %s", targetFile)
}

// GetEnvironment retrieves an environment by ID - mirrors how MCP tools work
// Each MCP tool call starts fresh by getting the environment from the repository
func (u *UserActions) GetEnvironment(envID string) *environment.Environment {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Should be able to get environment %s", envID)
	return env
}

// --- Direct manipulation methods for edge case testing ---

// WriteSourceFile writes directly to the source repository
func (u *UserActions) WriteSourceFile(path, content string) {
	require.NotEmpty(u.t, u.repoDir, "Need direct access for source file manipulation")
	fullPath := filepath.Join(u.repoDir, path)
	dir := filepath.Dir(fullPath)

	err := os.MkdirAll(dir, 0755)
	require.NoError(u.t, err, "Failed to create dir")

	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(u.t, err, "Failed to write source file")
}

// ReadWorktreeFile reads directly from an environment's worktree
func (u *UserActions) ReadWorktreeFile(envID, path string) string {
	// Get fresh environment to get current worktree path
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	fullPath := filepath.Join(env.Worktree, path)
	content, err := os.ReadFile(fullPath)
	require.NoError(u.t, err, "Failed to read worktree file")
	return string(content)
}

// CorruptWorktree simulates worktree corruption for recovery testing
func (u *UserActions) CorruptWorktree(envID string) {
	// Get fresh environment to get current worktree path
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	// Remove .git directory to corrupt the worktree
	gitDir := filepath.Join(env.Worktree, ".git")
	err = os.RemoveAll(gitDir)
	require.NoError(u.t, err, "Failed to corrupt worktree")
}

// GitCommand runs a git command in the source repository
func (u *UserActions) GitCommand(args ...string) string {
	require.NotEmpty(u.t, u.repoDir, "Need direct access for git commands")
	output, err := repository.RunGitCommand(u.ctx, u.repoDir, args...)
	require.NoError(u.t, err, "Git command failed: %v", args)
	return output
}
