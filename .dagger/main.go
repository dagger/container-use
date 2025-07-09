package main

import (
	"context"
	"dagger/container-use/internal/dagger"
)

type ContainerUse struct {
	Source *dagger.Directory
}

// dagger module for building container-use
func New(
	//+defaultPath="/"
	source *dagger.Directory,
) *ContainerUse {
	return &ContainerUse{
		Source: source,
	}
}

// Build creates a binary for the current platform
func (m *ContainerUse) Build(ctx context.Context,
	//+optional
	platform dagger.Platform,
) *dagger.File {
	return dag.Go(m.Source).Binary("./cmd/container-use", dagger.GoBinaryOpts{
		Platform: platform,
	})
}

// BuildMultiPlatform builds binaries for multiple platforms using GoReleaser
func (m *ContainerUse) BuildMultiPlatform(ctx context.Context) *dagger.Directory {
	return dag.Goreleaser(m.Source).Build().WithSnapshot().All()
}

// Release creates a release using GoReleaser
func (m *ContainerUse) Release(ctx context.Context,
	// Version tag for the release
	version string,
	// GitHub token for authentication
	githubToken *dagger.Secret,
	// GitHub org name for package publishing, set only if testing release process on a personal fork
	//+default="dagger"
	githubOrgName string,
) (string, error) {
	return dag.Goreleaser(m.Source).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		WithEnvVariable("GH_ORG_NAME", githubOrgName).
		Release().
		Run(ctx)
}

// Test runs the test suite
func (m *ContainerUse) Test(ctx context.Context,
	//+optional
	//+default="./..."
	// Package to test
	pkg string,
	//+optional
	// Run tests with verbose output
	verboseOutput bool,
	//+optional
	//+default=true
	// Run tests including integration tests
	integration bool,
) (string, error) {
	ctr := dag.Go(m.Source).
		Base().
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src").
		// Configure git for tests
		WithExec([]string{"git", "config", "--global", "user.email", "test@example.com"}).
		WithExec([]string{"git", "config", "--global", "user.name", "Test User"})

	args := []string{"go", "test"}
	if verboseOutput {
		args = append(args, "-v")
	}
	if !integration {
		args = append(args, "-short")
	}
	args = append(args, pkg)

	return ctr.
		WithExec(args, dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true}).
		Stdout(ctx)
}

// Lint runs the linter and custom checks
func (m *ContainerUse) Lint(ctx context.Context) error {
	// Run golangci-lint
	err := dag.
		Golangci().
		Lint(m.Source, dagger.GolangciLintOpts{}).
		Assert(ctx)
	if err != nil {
		return err
	}

	// Check for t.Parallel() in tests using WithRepository
	// This is a simple grep-based check that prevents race conditions
	// WithRepository uses SetTestConfigPath which modifies global state
	checkScript := `#!/bin/bash
set -e

echo "Checking for t.Parallel() in tests using WithRepository..."

# Find test files that use WithRepository
files_with_repo=$(grep -l "WithRepository" environment/integration/*_test.go 2>/dev/null || true)

if [ -z "$files_with_repo" ]; then
  echo "No test files found using WithRepository"
  exit 0
fi

# Check each file for t.Parallel()
found_issues=false
for file in $files_with_repo; do
  if grep -q "t\.Parallel()" "$file"; then
    echo "ERROR: $file uses both WithRepository and t.Parallel()"
    echo "  WithRepository modifies global state and is not safe for parallel execution"
    found_issues=true
  fi
done

if [ "$found_issues" = true ]; then
  echo ""
  echo "Tests using WithRepository must not call t.Parallel() as it causes race conditions."
  exit 1
fi

echo "✓ No parallel test issues found"
`

	_, err = dag.Go(m.Source).
		Base().
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithExec([]string{"bash", "-c", checkScript}).
		Sync(ctx)

	return err
}
