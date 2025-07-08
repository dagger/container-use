package integration

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/mcpserver"
	"github.com/dagger/container-use/repository"
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
//
// WARNING: Tests using WithRepository MUST NOT call t.Parallel() because
// SetTestConfigPath modifies a global variable. Parallel tests will race
// and cause unpredictable failures.
func WithRepository(t *testing.T, name string, setup RepositorySetup, fn func(t *testing.T, repo *repository.Repository, user *UserActions)) {
	// Initialize Dagger (needed for environment operations)
	initializeDaggerOnce(t)

	ctx := context.Background()

	// Create isolated temp directories
	repoDir, err := os.MkdirTemp("", "cu-test-"+name+"-*")
	require.NoError(t, err, "Failed to create repo dir")

	configDir, err := os.MkdirTemp("", "cu-test-config-"+name+"-*")
	require.NoError(t, err, "Failed to create config dir")

	// Override the global config path for this test
	cleanup := repository.SetTestConfigPath(configDir)
	t.Cleanup(cleanup)

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
		client, err := dagger.Connect(ctx)
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
}

func NewUserActions(ctx context.Context, t *testing.T, repo *repository.Repository, dag *dagger.Client) *UserActions {
	return &UserActions{
		t:    t,
		ctx:  ctx,
		repo: repo,
		dag:  dag,
	}
}

// WithDirectAccess adds direct filesystem access for edge case testing
func (u *UserActions) WithDirectAccess(repoDir, configDir string) *UserActions {
	u.repoDir = repoDir
	u.configDir = configDir
	return u
}

// FileWrite mirrors environment_file_write MCP tool behavior
func (u *UserActions) FileWrite(envID, targetFile, contents, explanation string) {
	err := mcpserver.WriteEnvironmentFile(u.ctx, u.dag, u.repoDir, envID, targetFile, contents, explanation)
	require.NoError(u.t, err, "FileWrite should succeed")
}

// RunCommand mirrors environment_run_cmd MCP tool behavior
func (u *UserActions) RunCommand(envID, command, explanation string) string {
	result, err := mcpserver.RunEnvironmentCommand(u.ctx, u.dag, u.repoDir, envID, command, "/bin/sh", explanation, false, false, nil)
	require.NoError(u.t, err, "Run command should succeed")
	require.NotNil(u.t, result, "Run command should return a result")

	// For non-background commands, result is a string
	output, ok := result.(string)
	require.True(u.t, ok, "Run command should return string output")
	return output
}

// CreateEnvironment mirrors environment_create MCP tool behavior
func (u *UserActions) CreateEnvironment(title, explanation string) *environment.Environment {
	_, env, err := mcpserver.CreateEnvironment(u.ctx, u.dag, u.repoDir, title, explanation)
	require.NoError(u.t, err, "Create environment should succeed")
	require.NotNil(u.t, env, "Create environment should return an environment")
	return env
}

// UpdateEnvironment mirrors environment_update MCP tool behavior
func (u *UserActions) UpdateEnvironment(envID, title, explanation string, config *environment.EnvironmentConfig) {
	_, err := mcpserver.UpdateEnvironment(u.ctx, u.dag, u.repoDir, envID, title, config.Instructions, config.BaseImage, explanation, config.SetupCommands, config.Env, config.Secrets)
	require.NoError(u.t, err, "UpdateEnvironment should succeed")
}

// FileDelete mirrors environment_file_delete MCP tool behavior
func (u *UserActions) FileDelete(envID, targetFile, explanation string) {
	err := mcpserver.DeleteEnvironmentFile(u.ctx, u.dag, u.repoDir, envID, targetFile, explanation)
	require.NoError(u.t, err, "FileDelete should succeed")
}

// FileRead mirrors environment_file_read MCP tool behavior (read-only, no update)
func (u *UserActions) FileRead(envID, targetFile string) string {
	content, err := mcpserver.ReadEnvironmentFile(u.ctx, u.dag, u.repoDir, envID, targetFile, true, 0, 0)
	require.NoError(u.t, err, "FileRead should succeed")
	return content
}

// FileReadExpectError is for testing expected failures
func (u *UserActions) FileReadExpectError(envID, targetFile string) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	_, err = env.FileRead(u.ctx, targetFile, true, 0, 0)
	assert.Error(u.t, err, "FileRead should fail for %s", targetFile)
}

// FileList mirrors environment_file_list MCP tool behavior
func (u *UserActions) FileList(envID, path string) string {
	content, err := mcpserver.ListEnvironmentFiles(u.ctx, u.dag, u.repoDir, envID, path)
	require.NoError(u.t, err, "FileList should succeed")
	return content
}

// GetEnvironment retrieves an environment by ID - mirrors how MCP tools work
// Each MCP tool call starts fresh by getting the environment from the repository
func (u *UserActions) GetEnvironment(envID string) *environment.Environment {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Should be able to get environment %s", envID)
	return env
}

// OpenEnvironment mirrors environment_open MCP tool behavior
func (u *UserActions) OpenEnvironment(envID string) *environment.Environment {
	env, err := mcpserver.GetEnvironmentFromSource(u.ctx, u.dag, u.repoDir, envID)
	require.NoError(u.t, err, "OpenEnvironment should succeed")
	require.NotNil(u.t, env, "OpenEnvironment should return an environment")
	return env
}

// AddService mirrors environment_add_service MCP tool behavior
func (u *UserActions) AddService(envID, name, image, command, explanation string, ports []int, envs []string, secrets []string) *environment.Service {
	service, err := mcpserver.AddEnvironmentService(u.ctx, u.dag, u.repoDir, envID, name, image, command, explanation, ports, envs, secrets)
	require.NoError(u.t, err, "AddService should succeed")
	require.NotNil(u.t, service, "AddService should return a service")
	return service
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

// WorktreePath returns the worktree path for an environment, handling errors
func (u *UserActions) WorktreePath(envID string) string {
	worktreePath, err := u.repo.WorktreePath(envID)
	require.NoError(u.t, err, "Failed to get worktree path for environment %s", envID)
	return worktreePath
}

// ReadWorktreeFile reads directly from an environment's worktree
func (u *UserActions) ReadWorktreeFile(envID, path string) string {
	worktreePath := u.WorktreePath(envID)
	fullPath := filepath.Join(worktreePath, path)
	content, err := os.ReadFile(fullPath)
	require.NoError(u.t, err, "Failed to read worktree file")
	return string(content)
}

// CorruptWorktree simulates worktree corruption for recovery testing
func (u *UserActions) CorruptWorktree(envID string) {
	worktreePath := u.WorktreePath(envID)

	// Remove .git directory to corrupt the worktree
	gitDir := filepath.Join(worktreePath, ".git")
	err := os.RemoveAll(gitDir)
	require.NoError(u.t, err, "Failed to corrupt worktree")
}

// GitCommand runs a git command in the source repository
func (u *UserActions) GitCommand(args ...string) string {
	require.NotEmpty(u.t, u.repoDir, "Need direct access for git commands")
	output, err := repository.RunGitCommand(u.ctx, u.repoDir, args...)
	require.NoError(u.t, err, "Git command failed: %v", args)
	return output
}
