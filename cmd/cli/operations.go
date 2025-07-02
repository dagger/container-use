// Package cli contains the core logic for container-use CLI commands.
package cli

import (
	"context"
	"fmt"

	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
)

// DeleteEnvironments deletes one or more environments.
func DeleteEnvironments(ctx context.Context, repoPath string, envIDs []string) error {
	for _, envID := range envIDs {
		repo, err := repository.Open(ctx, repoPath)
		if err != nil {
			return fmt.Errorf("failed to open repository: %w", err)
		}
		if err := repo.Delete(ctx, envID); err != nil {
			return fmt.Errorf("failed to delete environment: %w", err)
		}
		fmt.Printf("Environment '%s' deleted successfully.\n", envID)
	}
	return nil
}

// ListEnvironments returns all environments in the repository.
func ListEnvironments(ctx context.Context, repoPath string) ([]*environment.EnvironmentInfo, error) {
	repo, err := repository.Open(ctx, repoPath)
	if err != nil {
		return nil, err
	}
	return repo.List(ctx)
}

// CheckoutEnvironment switches to the git branch for an environment.
func CheckoutEnvironment(ctx context.Context, repoPath string, envID, branch string) (string, error) {
	repo, err := repository.Open(ctx, repoPath)
	if err != nil {
		return "", err
	}
	return repo.Checkout(ctx, envID, branch)
}

// GetEnvironmentLog returns git log for an environment (test helper).
func GetEnvironmentLog(ctx context.Context, repoPath string, envID string) (string, error) {
	if _, err := repository.Open(ctx, repoPath); err != nil {
		return "", err
	}

	ref := fmt.Sprintf("container-use/%s", envID)
	return repository.RunGitCommand(ctx, repoPath, "log", "--oneline", "-10", ref)
}

// MergeEnvironment performs a non-interactive merge (test helper).
func MergeEnvironment(ctx context.Context, repoPath string, envID string) error {
	repo, err := repository.Open(ctx, repoPath)
	if err != nil {
		return err
	}

	envs, err := repo.List(ctx)
	if err != nil {
		return err
	}

	found := false
	for _, env := range envs {
		if env.ID == envID {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("environment %s not found", envID)
	}

	ref := fmt.Sprintf("container-use/%s", envID)
	_, err = repository.RunGitCommand(ctx, repoPath, "merge", "--no-edit", ref)
	return err
}
