package main

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long: `Print the version, commit hash, and build date of the cu binary.
With --verbose, show extended diagnostic information including system details,
Docker/Dagger status, Git configuration, and more.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		verbose, _ := cmd.Flags().GetBool("verbose")
		jsonOutput, _ := cmd.Flags().GetBool("json")

		if !verbose {
			// Simple version output
			return printSimpleVersion(jsonOutput)
		}

		// Verbose mode - collect and display diagnostic information
		snapshot := Collect(cmd.Context())

		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(snapshot)
		}

		printDiagnostics(snapshot)
		return nil
	},
}

func init() {
	versionCmd.Flags().BoolP("verbose", "v", false, "Show extended diagnostic information")
	versionCmd.Flags().Bool("json", false, "Output in JSON format")
	rootCmd.AddCommand(versionCmd)
}

func printSimpleVersion(jsonOutput bool) error {
	currentVersion := version
	currentCommit := commit
	currentDate := date

	// For dev builds, try to extract build info from the binary
	if version == "dev" {
		if buildCommit, buildTime := getBuildInfoFromBinary(); buildCommit != "unknown" {
			currentCommit = buildCommit
			currentDate = buildTime
		}
	}

	if jsonOutput {
		info := VersionInfo{
			Version:   currentVersion,
			Commit:    currentCommit,
			BuildDate: currentDate,
			GoVersion: runtime.Version(),
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(info)
	}

	fmt.Printf("cu version %s\n", currentVersion)
	if currentCommit != "unknown" {
		fmt.Printf("commit: %s\n", currentCommit)
	}
	if currentDate != "unknown" {
		fmt.Printf("built: %s\n", currentDate)
	}
	return nil
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