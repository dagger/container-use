package main

import (
	"context"
	"os"
	"testing"

	"github.com/dagger/container-use/environment/integration"
	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveEnvironmentID(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("WithProvidedArgs", func(t *testing.T) {
		// When args are provided, should return the first arg directly
		ctx := context.Background()
		args := []string{"test-env"}

		// Don't need a real repository for this test
		envID, err := resolveEnvironmentID(ctx, nil, args)
		require.NoError(t, err)
		assert.Equal(t, "test-env", envID)
	})

	t.Run("NoEnvironments", func(t *testing.T) {
		integration.WithRepository(t, "no-envs", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Should return error when no environments exist
			_, err := resolveEnvironmentID(context.Background(), repo, []string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no environments found")
		})
	})

	t.Run("SingleMatchingEnvironment", func(t *testing.T) {
		integration.WithRepository(t, "single-env", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Create an environment that is a descendant of current HEAD
			env := user.CreateEnvironment("Test Environment", "Testing single environment selection")
			
			// Should auto-select the single environment
			envID, err := resolveEnvironmentID(context.Background(), repo, []string{})
			require.NoError(t, err)
			assert.Equal(t, env.ID, envID)
		})
	})

	t.Run("MultipleMatchingEnvironments", func(t *testing.T) {
		integration.WithRepository(t, "multi-env", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Create multiple environments that are descendants of current HEAD
			env1 := user.CreateEnvironment("Test Environment 1", "Testing multiple environment selection")
			env2 := user.CreateEnvironment("Test Environment 2", "Testing multiple environment selection")
			
			// Since we can't test interactive prompts, we'll just verify that the function
			// correctly identifies multiple environments (it would normally prompt the user)
			envs, err := repo.List(context.Background())
			require.NoError(t, err)
			assert.Len(t, envs, 2)
			
			// Check that both environments are descendants of current HEAD
			currentHead, err := repository.RunGitCommand(context.Background(), repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = currentHead[:len(currentHead)-1] // Remove newline
			
			isDescendant1 := isDescendantOfHead(context.Background(), repo, currentHead, env1.ID)
			isDescendant2 := isDescendantOfHead(context.Background(), repo, currentHead, env2.ID)
			assert.True(t, isDescendant1)
			assert.True(t, isDescendant2)
		})
	})

	t.Run("NoMatchingEnvironments", func(t *testing.T) {
		integration.WithRepository(t, "no-matching", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Create an environment
			env := user.CreateEnvironment("Test Environment", "Testing no matching environments")
			
			// Make a divergent commit on the main branch to create a scenario where
			// the environment is not a descendant of current HEAD
			user.GitCommand("commit", "--allow-empty", "-m", "Divergent commit")
			
			// The environment should not be considered a descendant anymore
			currentHead, err := repository.RunGitCommand(context.Background(), repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = currentHead[:len(currentHead)-1] // Remove newline
			
			isDescendant := isDescendantOfHead(context.Background(), repo, currentHead, env.ID)
			assert.False(t, isDescendant)
			
			// Should return error when no environments match
			_, err = resolveEnvironmentID(context.Background(), repo, []string{})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "no environments found that are descendants of the current HEAD")
		})
	})
}

func TestIsDescendantOfHead(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("EnvironmentIsDescendant", func(t *testing.T) {
		integration.WithRepository(t, "descendant-test", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Get current HEAD
			currentHead, err := repository.RunGitCommand(context.Background(), repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = currentHead[:len(currentHead)-1] // Remove newline
			
			// Create an environment (this creates a branch from current HEAD and adds commits)
			env := user.CreateEnvironment("Test Environment", "Testing descendant relationship")
			
			// The environment should be a descendant of the original HEAD
			isDescendant := isDescendantOfHead(context.Background(), repo, currentHead, env.ID)
			assert.True(t, isDescendant)
		})
	})

	t.Run("EnvironmentIsNotDescendant", func(t *testing.T) {
		integration.WithRepository(t, "not-descendant-test", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Create an environment from current HEAD
			env := user.CreateEnvironment("Test Environment", "Testing non-descendant relationship")
			
			// Make a new commit on the main branch
			user.GitCommand("commit", "--allow-empty", "-m", "New commit on main")
			
			// Get the new HEAD
			newHead, err := repository.RunGitCommand(context.Background(), repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			newHead = newHead[:len(newHead)-1] // Remove newline
			
			// The environment should NOT be a descendant of the new HEAD
			isDescendant := isDescendantOfHead(context.Background(), repo, newHead, env.ID)
			assert.False(t, isDescendant)
		})
	})
}

func TestEnvironmentSelectionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("LogCommandWithoutArgs", func(t *testing.T) {
		integration.WithRepository(t, "log-test", integration.SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *integration.UserActions) {
			// Create an environment
			env := user.CreateEnvironment("Test Environment", "Testing log command without args")
			
			// Add some content to make the log more interesting
			user.FileWrite(env.ID, "test.txt", "Hello World", "Add test file")
			
			// Test that we can resolve the environment ID without explicit args
			envID, err := resolveEnvironmentID(context.Background(), repo, []string{})
			require.NoError(t, err)
			assert.Equal(t, env.ID, envID)
		})
	})
}