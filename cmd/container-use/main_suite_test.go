package main_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

var (
	binaryPath     string
	binaryPathOnce sync.Once
)

// getContainerUseBinary builds the container-use binary once per test run
func getContainerUseBinary(t *testing.T) string {
	binaryPathOnce.Do(func() {
		t.Log("Building fresh container-use binary...")
		cmd := exec.Command("go", "build", "-o", "container-use", ".")
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		if err != nil {
			t.Fatalf("Failed to build container-use binary: %v", err)
		}

		abs, err := filepath.Abs("container-use")
		if err != nil {
			t.Fatalf("Failed to get absolute path: %v", err)
		}
		binaryPath = abs
	})
	return binaryPath
}

// setupGitRepo initializes a git repository in the given directory
func setupGitRepo(t *testing.T, repoDir string) {
	ctx := context.Background()

	cmds := [][]string{
		{"init"},
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test User"},
		{"config", "commit.gpgsign", "false"},
	}

	for _, cmd := range cmds {
		err := runGitCommand(ctx, repoDir, cmd...)
		require.NoError(t, err, "Failed to run git %v", cmd)
	}

	writeFile(t, repoDir, "README.md", "# E2E Test Repository\n")
	writeFile(t, repoDir, "package.json", `{
  "name": "e2e-test-project",
  "version": "1.0.0",
  "main": "index.js"
}`)

	err := runGitCommand(ctx, repoDir, "add", ".")
	require.NoError(t, err, "Failed to stage files")
	err = runGitCommand(ctx, repoDir, "commit", "-m", "Initial commit")
	require.NoError(t, err, "Failed to commit")
}

// writeFile creates a file with the given content
func writeFile(t *testing.T, repoDir, path, content string) {
	fullPath := filepath.Join(repoDir, path)
	dir := filepath.Dir(fullPath)
	err := os.MkdirAll(dir, 0755)
	require.NoError(t, err, "Failed to create dir")
	err = os.WriteFile(fullPath, []byte(content), 0644)
	require.NoError(t, err, "Failed to write file")
}

// runGitCommand runs a git command in the specified directory
func runGitCommand(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	_, err := cmd.CombinedOutput()
	return err
}
