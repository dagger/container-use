package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryOpen tests the Open function which initializes a Repository
func TestRepositoryOpen(t *testing.T) {
	ctx := context.Background()

	t.Run("not_a_git_repository", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := Open(ctx, tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "you must be in a git repository")
	})

	t.Run("valid_git_repository", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := t.TempDir() // Separate dir for container-use config

		// Initialize a git repo
		_, err := RunGitCommand(ctx, tempDir, "init")
		require.NoError(t, err)

		// Set git config
		_, err = RunGitCommand(ctx, tempDir, "config", "user.email", "test@example.com")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "config", "user.name", "Test User")
		require.NoError(t, err)

		// Make initial commit
		testFile := filepath.Join(tempDir, "README.md")
		err = os.WriteFile(testFile, []byte("# Test"), 0644)
		require.NoError(t, err)

		_, err = RunGitCommand(ctx, tempDir, "add", ".")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "commit", "-m", "Initial commit")
		require.NoError(t, err)

		// Open repository with isolated base path
		repo, err := OpenWithBasePath(ctx, tempDir, configDir)
		require.NoError(t, err)
		assert.NotNil(t, repo)
		// Git resolves symlinks, so repo.userRepoPath will be the canonical path
		// This is correct behavior - we should store what git reports
		assert.NotEmpty(t, repo.userRepoPath)
		assert.DirExists(t, repo.userRepoPath)
		assert.NotEmpty(t, repo.forkRepoPath)

		// Verify fork was created
		_, err = os.Stat(repo.forkRepoPath)
		assert.NoError(t, err)

		// Verify remote was added
		remote, err := RunGitCommand(ctx, tempDir, "remote", "get-url", "container-use")
		require.NoError(t, err)
		assert.Equal(t, repo.forkRepoPath, strings.TrimSpace(remote))
	})
}

// TestIsDescendantOfCommit tests the ancestry checking logic
func TestIsDescendantOfCommit(t *testing.T) {
	ctx := context.Background()

	t.Run("descendant_relationship", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize a git repo
		_, err := RunGitCommand(ctx, tempDir, "init")
		require.NoError(t, err)

		// Set git config
		_, err = RunGitCommand(ctx, tempDir, "config", "user.email", "test@example.com")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "config", "user.name", "Test User")
		require.NoError(t, err)

		// Make initial commit
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Get the initial commit hash
		initialCommit, err := RunGitCommand(ctx, tempDir, "rev-parse", "HEAD")
		require.NoError(t, err)
		initialCommit = strings.TrimSpace(initialCommit)

		// Open repository
		repo, err := OpenWithBasePath(ctx, tempDir, configDir)
		require.NoError(t, err)

		// Create a branch and make a commit (simulating environment creation)
		_, err = RunGitCommand(ctx, tempDir, "checkout", "-b", "test-env")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Environment commit")
		require.NoError(t, err)

		// Push to container-use remote
		_, err = RunGitCommand(ctx, tempDir, "push", "container-use", "test-env:test-env")
		require.NoError(t, err)

		// Check if the environment is a descendant of the initial commit
		isDescendant := repo.isDescendantOfCommit(ctx, initialCommit, "test-env")
		assert.True(t, isDescendant, "Environment should be a descendant of initial commit")
	})

	t.Run("non_descendant_relationship", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize a git repo
		_, err := RunGitCommand(ctx, tempDir, "init")
		require.NoError(t, err)

		// Set git config
		_, err = RunGitCommand(ctx, tempDir, "config", "user.email", "test@example.com")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "config", "user.name", "Test User")
		require.NoError(t, err)

		// Make initial commit
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Open repository
		repo, err := OpenWithBasePath(ctx, tempDir, configDir)
		require.NoError(t, err)

		// Create a branch and make a commit (simulating environment creation)
		_, err = RunGitCommand(ctx, tempDir, "checkout", "-b", "test-env")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Environment commit")
		require.NoError(t, err)

		// Push to container-use remote
		_, err = RunGitCommand(ctx, tempDir, "push", "container-use", "test-env:test-env")
		require.NoError(t, err)

		// Go back to main and make a divergent commit
		defaultBranch, err := RunGitCommand(ctx, tempDir, "branch", "--show-current")
		require.NoError(t, err)
		defaultBranch = strings.TrimSpace(defaultBranch)
		if defaultBranch != "test-env" {
			_, err = RunGitCommand(ctx, tempDir, "checkout", defaultBranch)
			require.NoError(t, err)
		} else {
			// If we're still on test-env, checkout main/master
			_, err = RunGitCommand(ctx, tempDir, "checkout", "-")
			require.NoError(t, err)
		}
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Divergent commit")
		require.NoError(t, err)

		// Get the new HEAD
		newHead, err := RunGitCommand(ctx, tempDir, "rev-parse", "HEAD")
		require.NoError(t, err)
		newHead = strings.TrimSpace(newHead)

		// Check if the environment is a descendant of the new HEAD (it shouldn't be)
		isDescendant := repo.isDescendantOfCommit(ctx, newHead, "test-env")
		assert.False(t, isDescendant, "Environment should not be a descendant of divergent commit")
	})
}

// TestListDescendantEnvironments tests the filtering of environments by ancestry
func TestListDescendantEnvironments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires environment creation")
	}

	ctx := context.Background()

	t.Run("no_environments", func(t *testing.T) {
		tempDir := t.TempDir()
		configDir := t.TempDir()

		// Initialize a git repo
		_, err := RunGitCommand(ctx, tempDir, "init")
		require.NoError(t, err)

		// Set git config
		_, err = RunGitCommand(ctx, tempDir, "config", "user.email", "test@example.com")
		require.NoError(t, err)
		_, err = RunGitCommand(ctx, tempDir, "config", "user.name", "Test User")
		require.NoError(t, err)

		// Make initial commit
		_, err = RunGitCommand(ctx, tempDir, "commit", "--allow-empty", "-m", "Initial commit")
		require.NoError(t, err)

		// Get the initial commit hash
		initialCommit, err := RunGitCommand(ctx, tempDir, "rev-parse", "HEAD")
		require.NoError(t, err)
		initialCommit = strings.TrimSpace(initialCommit)

		// Open repository
		repo, err := OpenWithBasePath(ctx, tempDir, configDir)
		require.NoError(t, err)

		// List descendant environments (should be empty)
		descendants, err := repo.ListDescendantEnvironments(ctx, initialCommit)
		require.NoError(t, err)
		assert.Empty(t, descendants, "Should have no descendant environments")
	})
}