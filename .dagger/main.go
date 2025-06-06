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

// Returns a container that echoes whatever string argument is provided
func (m *ContainerUse) Build(ctx context.Context) *dagger.File {
	return dag.Go(m.Source).Binary("./cmd/cu")
}
