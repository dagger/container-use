package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnvironmentSelection tests the environment selection logic
func TestEnvironmentSelection(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	t.Run("SingleDescendantEnvironment", func(t *testing.T) {
		WithRepository(t, "single-descendant", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Get current HEAD
			currentHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create an environment (this creates a branch from current HEAD and adds commits)
			env := user.CreateEnvironment("Test Environment", "Testing single descendant environment")

			// List descendant environments
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 1)
			assert.Equal(t, env.ID, descendantEnvs[0].ID)
			assert.Equal(t, "Test Environment", descendantEnvs[0].State.Title)
		})
	})

	t.Run("MultipleDescendantEnvironments", func(t *testing.T) {
		WithRepository(t, "multiple-descendants", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Get current HEAD
			currentHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			currentHead = strings.TrimSpace(currentHead)

			// Create multiple environments
			env1 := user.CreateEnvironment("First Environment", "Testing multiple descendants")
			env2 := user.CreateEnvironment("Second Environment", "Testing multiple descendants")

			// List descendant environments
			descendantEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
			require.NoError(t, err)
			assert.Len(t, descendantEnvs, 2)

			// Check that both environments are present (they're sorted by update time)
			envIDs := []string{descendantEnvs[0].ID, descendantEnvs[1].ID}
			assert.Contains(t, envIDs, env1.ID)
			assert.Contains(t, envIDs, env2.ID)
		})
	})

	t.Run("NoDescendantEnvironments", func(t *testing.T) {
		WithRepository(t, "no-descendants", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Create an environment first
			env := user.CreateEnvironment("Test Environment", "Testing no descendants")

			// Make a divergent commit on the main branch
			user.GitCommand("commit", "--allow-empty", "-m", "Divergent commit")

			// Get the new HEAD
			newHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
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

	t.Run("EnvironmentsSortedByUpdateTime", func(t *testing.T) {
		WithRepository(t, "sorted-envs", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Get current HEAD
			currentHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
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

	t.Run("MixedEnvironmentAncestry", func(t *testing.T) {
		WithRepository(t, "mixed-ancestry", SetupNodeRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
			ctx := context.Background()

			// Get initial HEAD
			initialHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			initialHead = strings.TrimSpace(initialHead)

			// Create first environment from initial HEAD
			user.CreateEnvironment("Environment 1", "Created from initial HEAD")

			// Make a commit on main
			user.GitCommand("commit", "--allow-empty", "-m", "New commit on main")

			// Get new HEAD
			newHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
			require.NoError(t, err)
			newHead = strings.TrimSpace(newHead)

			// Create second environment from new HEAD
			env2 := user.CreateEnvironment("Environment 2", "Created from new HEAD")

			// List descendants from initial HEAD - should include both environments
			descendantsFromInitial, err := repo.ListDescendantEnvironments(ctx, initialHead)
			require.NoError(t, err)
			assert.Len(t, descendantsFromInitial, 2) // Both environments are descendants of initial HEAD

			// List descendants from new HEAD - should only include env2
			descendantsFromNew, err := repo.ListDescendantEnvironments(ctx, newHead)
			require.NoError(t, err)
			assert.Len(t, descendantsFromNew, 1)
			assert.Equal(t, env2.ID, descendantsFromNew[0].ID)
		})
	})
}
