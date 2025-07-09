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

// Test runs the linter
func (m *ContainerUse) Lint(ctx context.Context) error {
	return dag.
		Golangci().
		Lint(m.Source, dagger.GolangciLintOpts{}).
		Assert(ctx)
}
