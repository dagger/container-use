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

// Git command error handling ensures we gracefully handle git failures
func TestGitCommandErrors(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()

	// Test invalid command
	_, err := RunGitCommand(ctx, tempDir, "invalid-command")
	assert.Error(t, err, "Should get error for invalid git command")

	// Test command in non-existent directory (use a cross-platform non-existent path)
	nonExistentDir := filepath.Join(tempDir, "nonexistent", "deeply", "nested")
	_, err = RunGitCommand(ctx, nonExistentDir, "status")
	assert.Error(t, err, "Should get error for non-existent directory")
}

// Selective file staging ensures problematic files are automatically excluded from commits
// This tests the actual user-facing behavior: "I want to commit my changes but not break git"
func TestSelectiveFileStaging(t *testing.T) {
	// Test real-world scenarios that users encounter
	scenarios := []struct {
		name        string
		setup       func(t *testing.T, dir string)
		shouldStage []string
		shouldSkip  []string
		reason      string
	}{
		{
			name: "python_project_with_pycache",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "main.py", "print('hello')")
				writeFile(t, dir, "utils.py", "def helper(): pass")
				createDir(t, dir, "__pycache__")
				writeBinaryFile(t, dir, "__pycache__/main.cpython-39.pyc", 150)
				writeBinaryFile(t, dir, "__pycache__/utils.cpython-39.pyc", 200)
			},
			shouldStage: []string{"main.py", "utils.py"},
			shouldSkip:  []string{"__pycache__"},
			reason:      "Python cache files should never be committed",
		},
		{
			name: "mixed_content_directory",
			setup: func(t *testing.T, dir string) {
				createDir(t, dir, "mydir")
				writeFile(t, dir, "mydir/readme.txt", "Documentation")
				writeBinaryFile(t, dir, "mydir/compiled.bin", 100)
				writeFile(t, dir, "mydir/script.sh", "#!/bin/bash\necho hello")
				writeBinaryFile(t, dir, "mydir/image.jpg", 5000)
			},
			shouldStage: []string{"mydir/readme.txt", "mydir/script.sh"},
			shouldSkip:  []string{"mydir/compiled.bin", "mydir/image.jpg"},
			reason:      "Binary files in directories should be automatically excluded",
		},
		{
			name: "node_modules_and_build_artifacts",
			setup: func(t *testing.T, dir string) {
				writeFile(t, dir, "index.js", "console.log('app')")
				createDir(t, dir, "node_modules/lodash")
				writeFile(t, dir, "node_modules/lodash/index.js", "module.exports = {}")
				createDir(t, dir, "build")
				writeBinaryFile(t, dir, "build/app.exe", 1024)
				writeFile(t, dir, "build/config.json", `{"prod": true}`)
			},
			shouldStage: []string{"index.js"},
			shouldSkip:  []string{"node_modules", "build"},
			reason:      "Dependencies and build outputs should be excluded",
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Create a test git repository
			dir := t.TempDir()
			ctx := context.Background()

			// Initialize git repo
			_, err := RunGitCommand(ctx, dir, "init")
			require.NoError(t, err)

			// Set git config to avoid errors
			_, err = RunGitCommand(ctx, dir, "config", "user.email", "test@example.com")
			require.NoError(t, err)
			_, err = RunGitCommand(ctx, dir, "config", "user.name", "Test User")
			require.NoError(t, err)

			// Setup the scenario
			scenario.setup(t, dir)

			// Create a Repository instance for testing
			repo := &Repository{
				lockManager: NewRepositoryLockManager(dir),
			}

			// Run the actual staging logic (testing the integration)
			err = repo.addNonBinaryFiles(ctx, dir, []string{})
			require.NoError(t, err, "Staging should not error")

			status, err := RunGitCommand(ctx, dir, "status", "--porcelain")
			require.NoError(t, err)

			// Verify expected behavior
			for _, file := range scenario.shouldStage {
				// Files should be staged (A  prefix)
				assert.Contains(t, status, "A  "+file, "%s should be staged - %s", file, scenario.reason)
			}

			for _, pattern := range scenario.shouldSkip {
				// Files should remain untracked (?? prefix), not staged (A  prefix)
				assert.NotContains(t, status, "A  "+pattern, "%s should not be staged - %s", pattern, scenario.reason)
				// They should appear as untracked
				if !strings.Contains(pattern, "/") {
					assert.Contains(t, status, "?? "+pattern, "%s should remain untracked - %s", pattern, scenario.reason)
				}
			}
		})
	}
}

// Test the commitWorktreeChanges function
func TestCommitWorktreeChanges(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()

	// Initialize git repo
	_, err := RunGitCommand(ctx, dir, "init")
	require.NoError(t, err)

	// Set git config
	_, err = RunGitCommand(ctx, dir, "config", "user.email", "test@example.com")
	require.NoError(t, err)
	_, err = RunGitCommand(ctx, dir, "config", "user.name", "Test User")
	require.NoError(t, err)

	repo := &Repository{
		lockManager: NewRepositoryLockManager(dir),
	}

	t.Run("empty_directory_handling", func(t *testing.T) {
		// Create empty directories (git doesn't track these)
		createDir(t, dir, "empty1")
		createDir(t, dir, "empty2/nested")

		// This verifies that commitWorktreeChanges handles empty directories gracefully
		// It should return nil (success) when there's nothing to commit
		err := repo.commitWorktreeChanges(ctx, dir, "Empty dirs", []string{})
		assert.NoError(t, err, "commitWorktreeChanges should handle empty dirs gracefully")
	})

	t.Run("commits_changes", func(t *testing.T) {
		// Create a file to commit
		writeFile(t, dir, "test.txt", "hello world")

		err := repo.commitWorktreeChanges(ctx, dir, "Testing commit functionality", []string{})
		require.NoError(t, err)

		// Verify commit was created
		log, err := RunGitCommand(ctx, dir, "log", "--oneline")
		require.NoError(t, err)
		assert.Contains(t, log, "Testing commit functionality")
	})
}

// Test helper functions
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	err := os.WriteFile(path, []byte(content), 0644)
	require.NoError(t, err)
}

func writeBinaryFile(t *testing.T, dir, name string, size int) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)

	// Create binary content
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	err := os.WriteFile(path, data, 0644)
	require.NoError(t, err)
}

func createDir(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.MkdirAll(path, 0755)
	require.NoError(t, err)
}
