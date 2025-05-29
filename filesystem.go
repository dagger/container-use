package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"dagger.io/dagger"
	"github.com/google/uuid"
)

type HostDirectory struct {
	ID      string         `json:"id"`
	Path    string         `json:"path"`
	History HostDirHistory `json:"history"`

	mu        sync.Mutex
	Directory *dagger.Directory
}

func (hd *HostDirectory) Checkpoint(ctx context.Context, reason string, explanation string) error {
	hd.mu.Lock()
	defer hd.mu.Unlock()

	name := fmt.Sprintf("%s %s", reason, hd.Path)
	err := hd.History.Checkpoint(ctx, name, explanation, hd.Directory)
	if err != nil {
		return fmt.Errorf("failed syncing host directory: %w", err)
	}
	err = saveHostDirState(hd)
	if err != nil {
		return fmt.Errorf("failed persisting host directory state: %w", err)
	}
	return nil
}

func (hd *HostDirectory) Revert(ctx context.Context, explanation string, version Version) error {
	hd.mu.Lock()
	defer hd.mu.Unlock()

	revision := hd.History.Get(version)
	if revision == nil {
		return errors.New("no revisions found")
	}
	
	// Export the reverted state back to the host filesystem
	if _, err := revision.state.Export(ctx, hd.Path); err != nil {
		return fmt.Errorf("failed exporting reverted state to host directory: %w", err)
	}
	
	hd.Directory = revision.state
	
	// Create a new checkpoint to record the revert
	name := fmt.Sprintf("Revert %s to %s", hd.Path, revision.Name)
	err := hd.History.Checkpoint(ctx, name, explanation, hd.Directory)
	if err != nil {
		return fmt.Errorf("failed syncing host directory after revert: %w", err)
	}
	
	err = saveHostDirState(hd)
	if err != nil {
		return fmt.Errorf("failed persisting host directory state after revert: %w", err)
	}
	
	return nil
}

type HostDirRevision struct {
	Version     Version   `json:"version"`
	Name        string    `json:"name"`
	Explanation string    `json:"explanation"`
	CreatedAt   time.Time `json:"created_at"`

	state *dagger.Directory
}

type HostDirHistory []*HostDirRevision

func (h HostDirHistory) Latest() *HostDirRevision {
	if len(h) == 0 {
		return nil
	}
	return h[len(h)-1]
}

func (h HostDirHistory) LatestVersion() Version {
	latest := h.Latest()
	if latest == nil {
		return 0
	}
	return latest.Version
}

func (h *HostDirHistory) Checkpoint(ctx context.Context, name string, explanation string, dir *dagger.Directory) error {
	state, err := dir.Sync(ctx)
	if err != nil {
		return err
	}
	*h = append(*h, &HostDirRevision{
		Version:     h.LatestVersion() + 1,
		Name:        name,
		Explanation: explanation,
		CreatedAt:   time.Now(),
		state:       state,
	})
	return nil
}

func (h HostDirHistory) Get(version Version) *HostDirRevision {
	for _, revision := range h {
		if revision.Version == version {
			return revision
		}
	}
	return nil
}

var hostDirectories = map[string]*HostDirectory{}
var hostDirectoriesMtx sync.Mutex

func LoadHostDirectories() error {
	hds, err := loadHostDirState()
	if err != nil {
		return err
	}
	hostDirectories = hds
	return nil
}

func GetHostDirectory(path string) *HostDirectory {
	// TODO: normalize path

	hostDirectoriesMtx.Lock()
	defer hostDirectoriesMtx.Unlock()

	if hd, ok := hostDirectories[path]; ok {
		return hd
	}

	hostDirectories[path] = &HostDirectory{
		ID:        uuid.New().String(),
		Path:      path,
		Directory: dag.Host().Directory(path, dagger.HostDirectoryOpts{NoCache: true}),
	}

	return hostDirectories[path]
}

func ListHostDirectories() []*HostDirectory {
	hostDirectoriesMtx.Lock()
	defer hostDirectoriesMtx.Unlock()
	
	hds := make([]*HostDirectory, 0, len(hostDirectories))
	for _, hd := range hostDirectories {
		hds = append(hds, hd)
	}
	return hds
}

func (s *Container) FileRead(ctx context.Context, targetFile string, shouldReadEntireFile bool, startLineOneIndexed int, endLineOneIndexedInclusive int) (string, error) {
	file, err := s.state.File(targetFile).Contents(ctx)
	if err != nil {
		return "", err
	}
	if shouldReadEntireFile {
		return string(file), err
	}

	lines := strings.Split(string(file), "\n")
	start := startLineOneIndexed - 1
	if start < 0 {
		start = 0
	}
	if start >= len(lines) {
		start = len(lines) - 1
	}
	end := endLineOneIndexedInclusive
	if end >= len(lines) {
		end = len(lines) - 1
	}
	if end < 0 {
		end = 0
	}
	return strings.Join(lines[start:end], "\n"), nil
}

func (s *Container) FileWrite(ctx context.Context, explanation, targetFile, contents string) error {
	return s.apply(ctx, "Write "+targetFile, explanation, s.state.WithNewFile(targetFile, contents))
}

func (s *Container) FileDelete(ctx context.Context, explanation, targetFile string) error {
	return s.apply(ctx, "Delete "+targetFile, explanation, s.state.WithoutFile(targetFile))
}

func (s *Container) FileList(ctx context.Context, path string) (string, error) {
	entries, err := s.state.Directory(path).Entries(ctx)
	if err != nil {
		return "", err
	}
	out := &strings.Builder{}
	for _, entry := range entries {
		fmt.Fprintf(out, "%s\n", entry)
	}
	return out.String(), nil
}

func urlToDirectory(url string) (*HostDirectory, *dagger.Directory) {
	switch {
	case strings.HasPrefix(url, "git://"):
		return nil, dag.Git(url[len("git://"):]).Head().Tree()
	case strings.HasPrefix(url, "https://"):
		return nil, dag.Git(url[len("https://"):]).Head().Tree()
	case strings.HasPrefix(url, "file://"):
		hd := GetHostDirectory(url[len("file://"):])
		return hd, hd.Directory
	default:
		hd := GetHostDirectory(url)
		return hd, hd.Directory
	}
}

func (s *Container) Upload(ctx context.Context, explanation string, source string, target string) error {
	// TODO: subpath disambiguation - /dir/subpath/subpath should give /dir when /dir already exists as a HostDirectory?
	hd, dir := urlToDirectory(source)
	if hd != nil {
		hd.Checkpoint(ctx, "Before Upload", explanation)
	}

	return s.apply(
		ctx,
		"Upload "+source+" to "+target,
		explanation,
		s.state.WithDirectory(target, dir),
	)
}

func (s *Container) Download(ctx context.Context, source string, target string) error {
	// TODO: subpath disambiguation - /dir/subpath/subpath should checkpoint /dir
	hd, _ := urlToDirectory(target)
	if hd != nil {
		hd.Checkpoint(ctx, "Before Download", "Downloaded "+source+", overwriting "+target)
	}

	if _, err := s.state.Directory(source).Export(ctx, target); err != nil {
		if strings.Contains(err.Error(), "not a directory") {
			if _, err := s.state.File(source).Export(ctx, target); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	return nil
}

func (s *Container) Diff(ctx context.Context, source string, target string) (string, error) {
	_, sourceDir := urlToDirectory(source)
	targetDir := s.state.Directory(target)

	diff, err := dag.Container().From("alpine").
		WithMountedDirectory("/source", sourceDir).
		WithMountedDirectory("/target", targetDir).
		WithExec([]string{"diff", "-burN", "/source", "/target"}, dagger.ContainerWithExecOpts{
			Expect: dagger.ReturnTypeAny,
		}).
		Stdout(ctx)
	if err != nil {
		var exitErr *dagger.ExecError
		if errors.As(err, &exitErr) {
			return fmt.Sprintf("command failed with exit code %d.\nstdout: %s\nstderr: %s", exitErr.ExitCode, exitErr.Stdout, exitErr.Stderr), nil
		}
		return "", err
	}
	return diff, nil
}
