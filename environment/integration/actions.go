package integration

import (
	"context"
	"testing"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// UserActions provides test helpers that mirror MCP tool behavior exactly
// These represent what a user would experience when using the MCP tools
type UserActions struct {
	t         *testing.T
	ctx       context.Context
	repo      *repository.Repository
	dag       *dagger.Client
	repoDir   string // Source directory (for direct manipulation)
	configDir string // Container-use config directory
}

func NewUserActions(t *testing.T, repo *repository.Repository, dag *dagger.Client) *UserActions {
	return &UserActions{
		t:    t,
		ctx:  context.Background(),
		repo: repo,
		dag:  dag,
	}
}

// WithDirectAccess adds direct filesystem access for edge case testing
func (u *UserActions) WithDirectAccess(repoDir, configDir string) *UserActions {
	u.repoDir = repoDir
	u.configDir = configDir
	return u
}

// FileWrite mirrors environment_file_write MCP tool behavior
func (u *UserActions) FileWrite(envID, targetFile, contents, explanation string) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	err = env.FileWrite(u.ctx, explanation, targetFile, contents)
	require.NoError(u.t, err, "FileWrite should succeed")

	err = u.repo.Update(u.ctx, env, explanation)
	require.NoError(u.t, err, "repo.Update after FileWrite should succeed")
}

// RunCommand mirrors environment_run_cmd MCP tool behavior
func (u *UserActions) RunCommand(envID, command, explanation string) string {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	output, err := env.Run(u.ctx, command, "/bin/sh", false)
	require.NoError(u.t, err, "Run command should succeed")

	err = u.repo.Update(u.ctx, env, explanation)
	require.NoError(u.t, err, "repo.Update after Run should succeed")

	return output
}

// CreateEnvironment mirrors environment_create MCP tool behavior
func (u *UserActions) CreateEnvironment(title, explanation string) *environment.Environment {
	env, err := u.repo.Create(u.ctx, u.dag, title, explanation, "HEAD")
	require.NoError(u.t, err, "Create environment should succeed")
	return env
}

// UpdateEnvironment mirrors environment_update MCP tool behavior
func (u *UserActions) UpdateEnvironment(envID, title, explanation string, config *environment.EnvironmentConfig) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	if title != "" {
		env.State.Title = title
	}

	err = env.UpdateConfig(u.ctx, config)
	require.NoError(u.t, err, "UpdateConfig should succeed")

	err = u.repo.Update(u.ctx, env, explanation)
	require.NoError(u.t, err, "repo.Update after UpdateConfig should succeed")
}

// FileDelete mirrors environment_file_delete MCP tool behavior
func (u *UserActions) FileDelete(envID, targetFile, explanation string) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	err = env.FileDelete(u.ctx, explanation, targetFile)
	require.NoError(u.t, err, "FileDelete should succeed")

	err = u.repo.Update(u.ctx, env, explanation)
	require.NoError(u.t, err, "repo.Update after FileDelete should succeed")
}

// FileRead mirrors environment_file_read MCP tool behavior (read-only, no update)
func (u *UserActions) FileRead(envID, targetFile string) string {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	content, err := env.FileRead(u.ctx, targetFile, true, 0, 0)
	require.NoError(u.t, err, "FileRead should succeed")
	return content
}

// FileReadExpectError is for testing expected failures
func (u *UserActions) FileReadExpectError(envID, targetFile string) {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Failed to get environment %s", envID)

	_, err = env.FileRead(u.ctx, targetFile, true, 0, 0)
	assert.Error(u.t, err, "FileRead should fail for %s", targetFile)
}

// GetEnvironment retrieves an environment by ID - mirrors how MCP tools work
// Each MCP tool call starts fresh by getting the environment from the repository
func (u *UserActions) GetEnvironment(envID string) *environment.Environment {
	env, err := u.repo.Get(u.ctx, u.dag, envID)
	require.NoError(u.t, err, "Should be able to get environment %s", envID)
	return env
}
