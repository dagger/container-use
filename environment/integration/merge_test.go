package integration

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryMerge tests merging an environment into the main branch
func TestRepositoryMerge(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

// TestRepositoryMergeSquash tests squash merging an environment
func TestRepositoryMergeSquash(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-merge-squash", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add some content
		env := user.CreateEnvironment("Test Squash Merge", "Squash merge test with multiple files")
		user.FileWrite(env.ID, "squash-test.txt", "content from environment", "Add squash test file")
		user.FileWrite(env.ID, "config.json", `{"version": "1.0"}`, "Add config file")

		// Get initial commit count
		initialLog, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-list", "--count", "HEAD")
		require.NoError(t, err)
		initialCommitCount := strings.TrimSpace(initialLog)

		// Perform squash merge
		var mergeOutput bytes.Buffer
		err = repo.MergeSquash(ctx, env.ID, &mergeOutput)
		require.NoError(t, err, "Squash merge should succeed: %s", mergeOutput.String())

		// Verify the files were merged into the working directory
		squashTestPath := filepath.Join(repo.SourcePath(), "squash-test.txt")
		content, err := os.ReadFile(squashTestPath)
		require.NoError(t, err)
		assert.Equal(t, "content from environment", string(content))

		configPath := filepath.Join(repo.SourcePath(), "config.json")
		configContent, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Equal(t, `{"version": "1.0"}`, string(configContent))

		// Verify we have exactly one new commit
		finalLog, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-list", "--count", "HEAD")
		require.NoError(t, err)
		finalCommitCount := strings.TrimSpace(finalLog)

		initialCount, err := strconv.Atoi(initialCommitCount)
		require.NoError(t, err)
		finalCount, err := strconv.Atoi(finalCommitCount)
		require.NoError(t, err)
		assert.Equal(t, initialCount+1, finalCount, "Should have exactly one new commit")

		// Verify the commit message uses the environment title
		commitMessage, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "-1", "--pretty=format:%s")
		require.NoError(t, err)
		assert.Equal(t, "Test Squash Merge", commitMessage)
	})
}

// TestRepositoryMergeSquashRepeated tests repeated squash merges with theirs strategy
// This should allow updating the same file multiple times without conflicts
func TestRepositoryMergeSquashRepeated(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-merge-squash-repeated", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add initial content
		env := user.CreateEnvironment("Test Repeated Squash", "First squash commit")
		user.FileWrite(env.ID, "repeated-squash.txt", "initial content", "Add initial file")

		// First squash merge
		var mergeOutput1 bytes.Buffer
		err := repo.MergeSquash(ctx, env.ID, &mergeOutput1)
		require.NoError(t, err, "First squash merge should succeed: %s", mergeOutput1.String())

		// Verify first squash merge content
		filePath := filepath.Join(repo.SourcePath(), "repeated-squash.txt")
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "initial content", string(content))

		// Update the same file in the environment - this should use theirs strategy
		user.FileWrite(env.ID, "repeated-squash.txt", "updated content", "Update file content")
		user.FileWrite(env.ID, "additional.txt", "new file content", "Add new file")

		var mergeOutput2 bytes.Buffer
		err = repo.MergeSquash(ctx, env.ID, &mergeOutput2)
		require.NoError(t, err, "Second squash merge should succeed: %s", mergeOutput2.String())

		// Verify second squash merge content - should have the updated content from environment
		content, err = os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "updated content", string(content))

		// Verify new file was added
		additionalPath := filepath.Join(repo.SourcePath(), "additional.txt")
		additionalContent, err := os.ReadFile(additionalPath)
		require.NoError(t, err)
		assert.Equal(t, "new file content", string(additionalContent))

		// Verify we have clean squash commits in the log
		log, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--oneline", "-5")
		require.NoError(t, err)
		// Both commits should have the same title since it's the same environment
		commitCount := strings.Count(log, "Test Repeated Squash")
		assert.Equal(t, 2, commitCount, "Should have two commits with environment title")
	})
}

// TestRepositoryMergeSquashTheirsStrategy tests that theirs strategy is used when appropriate
func TestRepositoryMergeSquashTheirsStrategy(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-merge-squash-theirs", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add initial content
		env := user.CreateEnvironment("Test Theirs Strategy", "Testing theirs strategy")
		user.FileWrite(env.ID, "conflict-file.txt", "environment version 1", "Add initial file")

		// First squash merge
		var mergeOutput1 bytes.Buffer
		err := repo.MergeSquash(ctx, env.ID, &mergeOutput1)
		require.NoError(t, err, "First squash merge should succeed: %s", mergeOutput1.String())

		// Verify first squash merge content
		filePath := filepath.Join(repo.SourcePath(), "conflict-file.txt")
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "environment version 1", string(content))

		// Now modify the same file in the environment to a different version
		user.FileWrite(env.ID, "conflict-file.txt", "environment version 2", "Update file to version 2")

		// Second squash merge - should use theirs strategy and favor environment version
		var mergeOutput2 bytes.Buffer
		err = repo.MergeSquash(ctx, env.ID, &mergeOutput2)
		require.NoError(t, err, "Second squash merge should succeed with theirs strategy: %s", mergeOutput2.String())

		// Verify the content is from the environment (theirs), not main branch (ours)
		content, err = os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "environment version 2", string(content), "Should use environment version (updated content)")

		// Verify we have clean squash commits in the log
		log, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--oneline", "-5")
		require.NoError(t, err)

		// Should have two squash commits with environment title
		commitCount := strings.Count(log, "Test Theirs Strategy")
		assert.Equal(t, 2, commitCount, "Should have two squash commits with environment title")
	})
}

// TestRepositoryMergeSquashConflictResolution tests that theirs strategy resolves conflicts correctly
func TestRepositoryMergeSquashConflictResolution(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-merge-squash-conflict", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create first environment and add initial content
		env1 := user.CreateEnvironment("First Environment", "First squash merge")
		user.FileWrite(env1.ID, "shared-file.txt", "content from first environment", "Add shared file")

		// First squash merge
		var mergeOutput1 bytes.Buffer
		err := repo.MergeSquash(ctx, env1.ID, &mergeOutput1)
		require.NoError(t, err, "First squash merge should succeed: %s", mergeOutput1.String())

		// Create second environment and modify the same file
		env2 := user.CreateEnvironment("Second Environment", "Second squash merge with conflict")
		user.FileWrite(env2.ID, "shared-file.txt", "content from second environment", "Update shared file")

		// Second squash merge - should use theirs strategy since all commits are squash merges
		var mergeOutput2 bytes.Buffer
		err = repo.MergeSquash(ctx, env2.ID, &mergeOutput2)
		require.NoError(t, err, "Second squash merge should succeed with theirs strategy: %s", mergeOutput2.String())

		// Verify the content is from the second environment (theirs)
		filePath := filepath.Join(repo.SourcePath(), "shared-file.txt")
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)
		assert.Equal(t, "content from second environment", string(content), "Should use second environment's content (theirs strategy)")

		// Verify we have clean squash commits in the log
		log, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--oneline", "-5")
		require.NoError(t, err)

		// Should have commits from both environments
		assert.Contains(t, log, "Second Environment")
		assert.Contains(t, log, "First Environment")
		assert.Contains(t, log, "Initial commit")
	})
}
