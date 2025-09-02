package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
)

// TestProjectSpecificGitConfiguration tests that project-specific git configurations
// are properly inherited and used within environments
func TestProjectSpecificGitConfiguration(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("UserEmailAndName", func(t *testing.T) {
		WithRepository(t, "git_config_user", SetupRepoWithGitConfig, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Git Config Test", "Testing git config inheritance")

			// Make a commit in the environment
			user.FileWrite(env.ID, "test.txt", "test content", "Test commit with project config")

			// Get the worktree path to run git commands directly
			worktreePath := user.WorktreePath(env.ID)

			// Check the commit author in the environment's git log
			ctx := context.Background()
			gitLog, err := repository.RunGitCommand(ctx, worktreePath, "log", "--format=%an <%ae>", "-n", "1")
			assert.NoError(t, err, "Should be able to get git log")

			// Should use project-specific user config, not global
			assert.Contains(t, gitLog, "Project User <project@example.com>", "Should use project git config for commits")
		})
	})

	t.Run("CommitGPGSign", func(t *testing.T) {
		WithRepository(t, "git_config_gpg", SetupRepoWithGPGConfig, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("GPG Config Test", "Testing GPG signing config")

			// Make a commit - this will either succeed with signing or fail appropriately
			user.FileWrite(env.ID, "gpg-test.txt", "content for gpg test", "Test commit with GPG config")

			// Get the worktree path to check git config
			worktreePath := user.WorktreePath(env.ID)

			// Verify the GPG signing configuration is present
			ctx := context.Background()
			gpgSignConfig, err := repository.RunGitCommand(ctx, worktreePath, "config", "commit.gpgsign")
			assert.NoError(t, err, "Should be able to read commit.gpgsign config")
			assert.Contains(t, gpgSignConfig, "true", "GPG signing should be enabled")
		})
	})

	t.Run("GitConfigPersistsAcrossEnvironments", func(t *testing.T) {
		WithRepository(t, "git_config_persist", SetupRepoWithGitConfig, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			// Create first environment
			env1 := user.CreateEnvironment("Config Test 1", "First environment")
			user.FileWrite(env1.ID, "file1.txt", "content 1", "Commit in env1")

			// Create second environment
			env2 := user.CreateEnvironment("Config Test 2", "Second environment")
			user.FileWrite(env2.ID, "file2.txt", "content 2", "Commit in env2")

			ctx := context.Background()

			// Both environments should use the same project config
			worktree1 := user.WorktreePath(env1.ID)
			gitLog1, err := repository.RunGitCommand(ctx, worktree1, "log", "--format=%an <%ae>", "-n", "1")
			assert.NoError(t, err)

			worktree2 := user.WorktreePath(env2.ID)
			gitLog2, err := repository.RunGitCommand(ctx, worktree2, "log", "--format=%an <%ae>", "-n", "1")
			assert.NoError(t, err)

			// Both should use project config
			assert.Contains(t, gitLog1, "Project User <project@example.com>")
			assert.Contains(t, gitLog2, "Project User <project@example.com>")
		})
	})

	t.Run("GlobalVsLocalConfigPrecedence", func(t *testing.T) {
		WithRepository(t, "git_config_precedence", SetupRepoWithConflictingConfig, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Precedence Test", "Testing config precedence")
			user.FileWrite(env.ID, "precedence.txt", "testing precedence", "Commit to test config precedence")

			worktreePath := user.WorktreePath(env.ID)
			ctx := context.Background()

			// Check that local repo config takes precedence over global
			userName, err := repository.RunGitCommand(ctx, worktreePath, "config", "user.name")
			assert.NoError(t, err)
			assert.Contains(t, userName, "Local Project User", "Local config should override global")

			userEmail, err := repository.RunGitCommand(ctx, worktreePath, "config", "user.email")
			assert.NoError(t, err)
			assert.Contains(t, userEmail, "local@project.com", "Local config should override global")
		})
	})
}

// TestProjectSpecificGitHooks tests that git hooks are properly ignored in environments
func TestProjectSpecificGitHooks(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("HooksAreIgnoredInEnvironment", func(t *testing.T) {
		WithRepository(t, "hooks_ignored_env", SetupRepoWithGitHooks, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Hook Ignore Test", "Testing that git hooks are ignored in environments")

			// This file would normally be blocked by the pre-commit hook in the source repo
			user.FileWrite(env.ID, "forbidden.txt", "This should be allowed", "Commit forbidden file")

			// Verify the file exists in the environment (commit succeeded in environment)
			content := user.FileRead(env.ID, "forbidden.txt")
			assert.Equal(t, "This should be allowed", content)
		})
	})

	t.Run("HooksAreIgnoredInSourceRepo", func(t *testing.T) {
		WithRepository(t, "hooks_ignored_source", SetupRepoWithGitHooks, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Hook Source Test", "Testing that git hooks are ignored when updating source repo")

			// Create a file that would be blocked by pre-commit hook
			user.FileWrite(env.ID, "forbidden.txt", "This should be allowed", "Commit forbidden file")

			// Now checkout the environment branch to the source repo - this should also ignore hooks
			ctx := context.Background()
			branch, err := repo.Checkout(ctx, env.ID, "")
			assert.NoError(t, err, "Checkout should succeed even with hooks that would block it")
			assert.NotEmpty(t, branch)

			// Verify the forbidden file exists in the source repo now
			sourcePath := repo.SourcePath()
			forbiddenPath := filepath.Join(sourcePath, "forbidden.txt")
			sourceContent, err := os.ReadFile(forbiddenPath)
			assert.NoError(t, err, "forbidden.txt should exist in source repo after checkout")
			assert.Equal(t, "This should be allowed", string(sourceContent))
		})
	})

	t.Run("CommitsSucceedDespiteFailingHooks", func(t *testing.T) {
		WithRepository(t, "commits_despite_failing_hooks", SetupRepoWithFailingHooks, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Failing Hook Test", "Testing commits work despite failing hooks")

			// Make commits that would be blocked by failing hooks
			user.FileWrite(env.ID, "should-fail-1.txt", "content 1", "First commit that hooks would block")
			user.FileWrite(env.ID, "should-fail-2.txt", "content 2", "Second commit that hooks would block")

			// Verify files exist (commits succeeded despite failing hooks)
			assert.Equal(t, "content 1", user.FileRead(env.ID, "should-fail-1.txt"))
			assert.Equal(t, "content 2", user.FileRead(env.ID, "should-fail-2.txt"))

			// Now test that checkout to source repo also works despite failing hooks
			ctx := context.Background()
			branch, err := repo.Checkout(ctx, env.ID, "")
			assert.NoError(t, err, "Checkout should succeed even with failing hooks")
			assert.NotEmpty(t, branch)

			// Verify files exist in source repo
			sourcePath := repo.SourcePath()
			file1Path := filepath.Join(sourcePath, "should-fail-1.txt")
			file1Content, err := os.ReadFile(file1Path)
			assert.NoError(t, err, "Files should exist in source repo after checkout")
			assert.Equal(t, "content 1", string(file1Content))
		})
	})

	t.Run("HookSideEffectsDoNotOccurAnywhere", func(t *testing.T) {
		WithRepository(t, "no_hook_side_effects_anywhere", SetupRepoWithGitHooks, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("No Side Effects Test", "Testing that hook side effects don't occur")

			// Make a commit that would trigger post-commit hook
			user.FileWrite(env.ID, "trigger-hooks.txt", "This should trigger hooks", "Commit to trigger hooks")

			// Verify no hook evidence file in environment
			user.FileReadExpectError(env.ID, ".hook-evidence")

			// Verify no hook evidence file in environment worktree
			worktreePath := user.WorktreePath(env.ID)
			hookEvidencePath := filepath.Join(worktreePath, ".hook-evidence")
			_, err := os.Stat(hookEvidencePath)
			assert.True(t, os.IsNotExist(err), "Hook evidence file should not exist in worktree")

			// Checkout to source repo
			ctx := context.Background()
			_, err = repo.Checkout(ctx, env.ID, "")
			assert.NoError(t, err, "Checkout should succeed")

			// Verify no hook evidence file in source repo either
			sourcePath := repo.SourcePath()
			sourceHookEvidencePath := filepath.Join(sourcePath, ".hook-evidence")
			_, err = os.Stat(sourceHookEvidencePath)
			assert.True(t, os.IsNotExist(err), "Hook evidence file should not exist in source repo after checkout")
		})
	})
}