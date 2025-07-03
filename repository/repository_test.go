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
