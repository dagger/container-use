package repository

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	petname "github.com/dustinkirkland/golang-petname"
)

const (
	cuGlobalConfigPath = "~/.config/container-use"
	cuRepoPath         = cuGlobalConfigPath + "/repos"
	cuWorktreePath     = cuGlobalConfigPath + "/worktrees"
	containerUseRemote = "container-use"
	gitNotesLogRef     = "container-use"
	gitNotesStateRef   = "container-use-state"
)

type Repository struct {
	userRepoPath string
	forkRepoPath string
}

func Open(ctx context.Context, repo string) (*Repository, error) {
	output, err := runGitCommand(ctx, repo, "rev-parse", "--show-toplevel")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return nil, errors.New("you must be in a git repository to use container-use")
		}
		return nil, err
	}
	userRepoPath := strings.TrimSpace(output)

	forkRepoPath, err := getContainerUseRemote(ctx, userRepoPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		forkRepoPath, err = normalizeForkPath(ctx, userRepoPath)
		if err != nil {
			return nil, err
		}
	}

	r := &Repository{
		userRepoPath: userRepoPath,
		forkRepoPath: forkRepoPath,
	}

	if err := r.ensureFork(ctx); err != nil {
		return nil, fmt.Errorf("unable to fork the repository: %w", err)
	}
	if err := r.ensureUserRemote(ctx); err != nil {
		return nil, fmt.Errorf("unable to set container-use remote: %w", err)
	}

	return r, nil
}

func (r *Repository) ensureFork(ctx context.Context) error {
	// Make sure the fork repo path exists, otherwise create it
	_, err := os.Stat(r.forkRepoPath)
	if err == nil {
		return nil
	}
	if !os.IsNotExist(err) {
		return err
	}

	slog.Info("Initializing local remote", "user-repo", r.userRepoPath, "fork-repo", r.forkRepoPath)
	if err := os.MkdirAll(r.forkRepoPath, 0755); err != nil {
		return err
	}
	_, err = runGitCommand(ctx, r.forkRepoPath, "init", "--bare")
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) ensureUserRemote(ctx context.Context) error {
	currentForkPath, err := getContainerUseRemote(ctx, r.userRepoPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		_, err := runGitCommand(ctx, r.userRepoPath, "remote", "add", containerUseRemote, r.forkRepoPath)
		return err
	}

	if currentForkPath != r.forkRepoPath {
		_, err := runGitCommand(ctx, r.userRepoPath, "remote", "set-url", containerUseRemote, r.forkRepoPath)
		return err
	}

	return nil
}

func (r *Repository) SourcePath() string {
	return r.userRepoPath
}

func (r *Repository) exists(ctx context.Context, id string) error {
	if _, err := runGitCommand(ctx, r.forkRepoPath, "rev-parse", "--verify", id); err != nil {
		if strings.Contains(err.Error(), "Needed a single revision") {
			return fmt.Errorf("environment %q not found", id)
		}
		return err
	}
	return nil
}

// Create creates a new environment with the given description and explanation.
// Requires a dagger client for container operations during environment initialization.
func (r *Repository) Create(ctx context.Context, dag *dagger.Client, description, explanation string) (*environment.Environment, error) {
	id := petname.Generate(2, "-")
	worktree, err := r.initializeWorktree(ctx, id)
	if err != nil {
		return nil, err
	}

	env, err := environment.New(ctx, dag, id, description, worktree)
	if err != nil {
		return nil, err
	}

	if err := r.propagateToWorktree(ctx, env, "Create env "+id, explanation); err != nil {
		return nil, err
	}

	return env, nil
}

// Get retrieves a full Environment with dagger client embedded for container operations.
// Use this when you need to perform container operations like running commands, terminals, etc.
// For basic metadata access without container operations, use Info() instead.
func (r *Repository) Get(ctx context.Context, dag *dagger.Client, id string) (*environment.Environment, error) {
	if err := r.exists(ctx, id); err != nil {
		return nil, err
	}

	worktree, err := r.initializeWorktree(ctx, id)
	if err != nil {
		return nil, err
	}

	state, err := r.loadState(ctx, worktree)
	if err != nil {
		return nil, err
	}

	env, err := environment.Load(ctx, dag, id, state, worktree)
	if err != nil {
		return nil, err
	}

	return env, nil
}

// Info retrieves environment metadata without requiring dagger operations.
// This is more efficient than Get() when you only need access to configuration,
// state, and other metadata without performing container operations.
func (r *Repository) Info(ctx context.Context, id string) (*environment.EnvironmentInfo, error) {
	if err := r.exists(ctx, id); err != nil {
		return nil, err
	}

	worktree, err := r.initializeWorktree(ctx, id)
	if err != nil {
		return nil, err
	}

	state, err := r.loadState(ctx, worktree)
	if err != nil {
		return nil, err
	}

	envInfo, err := environment.LoadInfo(ctx, id, state, worktree)
	if err != nil {
		return nil, err
	}

	return envInfo, nil
}

// List returns information about all environments in the repository.
// Returns EnvironmentInfo slice avoiding dagger client initialization.
// Use Get() on individual environments when you need full Environment with container operations.
func (r *Repository) List(ctx context.Context) ([]*environment.EnvironmentInfo, error) {
	branches, err := runGitCommand(ctx, r.forkRepoPath, "branch", "--format", "%(refname:short)")
	if err != nil {
		return nil, err
	}

	envs := []*environment.EnvironmentInfo{}
	for branch := range strings.SplitSeq(branches, "\n") {
		branch = strings.TrimSpace(branch)

		// FIXME(aluzzardi): This is a hack to make sure the branch is actually an environment.
		// There must be a better way to do this.
		worktree, err := worktreePath(branch)
		if err != nil {
			return nil, err
		}
		state, err := r.loadState(ctx, worktree)
		if err != nil || state == nil {
			continue
		}

		envInfo, err := r.Info(ctx, branch)
		if err != nil {
			return nil, err
		}

		envs = append(envs, envInfo)
	}

	// Sort by most recently updated environments first
	sort.Slice(envs, func(i, j int) bool {
		return envs[i].State.UpdatedAt.After(envs[j].State.UpdatedAt)
	})

	return envs, nil
}

// Update saves the provided environment to the repository.
// Writes configuration and source code changes to the worktree and history + state to git notes.
func (r *Repository) Update(ctx context.Context, env *environment.Environment, operation, explanation string) error {
	note := env.Notes.Pop()
	if strings.TrimSpace(note) != "" {
		if err := r.addGitNote(ctx, env, note); err != nil {
			return err
		}
	}
	return r.propagateToWorktree(ctx, env, operation, explanation)
}

// Delete removes an environment from the repository.
func (r *Repository) Delete(ctx context.Context, id string) error {
	if err := r.exists(ctx, id); err != nil {
		return err
	}

	if err := r.deleteWorktree(id); err != nil {
		return err
	}
	if err := r.deleteLocalRemoteBranch(id); err != nil {
		return err
	}
	return nil
}

// Checkout changes the user's current branch to that of the identified environment.
// It attempts to get the most recent commit from the environment without discarding any user changes.
func (r *Repository) Checkout(ctx context.Context, id string) (string, error) {
	if err := r.exists(ctx, id); err != nil {
		return "", err
	}

	branch := "cu-" + id

	// set up remote tracking branch if it's not already there
	_, err := runGitCommand(ctx, r.userRepoPath, "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branch))
	localBranchExists := err == nil
	if !localBranchExists {
		_, err = runGitCommand(ctx, r.userRepoPath, "branch", "--track", branch, fmt.Sprintf("%s/%s", containerUseRemote, id))
		if err != nil {
			return "", err
		}
	}

	_, err = runGitCommand(ctx, r.userRepoPath, "checkout", id)
	if err != nil {
		return "", err
	}

	if localBranchExists {
		remoteRef := fmt.Sprintf("%s/%s", containerUseRemote, id)

		counts, err := runGitCommand(ctx, r.userRepoPath, "rev-list", "--left-right", "--count", fmt.Sprintf("HEAD...%s", remoteRef))
		if err != nil {
			return branch, err
		}

		parts := strings.Split(strings.TrimSpace(counts), "\t")
		if len(parts) != 2 {
			return branch, fmt.Errorf("unexpected git rev-list output: %s", counts)
		}
		aheadCount, behindCount := parts[0], parts[1]

		if behindCount != "0" && aheadCount == "0" {
			_, err = runGitCommand(ctx, r.userRepoPath, "merge", "--ff-only", remoteRef)
			if err != nil {
				return branch, err
			}
		} else if behindCount != "0" {
			return branch, fmt.Errorf("switched to %s, but %s is %s ahead and container-use/ remote has %s additional commits", branch, branch, aheadCount, behindCount)
		}
	}

	return branch, err
}
