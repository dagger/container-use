package main

import (
	"context"
	"strings"

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

// goContainer returns a plain Debian-based Go build container with the source
// mounted and module/build caches configured. We avoid Wolfi-based module
// containers because Wolfi's rolling package repository routinely breaks
// pinned version constraints (go, protobuf, ...).
func (m *ContainerUse) goContainer() *dagger.Container {
	return dag.Container().
		From("golang:1.26").
		WithEnvVariable("CGO_ENABLED", "0").
		WithMountedCache("/go/pkg/mod", dag.CacheVolume("container-use-go-mod")).
		WithMountedCache("/root/.cache/go-build", dag.CacheVolume("container-use-go-build")).
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src")
}

// Build creates a binary for the current platform
func (m *ContainerUse) Build(ctx context.Context,
	//+optional
	platform dagger.Platform,
) *dagger.File {
	ctr := m.goContainer()
	if platform != "" {
		parts := strings.SplitN(string(platform), "/", 2)
		ctr = ctr.
			WithEnvVariable("GOOS", parts[0]).
			WithEnvVariable("GOARCH", parts[1])
	}
	return ctr.
		WithExec([]string{"go", "build", "-o", "/out/container-use", "./cmd/container-use"}).
		File("/out/container-use")
}

// BuildMultiPlatform builds binaries for multiple platforms using GoReleaser
func (m *ContainerUse) BuildMultiPlatform(ctx context.Context,
	// GitHub org name for package publishing, set only if testing release process on a personal fork
	//+optional
	//+default="dagger"
	githubOrgName string,
) *dagger.Directory {
	return dag.Goreleaser(m.Source).
		WithEnvVariable("GH_ORG_NAME", githubOrgName).
		Build().
		WithSnapshot().
		All()
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
	// Create custom container with nix package for nix-hash binary
	customContainer := dag.Container().
		From("ghcr.io/goreleaser/goreleaser:latest").
		WithExec([]string{"apk", "add", "nix"})

	// Use custom container with Goreleaser
	return dag.Goreleaser(m.Source, dagger.GoreleaserOpts{
		Container: customContainer,
	}).
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
	// Use a plain Debian-based Go image to avoid Wolfi package constraints (e.g. protoc version pins)
	ctr := m.goContainer().
		// Configure git for tests
		WithExec([]string{"git", "config", "--global", "user.email", "test@example.com"}).
		WithExec([]string{"git", "config", "--global", "user.name", "Test User"})

	args := []string{"go", "test"}
	if verboseOutput {
		args = append(args, "-v")
	}
	if !integration {
		args = append(args, "-short")
	} else {
		// Integration tests spin up real containers, so we need a Docker daemon
		// inside the (privileged) test container.
		ctr = ctr.WithExec([]string{"sh", "-c", "apt-get update -qq && apt-get install -y -qq docker.io > /dev/null"})
		args = []string{"sh", "-c", "dockerd > /tmp/dockerd.log 2>&1 & for i in $(seq 1 30); do docker info > /dev/null 2>&1 && break; sleep 1; done; " + strings.Join(args, " ") + " \"$@\"", "sh"}
	}
	args = append(args, pkg)

	return ctr.
		WithExec(args, dagger.ContainerWithExecOpts{ExperimentalPrivilegedNesting: true}).
		Stdout(ctx)
}

// TestNixHash tests if nix-hash binary is available in our custom container
func (m *ContainerUse) TestNixHash(ctx context.Context) (string, error) {
	// Create the same custom container we use for releases
	customContainer := dag.Container().
		From("ghcr.io/goreleaser/goreleaser:latest").
		WithExec([]string{"apk", "add", "nix"})

	// Test if nix-hash is available
	return customContainer.
		WithExec([]string{"which", "nix-hash"}).
		Stdout(ctx)
}

// Lint runs the linter
func (m *ContainerUse) Lint(ctx context.Context) error {
	_, err := dag.Container().
		From("golangci/golangci-lint:latest").
		WithMountedDirectory("/src", m.Source).
		WithWorkdir("/src").
		WithExec([]string{"golangci-lint", "run", "./..."}).
		Sync(ctx)
	return err
}
