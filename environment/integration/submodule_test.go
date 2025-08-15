package integration

import (
	"context"
	"os"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubmoduleFileWriteErrors verifies that file write operations to submodules are blocked
func TestSubmoduleFileWriteErrors(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	WithRepository(t, "submodule_errors", SetupSimpleRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		env := user.CreateEnvironment("Submodule Test", "Testing submodule file write protection")

		// Manually set submodule paths since git submodule foreach won't work without proper submodule init
		envObj, err := repo.Get(context.Background(), user.dag, env.ID)
		require.NoError(t, err)

		// Override the submodule paths that were detected (which might be empty)
		envObj.State.SubmodulePaths = []string{"vendor/example-lib"}

		// Try to write to a file in the submodule - should fail
		targetFile := "vendor/example-lib/src/main.js"
		contents := "console.log('modified');"
		explanation := "Attempt to modify submodule file"

		err = envObj.FileWrite(context.Background(), explanation, targetFile, contents)
		assert.Error(t, err, "FileWrite should fail for submodule files")
		assert.Contains(t, err.Error(), "cannot modify file", "Error should mention file modification is blocked")
		assert.Contains(t, err.Error(), "within a git submodule", "Error should mention submodule")
		assert.Contains(t, err.Error(), "read-only", "Error should mention read-only restriction")

		// Try to write to a file in the main repository - should succeed
		mainRepoFile := "main.js"
		mainContents := "console.log('main repo file');"

		err = envObj.FileWrite(context.Background(), "Write to main repo", mainRepoFile, mainContents)
		assert.NoError(t, err, "FileWrite should succeed for main repository files")

		// Try to delete a file in the submodule - should also fail
		err = envObj.FileDelete(context.Background(), "Try to delete submodule file", targetFile)
		assert.Error(t, err, "FileDelete should fail for submodule files")
		assert.Contains(t, err.Error(), "cannot modify file", "Delete error should mention file modification is blocked")

		// Try to edit a file in the submodule - should also fail
		err = envObj.FileEdit(context.Background(), "Try to edit submodule file", targetFile, "example", "modified", "")
		assert.Error(t, err, "FileEdit should fail for submodule files")
		assert.Contains(t, err.Error(), "cannot modify file", "Edit error should mention file modification is blocked")

		// Verify we can still read from submodule files
		content, err := envObj.FileRead(context.Background(), targetFile, true, 0, 0)
		assert.NoError(t, err, "FileRead should work for submodule files")
		assert.Contains(t, content, "example library", "Should be able to read submodule content")
	})
}

// SetupSimpleRepoWithSubmoduleStructure creates a repository with submodule structure for testing
var SetupSimpleRepoWithSubmoduleStructure = func(t *testing.T, repoDir string) {

	// Create main repository files
	writeFile(t, repoDir, "main.js", "console.log('Main application');")
	writeFile(t, repoDir, "package.json", `{
  "name": "main-app",
  "version": "1.0.0", 
  "main": "main.js"
}`)

	// Create submodule directory structure (without actual submodule setup to avoid initialization issues)
	err := os.MkdirAll(repoDir+"/vendor/example-lib/src", 0755)
	require.NoError(t, err, "Failed to create submodule directory")

	// Create submodule content
	submoduleContent := `// Example library
console.log('This is an example library');
module.exports = { version: '1.0.0' };`

	writeFile(t, repoDir, "vendor/example-lib/src/main.js", submoduleContent)
	writeFile(t, repoDir, "vendor/example-lib/package.json", `{
  "name": "example-lib", 
  "version": "1.0.0",
  "main": "src/main.js"
}`)
	writeFile(t, repoDir, "vendor/example-lib/README.md", "# Example Library\nThis is a submodule library")

	// Commit all files (including the "submodule" directory as regular files)
	gitCommit(t, repoDir, "Add main project and vendor files")
}
