package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironmentID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("WithProvidedArgs", func(t *testing.T) {
		// When args are provided, should return the first arg directly
		ctx := context.Background()
		args := []string{"test-env"}

		// Don't need a real repository for this test
		envID, err := resolveEnvironmentID(ctx, nil, args)
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	t.Run("NoEnvironments", func(t *testing.T) {
		// Create a temporary repository with no environments
		ctx := context.Background()
		repoDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize git repo
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
			require.NoError(t, err)
		}

		// Create initial commit
		_, err := repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		repo, err := repository.OpenWithBasePath(ctx, repoDir, configDir)
		require.NoError(t, err)

		// Should return error when no environments exist
		_, err = resolveEnvironmentID(ctx, repo, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no environments found")
	})

	t.Run("SingleMatchingEnvironment", func(t *testing.T) {
		// Create a temporary repository with a single environment
		ctx := context.Background()
		repoDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize git repo
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
			require.NoError(t, err)
		}

		// Create initial commit
		_, err := repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Get the default branch name
		defaultBranch, err := repository.RunGitCommand(ctx, repoDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = defaultBranch[:len(defaultBranch)-1] // Remove newline

		repo, err := repository.OpenWithBasePath(ctx, repoDir, configDir)
		require.NoError(t, err)

		// Create mock environment by simulating the git branch structure
		// In a real scenario, we would create an environment through the repository API
		// For this test, we'll create a branch in the container-use remote
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", "-b", "test-branch")
		require.NoError(t, err)

		// Write a simple file and commit
		testFile := repoDir + "/test.txt"
		err = writeTestFile(testFile, "test content")
		require.NoError(t, err)

		_, err = repository.RunGitCommand(ctx, repoDir, "add", "test.txt")
		require.NoError(t, err)

		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "-m", "Add test file")
		require.NoError(t, err)

		// Switch back to default branch
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", defaultBranch)
		require.NoError(t, err)

		// Push test-branch to container-use remote (simulating environment creation)
		_, err = repository.RunGitCommand(ctx, repoDir, "push", "container-use", "test-branch:test-env")
		require.NoError(t, err)

		// Create environment state file in the worktree
		worktreePath := configDir + "/worktrees/test-env"
		err = createMockEnvironmentState(worktreePath, "test-env", "Test Environment")
		require.NoError(t, err)

		// Test that single environment is auto-selected
		envID, err := resolveEnvironmentID(ctx, repo, []string{})
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	t.Run("NoMatchingEnvironments", func(t *testing.T) {
		// Create a temporary repository with environments that don't match the current HEAD
		ctx := context.Background()
		repoDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize git repo
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
			require.NoError(t, err)
		}

		// Create initial commit
		_, err := repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Get the default branch name
		defaultBranch, err := repository.RunGitCommand(ctx, repoDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = defaultBranch[:len(defaultBranch)-1] // Remove newline

		repo, err := repository.OpenWithBasePath(ctx, repoDir, configDir)
		require.NoError(t, err)

		// Create a branch that diverges from main
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", "-b", "feature-branch")
		require.NoError(t, err)

		// Make a commit on feature-branch
		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Feature commit")
		require.NoError(t, err)

		// Switch back to default branch
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", defaultBranch)
		require.NoError(t, err)

		// Make a different commit on default branch (creating divergence)
		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Default branch commit")
		require.NoError(t, err)

		// Push feature-branch to container-use remote as an environment
		_, err = repository.RunGitCommand(ctx, repoDir, "push", "container-use", "feature-branch:test-env")
		require.NoError(t, err)

		// Create environment state file
		worktreePath := configDir + "/worktrees/test-env"
		err = createMockEnvironmentState(worktreePath, "test-env", "Test Environment")
		require.NoError(t, err)

		// Test that no matching environments are found
		_, err = resolveEnvironmentID(ctx, repo, []string{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no environments found that are descendants of the current HEAD")
	})
}

func TestIsParentOfEnvironment(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("CurrentHeadIsParent", func(t *testing.T) {
		ctx := context.Background()
		repoDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize git repo
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
			require.NoError(t, err)
		}

		// Create initial commit
		_, err := repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		currentHead, err := repository.RunGitCommand(ctx, repoDir, "rev-parse", "HEAD")
		require.NoError(t, err)
		currentHead = currentHead[:len(currentHead)-1] // Remove newline

		// Get the default branch name
		defaultBranch, err := repository.RunGitCommand(ctx, repoDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = defaultBranch[:len(defaultBranch)-1] // Remove newline

		repo, err := repository.OpenWithBasePath(ctx, repoDir, configDir)
		require.NoError(t, err)

		// Create a branch from current HEAD
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", "-b", "child-branch")
		require.NoError(t, err)

		// Make a commit on child-branch
		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Child commit")
		require.NoError(t, err)

		// Push child-branch to container-use remote
		_, err = repository.RunGitCommand(ctx, repoDir, "push", "container-use", "child-branch:test-env")
		require.NoError(t, err)

		// Switch back to default branch
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", defaultBranch)
		require.NoError(t, err)

		// Test that current HEAD is parent of environment
		isParent := isParentOfEnvironment(ctx, repo, currentHead, "test-env")
		assert.True(t, isParent)
	})

	t.Run("CurrentHeadIsNotParent", func(t *testing.T) {
		ctx := context.Background()
		repoDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize git repo
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir, cmd...)
			require.NoError(t, err)
		}

		// Create initial commit
		_, err := repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Get the default branch name
		defaultBranch, err := repository.RunGitCommand(ctx, repoDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = defaultBranch[:len(defaultBranch)-1] // Remove newline

		repo, err := repository.OpenWithBasePath(ctx, repoDir, configDir)
		require.NoError(t, err)

		// Create a branch and make a commit
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", "-b", "branch1")
		require.NoError(t, err)

		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Branch1 commit")
		require.NoError(t, err)

		// Push branch1 to container-use remote
		_, err = repository.RunGitCommand(ctx, repoDir, "push", "container-use", "branch1:test-env")
		require.NoError(t, err)

		// Switch back to default branch and make a different commit
		_, err = repository.RunGitCommand(ctx, repoDir, "checkout", defaultBranch)
		require.NoError(t, err)

		_, err = repository.RunGitCommand(ctx, repoDir, "commit", "--allow-empty", "-m", "Default branch commit")
		require.NoError(t, err)

		currentHead, err := repository.RunGitCommand(ctx, repoDir, "rev-parse", "HEAD")
		require.NoError(t, err)
		currentHead = currentHead[:len(currentHead)-1] // Remove newline

		// Test that current HEAD is not parent of environment
		isParent := isParentOfEnvironment(ctx, repo, currentHead, "test-env")
		assert.False(t, isParent)
	})
}

// Helper functions

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

func createMockEnvironmentState(worktreePath, envID, title string) error {
	// Create worktree directory
	err := os.MkdirAll(worktreePath, 0755)
	if err != nil {
		return err
	}

	// Create a mock environment state
	state := &environment.State{
		Container: "mock-container",
		Title:     title,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	stateBytes, err := state.Marshal()
	if err != nil {
		return err
	}

	// Write state to git notes (simulated)
	// In a real scenario, this would be handled by the git notes system
	// For testing, we'll just create a temporary file
	return os.WriteFile(worktreePath+"/state.json", stateBytes, 0644)
}
