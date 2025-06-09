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
	return dag.Go(m.Source).Binary("./cmd/cu", dagger.GoBinaryOpts{
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
) (string, error) {
	return dag.Goreleaser(m.Source).
		WithSecretVariable("GITHUB_TOKEN", githubToken).
		Release().
		Run(ctx)
}
