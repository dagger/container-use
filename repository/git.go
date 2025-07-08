package repository

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"dagger.io/dagger"
	"github.com/dagger/container-use/environment"
	"github.com/mitchellh/go-homedir"
)

const (
	maxFileSizeForTextCheck = 10 * 1024 * 1024 // 10MB
)

var (
	urlSchemeRegExp  = regexp.MustCompile(`^[^:]+://`)
	scpLikeURLRegExp = regexp.MustCompile(`^(?:(?P<user>[^@]+)@)?(?P<host>[^:\s]+):(?:(?P<port>[0-9]{1,5})(?:\/|:))?(?P<path>[^\\].*\/[^\\].*)$`)
)

// RunGitCommand executes a git command in the specified directory.
// This is exported for use in tests and other packages that need direct git access.
func RunGitCommand(ctx context.Context, dir string, args ...string) (out string, rerr error) {
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

// RunInteractiveGitCommand executes a git command in the specified directory in interactive mode.
func RunInteractiveGitCommand(ctx context.Context, dir string, w io.Writer, args ...string) (rerr error) {
	slog.Info(fmt.Sprintf("[%s] $ git %s", dir, strings.Join(args, " ")))
	defer func() {
		slog.Info(fmt.Sprintf("[%s] $ git %s (DONE)", dir, strings.Join(args, " ")), "err", rerr)
	}()

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Stdout = w
	cmd.Stderr = w

	return cmd.Run()
}

func getContainerUseRemote(ctx context.Context, repo string) (string, error) {
	// Check if we already have a container-use remote
	cuRemote, err := RunGitCommand(ctx, repo, "remote", "get-url", "container-use")
	if err != nil {
		// Check for exit code 2 which means the remote doesn't exist
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return "", os.ErrNotExist
		}
		return "", err
	}

	return strings.TrimSpace(cuRemote), nil
}

func (r *Repository) WorktreePath(id string) (string, error) {
	return homedir.Expand(path.Join(r.getWorktreePath(), id))
}

func (r *Repository) deleteWorktree(id string) error {
	worktreePath, err := r.WorktreePath(id)
	if err != nil {
		return err
	}
	fmt.Printf("Deleting worktree at %s\n", worktreePath)
	return os.RemoveAll(worktreePath)
}

func (r *Repository) deleteLocalRemoteBranch(id string) error {
	slog.Info("Pruning git worktrees", "repo", r.forkRepoPath)
	if _, err := RunGitCommand(context.Background(), r.forkRepoPath, "worktree", "prune"); err != nil {
		slog.Error("Failed to prune git worktrees", "repo", r.forkRepoPath, "err", err)
		return err
	}

	slog.Info("Deleting local branch", "repo", r.forkRepoPath, "branch", id)
	if _, err := RunGitCommand(context.Background(), r.forkRepoPath, "branch", "-D", id); err != nil {
		slog.Error("Failed to delete local branch", "repo", r.forkRepoPath, "branch", id, "err", err)
		return err
	}

	if _, err := RunGitCommand(context.Background(), r.userRepoPath, "remote", "prune", containerUseRemote); err != nil {
		slog.Error("Failed to fetch and prune container-use remote", "local-repo", r.userRepoPath, "err", err)
		return err
	}

	return nil
}

func (r *Repository) initializeWorktree(ctx context.Context, id string) (string, error) {
	worktreePath, err := r.WorktreePath(id)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return worktreePath, nil
	}

	slog.Info("Initializing worktree", "repository", r.userRepoPath, "container-id", id)

	currentHead, err := RunGitCommand(ctx, r.userRepoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	currentHead = strings.TrimSpace(currentHead)

	_, err = RunGitCommand(ctx, r.userRepoPath, "push", containerUseRemote, fmt.Sprintf("%s:refs/heads/%s", currentHead, id))
	if err != nil {
		return "", err
	}

	_, err = RunGitCommand(ctx, r.forkRepoPath, "worktree", "add", worktreePath, id)
	if err != nil {
		return "", err
	}

	_, err = RunGitCommand(ctx, r.userRepoPath, "fetch", containerUseRemote, id)
	if err != nil {
		return "", err
	}

	return worktreePath, nil
}

func (r *Repository) propagateToWorktree(ctx context.Context, env *environment.Environment, explanation string) (rerr error) {
	slog.Info("Propagating to worktree...",
		"environment.id", env.ID,
		"workdir", env.Config.Workdir,
		"id", env.ID)
	defer func() {
		slog.Info("Propagating to worktree... (DONE)",
			"environment.id", env.ID,
			"workdir", env.Config.Workdir,
			"id", env.ID,
			"err", rerr)
	}()

	if err := r.exportEnvironment(ctx, env); err != nil {
		return err
	}

	worktreePath, err := r.WorktreePath(env.ID)
	if err != nil {
		return fmt.Errorf("failed to get worktree path: %w", err)
	}
	if err := r.commitWorktreeChanges(ctx, worktreePath, explanation); err != nil {
		return fmt.Errorf("failed to commit worktree changes: %w", err)
	}

	if err := r.saveState(ctx, env); err != nil {
		return fmt.Errorf("failed to add notes: %w", err)
	}

	slog.Info("Fetching container-use remote in source repository")
	if _, err := RunGitCommand(ctx, r.userRepoPath, "fetch", containerUseRemote, env.ID); err != nil {
		return err
	}

	if err := r.propagateGitNotes(ctx, gitNotesStateRef); err != nil {
		return err
	}

	return nil
}

func (r *Repository) exportEnvironment(ctx context.Context, env *environment.Environment) error {
	worktreePointer := fmt.Sprintf("gitdir: %s/worktrees/%s", r.forkRepoPath, env.ID)

	worktreePath, err := r.WorktreePath(env.ID)
	if err != nil {
		return fmt.Errorf("failed to get worktree path: %w", err)
	}

	_, err = env.Workdir().
		WithNewFile(".git", worktreePointer).
		Export(
			ctx,
			worktreePath,
			dagger.DirectoryExportOpts{Wipe: true},
		)
	if err != nil {
		return err
	}

	slog.Info("Saving environment")
	if err := env.Config.Save(worktreePath); err != nil {
		return err
	}
	return nil
}
func (r *Repository) propagateGitNotes(ctx context.Context, ref string) error {
	fullRef := fmt.Sprintf("refs/notes/%s", ref)
	fetch := func() error {
		_, err := RunGitCommand(ctx, r.userRepoPath, "fetch", containerUseRemote, fullRef+":"+fullRef)
		return err
	}

	if err := fetch(); err != nil {
		if strings.Contains(err.Error(), "[rejected]") {
			if _, err := RunGitCommand(ctx, r.userRepoPath, "update-ref", "-d", fullRef); err == nil {
				return fetch()
			}
		}
		return err
	}
	return nil
}

func (r *Repository) saveState(ctx context.Context, env *environment.Environment) error {
	state, err := env.State.Marshal()
	if err != nil {
		return err
	}
	worktreePath, err := r.WorktreePath(env.ID)
	if err != nil {
		return fmt.Errorf("failed to get worktree path: %w", err)
	}

	f, err := os.CreateTemp(os.TempDir(), ".container-use-git-notes-*")
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(state); err != nil {
		return err
	}

	_, err = RunGitCommand(ctx, worktreePath, "notes", "--ref", gitNotesStateRef, "add", "-f", "-F", f.Name())
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) loadState(ctx context.Context, worktreePath string) ([]byte, error) {
	buff, err := RunGitCommand(ctx, worktreePath, "notes", "--ref", gitNotesStateRef, "show")
	if err != nil {
		if strings.Contains(err.Error(), "no note found") {
			return nil, nil
		}
		return nil, err
	}
	return []byte(buff), nil
}

func (r *Repository) addGitNote(ctx context.Context, env *environment.Environment, note string) error {
	worktreePath, err := r.WorktreePath(env.ID)
	if err != nil {
		return fmt.Errorf("failed to get worktree path: %w", err)
	}
	_, err = RunGitCommand(ctx, worktreePath, "notes", "--ref", gitNotesLogRef, "append", "-m", note)
	if err != nil {
		return err
	}
	return r.propagateGitNotes(ctx, gitNotesLogRef)
}

func (r *Repository) currentUserBranch(ctx context.Context) (string, error) {
	return RunGitCommand(ctx, r.userRepoPath, "branch", "--show-current")
}

func (r *Repository) mergeBase(ctx context.Context, env *environment.EnvironmentInfo) (string, error) {
	currentBranch, err := r.currentUserBranch(ctx)
	if err != nil {
		return "", err
	}
	currentBranch = strings.TrimSpace(currentBranch)
	if currentBranch == "" {
		currentBranch = "HEAD"
	}
	envGitRef := fmt.Sprintf("%s/%s", containerUseRemote, env.ID)
	mergeBase, err := RunGitCommand(ctx, r.userRepoPath, "merge-base", currentBranch, envGitRef)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(mergeBase), nil
}

func (r *Repository) revisionRange(ctx context.Context, env *environment.EnvironmentInfo) (string, error) {
	mergeBase, err := r.mergeBase(ctx, env)
	if err != nil {
		return "", err
	}
	envGitRef := fmt.Sprintf("%s/%s", containerUseRemote, env.ID)
	return fmt.Sprintf("%s..%s", mergeBase, envGitRef), nil
}

func (r *Repository) commitWorktreeChanges(ctx context.Context, worktreePath, explanation string) error {
	status, err := RunGitCommand(ctx, worktreePath, "status", "--porcelain")
	if err != nil {
		return err
	}

	if strings.TrimSpace(status) == "" {
		return nil
	}

	if err := r.addNonBinaryFiles(ctx, worktreePath); err != nil {
		return err
	}

	_, err = RunGitCommand(ctx, worktreePath, "commit", "--allow-empty", "--allow-empty-message", "-m", explanation)
	return err
}

// AI slop below!
// this is just to keep us moving fast because big git repos get hard to work with
// and our demos like to download large dependencies.
func (r *Repository) addNonBinaryFiles(ctx context.Context, worktreePath string) error {
	statusOutput, err := RunGitCommand(ctx, worktreePath, "status", "--porcelain")
	if err != nil {
		return err
	}

	for line := range strings.SplitSeq(strings.TrimSpace(statusOutput), "\n") {
		if line == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]
		fileName := strings.TrimSpace(line[2:])
		if fileName == "" {
			continue
		}

		if r.shouldSkipFile(fileName) {
			continue
		}

		switch {
		case indexStatus == '?' && workTreeStatus == '?':
			// ?? = untracked files or directories
			if strings.HasSuffix(fileName, "/") {
				// Untracked directory - traverse and add non-binary files
				dirName := strings.TrimSuffix(fileName, "/")
				if err := r.addFilesFromUntrackedDirectory(ctx, worktreePath, dirName); err != nil {
					return err
				}
			} else if !r.isBinaryFile(worktreePath, fileName) {
				// Untracked file - add if not binary

				_, err = RunGitCommand(ctx, worktreePath, "add", fileName)
				if err != nil {
					return err
				}
			}
		case indexStatus == 'A':
			// A = already staged, skip
			continue
		case indexStatus == 'D' || workTreeStatus == 'D':
			// D = deleted files (always stage deletion)
			_, err = RunGitCommand(ctx, worktreePath, "add", fileName)
			if err != nil {
				return err
			}
		default:
			// M, R, C and other statuses - add if not binary
			if !r.isBinaryFile(worktreePath, fileName) {
				_, err = RunGitCommand(ctx, worktreePath, "add", fileName)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *Repository) shouldSkipFile(fileName string) bool {
	skipExtensions := []string{
		".tar", ".tar.gz", ".tgz", ".tar.bz2", ".tbz2", ".tar.xz", ".txz",
		".zip", ".rar", ".7z", ".gz", ".bz2", ".xz",
		".exe", ".bin", ".dmg", ".pkg", ".msi",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".tiff", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wmv", ".flv", ".mkv",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".so", ".dylib", ".dll", ".a", ".lib",
	}

	lowerName := strings.ToLower(fileName)
	for _, ext := range skipExtensions {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}

	skipPatterns := []string{
		"node_modules/", ".git/", "__pycache__/", ".DS_Store",
		"venv/", ".venv/", "env/", ".env/",
		"target/", "build/", "dist/", ".next/",
		"*.tmp", "*.temp", "*.cache", "*.log",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(lowerName, strings.ToLower(pattern)) {
			return true
		}
	}

	return false
}

func (r *Repository) IsDirty(ctx context.Context) (bool, string, error) {
	status, err := RunGitCommand(ctx, r.userRepoPath, "status", "--porcelain")
	if err != nil {
		return false, "", err
	}

	if strings.TrimSpace(status) == "" {
		return false, "", nil
	}

	return true, status, nil
}

func (r *Repository) addFilesFromUntrackedDirectory(ctx context.Context, worktreePath, dirName string) error {
	dirPath := filepath.Join(worktreePath, dirName)

	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(worktreePath, path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			if r.shouldSkipFile(relPath + "/") {
				return filepath.SkipDir
			}
			return nil
		}

		if r.shouldSkipFile(relPath) {
			return nil
		}

		if !r.isBinaryFile(worktreePath, relPath) {
			_, err = RunGitCommand(ctx, worktreePath, "add", relPath)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) isBinaryFile(worktreePath, fileName string) bool {
	fullPath := filepath.Join(worktreePath, fileName)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return true
	}

	if stat.IsDir() {
		return false
	}

	if stat.Size() > maxFileSizeForTextCheck {
		return true
	}

	// Empty files should be treated as text files so `touch .gitkeep` and friends work correctly
	if stat.Size() == 0 {
		return false
	}

	file, err := os.Open(fullPath)
	if err != nil {
		slog.Error("Error opening file", "err", err)
		return true
	}
	defer file.Close()

	buffer := make([]byte, 8000)
	n, err := file.Read(buffer)
	if err != nil && n == 0 {
		return true
	}

	buffer = buffer[:n]
	return slices.Contains(buffer, 0)
}

func (r *Repository) normalizeForkPath(ctx context.Context, repo string) (string, error) {
	// Check if there's an origin remote
	origin, err := RunGitCommand(ctx, repo, "remote", "get-url", "origin")
	if err != nil {
		// If not -- this repository is a local one, we're going to use the filesystem path for the container-use repo
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			// Exit code 2 means the remote doesn't exist
			return homedir.Expand(filepath.Join(r.getRepoPath(), repo))
		}
		return "", err
	}

	// Otherwise, let's use the normalized origin as path
	normalizedOrigin, err := normalizeGitURL(strings.TrimSpace(origin))
	if err != nil {
		return "", err
	}
	return homedir.Expand(filepath.Join(r.getRepoPath(), normalizedOrigin))
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
