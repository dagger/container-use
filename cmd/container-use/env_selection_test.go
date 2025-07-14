package main

import (
	"context"
	"os"
	"strings"
	"testing"

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
}

func TestIsDescendantOfHead(t *testing.T) {
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
		currentHead = strings.TrimSpace(currentHead)

		// Get the default branch name
		defaultBranch, err := repository.RunGitCommand(ctx, repoDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = strings.TrimSpace(defaultBranch)

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
		isDescendant := isDescendantOfHead(ctx, repo, currentHead, "test-env")
		assert.True(t, isDescendant)
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
		defaultBranch = strings.TrimSpace(defaultBranch)

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
		currentHead = strings.TrimSpace(currentHead)

		// Test that current HEAD is not parent of environment
		isDescendant := isDescendantOfHead(ctx, repo, currentHead, "test-env")
		assert.False(t, isDescendant)
	})
}

// Helper functions

func writeTestFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}