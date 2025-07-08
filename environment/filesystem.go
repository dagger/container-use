package environment

import (
	"context"
	"fmt"
	"strings"
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

func (env *Environment) FileWrite(ctx context.Context, explanation, targetFile, contents string) error {
	err := env.apply(ctx, env.container().WithNewFile(targetFile, contents))
	if err != nil {
		return fmt.Errorf("failed applying file write, skipping git propagation: %w", err)
	}
	env.Notes.Add("Write %s", targetFile)
	return nil
}

func (env *Environment) FileDelete(ctx context.Context, explanation, targetFile string) error {
	err := env.apply(ctx, env.container().WithoutFile(targetFile))
	if err != nil {
		return fmt.Errorf("failed applying file delete, skipping git propagation: %w", err)
	}
	env.Notes.Add("Delete %s", targetFile)
	return nil
}

func (env *Environment) FileList(ctx context.Context, path string) (string, error) {
	entries, err := env.container().Directory(path).Entries(ctx)
	if err != nil {
		return "", err
	}
	out := &strings.Builder{}
	for _, entry := range entries {
		fmt.Fprintf(out, "%s\n", entry)
	}
	return out.String(), nil
}
