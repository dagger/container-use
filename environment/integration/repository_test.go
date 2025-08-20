package integration

import (
	"bytes"
	"os"
	"path/filepath"
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
		ctx := t.Context()

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
		ctx := t.Context()

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
		ctx := t.Context()

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
		ctx := t.Context()

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
		ctx := t.Context()

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
		ctx := t.Context()

		// Create initial commit with test content
		user.WriteFileInSourceRepo("initial.txt", "initial content", "Initial commit")

		// Create a feature branch and add content
		user.CreateBranchInSourceRepo("feature-branch")
		user.WriteFileInSourceRepo("feature.txt", "feature content", "Add feature")

		// Go back to main and add different content
		user.CheckoutBranchInSourceRepo("main")
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

func TestRepositoryWithSubmodule(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-with-submodule", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := t.Context()

		user.GitCommand("submodule", "add", "https://github.com/dagger/dagger-test-modules.git", "submodule")
		user.GitCommand("submodule", "update", "--init")
		// test that everything works regardless of user's submodule init status
		user.GitCommand("submodule", "add", "https://github.com/dagger/dagger-test-modules.git", "submodule-2")

		user.GitCommand("commit", "-am", "add submodules")

		env := user.CreateEnvironment("Test Submodule", "Testing repository with submodule")

		// Add a file to the base repo
		user.FileWrite(env.ID, "test.txt", "initial content\n", "Initial commit")

		// Add a file to the submodule
		require.Error(t, env.FileWrite(
			ctx,
			"attempt to write a file to the submodule",
			"submodule/test.txt",
			"This should fail",
		))

		assert.NoError(t, repo.Update(ctx, env, "write the env back to the repo"))

		// Assert that submodule/test.txt doesn't exist on the host
		hostSubmoduleTestPath := filepath.Join(repo.SourcePath(), "submodule", "test.txt")
		_, statErr := os.Stat(hostSubmoduleTestPath)
		assert.True(t, os.IsNotExist(statErr), "submodule/test.txt should not exist on the host")

		// check that the contents of the repo are being cloned into the env
		checkSubmoduleReadme := func(submodulePath string) {
			readmeContent, readErr := env.FileRead(ctx, submodulePath+"/README.md", true, 0, 0)
			require.NoError(t, readErr, "Should be able to read %s/README.md from inside container", submodulePath)
			assert.Contains(t, readmeContent, "Test fixtures used by dagger integration tests.")
		}

		checkSubmoduleReadme("submodule")
		checkSubmoduleReadme("submodule-2")

		// Below we document the behavior of env.Run-instigated file writes to submodules.
		// Ideally, these would error, but practically we don't have an easy way to detect them.
		// env.Run-instigated submodules writes do not error, but they also do not propagate outwards to the fork repository.
		_, err := env.Run(ctx, "echo 'content from env_run_cmd' > submodule/test-from-cmd.txt", "sh", false)
		require.NoError(t, err, "env_run_cmd should be able to write files in submodules")

		// Verify the file was created inside the container
		fileContent, err := env.FileRead(ctx, "submodule/test-from-cmd.txt", true, 0, 0)
		require.NoError(t, err, "Should be able to read the file created by env_run_cmd")
		assert.Contains(t, fileContent, "content from env_run_cmd")

		// However, after update, the file should not exist on the host (same behavior as blocked FileWrite)
		assert.NoError(t, repo.Update(ctx, env, "update the env back to the repo"))
		hostCmdTestPath := filepath.Join(repo.SourcePath(), "submodule", "test-from-cmd.txt")
		_, statErr = os.Stat(hostCmdTestPath)
		assert.True(t, os.IsNotExist(statErr), "submodule/test-from-cmd.txt should not exist on the host after update")

		// Verify that the git working tree remains clean (no uncommitted changes)
		gitStatus, err := repository.RunGitCommand(ctx, repo.SourcePath(), "status", "--porcelain")
		require.NoError(t, err, "Should be able to check git status")
		assert.Empty(t, strings.TrimSpace(gitStatus), "Git working tree should remain clean after env_run_cmd writes to submodule")
	})
}

func TestRepositoryWithRecursiveSubmodule(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-with-submodule", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := t.Context()

		user.GitCommand("submodule", "add", "https://github.com/git-up/test-repo-recursive-submodules.git", "submodule")
		user.GitCommand("submodule", "update", "--init")
		user.GitCommand("submodule", "add", "https://github.com/git-up/test-repo-recursive-submodules.git", "submodule-2")

		user.GitCommand("commit", "-am", "add submodules")

		env := user.CreateEnvironment("Test Submodule", "Testing repository with submodule")

		// Add a file to the base repo
		user.FileWrite(env.ID, "test.txt", "initial content\n", "Initial commit")

		// Add a file to the submodule
		require.Error(t, env.FileWrite(
			ctx,
			"attempt to write a file to the submodule",
			"submodule/test.txt",
			"This should fail",
		))

		assert.NoError(t, repo.Update(ctx, env, "write the env back to the repo"))

		// Assert that submodule/test.txt doesn't exist on the host
		hostSubmoduleTestPath := filepath.Join(repo.SourcePath(), "submodule", "test.txt")
		_, statErr := os.Stat(hostSubmoduleTestPath)
		assert.True(t, os.IsNotExist(statErr), "submodule/test.txt should not exist on the host")

		// check that the contents of the repo are being cloned into the env
		checkSubmoduleReadme := func(submodulePath string) {
			readmeContent, readErr := env.FileRead(ctx, submodulePath+"/README.md", true, 0, 0)
			require.NoError(t, readErr, "Should be able to read %s/README.md from inside container", submodulePath)
			assert.Contains(t, readmeContent, "A test repository that uses submodules")
		}

		// Check first-level submodules
		checkSubmoduleReadme("submodule")
		checkSubmoduleReadme("submodule-2")

		// Check nested submodules (recursive submodules)
		checkNestedSubmoduleReadme := func(submodulePath string) {
			nestedReadmeContent, readErr := env.FileRead(ctx, submodulePath+"/rebase/base/README.md", true, 0, 0)
			require.NoError(t, readErr, "Should be able to read %s/rebase/base/README.md from inside container", submodulePath)
			assert.Contains(t, nestedReadmeContent, "A simple test repository")
		}

		checkNestedSubmoduleReadme("submodule")
		checkNestedSubmoduleReadme("submodule-2")
	})
}

// TestRepositoryDiff tests retrieving changes between commits
func TestRepositoryDiff(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-diff", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := t.Context()

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
