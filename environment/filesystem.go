package environment

import (
	"context"
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	godiffpatch "github.com/sourcegraph/go-diff-patch"
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
	// Check if the file is within a submodule
	if err := env.validateNotSubmoduleFile(targetFile); err != nil {
		return err
	}

	err := env.apply(ctx, env.container().WithNewFile(targetFile, contents))
	if err != nil {
		return fmt.Errorf("failed applying file write, skipping git propagation: %w", err)
	}
	env.Notes.Add("Write %s", targetFile)
	return nil
}

func (env *Environment) FileEdit(ctx context.Context, explanation, targetFile, search, replace, matchID string) error {
	// Check if the file is within a submodule
	if err := env.validateNotSubmoduleFile(targetFile); err != nil {
		return err
	}

	contents, err := env.container().File(targetFile).Contents(ctx)
	if err != nil {
		return err
	}

	// Find all matches of the search text
	matches := []int{}
	cursor := 0
	for {
		index := strings.Index(contents[cursor:], search)
		if index == -1 {
			break
		}
		actualIndex := cursor + index
		matches = append(matches, actualIndex)
		cursor = actualIndex + 1
	}

	if len(matches) == 0 {
		return fmt.Errorf("search text not found in file %s", targetFile)
	}

	// If there are multiple matches and no matchID is provided, return an error with all matches
	if len(matches) > 1 && matchID == "" {
		var matchDescriptions []string
		for i, matchIndex := range matches {
			// Generate a unique ID for each match
			id := generateMatchID(targetFile, search, replace, i)

			// Get context around the match (3 lines before and after)
			context := getMatchContext(contents, matchIndex)

			matchDescriptions = append(matchDescriptions, fmt.Sprintf("Match %d (ID: %s):\n%s", i+1, id, context))
		}

		return fmt.Errorf("multiple matches found for search text in %s. Please specify which_match parameter with one of the following IDs:\n\n%s",
			targetFile, strings.Join(matchDescriptions, "\n\n"))
	}

	// Determine which match to replace
	var targetMatchIndex int
	if len(matches) == 1 {
		targetMatchIndex = matches[0]
	} else {
		// Find the match with the specified ID
		found := false
		for i, matchIndex := range matches {
			id := generateMatchID(targetFile, search, replace, i)
			if id == matchID {
				targetMatchIndex = matchIndex
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("match ID %s not found", matchID)
		}
	}

	// Replace the specific match
	newContents := contents[:targetMatchIndex] + replace + contents[targetMatchIndex+len(search):]

	// Apply the changes using `Directory.withPatch` so we don't have to spit out
	// the entire contents
	patch := godiffpatch.GeneratePatch(targetFile, contents, newContents)
	ctr := env.container()
	err = env.apply(ctx, ctr.WithDirectory(".", ctr.Directory(".").WithPatch(patch)))
	if err != nil {
		return fmt.Errorf("failed applying file edit, skipping git propagation: %w", err)
	}
	env.Notes.Add("Edit %s", targetFile)
	return nil
}

func (env *Environment) FileDelete(ctx context.Context, explanation, targetFile string) error {
	// Check if the file is within a submodule
	if err := env.validateNotSubmoduleFile(targetFile); err != nil {
		return err
	}

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

// generateMatchID creates a unique ID for a match based on file, search, replace, and index
func generateMatchID(targetFile, search, replace string, index int) string {
	data := fmt.Sprintf("%s:%s:%s:%d", targetFile, search, replace, index)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("%x", hash)[:8] // Use first 8 characters of hash
}

// getMatchContext returns the context around a match (3 lines before and after)
func getMatchContext(contents string, matchIndex int) string {
	lines := strings.Split(contents, "\n")

	// Find which line contains the match
	currentPos := 0
	matchLine := 0
	for i, line := range lines {
		if currentPos+len(line) >= matchIndex {
			matchLine = i
			break
		}
		currentPos += len(line) + 1 // +1 for newline
	}

	// Get context lines (3 before, match line, 3 after)
	start := max(0, matchLine-3)
	end := min(len(lines), matchLine+4)

	contextLines := make([]string, 0, end-start)
	for i := start; i < end; i++ {
		prefix := "  "
		if i == matchLine {
			prefix = "> " // Mark the line containing the match
		}
		// Include line numbers, which may help the model determine the right match
		prefix += fmt.Sprintf("%4d | ", i+1)
		contextLines = append(contextLines, fmt.Sprintf("%s%s", prefix, lines[i]))
	}

	return strings.Join(contextLines, "\n")
}

// isWithinSubmodule checks if a file path is within any of the submodule directories
func (env *Environment) isWithinSubmodule(filePath string, submodulePaths []string) bool {
	// Convert absolute paths to relative paths within workdir
	workdir := env.State.Config.Workdir
	if filepath.IsAbs(filePath) {
		var err error
		filePath, err = filepath.Rel(workdir, filePath)
		if err != nil || strings.HasPrefix(filePath, "..") {
			// If the file is outside workdir, it's not in a submodule
			return false
		}
	}

	cleanFilePath := filepath.Clean(filePath)

	for _, submodulePath := range submodulePaths {
		cleanSubmodulePath := filepath.Clean(submodulePath)

		// Check if the file is exactly the submodule path or within it
		if cleanFilePath == cleanSubmodulePath {
			return true
		}

		// Check if the file is within the submodule directory
		if strings.HasPrefix(cleanFilePath, cleanSubmodulePath+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// validateNotSubmoduleFile checks if a file path is within a submodule and returns an error if it is
func (env *Environment) validateNotSubmoduleFile(filePath string) error {
	// Use cached submodule paths from state (detected once during creation)
	submodulePaths := env.State.SubmodulePaths

	if env.isWithinSubmodule(filePath, submodulePaths) {
		return fmt.Errorf("cannot modify file '%s': it is within a git submodule. Submodule files are read-only to prevent accidental changes", filePath)
	}

	return nil
}
