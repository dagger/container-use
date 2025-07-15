package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitAuditTrail verifies that all operations are tracked in git
func TestGitAuditTrail(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	mustParseInt64 := func(t *testing.T, s string) int64 {
		n, err := strconv.ParseInt(s, 10, 64)
		require.NoError(t, err)
		return n
	}

	WithRepository(t, "audit", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		env := user.CreateEnvironment("Audit Test", "Testing git tracking")

		// User performs various operations
		user.FileWrite(env.ID, "config.json", `{"name": "test", "version": "1.0.0"}`, "Initial config")
		user.RunCommand(env.ID, "npm install", "Install dependencies")
		user.FileWrite(env.ID, "src/app.js", "console.log('Hello');", "Add app code")

		// Verify all operations are tracked in git
		gitLog, err := repository.RunGitCommand(context.Background(), user.WorktreePath(env.ID), "log", "--oneline", "-5")
		require.NoError(t, err)

		// Operations that create git-trackable changes should have commits
		assert.Contains(t, gitLog, "Initial config")
		// "Run npm install" won't create a commit (npm not installed, would create gitignored files)
		assert.Contains(t, gitLog, "Add app code")

		// Verify file contents are in git
		configFromGit, err := repository.RunGitCommand(context.Background(), user.WorktreePath(env.ID), "show", "HEAD~1:config.json")
		require.NoError(t, err)
		assert.Contains(t, configFromGit, `"version": "1.0.0"`)

		// Verify commit timestamps are reasonable
		gitTime, err := repository.RunGitCommand(context.Background(), user.WorktreePath(env.ID), "log", "-1", "--pretty=format:%ct")
		require.NoError(t, err)
		commitTime := time.Unix(mustParseInt64(t, strings.TrimSpace(gitTime)), 0)
		assert.WithinDuration(t, time.Now(), commitTime, 5*time.Second)
	})
}

// TestEnvironmentIsolation verifies that changes in one environment don't affect others
func TestEnvironmentIsolation(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	WithRepository(t, "isolation", SetupPythonRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// User creates two environments from the same repository
		dev := user.CreateEnvironment("Development", "Creating dev environment")
		staging := user.CreateEnvironment("Staging", "Creating staging environment")

		// User writes different files in each environment
		user.FileWrite(dev.ID, "config.json", `{"env": "dev", "debug": true}`, "Dev config")
		user.FileWrite(staging.ID, "config.json", `{"env": "staging", "debug": false}`, "Staging config")

		// Verify each environment has its own config
		devConfig := user.FileRead(dev.ID, "config.json")
		assert.Contains(t, devConfig, `"env": "dev"`)
		assert.Contains(t, devConfig, `"debug": true`)

		stagingConfig := user.FileRead(staging.ID, "config.json")
		assert.Contains(t, stagingConfig, `"env": "staging"`)
		assert.Contains(t, stagingConfig, `"debug": false`)

		// Get fresh environments to access Worktree paths for git commands
		dev = user.GetEnvironment(dev.ID)
		staging = user.GetEnvironment(staging.ID)

		// Verify git histories are independent
		devLog, _ := repository.RunGitCommand(context.Background(), user.WorktreePath(dev.ID), "log", "--oneline", "-2")
		stagingLog, _ := repository.RunGitCommand(context.Background(), user.WorktreePath(staging.ID), "log", "--oneline", "-2")

		assert.Contains(t, devLog, "Dev config")
		assert.Contains(t, stagingLog, "Staging config")

		// Verify complete isolation - dev files don't exist in staging
		user.FileReadExpectError(staging.ID, "dev-only.txt")
		user.FileWrite(dev.ID, "dev-only.txt", "Only in dev", "Dev only file")

		// Staging still shouldn't see dev files
		user.FileReadExpectError(staging.ID, "dev-only.txt")
	})
}

// TestSystemHandlesProblematicFiles verifies edge cases don't break the system
func TestSystemHandlesProblematicFiles(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Verify Python __pycache__ directories don't interfere with operations
	t.Run("PythonDevelopmentWorkflow", func(t *testing.T) {
		WithRepository(t, "python_cache", SetupPythonRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Python Dev", "Testing Python cache handling")

			// Simulate Python cache directories
			output := user.RunCommand(env.ID,
				"mkdir -p __pycache__ && "+
					"echo 'binary content' > __pycache__/main.cpython-311.pyc && "+
					"echo 'binary content' > __pycache__/utils.cpython-311.pyc",
				"Simulate Python cache")
			_ = output

			// Continue development
			user.FileWrite(env.ID, "feature.py", "def new_feature():\n    return True", "Add feature")
			user.FileWrite(env.ID, "main.py", "# Updated\nprint('Hello, Updated World!')", "Update main")

			// Verify __pycache__ doesn't interfere

			// Create more files to ensure continued functionality
			user.RunCommand(env.ID, "touch __pycache__/feature.cpython-311.pyc", "Create more cache")

			// Verify we can still read files
			content := user.FileRead(env.ID, "feature.py")
			assert.Contains(t, content, "new_feature")
		})
	})

	t.Run("BinaryDirectories", func(t *testing.T) {
		WithRepository(t, "binary_dirs", SetupPythonRepoNoGitignore, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Test", "Testing binary directory handling")

			// Create directories with only binary files
			_ = user.RunCommand(env.ID,
				"mkdir -p __pycache__ && "+
					"dd if=/dev/urandom of=__pycache__/main.cpython-39.pyc bs=1024 count=1 2>/dev/null && "+
					"dd if=/dev/urandom of=__pycache__/utils.cpython-39.pyc bs=1024 count=1 2>/dev/null",
				"Create binary directory")

			// Should still handle text files
			user.FileWrite(env.ID, "notes.txt", "System should handle binary directories gracefully", "Add text file")

			// Verify the text file was written and can be read
			content := user.FileRead(env.ID, "notes.txt")
			assert.Equal(t, "System should handle binary directories gracefully", content)
		})
	})

	t.Run("LargeFiles", func(t *testing.T) {
		WithRepository(t, "large_files", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Large Files", "Testing large file handling")

			// Create a large file
			output := user.RunCommand(env.ID,
				"dd if=/dev/urandom of=large.dat bs=1M count=5 2>/dev/null",
				"Create large file")

			// System should handle large files
			_ = output

			// Should still work with normal files
			user.FileWrite(env.ID, "config.json", `{"maxFileSize": "5MB"}`, "Add config")

			// Verify we can read the config
			content := user.FileRead(env.ID, "config.json")
			assert.Contains(t, content, "maxFileSize")
		})
	})
}

// Large project performance ensures the system scales to real-world codebases
func TestLargeProjectPerformance(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping performance test")
	}

	t.Run("large_project_performance", func(t *testing.T) {
		// Create many files for performance testing
		largeProjectSetup := func(t *testing.T, repoDir string) {
			// Create 100 test files
			for i := range 100 {
				writeFile(t, repoDir, filepath.Join("src", fmt.Sprintf("file%d.js", i)),
					fmt.Sprintf("// File %d\nconsole.log('test');", i))
			}
			gitCommit(t, repoDir, "Large project")
		}

		WithRepository(t, "large_project_performance", largeProjectSetup, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Performance Test", "Testing performance with large project")

			// Time file operations
			start := time.Now()
			user.FileWrite(env.ID, "new.txt", "test", "Test write performance")
			writeTime := time.Since(start)

			t.Logf("File write took: %v", writeTime)

			assert.LessOrEqual(t, writeTime, 2*time.Second, "File write should be fast")
		})
	})
}

// TestWorktreeUpdatesAreVisibleAfterRebuild verifies file changes persist through rebuilds
func TestWorktreeUpdatesAreVisibleAfterRebuild(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("worktree_cache", func(t *testing.T) {
		WithRepository(t, "worktree_cache", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			env := user.CreateEnvironment("Worktree Test", "Testing worktree updates after rebuild")

			initialScript := `echo "Version 1"`
			user.FileWrite(env.ID, "script.sh", initialScript, "Create script")

			// Initial version
			output := user.RunCommand(env.ID, "sh script.sh", "Run initial version")
			assert.Contains(t, output, "Version 1")

			// Update script
			updatedScript := `echo "Version 2"`
			user.FileWrite(env.ID, "script.sh", updatedScript, "Update script")

			// Rebuild environment
			env = user.GetEnvironment(env.ID)

			// Update config to force rebuild
			config := env.State.Config.Copy()
			user.UpdateEnvironment(env.ID, env.State.Title, "Force rebuild", config)

			// Check script after rebuild
			catOutput := user.RunCommand(env.ID, "cat script.sh", "Check script content")
			t.Logf("Script content after rebuild: %s", catOutput)

			// Version 2 should be active
			output = user.RunCommand(env.ID, "sh script.sh", "Run after rebuild")
			assert.Contains(t, output, "Version 2", "Updated version should be used after rebuild")
		})
	})
}

// TestWeirdUserScenarios verifies edge case handling
func TestWeirdUserScenarios(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("EnvironmentNameCollisions", func(t *testing.T) {
		WithRepository(t, "name_collisions", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			// Create first environment
			env1 := user.CreateEnvironment("My App", "Creating first app environment")

			// Create second environment with SAME name
			env2 := user.CreateEnvironment("My App", "Creating second app environment")

			// They should have different IDs despite same name
			assert.NotEqual(t, env1.ID, env2.ID, "Same-named environments should get unique IDs")
			// IDs are generated using random pet names, not derived from the environment name
			assert.NotEmpty(t, env1.ID, "ID should not be empty")
			assert.NotEmpty(t, env2.ID, "ID should not be empty")

			// Both should be independently accessible
			ctx := context.Background()
			retrieved1, err := repo.Get(ctx, user.dag, env1.ID)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved1, "First env should be retrievable")

			retrieved2, err := repo.Get(ctx, user.dag, env2.ID)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved2, "Second env should be retrievable")

			// Write different content to verify isolation
			user.FileWrite(env1.ID, "app.txt", "Environment 1", "Write to env1")
			user.FileWrite(env2.ID, "app.txt", "Environment 2", "Write to env2")

			// Verify content
			content1 := user.FileRead(env1.ID, "app.txt")
			content2 := user.FileRead(env2.ID, "app.txt")

			assert.Equal(t, "Environment 1", content1)
			assert.Equal(t, "Environment 2", content2)
		})
	})

	t.Run("OrphanedWorktreeRecovery", func(t *testing.T) {
		WithRepository(t, "orphaned_worktree", SetupPythonRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			newEnv := user.CreateEnvironment("Test", "Creating test environment")

			// Save worktree path for later verification
			envID := newEnv.ID
			worktreePath := user.WorktreePath(newEnv.ID)

			// Simulate corruption by removing the .git directory in the worktree
			user.CorruptWorktree(newEnv.ID)

			// Verify worktree still exists on disk but is corrupted
			_, err := os.Stat(worktreePath)
			assert.NoError(t, err, "Worktree should still exist")

			gitDir := filepath.Join(worktreePath, ".git")
			_, err = os.Stat(gitDir)
			assert.Error(t, err, ".git should be removed")

			// Try to create new environment with same name - should work
			env2 := user.CreateEnvironment("Test", "Creating test environment after orphan")

			// New environment should have different ID and worktree
			assert.NotEqual(t, envID, env2.ID)
			assert.NotEqual(t, worktreePath, user.WorktreePath(env2.ID))

			// New environment should be functional
			user.FileWrite(env2.ID, "test.txt", "New environment works", "Verify new env works")
			content := user.FileRead(env2.ID, "test.txt")
			assert.Equal(t, "New environment works", content)
		})
	})

	t.Run("CrossRepositoryConfusion", func(t *testing.T) {
		initializeDaggerOnce(t)

		// Create two separate repositories
		ctx := context.Background()

		// Create first repository
		repoDir1, err := os.MkdirTemp("", "cu-test-repo1-*")
		require.NoError(t, err)
		defer os.RemoveAll(repoDir1)

		configDir1, err := os.MkdirTemp("", "cu-test-config1-*")
		require.NoError(t, err)
		defer os.RemoveAll(configDir1)

		// Initialize git repo1
		cmds := [][]string{
			{"init"},
			{"config", "user.email", "test@example.com"},
			{"config", "user.name", "Test User"},
			{"config", "commit.gpgsign", "false"},
		}
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir1, cmd...)
			require.NoError(t, err)
		}
		SetupNodeRepo(t, repoDir1)

		// Create second repository
		repoDir2, err := os.MkdirTemp("", "cu-test-repo2-*")
		require.NoError(t, err)
		defer os.RemoveAll(repoDir2)

		configDir2, err := os.MkdirTemp("", "cu-test-config2-*")
		require.NoError(t, err)
		defer os.RemoveAll(configDir2)

		// Initialize git repo2
		for _, cmd := range cmds {
			_, err := repository.RunGitCommand(ctx, repoDir2, cmd...)
			require.NoError(t, err)
		}
		SetupPythonRepo(t, repoDir2)

		// Open repository and create environment in repo1
		repo1, err := repository.OpenWithBasePath(ctx, repoDir1, configDir1)
		require.NoError(t, err)

		env1, err := repo1.Create(ctx, testDaggerClient, "App", "Creating app in repo1")
		require.NoError(t, err)
		defer repo1.Delete(ctx, env1.ID)

		// Write file in env1
		err = env1.FileWrite(ctx, "Add file", "app.js", "console.log('repo1');")
		require.NoError(t, err)

		// Try to use env1 while in repo2 (should fail)
		_, err = env1.FileRead(ctx, "main.py", true, 0, 0)
		assert.Error(t, err, "Should fail to read repo2 files from repo1 environment")

		// The environment is still tied to repo1
		jsContent, err := env1.FileRead(ctx, "app.js", true, 0, 0)
		require.NoError(t, err)
		assert.Contains(t, jsContent, "repo1", "Environment should still access its original repo")
	})
}

// TestEnvironmentConfigurationPersists verifies configuration persistence
func TestEnvironmentConfigurationPersists(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("BaseImagePersists", func(t *testing.T) {
		WithRepository(t, "base_image", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			newEnv := user.CreateEnvironment("Test environment", "Creating Alpine-based test environment")

			// Update to Alpine with git
			updatedConfig := newEnv.State.Config.Copy()
			updatedConfig.BaseImage = "alpine:latest"
			updatedConfig.SetupCommands = []string{"apk add --no-cache git"}

			user.UpdateEnvironment(newEnv.ID, "Test environment", "Use Alpine Linux", updatedConfig)

			// Save and reload config
			newEnv = user.GetEnvironment(newEnv.ID)
			newConfig := newEnv.State.Config.Copy()

			assert.Equal(t, "alpine:latest", newConfig.BaseImage, "Base image should persist")
			assert.Equal(t, []string{"apk add --no-cache git"}, newConfig.SetupCommands, "Setup commands should persist")
		})
	})

	t.Run("SetupCommandsPersist", func(t *testing.T) {
		WithRepository(t, "setup_commands", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			newEnv := user.CreateEnvironment("Test with setup", "Creating environment with setup commands")

			setupCmds := []string{
				"apk add --no-cache curl git",
				"echo 'Setup complete' > /setup.log",
			}
			updatedConfig := newEnv.State.Config.Copy()
			updatedConfig.BaseImage = "alpine:latest"
			updatedConfig.SetupCommands = setupCmds

			user.UpdateEnvironment(newEnv.ID, "Test with setup", "Install development tools", updatedConfig)

			// Reload config
			newEnv = user.GetEnvironment(newEnv.ID)
			newConfig := newEnv.State.Config.Copy()
			assert.Equal(t, setupCmds, newConfig.SetupCommands, "Setup commands should persist")
		})
	})

	t.Run("InstallCommandsPersist", func(t *testing.T) {
		WithRepository(t, "install_commands", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			newEnv := user.CreateEnvironment("Test with install", "Creating environment with install commands")

			installCmds := []string{
				"npm install --save-dev jest",
				"echo 'Dependencies installed' > /install.log",
			}
			updatedConfig := newEnv.State.Config.Copy()
			updatedConfig.BaseImage = "node:18"
			updatedConfig.InstallCommands = installCmds

			user.UpdateEnvironment(newEnv.ID, "Test with install", "Install project dependencies", updatedConfig)

			// Reload config
			newEnv = user.GetEnvironment(newEnv.ID)
			newConfig := newEnv.State.Config.Copy()
			assert.Equal(t, installCmds, newConfig.InstallCommands, "Install commands should persist")
		})
	})

	t.Run("EnvironmentVariable", func(t *testing.T) {
		t.Run("Persistence", func(t *testing.T) {
			WithRepository(t, "envvar_persistence", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
				// User: "Create a Node.js development environment"
				env := user.CreateEnvironment("Node.js Dev", "Setting up Node.js development environment")
				envID := env.ID // LLM only keeps the ID, not the whole object

				// User: "I need to set environment variables for my API"
				// LLM has to provide ALL required fields because the tool requires them
				user.UpdateEnvironment(envID, "Node.js Dev", "Configure API environment variables", &environment.EnvironmentConfig{
					BaseImage:     "ubuntu:24.04",
					SetupCommands: []string{},
					Workdir:       "/workdir",
					Env: []string{
						"API_URL=https://api.example.com",
						"NODE_ENV=production",
						"PORT=3000",
					},
					Secrets: []string{},
				})

				// User: "Check if my environment variables are set"
				output := user.RunCommand(envID, "echo API_URL=$API_URL NODE_ENV=$NODE_ENV PORT=$PORT", "Verify env vars")
				assert.Contains(t, output, "API_URL=https://api.example.com")
				assert.Contains(t, output, "NODE_ENV=production")
				assert.Contains(t, output, "PORT=3000")

				// User: "Add a simple setup command"
				// Again, LLM must provide ALL fields, potentially losing env vars if not careful
				user.UpdateEnvironment(envID, "Node.js Dev", "Add setup command", &environment.EnvironmentConfig{
					BaseImage: "ubuntu:24.04",
					SetupCommands: []string{
						"echo 'Setup complete' > /tmp/setup.log",
					},
					Workdir: "/workdir",
					Env: []string{
						"API_URL=https://api.example.com",
						"NODE_ENV=production",
						"PORT=3000",
					},
					Secrets: []string{},
				})

				// User: "Are my environment variables still there?"
				output = user.RunCommand(envID, "echo API_URL=$API_URL", "Check API_URL after rebuild")
				assert.Contains(t, output, "API_URL=https://api.example.com")

				output = user.RunCommand(envID, "echo NODE_ENV=$NODE_ENV", "Check NODE_ENV after rebuild")
				assert.Contains(t, output, "NODE_ENV=production")

				output = user.RunCommand(envID, "echo PORT=$PORT", "Check PORT after rebuild")
				assert.Contains(t, output, "PORT=3000")
			})
		})

		t.Run("Loss", func(t *testing.T) {
			// This test shows what happens when an LLM forgets to include env vars
			WithRepository(t, "envvar_loss", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
				// User: "Create a Node.js environment with env vars"
				env := user.CreateEnvironment("Node.js API", "Create Node.js API environment")
				envID := env.ID

				// User: "Set up my API environment variables"
				user.UpdateEnvironment(envID, "Node.js API", "Configure environment", &environment.EnvironmentConfig{
					BaseImage:     "ubuntu:24.04",
					SetupCommands: []string{},
					Workdir:       "/workdir",
					Env: []string{
						"DATABASE_URL=postgres://localhost:5432/mydb",
						"REDIS_URL=redis://localhost:6379",
						"API_KEY=secret123",
					},
					Secrets: []string{},
				})

				// Verify env vars are set
				output := user.RunCommand(envID, "echo DATABASE_URL=$DATABASE_URL", "Check database URL")
				assert.Contains(t, output, "DATABASE_URL=postgres://localhost:5432/mydb")

				// User: "Add a marker file"
				// LLM forgets to include the env vars when updating!
				user.UpdateEnvironment(envID, "Node.js API", "Add marker file", &environment.EnvironmentConfig{
					BaseImage: "ubuntu:24.04",
					SetupCommands: []string{
						"touch /tmp/marker.txt",
					},
					Workdir: "/workdir",
					Env:     []string{}, // Oops! LLM forgot to include existing env vars
					Secrets: []string{},
				})

				// Check if env vars are lost
				output = user.RunCommand(envID, "echo DATABASE_URL=$DATABASE_URL", "Check if database URL survived")
				assert.NotContains(t, output, "postgres://localhost:5432/mydb", "Environment variables were lost!")
				assert.Equal(t, "DATABASE_URL=\n", output, "DATABASE_URL should be empty")
			})
		})
	})

	t.Run("LifecycleOperations", func(t *testing.T) {
		WithRepository(t, "lifecycle", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Test Create
			newEnv := user.CreateEnvironment("Test lifecycle", "Creating environment for lifecycle testing")
			require.NotNil(t, newEnv)

			envID := newEnv.ID
			originalWorktree := user.WorktreePath(newEnv.ID)

			// Environment is registered
			retrieved, err := repo.Get(ctx, user.dag, envID)
			assert.NoError(t, err)
			assert.NotNil(t, retrieved, "Environment should be retrievable")

			// Worktree at predictable location
			assert.Contains(t, originalWorktree, envID, "Worktree path should contain environment ID")

			// Test Update with Alpine base image
			setupCmds := []string{"apk add --no-cache git nodejs npm"}
			updatedConfig := newEnv.State.Config.Copy()
			updatedConfig.BaseImage = "alpine:latest"
			updatedConfig.SetupCommands = setupCmds

			user.UpdateEnvironment(newEnv.ID, "Test lifecycle", "Install development tools", updatedConfig)

			// Setup command executed
			output := user.RunCommand(newEnv.ID, "node --version", "Check node installed")
			assert.Contains(t, output, "v", "Node should be installed")

			// Worktree location stable
			assert.Equal(t, originalWorktree, user.WorktreePath(newEnv.ID), "Worktree location should not change")

			// Test Delete
			err = repo.Delete(ctx, envID)
			require.NoError(t, err, "Should delete environment")

			// Verify cleanup
			_, err = repo.Get(ctx, user.dag, envID)
			assert.Error(t, err, "Environment should not be retrievable after deletion")

			// Worktree deleted
			_, err = os.Stat(user.WorktreePath(envID))
			assert.True(t, os.IsNotExist(err), "Worktree should be deleted")
		})
	})
}
