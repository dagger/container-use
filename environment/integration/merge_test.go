package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryMerge tests merging an environment into the main branch
func TestRepositoryMerge(t *testing.T) {
	WithRepository(t, "repository-merge", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add some content
		env := user.CreateEnvironment("Test Merge", "Testing repository merge")
		user.FileWrite(env.ID, "merge-test.txt", "content from environment", "Add merge test file")
		user.FileWrite(env.ID, "config.json", `{"version": "1.0"}`, "Add config file")

		// Get initial branch
		initialBranch, err := repository.RunGitCommand(ctx, repo.SourcePath(), "branch", "--show-current")
		require.NoError(t, err)
		initialBranch = strings.TrimSpace(initialBranch)

		// Merge the environment (without squash)
		var mergeOutput bytes.Buffer
		err = repo.Merge(ctx, env.ID, &mergeOutput)
		require.NoError(t, err, "Merge should succeed: %s", mergeOutput.String())

		// Verify we're still on the initial branch
		currentBranch, err := repository.RunGitCommand(ctx, repo.SourcePath(), "branch", "--show-current")
		require.NoError(t, err)
		assert.Equal(t, initialBranch, strings.TrimSpace(currentBranch))

		// Verify the files were merged into the working directory
		mergeTestPath := filepath.Join(repo.SourcePath(), "merge-test.txt")
		content, err := os.ReadFile(mergeTestPath)
		require.NoError(t, err)
		assert.Equal(t, "content from environment", string(content))

		configPath := filepath.Join(repo.SourcePath(), "config.json")
		configContent, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, `{"version": "1.0"}`, string(configContent))

		// Verify commit history includes the environment changes
		log, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--oneline", "-10")
		require.NoError(t, err)
		// The merge might be fast-forward, so check for either merge commit or environment commits
		assert.True(t,
			strings.Contains(log, "Merge environment "+env.ID) ||
				(strings.Contains(log, "Add merge test file") && strings.Contains(log, "Add config file")),
			"Log should contain merge commit or environment commits: %s", log)
	})
}

// TestRepositoryMergeNonExistent tests merging a non-existent environment
func TestRepositoryMergeNonExistent(t *testing.T) {
	WithRepository(t, "repository-merge-nonexistent", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Try to merge non-existent environment
		var mergeOutput bytes.Buffer
		err := repo.Merge(ctx, "non-existent-env", &mergeOutput)
		assert.Error(t, err, "Merging non-existent environment should fail")
		assert.Contains(t, err.Error(), "not found")
	})
}

// TestRepositoryMergeWithConflicts tests merge behavior when there are conflicts
func TestRepositoryMergeWithConflicts(t *testing.T) {
	WithRepository(t, "repository-merge-conflicts", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and modify the same file
		env := user.CreateEnvironment("Test Merge Conflicts", "Testing merge conflicts")
		user.FileWrite(env.ID, "conflict.txt", "environment branch content", "Modify conflict file")

		conflictFile := filepath.Join(repo.SourcePath(), "conflict.txt")
		err := os.WriteFile(conflictFile, []byte("main branch content"), 0644)
		require.NoError(t, err)

		_, err = repository.RunGitCommand(ctx, repo.SourcePath(), "add", "conflict.txt")
		require.NoError(t, err)
		_, err = repository.RunGitCommand(ctx, repo.SourcePath(), "commit", "-m", "Add conflict file in main")
		require.NoError(t, err)

		// Try to merge - this should either succeed with conflict resolution or fail gracefully
		var mergeOutput bytes.Buffer
		err = repo.Merge(ctx, env.ID, &mergeOutput)

		// The merge should fail due to conflict
		assert.Error(t, err, "Merge should fail due to conflict")
		outputStr := mergeOutput.String()
		assert.Contains(t, outputStr, "conflict", "Merge output should mention conflict: %s", outputStr)
	})
}

// TestRepositoryMergeCompleted tests merging the same environment multiple times
// This should result in fast-forward merges since the main branch doesn't diverge
func TestRepositoryMergeCompleted(t *testing.T) {
	WithRepository(t, "repository-merge-completed", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add initial content
		env := user.CreateEnvironment("Test Repeated Merge", "Testing repeated merges")
		user.FileWrite(env.ID, "repeated-file.txt", "initial content", "Add initial file")

		// First merge
		var mergeOutput1 bytes.Buffer
		err := repo.Merge(ctx, env.ID, &mergeOutput1)
		require.NoError(t, err, "First merge should succeed: %s", mergeOutput1.String())

		// Verify first merge content
		filePath := filepath.Join(repo.SourcePath(), "repeated-file.txt")
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "initial content", string(content))

		// Update the same file in the environment
		user.FileWrite(env.ID, "repeated-file.txt", "updated content", "Update file content")

		// Second merge
		var mergeOutput2 bytes.Buffer
		err = repo.Merge(ctx, env.ID, &mergeOutput2)
		require.NoError(t, err, "Second merge should succeed: %s", mergeOutput2.String())

		// Verify second merge content
		content, err = os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "updated content", string(content))

		// Verify commit history includes both merges
		log, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--oneline", "-10")
		require.NoError(t, err)
		// Should have commits for both merges or their individual commits
		assert.Contains(t, log, "Add initial file", "Log should contain initial commit")
		assert.Contains(t, log, "Update file content", "Log should contain update commit")
	})
}
