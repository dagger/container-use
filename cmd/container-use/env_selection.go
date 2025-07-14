package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/dagger/container-use/environment"
	"github.com/dagger/container-use/repository"
)

// resolveEnvironmentID resolves the environment ID for commands that take env_id as the only positional argument.
// If no args are provided, it filters environments to those where the local repo head is a parent of the environment's head,
// then either auto-selects if there's only one match or prompts the user to select from multiple options.
func resolveEnvironmentID(ctx context.Context, repo *repository.Repository, args []string) (string, error) {
	if len(args) == 1 {
		return args[0], nil
	}
	if len(args) > 1 {
		return "", errors.New("too many arguments")
	}

	// Get current user repo head - this could easily go inside ListDescendantEnvironments, but keeping it outside simplifies testing
	currentHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current HEAD: %w", err)
	}
	currentHead = strings.TrimSpace(currentHead)

	// Get environments that are descendants of current HEAD
	filteredEnvs, err := repo.ListDescendantEnvironments(ctx, currentHead)
	if err != nil {
		return "", fmt.Errorf("failed to list descendant environments: %w", err)
	}

	if len(filteredEnvs) == 0 {
		return "", errors.New("no environments found that are descendants of the current HEAD")
	}

	// If only one environment matches, use it
	if len(filteredEnvs) == 1 {
		return filteredEnvs[0].ID, nil
	}

	// Multiple environments - prompt user to select
	return promptForEnvironmentSelection(filteredEnvs)
}

// promptForEnvironmentSelection prompts the user to select from multiple environments
func promptForEnvironmentSelection(envs []*environment.EnvironmentInfo) (string, error) {
	var options []huh.Option[string]

	for _, env := range envs {
		title := env.State.Title
		if title == "" {
			title = "No description"
		}

		label := fmt.Sprintf("%s - %s", env.ID, title)
		options = append(options, huh.NewOption(label, env.ID))
	}

	var selectedID string
	prompt := huh.NewSelect[string]().
		Title("Select an environment:").
		Options(options...).
		Value(&selectedID)

	if err := prompt.Run(); err != nil {
		return "", err
	}

	return selectedID, nil
}
