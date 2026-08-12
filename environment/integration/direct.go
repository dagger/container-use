package integration

import (
	"os"
	"path/filepath"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/require"
)

// --- Direct manipulation methods for edge case testing ---

// WriteSourceFile writes directly to the source repository
func (u *UserActions) WriteSourceFile(path, content string) {
	require.NotEmpty(u.t, u.repoDir, "Need direct access for source file manipulation")
	fullPath := filepath.Join(u.repoDir, path)
	dir := filepath.Dir(fullPath)

	err := os.MkdirAll(dir, 0755)
	require.NoError(u.t, err, "Failed to create dir")

	err = os.WriteFile(fullPath, []byte(content), 0600)
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

// WriteFileInSourceRepo writes a file to the source repo and commits it
func (u *UserActions) WriteFileInSourceRepo(path, content, commitMessage string) {
	require.NotEmpty(u.t, u.repoDir, "Need direct access for source file manipulation")
	writeFile(u.t, u.repoDir, path, content)
	gitCommit(u.t, u.repoDir, commitMessage)
}

// CreateBranchInSourceRepo creates and checks out a new branch in the source repo
func (u *UserActions) CreateBranchInSourceRepo(branchName string) {
	u.GitCommand("checkout", "-b", branchName)
}

// CheckoutBranchInSourceRepo checks out an existing branch in the source repo
func (u *UserActions) CheckoutBranchInSourceRepo(branchName string) {
	u.GitCommand("checkout", branchName)
}
