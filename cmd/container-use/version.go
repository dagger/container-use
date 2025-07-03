package main

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func init() {
	if version == "dev" {
		if buildCommit, buildTime := getBuildInfoFromBinary(); buildCommit != "unknown" {
			commit = buildCommit
			date = buildTime
		}
	}
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print the version, commit hash, and build date of the container-use binary.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("container-use version %s\n", version)
		if commit != "unknown" {
			fmt.Printf("commit: %s\n", commit)
		}
		if date != "unknown" {
			fmt.Printf("built: %s\n", date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
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

	// Format commit hash (use short version)
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
