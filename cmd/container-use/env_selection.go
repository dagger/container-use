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
	if len(args) > 0 {
		return args[0], nil
	}

	// Get current user repo head
	currentHead, err := repository.RunGitCommand(ctx, repo.SourcePath(), "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("failed to get current HEAD: %w", err)
	}
	currentHead = strings.TrimSpace(currentHead)

	// Get environment IDs that are descendants of current HEAD using git-native filtering
	envIDs, err := getDescendantEnvironments(ctx, repo, currentHead)
	if err != nil {
		return "", fmt.Errorf("failed to get descendant environments: %w", err)
	}

	if len(envIDs) == 0 {
		return "", errors.New("no environments found that are descendants of the current HEAD")
	}

	// If only one environment matches, use it
	if len(envIDs) == 1 {
		return envIDs[0], nil
	}

	// Multiple environments - get their info and prompt user to select
	var envs []*environment.EnvironmentInfo
	for _, envID := range envIDs {
		envInfo, err := repo.Info(ctx, envID)
		if err != nil {
			// Skip environments that can't be loaded
			continue
		}
		envs = append(envs, envInfo)
	}

	if len(envs) == 0 {
		return "", errors.New("no environments found")
	}

	if len(envs) == 1 {
		return envs[0].ID, nil
	}

	return promptForEnvironmentSelection(envs)
}

// getDescendantEnvironments uses git-native commands to efficiently filter environments
// that are descendants of the current HEAD
func getDescendantEnvironments(ctx context.Context, repo *repository.Repository, currentHead string) ([]string, error) {
	// Use git for-each-ref to get all container-use branches with their commit hashes
	// Format: <commit-hash> <refname>
	output, err := repository.RunGitCommand(ctx, repo.SourcePath(),
		"for-each-ref",
		"--format=%(objectname) %(refname:short)",
		"refs/remotes/container-use/")
	if err != nil {
		return nil, err
	}

	if strings.TrimSpace(output) == "" {
		return []string{}, nil
	}

	var candidateEnvs []string
	var candidateCommits []string

	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}

		commit := parts[0]
		refname := parts[1]

		// Extract environment ID from refname (container-use/env-id -> env-id)
		if strings.HasPrefix(refname, "container-use/") {
			envID := strings.TrimPrefix(refname, "container-use/")
			candidateEnvs = append(candidateEnvs, envID)
			candidateCommits = append(candidateCommits, commit)
		}
	}

	if len(candidateEnvs) == 0 {
		return []string{}, nil
	}

	// Use git merge-base to batch-check which environments are descendants of currentHead
	// We'll use git merge-base --is-ancestor for each candidate
	var descendantEnvs []string
	for i, envID := range candidateEnvs {
		envCommit := candidateCommits[i]

		// Check if currentHead is an ancestor of envCommit
		// git merge-base --is-ancestor returns 0 if currentHead is an ancestor of envCommit
		_, err := repository.RunGitCommand(ctx, repo.SourcePath(),
			"merge-base", "--is-ancestor", currentHead, envCommit)
		if err == nil {
			// currentHead is an ancestor of envCommit, so envCommit is a descendant
			descendantEnvs = append(descendantEnvs, envID)
		}
	}

	return descendantEnvs, nil
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
