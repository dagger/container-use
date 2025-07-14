package repository

import (
	"context"
	"strings"
	"testing"

	"github.com/dagger/container-use/environment/integration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListDescendantEnvironments(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("SingleDescendantEnvironment", func(t *testing.T) {
		integration.WithRepository(t, "single-descendant", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Get current HEAD
			ctx := context.Background()
			currentHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create an environment (this creates a branch from current HEAD and adds commits)
			env := user.CreateEnvironment("Test Environment", "Testing descendant environment selection")

			// List descendant environments
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 1)
			assert.Equal(t, env.ID, descendantEnvs[0].ID)
		})
	})

	t.Run("MultipleDescendantEnvironments", func(t *testing.T) {
		integration.WithRepository(t, "multiple-descendants", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Get current HEAD
			ctx := context.Background()
			currentHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create multiple environments
			env1 := user.CreateEnvironment("Test Environment 1", "Testing multiple descendant environments")
			env2 := user.CreateEnvironment("Test Environment 2", "Testing multiple descendant environments")

			// List descendant environments
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 2)

			// Check that both environments are present
			envIDs := []string{descendantEnvs[0].ID, descendantEnvs[1].ID}
			assert.Contains(t, envIDs, env1.ID)
			assert.Contains(t, envIDs, env2.ID)
		})
	})

	t.Run("NoDescendantEnvironments", func(t *testing.T) {
		integration.WithRepository(t, "no-descendants", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Create an environment first
			env := user.CreateEnvironment("Test Environment", "Testing no descendant environments")

			// Make a divergent commit on the main branch
			ctx := context.Background()
			user.GitCommand("commit", "--allow-empty", "-m", "Divergent commit")

			// Get the new HEAD
			newHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			newHead = strings.TrimSpace(newHead)

			// List descendant environments from the new HEAD
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, newHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 0)

			// Verify that the environment still exists but is not a descendant
			allEnvs, err := repo.List(ctx)
			require.NoError(t, err)
			assert.Len(t, allEnvs, 1)
			assert.Equal(t, env.ID, allEnvs[0].ID)
		})
	})

	t.Run("SortedByMostRecentlyUpdated", func(t *testing.T) {
		integration.WithRepository(t, "sorted-descendants", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Get current HEAD
			ctx := context.Background()
			currentHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create environments with some time between them
			env1 := user.CreateEnvironment("First Environment", "Creating first environment")
			env2 := user.CreateEnvironment("Second Environment", "Creating second environment")

			// Update the first environment to make it more recent
			user.FileWrite(env1.ID, "update.txt", "Updated content", "Update first environment")

			// List descendant environments
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 2)

			// First environment should be first (most recently updated)
			assert.Equal(t, env1.ID, descendantEnvs[0].ID)
			assert.Equal(t, env2.ID, descendantEnvs[1].ID)
		})
	})
}

func TestIsDescendantOfCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("EnvironmentIsDescendant", func(t *testing.T) {
		integration.WithRepository(t, "is-descendant", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Get current HEAD
			ctx := context.Background()
			currentHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create an environment
			env := user.CreateEnvironment("Test Environment", "Testing descendant check")

			// Check if environment is descendant of current HEAD
			isDescendant := repo.isDescendantOfCommit(ctx, currentHead, env.ID)
			assert.True(t, isDescendant)
		})
	})

	t.Run("EnvironmentIsNotDescendant", func(t *testing.T) {
		integration.WithRepository(t, "not-descendant", integration.SetupNodeRepo, func(t *testing.T, repo *Repository, user *integration.UserActions) {
			// Create an environment from current HEAD
			env := user.CreateEnvironment("Test Environment", "Testing non-descendant check")

			// Make a new commit on the main branch
			ctx := context.Background()
			user.GitCommand("commit", "--allow-empty", "-m", "New commit on main")

			// Get the new HEAD
			newHead, err := RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			newHead = strings.TrimSpace(newHead)

			// Check if environment is descendant of new HEAD (it shouldn't be)
			isDescendant := repo.isDescendantOfCommit(ctx, newHead, env.ID)
			assert.False(t, isDescendant)
		})
	})
}