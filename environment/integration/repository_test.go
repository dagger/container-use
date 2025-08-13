package integration

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRepositoryCreate tests creating a new environment
func TestRepositoryCreate(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-create", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create an environment
		env := user.CreateEnvironment("Test Create", "Testing repository create")

		// Verify environment was created properly
		assert.NotNil(t, env)
		assert.NotEmpty(t, env.ID)
		assert.Equal(t, "Test Create", env.State.Title)
		worktreePath := user.WorktreePath(env.ID)
		assert.NotEmpty(t, worktreePath)

		// Verify worktree was created
		_, err := os.Stat(worktreePath)
		assert.NoError(t, err)
	})
}

// TestRepositoryGet tests retrieving an existing environment
func TestRepositoryGet(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-get", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment
		env := user.CreateEnvironment("Test Get", "Testing repository get")

		// Get the environment using repository directly
		retrieved, err := repo.Get(ctx, user.dag, env.ID)
		require.NoError(t, err)
		assert.NotNil(t, retrieved)
		assert.Equal(t, env.ID, retrieved.ID)
		assert.Equal(t, env.State.Title, retrieved.State.Title)

		// Test getting non-existent environment
		_, err = repo.Get(ctx, user.dag, "non-existent-env")
		assert.Error(t, err)
	})
}

// TestRepositoryList tests listing all environments
func TestRepositoryList(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-list", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create two environments
		env1 := user.CreateEnvironment("Environment 1", "First test environment")
		env2 := user.CreateEnvironment("Environment 2", "Second test environment")

		// List should return at least 2
		envs, err := repo.List(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(envs), 2)

		// Verify the environments are in the list
		var foundIDs []string
		for _, e := range envs {
			foundIDs = append(foundIDs, e.ID)
		}
		assert.Contains(t, foundIDs, env1.ID)
		assert.Contains(t, foundIDs, env2.ID)
	})
}

// TestRepositoryDelete tests deleting an environment
func TestRepositoryDelete(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-delete", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment
		env := user.CreateEnvironment("Test Delete", "Testing repository delete")
		worktreePath := user.WorktreePath(env.ID)
		envID := env.ID

		// Delete it
		err := repo.Delete(ctx, envID)
		require.NoError(t, err)

		// Verify it's gone
		_, err = repo.Get(ctx, user.dag, envID)
		assert.Error(t, err)

		// Verify worktree is deleted
		_, err = os.Stat(worktreePath)
		assert.True(t, os.IsNotExist(err))
	})
}

// TestRepositoryCheckout tests checking out an environment branch
func TestRepositoryCheckout(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-checkout", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add content
		env := user.CreateEnvironment("Test Checkout", "Testing repository checkout")
		user.FileWrite(env.ID, "test.txt", "test content", "Add test file")

		// Checkout the environment branch in the source repo
		branch, err := repo.Checkout(ctx, env.ID, "")
		require.NoError(t, err)
		assert.NotEmpty(t, branch)

		// Verify we're on the correct branch
		currentBranch, err := repository.RunGitCommand(ctx, repo.SourcePath(), "branch", "--show-current")
		require.NoError(t, err)
		// Branch name could be either env.ID or cu-env.ID depending on the logic
		actualBranch := strings.TrimSpace(currentBranch)
		assert.True(t, actualBranch == env.ID || actualBranch == "cu-"+env.ID,
			"Expected branch to be %s or cu-%s, got %s", env.ID, env.ID, actualBranch)
	})
}

// TestRepositoryLog tests retrieving commit history for an environment
func TestRepositoryLog(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-log", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and add some commits
		env := user.CreateEnvironment("Test Log", "Testing repository log")
		user.FileWrite(env.ID, "file1.txt", "initial content", "Initial commit")
		user.FileWrite(env.ID, "file1.txt", "updated content", "Update file")
		user.FileWrite(env.ID, "file2.txt", "new file", "Add second file")

		// Get commit log without patches
		var logBuf bytes.Buffer
		err := repo.Log(ctx, env.ID, false, &logBuf)
		logOutput := logBuf.String()
		require.NoError(t, err, logOutput)

		// Verify commit messages are present
		assert.Contains(t, logOutput, "Add second file")
		assert.Contains(t, logOutput, "Update file")
		assert.Contains(t, logOutput, "Initial commit")

		// Get commit log with patches
		logBuf.Reset()
		err = repo.Log(ctx, env.ID, true, &logBuf)
		logWithPatchOutput := logBuf.String()
		require.NoError(t, err, logWithPatchOutput)

		// Verify patch information is included
		assert.Contains(t, logWithPatchOutput, "diff --git")
		assert.Contains(t, logWithPatchOutput, "+updated content")

		// Test log for non-existent environment
		err = repo.Log(ctx, "non-existent-env", false, &logBuf)
		assert.Error(t, err)
	})
}

// TestRepositoryCreateFromGitRef tests creating environments from specific git references
func TestRepositoryCreateFromGitRef(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-create-from-ref", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create initial commit with test content
		user.WriteFileInSourceRepo("initial.txt", "initial content", "Initial commit")

		// Create a feature branch and add content
		user.CreateBranchInSourceRepo("feature-branch")
		user.WriteFileInSourceRepo("feature.txt", "feature content", "Add feature")

		// Go back to master and add different content
		user.CheckoutBranchInSourceRepo("master")
		user.WriteFileInSourceRepo("main.txt", "main content", "Add main file")

		// Get the SHA of the initial commit (before the main.txt was added)
		initialSHA, err := repository.RunGitCommand(ctx, repo.SourcePath(), "log", "--format=%H", "-n", "2", "--reverse")
		require.NoError(t, err)
		initialCommitSHA := strings.Split(strings.TrimSpace(initialSHA), "\n")[0]

		// Test creating environment from HEAD (default behavior)
		envFromHead := user.CreateEnvironment("From HEAD", "Environment from HEAD")
		content, err := envFromHead.FileRead(ctx, "main.txt", true, 0, 0)
		require.NoError(t, err)
		assert.Contains(t, content, "main content")

		// Test creating environment from feature branch
		envFromBranch, err := repo.Create(ctx, user.dag, "From Feature", "Environment from feature branch", "feature-branch")
		require.NoError(t, err)
		assert.NotNil(t, envFromBranch)

		// Should have feature.txt but not main.txt
		featureContent, err := envFromBranch.FileRead(ctx, "feature.txt", true, 0, 0)
		require.NoError(t, err)
		assert.Contains(t, featureContent, "feature content")

		_, err = envFromBranch.FileRead(ctx, "main.txt", true, 0, 0)
		assert.Error(t, err, "main.txt should not exist in feature branch environment")

		// Test creating environment from specific SHA
		envFromSHA, err := repo.Create(ctx, user.dag, "From SHA", "Environment from initial commit", initialCommitSHA)
		require.NoError(t, err)
		assert.NotNil(t, envFromSHA)

		// Should have only initial.txt
		initialContent, err := envFromSHA.FileRead(ctx, "initial.txt", true, 0, 0)
		require.NoError(t, err)
		assert.Contains(t, initialContent, "initial content")

		_, err = envFromSHA.FileRead(ctx, "main.txt", true, 0, 0)
		assert.Error(t, err, "main.txt should not exist in SHA environment")

		_, err = envFromSHA.FileRead(ctx, "feature.txt", true, 0, 0)
		assert.Error(t, err, "feature.txt should not exist in SHA environment")

		// Test invalid git ref
		_, err = repo.Create(ctx, user.dag, "Invalid Ref", "Environment from invalid ref", "nonexistent-ref")
		assert.Error(t, err, "Should fail with invalid git ref")
	})
}

// TestRepositoryDiff tests retrieving changes between commits
func TestRepositoryDiff(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-diff", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()

		// Create an environment and make some changes
		env := user.CreateEnvironment("Test Diff", "Testing repository diff")

		// First commit - add a file
		user.FileWrite(env.ID, "test.txt", "initial content\n", "Initial commit")

		// Make changes to the file
		user.FileWrite(env.ID, "test.txt", "initial content\nupdated content\n", "Update file")

		// Get diff output
		var diffBuf bytes.Buffer
		err := repo.Diff(ctx, env.ID, &diffBuf)
		diffOutput := diffBuf.String()
		require.NoError(t, err, diffOutput)

		// Verify diff contains expected changes
		assert.Contains(t, diffOutput, "+updated content")

		// Test diff with non-existent environment
		err = repo.Diff(ctx, "non-existent-env", &diffBuf)
		assert.Error(t, err)
	})
}
