package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"runtime/debug"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

const defaultTimeout = 2 * time.Second

func init() {
	if version == "dev" {
		if buildCommit, buildTime := getBuildInfoFromBinary(); buildCommit != "unknown" {
			commit = buildCommit
			date = buildTime
		}
	}

	versionCmd.Flags().BoolP("system", "s", false, "Show system information")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, and build date of the container-use binary.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		showSystem, _ := cmd.Flags().GetBool("system")

		// Always show basic version info
		cmd.Printf("container-use version %s\n", version)
		if commit != "unknown" {
			cmd.Printf("commit: %s\n", commit)
		}
		if date != "unknown" {
			cmd.Printf("built: %s\n", date)
		}

		if showSystem {
			cmd.Printf("\nSystem:\n")
			cmd.Printf("  OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)

			// Check container runtime
			if runtime := detectContainerRuntime(cmd.Context()); runtime != nil {
				cmd.Printf("  Container Runtime: %s\n", runtime)
			} else {
				cmd.Printf("  Container Runtime: not found\n")
			}

			// Check Git
			if version := getToolVersion(cmd.Context(), "git", "--version"); version != "" {
				cmd.Printf("  Git: %s\n", version)
			} else {
				cmd.Printf("  Git: not found\n")
			}

			// Check Dagger CLI
			if version := getToolVersion(cmd.Context(), "dagger", "version"); version != "" {
				cmd.Printf("  Dagger CLI: %s\n", version)
			} else {
				cmd.Printf("  Dagger CLI: not found (needed for 'terminal' command)\n")
			}
		}

		return nil
	},
}

// runtimeInfo holds container runtime information
type runtimeInfo struct {
	Name    string
	Version string
	Running bool
}

func (r *runtimeInfo) String() string {
	if !r.Running {
		return fmt.Sprintf("%s %s (daemon not running)", r.Name, r.Version)
	}
	return fmt.Sprintf("%s %s", r.Name, r.Version)
}

// detectContainerRuntime finds the first available container runtime
func detectContainerRuntime(ctx context.Context) *runtimeInfo {
	// Check in the same order as Dagger
	runtimes := []struct {
		command string
		name    string
	}{
		{"docker", "Docker"},
		{"podman", "Podman"},
		{"nerdctl", "nerdctl"},
		{"finch", "finch"},
	}

	for _, rt := range runtimes {
		if info := checkRuntime(ctx, rt.command, rt.name); info != nil {
			return info
		}
	}
	return nil
}

// checkRuntime checks if a specific runtime is available
func checkRuntime(ctx context.Context, command, name string) *runtimeInfo {
	// Check if command exists
	if _, err := exec.LookPath(command); err != nil {
		return nil
	}

	info := &runtimeInfo{
		Name:    name,
		Version: "unknown",
	}

	// Get version
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	if out, err := exec.CommandContext(ctx, command, "--version").Output(); err == nil {
		info.Version = extractVersion(string(out))
	}

	// Check if daemon is running
	cmd := exec.CommandContext(ctx, command, "info")
	cmd.Stdout = nil // discard output
	cmd.Stderr = nil
	info.Running = cmd.Run() == nil

	return info
}

var versionRegex = regexp.MustCompile(`v?(\d+\.\d+(?:\.\d+)?)`)

// extractVersion finds a version number in the output
func extractVersion(output string) string {
	if matches := versionRegex.FindStringSubmatch(output); len(matches) > 1 {
		return matches[1]
	}
	return "unknown"
}

// getToolVersion runs a command and returns its version output
func getToolVersion(ctx context.Context, tool string, args ...string) string {
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, tool, args...).Output()
	if err != nil {
		return ""
	}

	output := strings.TrimSpace(string(out))

	// Handle specific tools
	switch tool {
	case "git":
		// "git version 2.39.3" -> "2.39.3"
		return strings.TrimPrefix(output, "git version ")
	case "dagger":
		// "dagger vX.Y.Z (...)" -> "vX.Y.Z"
		fields := strings.Fields(output)
		if len(fields) > 1 {
			return fields[1]
		}
	}

	return output
}

func getBuildInfoFromBinary() (string, string) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown", "unknown"
	}

	var revision, buildTime, modified string
	for _, setting := range buildInfo.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.time":
			buildTime = setting.Value
		case "vcs.modified":
			modified = setting.Value
		}
	}

	// Format commit hash
	if len(revision) > 7 {
		revision = revision[:7]
	}
	if modified == "true" {
		revision += "-dirty"
	}

	if revision == "" {
		revision = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}

	return revision, buildTime
}
