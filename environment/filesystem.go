package environment

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"dagger.io/dagger"
	"github.com/dagger/container-use/edit"
)

func (env *Environment) FileRead(ctx context.Context, targetFile string, shouldReadEntireFile bool, startLineOneIndexedInclusive int, endLineOneIndexedInclusive int) (string, error) {
	file, err := env.container().File(targetFile).Contents(ctx)
	if err != nil {
		return "", err
	}
	if shouldReadEntireFile {
		return file, err
	}

	lines := strings.Split(file, "\n")
	start := startLineOneIndexedInclusive - 1
	start = max(start, 0)
	if start >= len(lines) {
		start = len(lines) - 1
	}
	if start < 0 {
		return "", fmt.Errorf("error reading file: start_line_one_indexed_inclusive (%d) cannot be less than 1", startLineOneIndexedInclusive)
	}
	end := endLineOneIndexedInclusive

	if end >= len(lines) {
		end = len(lines) - 1
	}
	if end < start {
		return "", fmt.Errorf("error reading file: end_line_one_indexed_inclusive (%d) must be greater than start_line_one_indexed_inclusive (%d)", endLineOneIndexedInclusive, startLineOneIndexedInclusive)
	}

	return strings.Join(lines[start:end], "\n"), nil
}

func (env *Environment) FileWrite(ctx context.Context, targetFile, contents string) error {
	err := env.apply(ctx, env.container().WithNewFile(targetFile, contents))
	if err != nil {
		return fmt.Errorf("failed applying file write, skipping git propagation: %w", err)
	}
	env.Notes.Add("Write %s", targetFile)
	return nil
}

func (env *Environment) FileDelete(ctx context.Context, targetFile string) error {
	err := env.apply(ctx, env.container().WithoutFile(targetFile))
	if err != nil {
		return fmt.Errorf("failed applying file delete, skipping git propagation: %w", err)
	}
	env.Notes.Add("Delete %s", targetFile)
	return nil
}

func (env *Environment) FileList(ctx context.Context, path string, ignore []string) (string, error) {
	filter := dagger.DirectoryFilterOpts{Exclude: ignore}
	return env.ls(ctx, path, filter)
}

func (env *Environment) FileGlob(ctx context.Context, path string, pattern string) (string, error) {
	filter := dagger.DirectoryFilterOpts{Include: []string{pattern}}
	return env.ls(ctx, path, filter)
}

func (env *Environment) ls(ctx context.Context, path string, filter dagger.DirectoryFilterOpts) (string, error) {
	entries, err := env.container().Directory(path).Filter(filter).Entries(ctx)
	if err != nil {
		return "", err
	}
	out := &strings.Builder{}
	for _, entry := range entries {
		fmt.Fprintf(out, "%s\n", entry)
	}
	return out.String(), nil
}

func (env *Environment) FileGrep(ctx context.Context, path, pattern, include string) (string, error) {
	// Hack: use busybox to run `sed` since dagger doesn't have native file editing primitives.
	args := []string{"/usr/bin/rg", "--no-unicode", "-g", include, "--", pattern, "."}

	dir := env.container().Directory(path)
	out, err := GrepUtil(env.dag).
		WithMountedDirectory("/workdir", dir).
		WithWorkdir("/workdir").
		WithExec(args, dagger.ContainerWithExecOpts{Expect: dagger.ReturnTypeAny}).
		Stdout(ctx)
	if err != nil {
		return "", err
	}
	return out, nil
}

type FileEdit struct {
	OldString  string
	NewString  string
	ReplaceAll bool
}

func EditUtil(dag *dagger.Client) *dagger.Container {
	editBin := dag.Container().From(golangImage).
		WithNewFile("/go/src/edit.go", edit.Src).
		WithNewFile("/go/src/go.mod", edit.GoMod).
		WithNewFile("/go/src/go.sum", edit.GoSum).
		WithEnvVariable("CGO_ENABLED", "0").
		WithWorkdir("/go/src").
		WithExec([]string{"go", "build", "-o", "/edit", "-ldflags", "-w -s", "/go/src/edit.go"}).File("/edit")
	return dag.Container().From(alpineImage).WithFile("/edit", editBin).WithEntrypoint([]string{"/edit"})
}

func GrepUtil(dag *dagger.Client) *dagger.Container {
	return dag.Container().From(alpineImage).WithExec([]string{"apk", "add", "-U", "ripgrep"})
}

func (env *Environment) FileEdit(ctx context.Context, targetFile string, edits []FileEdit) error {
	// Hack: use busybox to run `sed` since dagger doesn't have native file editing primitives.
	args := []string{"/edit", "/target", "/new"}
	for _, edit := range edits {
		replaceCount := "1"
		if edit.ReplaceAll {
			replaceCount = "-1"
		}
		args = append(args, edit.OldString, edit.NewString, replaceCount)
	}

	newFile := EditUtil(env.dag).WithMountedFile("/target", env.container().File(targetFile)).WithExec(args).File("/new")
	err := env.apply(ctx, env.container().WithFile(targetFile, newFile))
	if err != nil {
		return fmt.Errorf("failed applying file edit, skipping git propagation: %w", err)
	}
	env.Notes.Add("Edit %s", targetFile)
	return nil
}
