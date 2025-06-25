package integration

import (
	"context"
	"os"
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
		assert.NotEmpty(t, env.Worktree)
		
		// Verify worktree was created
		_, err := os.Stat(env.Worktree)
		assert.NoError(t, err)
	})
}

// TestRepositoryGet tests retrieving an existing environment
func TestRepositoryGet(t *testing.T) {
	t.Parallel()
	WithRepository(t, "repository-get", SetupEmptyRepo, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		ctx := context.Background()
		
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
		ctx := context.Background()
		
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
		ctx := context.Background()
		
		// Create an environment
		env := user.CreateEnvironment("Test Delete", "Testing repository delete")
		worktreePath := env.Worktree
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
		ctx := context.Background()
		
		// Create an environment and add content
		env := user.CreateEnvironment("Test Checkout", "Testing repository checkout")
		user.FileWrite(env.ID, "test.txt", "test content", "Add test file")
		
		// Checkout the environment branch in the source repo
		branch, err := repo.Checkout(ctx, env.ID)
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