package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/dagger/container-use/repository"
)

// VersionInfo contains version information.
type VersionInfo struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	OS        string `json:"os"`
}

// Snapshot captures the complete diagnostic state.
type Snapshot struct {
	Version         VersionInfo                `json:"version"`
	Git             GitInfo                    `json:"git"`
	Docker          DockerInfo                 `json:"docker"`
	Filesystem      FilesystemInfo             `json:"filesystem"`
	Environments    map[string]EnvironmentInfo `json:"environments"`
	Inconsistencies []string                   `json:"inconsistencies,omitempty"`
	RecentErrors    []string                   `json:"recent_errors,omitempty"`
}

// GitInfo holds Git repository information.
type GitInfo struct {
	InRepo            bool              `json:"in_repo"`
	Branch            string            `json:"branch,omitempty"`
	Remotes           map[string]string `json:"remotes,omitempty"`
	WorktreeCount     int               `json:"worktree_count"`
	CURemoteExists    bool              `json:"cu_remote_exists"`
	CURemoteReachable bool              `json:"cu_remote_reachable"`
	CUBranchCount     int               `json:"cu_branch_count"`
	CUBranches        []string          `json:"cu_branches,omitempty"`
}

// DockerInfo holds Docker and Dagger information.
type DockerInfo struct {
	Available        bool              `json:"available"`
	DaggerSDKVersion string            `json:"dagger_sdk_version,omitempty"`
	DaggerEngines    []DaggerEngine    `json:"dagger_engines,omitempty"`
	DaggerEnvVars    map[string]string `json:"dagger_env_vars,omitempty"`
}

// DaggerEngine represents a running Dagger engine.
type DaggerEngine struct {
	Name    string `json:"name"`
	Image   string `json:"image"`
	Version string `json:"version"`
}

// FilesystemInfo holds filesystem state.
type FilesystemInfo struct {
	ConfigDirExists   bool     `json:"config_dir_exists"`
	WorktreeDirExists bool     `json:"worktree_dir_exists"`
	WorktreeNames     []string `json:"worktree_names,omitempty"`
}

// EnvironmentInfo represents a single environment's state.
type EnvironmentInfo struct {
	ID           string `json:"id"`
	HasWorktree  bool   `json:"has_worktree"`
	HasCUDir     bool   `json:"has_cu_dir"`
	HasEnvJSON   bool   `json:"has_environment_json"`
	HasAgentMD   bool   `json:"has_agent_md"`
	HasGitBranch bool   `json:"has_git_branch"`
	HasGitNotes  bool   `json:"has_git_notes"`
}

// Collect gathers all diagnostic information.
func Collect(ctx context.Context) Snapshot {
	c := &collector{ctx: ctx}
	snapshot := Snapshot{
		Version:      c.version(),
		Git:          c.git(),
		Docker:       c.docker(),
		Filesystem:   c.filesystem(),
		Environments: make(map[string]EnvironmentInfo),
	}
	
	c.environments(&snapshot)
	snapshot.Inconsistencies = c.inconsistencies(&snapshot)
	snapshot.RecentErrors = c.recentErrors()
	return snapshot
}

type collector struct {
	ctx context.Context
}

func (c *collector) version() VersionInfo {
	return VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildDate: date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS + "/" + runtime.GOARCH,
	}
}

func (c *collector) git() GitInfo {
	info := GitInfo{Remotes: make(map[string]string)}
	
	// Check if we're in a git repository
	if _, err := c.run("git", "rev-parse", "--git-dir"); err != nil {
		return info
	}
	info.InRepo = true
	
	// Get current branch
	if branch, _ := c.run("git", "branch", "--show-current"); branch != "" {
		info.Branch = strings.TrimSpace(branch)
	}
	
	// Get remotes
	if out, _ := c.run("git", "remote", "-v"); out != "" {
		for line := range strings.SplitSeq(out, "\n") {
			parts := strings.Fields(line)
			if len(parts) >= 2 && strings.HasSuffix(line, "(fetch)") {
				info.Remotes[parts[0]] = parts[1]
			}
		}
	}
	
	// Check container-use remote
	if _, ok := info.Remotes["container-use"]; ok {
		info.CURemoteExists = true
		if _, err := c.run("git", "ls-remote", "--exit-code", "container-use", "HEAD"); err == nil {
			info.CURemoteReachable = true
		}
	}
	
	// Count worktrees
	if out, _ := c.run("git", "worktree", "list", "--porcelain"); out != "" {
		info.WorktreeCount = strings.Count(out, "worktree ")
	}
	
	// Get container-use branches
	if info.CURemoteExists {
		if out, _ := c.run("git", "branch", "-r", "--list", "container-use/*"); out != "" {
			const prefix = "container-use/"
			for line := range strings.SplitSeq(out, "\n") {
				line = strings.TrimSpace(line)
				if line != "" && strings.Contains(line, prefix) {
					info.CUBranches = append(info.CUBranches, line[strings.Index(line, prefix)+len(prefix):])
				}
			}
			info.CUBranchCount = len(info.CUBranches)
		}
	}
	
	return info
}


func (c *collector) docker() DockerInfo {
	info := DockerInfo{DaggerEnvVars: make(map[string]string)}
	
	// Check if Docker is available
	if _, err := c.runCmd("docker", "version", "--format", "json"); err != nil {
		return info
	}
	info.Available = true
	
	// Get Dagger SDK version from go.mod
	if data, _ := os.ReadFile("go.mod"); len(data) > 0 {
		for line := range strings.SplitSeq(string(data), "\n") {
			if strings.Contains(line, "dagger.io/dagger") {
				if parts := strings.Fields(line); len(parts) >= 2 {
					info.DaggerSDKVersion = parts[1]
					break
				}
			}
		}
	}
	
	// Find running Dagger engines
	out, _ := c.runCmd("docker", "ps", "--filter", "name=dagger-engine", "--format", "{{.Names}}\t{{.Image}}")
	for line := range strings.SplitSeq(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Split(line, "\t")
		if len(parts) == 2 {
			engine := DaggerEngine{
				Name:  parts[0],
				Image: parts[1],
			}
			// Extract version from image tag
			if idx := strings.LastIndex(parts[1], ":"); idx != -1 {
				engine.Version = parts[1][idx+1:]
			}
			info.DaggerEngines = append(info.DaggerEngines, engine)
		}
	}
	
	// Check Dagger environment variables
	for _, env := range daggerEnvironmentVars {
		if v := os.Getenv(env); v != "" {
			info.DaggerEnvVars[env] = v
		}
	}
	
	return info
}

var daggerEnvironmentVars = []string{
	"_EXPERIMENTAL_DAGGER_RUNNER_HOST",
	"_EXPERIMENTAL_DAGGER_CLI_BIN",
}

func (c *collector) filesystem() FilesystemInfo {
	info := FilesystemInfo{}
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "container-use")
	
	if _, err := os.Stat(configDir); err == nil {
		info.ConfigDirExists = true
		
		if entries, err := os.ReadDir(filepath.Join(configDir, "worktrees")); err == nil {
			info.WorktreeDirExists = true
			for _, entry := range entries {
				if entry.IsDir() {
					info.WorktreeNames = append(info.WorktreeNames, entry.Name())
				}
			}
		}
	}
	return info
}

func (c *collector) environments(snapshot *Snapshot) {
	homeDir, _ := os.UserHomeDir()
	baseDir := filepath.Join(homeDir, ".config", "container-use")
	
	// Collect environments from filesystem worktrees
	for _, name := range snapshot.Filesystem.WorktreeNames {
		env := EnvironmentInfo{
			ID:          name,
			HasWorktree: true,
		}
		
		cuDir := filepath.Join(baseDir, "worktrees", name, ".container-use")
		if _, err := os.Stat(cuDir); err == nil {
			env.HasCUDir = true
			env.HasEnvJSON = fileExists(filepath.Join(cuDir, "environment.json"))
			env.HasAgentMD = fileExists(filepath.Join(cuDir, "AGENT.md"))
		}
		
		snapshot.Environments[name] = env
	}
	
	// Mark environments that have git branches
	for _, branch := range snapshot.Git.CUBranches {
		if env, ok := snapshot.Environments[branch]; ok {
			env.HasGitBranch = true
			snapshot.Environments[branch] = env
		} else {
			snapshot.Environments[branch] = EnvironmentInfo{
				ID:           branch,
				HasGitBranch: true,
			}
		}
	}
	
	// Check for git notes in the container-use repository
	if snapshot.Git.CURemoteExists {
		reposDir := filepath.Join(baseDir, "repos")
		if entries, _ := os.ReadDir(reposDir); len(entries) > 0 {
			repoDir := filepath.Join(reposDir, entries[0].Name())
			for id, env := range snapshot.Environments {
				cmd := fmt.Sprintf("cd %s && git notes --ref container-use-state show %s 2>/dev/null", repoDir, id)
				if _, err := c.runCmd("sh", "-c", cmd); err == nil {
					env.HasGitNotes = true
					snapshot.Environments[id] = env
				}
			}
		}
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (c *collector) inconsistencies(s *Snapshot) []string {
	var results []string
	
	// Check worktree/branch count mismatch
	worktreeCount := len(s.Filesystem.WorktreeNames)
	branchCount := s.Git.CUBranchCount
	if worktreeCount != branchCount {
		results = append(results, fmt.Sprintf("Worktree count (%d) != Git branch count (%d)", worktreeCount, branchCount))
	}
	
	// Check environment state inconsistencies
	for id, env := range s.Environments {
		switch {
		case env.HasWorktree && !env.HasGitBranch:
			results = append(results, fmt.Sprintf("'%s': has worktree but no git branch", id))
		case env.HasGitBranch && !env.HasWorktree:
			results = append(results, fmt.Sprintf("'%s': has git branch but no worktree", id))
		case env.HasWorktree && !env.HasGitNotes:
			results = append(results, fmt.Sprintf("'%s': has worktree but no git notes (invisible to 'cu list')", id))
		case env.HasGitNotes && !env.HasWorktree:
			results = append(results, fmt.Sprintf("'%s': has git notes but no worktree", id))
		case env.HasWorktree && !env.HasCUDir:
			results = append(results, fmt.Sprintf("'%s': worktree missing .container-use directory", id))
		case env.HasCUDir && !env.HasEnvJSON:
			results = append(results, fmt.Sprintf("'%s': .container-use directory missing environment.json", id))
		}
	}
	
	// Check git repository and remote status
	if !s.Git.InRepo {
		results = append(results, "Not in a git repository")
	} else if !s.Git.CURemoteExists && len(s.Environments) > 0 {
		results = append(results, "Git remote 'container-use' missing but environments exist")
	} else if s.Git.CURemoteExists && !s.Git.CURemoteReachable {
		results = append(results, "Git remote 'container-use' unreachable")
	}
	
	// Check Dagger version compatibility
	if s.Docker.DaggerSDKVersion != "" && len(s.Docker.DaggerEngines) > 0 {
		hasMatchingEngine := slices.ContainsFunc(s.Docker.DaggerEngines, func(e DaggerEngine) bool {
			return e.Version == s.Docker.DaggerSDKVersion
		})
		
		if !hasMatchingEngine {
			versions := make([]string, len(s.Docker.DaggerEngines))
			for i, e := range s.Docker.DaggerEngines {
				versions[i] = e.Version
			}
			results = append(results, fmt.Sprintf("Dagger SDK %s expects matching engine, but found: %s",
				s.Docker.DaggerSDKVersion, strings.Join(versions, ", ")))
		}
		
		if len(s.Docker.DaggerEngines) > 1 {
			results = append(results, fmt.Sprintf("Multiple Dagger engines running (%d), may cause connection issues",
				len(s.Docker.DaggerEngines)))
		}
	}
	
	return results
}

func (c *collector) recentErrors() []string {
	logPath := os.Getenv("CU_STDERR_FILE")
	if logPath == "" {
		if runtime.GOOS == "windows" {
			logPath = filepath.Join(os.TempDir(), "cu.debug.stderr.log")
		} else {
			logPath = "/tmp/cu.debug.stderr.log"
		}
	}
	
	content, err := os.ReadFile(logPath)
	if err != nil {
		return nil
	}
	
	if len(content) > 1024 {
		content = content[len(content)-1024:]
	}
	
	errors := make([]string, 0, 3)
	for line := range strings.SplitSeq(string(content), "\n") {
		if strings.Contains(line, "ERROR") || strings.Contains(line, "exit code 128") {
			errors = append(errors, strings.TrimSpace(line))
			if len(errors) >= 3 {
				break
			}
		}
	}
	return errors
}

func (c *collector) run(name string, args ...string) (string, error) {
	if name == "git" {
		ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
		defer cancel()
		return repository.RunGitCommand(ctx, ".", args...)
	}
	return c.runCmd(name, args...)
}

func (c *collector) runCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(c.ctx, 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).Output()
	return string(out), err
}

// printDiagnostics prints a clean state snapshot.
func printDiagnostics(snapshot Snapshot) {
	fmt.Printf("VERSION: %s (%s)\n", snapshot.Version.Version, snapshot.Version.OS)
	fmt.Printf("BUILD: %s\n\n", snapshot.Version.Commit)
	
	printGitInfo(snapshot.Git)
	printDockerInfo(snapshot.Docker)
	printFilesystemInfo(snapshot.Filesystem)
	printEnvironments(snapshot.Environments)
	printInconsistencies(snapshot.Inconsistencies)
	printRecentErrors(snapshot.RecentErrors)
}

func printGitInfo(git GitInfo) {
	fmt.Println("GIT:")
	fmt.Printf("  in_repo: %v\n", git.InRepo)
	if !git.InRepo {
		return
	}
	
	fmt.Printf("  branch: %s\n", git.Branch)
	fmt.Printf("  remotes: %d\n", len(git.Remotes))
	fmt.Printf("  cu_remote: %v (reachable: %v)\n", git.CURemoteExists, git.CURemoteReachable)
	fmt.Printf("  cu_branches: %d\n", git.CUBranchCount)
	fmt.Printf("  worktrees: %d\n", git.WorktreeCount)
}

func printDockerInfo(docker DockerInfo) {
	fmt.Println("\nDOCKER:")
	fmt.Printf("  available: %v\n", docker.Available)
	
	if docker.DaggerSDKVersion != "" {
		fmt.Printf("  dagger_sdk: %s\n", docker.DaggerSDKVersion)
	}
	
	if len(docker.DaggerEngines) > 0 {
		fmt.Printf("  dagger_engines: %d running\n", len(docker.DaggerEngines))
		for _, engine := range docker.DaggerEngines {
			fmt.Printf("    - %s (%s)\n", engine.Name, engine.Version)
		}
	} else {
		fmt.Printf("  dagger_engines: none running\n")
	}
	
	if len(docker.DaggerEnvVars) > 0 {
		fmt.Printf("  dagger_env_vars:\n")
		for k, v := range docker.DaggerEnvVars {
			fmt.Printf("    %s: %s\n", k, v)
		}
	}
}

func printFilesystemInfo(fs FilesystemInfo) {
	fmt.Println("\nFILESYSTEM:")
	fmt.Printf("  config_dir: %v\n", fs.ConfigDirExists)
	fmt.Printf("  worktree_count: %d\n", len(fs.WorktreeNames))
}

func printEnvironments(environments map[string]EnvironmentInfo) {
	if len(environments) == 0 {
		return
	}
	
	fmt.Println("\nENVIRONMENTS:")
	for id, env := range environments {
		fmt.Printf("  %s:\n", id)
		fmt.Printf("    worktree: %v, cu_dir: %v, env.json: %v, agent.md: %v\n",
			env.HasWorktree, env.HasCUDir, env.HasEnvJSON, env.HasAgentMD)
		fmt.Printf("    git_branch: %v, git_notes: %v\n",
			env.HasGitBranch, env.HasGitNotes)
	}
}

func printInconsistencies(inconsistencies []string) {
	if len(inconsistencies) == 0 {
		return
	}
	
	fmt.Println("\nINCONSISTENCIES:")
	for _, inc := range inconsistencies {
		fmt.Printf("  - %s\n", inc)
	}
}

func printRecentErrors(errors []string) {
	if len(errors) == 0 {
		return
	}
	
	fmt.Println("\nRECENT ERRORS:")
	for _, err := range errors {
		fmt.Printf("  %s\n", err)
	}
}