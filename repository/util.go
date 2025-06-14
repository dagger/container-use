package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mitchellh/go-homedir"
)

var (
	urlSchemeRegExp  = regexp.MustCompile(`^[^:]+://`)
	scpLikeURLRegExp = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:\s]+):(?:(?P<port>[0-9]{1,5})(?:\/|:))?(?P<path>[^\\].*\/[^\\].*)$`)
)

func getContainerUseRemote(ctx context.Context, repo string) (string, error) {
	// Check if we already have a container-use remote
	cuRemote, err := runGitCommand(ctx, repo, "remote", "get-url", "container-use")
	if err != nil {
		if strings.Contains(err.Error(), "No such remote") {
			return "", os.ErrNotExist
		}
		return "", err
	}

	return strings.TrimSpace(cuRemote), nil
}

func worktreePath(id string) (string, error) {
	return homedir.Expand(path.Join(cuWorktreePath, id))
}

func normalizeForkPath(ctx context.Context, repo string) (string, error) {
	// Check if there's an origin remote
	origin, err := runGitCommand(ctx, repo, "remote", "get-url", "origin")
	if err != nil {
		// If not -- this repository is a local one, we're going to use the filesystem path for the container-use repo
		if strings.Contains(err.Error(), "No such remote") {
			return homedir.Expand(filepath.Join(cuRepoPath, repo))
		}
		return "", err
	}

	// Otherwise, let's use the normalized origin as path
	normalizedOrigin, err := normalizeGitURL(strings.TrimSpace(origin))
	if err != nil {
		return "", err
	}
	return homedir.Expand(filepath.Join(cuRepoPath, normalizedOrigin))
}

func normalizeGitURL(endpoint string) (string, error) {
	if e, ok := normalizeSCPLike(endpoint); ok {
		return e, nil
	}

	return normalizeURL(endpoint)
}

func normalizeURL(endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}

	if !u.IsAbs() {
		return "", fmt.Errorf(
			"invalid endpoint: %s", endpoint,
		)
	}

	return fmt.Sprintf("%s%s", u.Hostname(), strings.TrimSuffix(u.Path, ".git")), nil
}

func normalizeSCPLike(endpoint string) (string, bool) {
	if matchesURLScheme(endpoint) || !matchesScpLike(endpoint) {
		return "", false
	}

	_, host, _, path := findScpLikeComponents(endpoint)

	return fmt.Sprintf("%s/%s", host, strings.TrimSuffix(path, ".git")), true
}

// matchesURLScheme returns true if the given string matches a URL-like
// format scheme.
func matchesURLScheme(url string) bool {
	return urlSchemeRegExp.MatchString(url)
}

// matchesScpLike returns true if the given string matches an SCP-like
// format scheme.
func matchesScpLike(url string) bool {
	return scpLikeURLRegExp.MatchString(url)
}

// findScpLikeComponents returns the user, host, port and path of the
// given SCP-like URL.
func findScpLikeComponents(url string) (user, host, port, path string) {
	m := scpLikeURLRegExp.FindStringSubmatch(url)
	return m[1], m[2], m[3], m[4]
}

// FIXME(aluzzardi): This is a copy of the function in the environment package
func runGitCommand(ctx context.Context, dir string, args ...string) (out string, rerr error) {
	slog.Info(fmt.Sprintf("[%s] $ git %s", dir, strings.Join(args, " ")))
	defer func() {
		slog.Info(fmt.Sprintf("[%s] $ git %s (DONE)", dir, strings.Join(args, " ")), "err", rerr)
	}()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("git command failed (exit code %d): %w\nOutput: %s",
				exitErr.ExitCode(), err, string(output))
		}
		return "", fmt.Errorf("git command failed: %w", err)
	}

	return string(output), nil
}
