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

func TestValidateGitRefComponent(t *testing.T) {
	valid := []string{
		"main",
		"cu-adverb-animal",
		"v1.0.0",
		"feature/test_123",
		"HEAD",
	}
	for _, name := range valid {
		assert.NoError(t, validateGitRefComponent(name), "expected %q to be valid", name)
	}

	invalid := []string{
		"",
		"--help",
		"-foo",
		"feature test",
		"foo\nbar",
		"foo\x00bar",
	}
	for _, name := range invalid {
		assert.Error(t, validateGitRefComponent(name), "expected %q to be invalid", name)
	}
}

func TestValidateExportFilePath(t *testing.T) {
	tmp := t.TempDir()
	worktreePath := filepath.Join(tmp, "worktree")
	require.NoError(t, os.MkdirAll(worktreePath, 0755))

	t.Run("rejects_absolute_paths", func(t *testing.T) {
		_, _, err := validateExportFilePath(worktreePath, "/etc/passwd")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "must be relative to the workdir")
	})

	t.Run("rejects_paths_escaping_worktree", func(t *testing.T) {
		for _, filePath := range []string{
			"../../../etc/cron.d/evil",
			"foo/../../etc/passwd",
			"../secret.txt",
		} {
			_, _, err := validateExportFilePath(worktreePath, filePath)
			require.Error(t, err, "expected %q to be rejected", filePath)
			assert.Contains(t, err.Error(), "escapes workdir", filePath)
		}
	})

	t.Run("accepts_valid_relative_paths", func(t *testing.T) {
		clean, abs, err := validateExportFilePath(worktreePath, "foo/bar.txt")
		require.NoError(t, err)
		assert.Equal(t, "foo/bar.txt", clean)
		assert.Equal(t, filepath.Join(worktreePath, "foo/bar.txt"), abs)
	})

	t.Run("normalizes_relative_paths", func(t *testing.T) {
		clean, abs, err := validateExportFilePath(worktreePath, "./foo/../baz.txt")
		require.NoError(t, err)
		assert.Equal(t, "baz.txt", clean)
		assert.Equal(t, filepath.Join(worktreePath, "baz.txt"), abs)
	})
}

func TestNormalizeGitURL(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"https://github.com/user/repo.git", "github.com/user/repo"},
		{"https://github.com/user/repo", "github.com/user/repo"},
		{"git@github.com:user/repo.git", "github.com/user/repo"},
		{"git@github.com:user/repo", "github.com/user/repo"},
	}

	for _, tc := range cases {
		got, err := normalizeGitURL(tc.input)
		require.NoError(t, err, tc.input)
		assert.Equal(t, tc.expected, got, tc.input)
	}
}

func TestNormalizeGitURLInvalid(t *testing.T) {
	_, err := normalizeGitURL("not-a-url")
	assert.Error(t, err)
}

func TestCreateSafePathFromAbsolute(t *testing.T) {
	assert.Equal(t, "home/user/project", createSafePathFromAbsolute("/home/user/project"))
	assert.Equal(t, "C_\\Users\\user\\project", createSafePathFromAbsolute("C:\\Users\\user\\project"))
	assert.Equal(t, "path______", createSafePathFromAbsolute("/path<>|?*\""))
}

func TestIsBinaryFile(t *testing.T) {
	tmp := t.TempDir()
	textFile := filepath.Join(tmp, "text.txt")
	binFile := filepath.Join(tmp, "binary.bin")
	emptyFile := filepath.Join(tmp, "empty")

	require.NoError(t, os.WriteFile(textFile, []byte("hello world\n"), 0644))
	require.NoError(t, os.WriteFile(binFile, []byte{0x00, 0x01, 0x02}, 0644))
	require.NoError(t, os.WriteFile(emptyFile, []byte{}, 0644))

	r := &Repository{}
	assert.False(t, r.isBinaryFile(tmp, "text.txt"))
	assert.True(t, r.isBinaryFile(tmp, "binary.bin"))
	assert.False(t, r.isBinaryFile(tmp, "empty"))
}
