package integration

import (
	"context"
	"testing"

	"github.com/dagger/container-use/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SetupRepoWithSubmoduleStructure creates a repository with submodule-like structure
// This avoids the issues with file:// URLs in Dagger by manually creating the structure
var SetupRepoWithSubmoduleStructure = func(t *testing.T, repoDir string) {
	// Create a vendor directory structure that simulates initialized submodules
	writeFile(t, repoDir, "vendor/submodule/submodule.txt", "This is content from the submodule\n")
	writeFile(t, repoDir, "vendor/submodule/lib/helper.go", "package lib\n\nfunc Helper() string {\n\treturn \"helper function\"\n}\n")

	// Create a .gitmodules file to simulate submodule configuration
	gitmodulesContent := `[submodule "vendor/submodule"]
	path = vendor/submodule
	url = https://github.com/example/submodule.git
`
	writeFile(t, repoDir, ".gitmodules", gitmodulesContent)

	// Add main repository content
	writeFile(t, repoDir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from main repo\")\n}\n")
	writeFile(t, repoDir, "README.md", "# Main Repository\n\nThis repository contains a submodule in vendor/submodule\n")

	// Commit everything
	gitCommit(t, repoDir, "Add submodule structure and main content")
}

// SetupRepoWithRealSubmodule creates a repository with an actual Git submodule using HTTPS
// This creates a real scenario but requires network access
var SetupRepoWithRealSubmodule = func(t *testing.T, repoDir string) {
	ctx := context.Background()

	// Add a real submodule from GitHub (small, well-known repository)
	_, err := repository.RunGitCommand(ctx, repoDir, "submodule", "add", "https://github.com/octocat/Hello-World.git", "vendor/hello-world")
	require.NoError(t, err, "Failed to add real submodule")

	// Add main repository content
	writeFile(t, repoDir, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from main repo\")\n}\n")
	writeFile(t, repoDir, "README.md", "# Main Repository\n\nThis repository contains a real submodule\n")

	// Commit the submodule addition
	gitCommit(t, repoDir, "Add real submodule and main content")
}

// TestSubmoduleBasicWorkflow tests that users can work with submodule content normally
func TestSubmoduleBasicWorkflow(t *testing.T) {
	t.Parallel()
	WithRepository(t, "submodule-basic", SetupRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Basic Submodule Workflow", "Testing basic submodule usage")

		// User should be able to read submodule files
		content := user.FileRead(env.ID, "vendor/submodule/submodule.txt")
		assert.Contains(t, content, "This is content from the submodule")

		// User should be able to read nested submodule files
		helperContent := user.FileRead(env.ID, "vendor/submodule/lib/helper.go")
		assert.Contains(t, helperContent, "func Helper()")

		// User should be able to modify submodule files
		user.FileWrite(env.ID, "vendor/submodule/config.json", `{"version": "1.0"}`, "Add config to submodule")

		// User should be able to read the modified file
		configContent := user.FileRead(env.ID, "vendor/submodule/config.json")
		assert.Contains(t, configContent, `"version": "1.0"`)

		// User should be able to work with main repo files alongside submodule
		mainContent := user.FileRead(env.ID, "main.go")
		assert.Contains(t, mainContent, "Hello from main repo")

		// User should be able to modify main repo files
		user.FileWrite(env.ID, "main.go", "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello from updated main repo\")\n}\n", "Update main file")

		// Changes should persist
		updatedMain := user.FileRead(env.ID, "main.go")
		assert.Contains(t, updatedMain, "Hello from updated main repo")
	})
}

// TestSubmoduleMultipleUpdates tests that submodule content persists across multiple environment updates
func TestSubmoduleMultipleUpdates(t *testing.T) {
	t.Parallel()
	WithRepository(t, "submodule-updates", SetupRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Multiple Updates", "Testing persistence across updates")

		// Initial state - submodule files should be present
		content := user.FileRead(env.ID, "vendor/submodule/submodule.txt")
		assert.Contains(t, content, "This is content from the submodule")

		// First update - modify submodule
		user.FileWrite(env.ID, "vendor/submodule/step1.txt", "First update", "Add step1 file")

		// Second update - modify main repo
		user.FileWrite(env.ID, "step1.txt", "Main repo update", "Add main repo file")

		// Third update - modify submodule again
		user.FileWrite(env.ID, "vendor/submodule/step2.txt", "Second update", "Add step2 file")

		// All content should still be accessible
		assert.Contains(t, user.FileRead(env.ID, "vendor/submodule/submodule.txt"), "This is content from the submodule")
		assert.Contains(t, user.FileRead(env.ID, "vendor/submodule/step1.txt"), "First update")
		assert.Contains(t, user.FileRead(env.ID, "vendor/submodule/step2.txt"), "Second update")
		assert.Contains(t, user.FileRead(env.ID, "step1.txt"), "Main repo update")

		// Original submodule structure should still work
		helperContent := user.FileRead(env.ID, "vendor/submodule/lib/helper.go")
		assert.Contains(t, helperContent, "func Helper()")
	})
}

// TestSubmoduleCommandExecution tests that users can run commands that depend on submodule content
func TestSubmoduleCommandExecution(t *testing.T) {
	t.Parallel()
	WithRepository(t, "submodule-commands", SetupRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Command Execution", "Testing command execution with submodules")

		// User should be able to list submodule contents
		output := user.RunCommand(env.ID, "ls -la vendor/submodule/", "List submodule directory")
		assert.Contains(t, output, "submodule.txt")
		assert.Contains(t, output, "lib")

		// User should be able to navigate into submodule directories
		output = user.RunCommand(env.ID, "ls vendor/submodule/lib/", "List submodule lib directory")
		assert.Contains(t, output, "helper.go")

		// User should be able to read submodule files via command line
		output = user.RunCommand(env.ID, "cat vendor/submodule/submodule.txt", "Read submodule file")
		assert.Contains(t, output, "This is content from the submodule")

		// User should be able to modify submodule files via command line
		user.RunCommand(env.ID, "echo 'Command line edit' > vendor/submodule/cmdline.txt", "Edit via command line")

		// Changes should be visible
		content := user.FileRead(env.ID, "vendor/submodule/cmdline.txt")
		assert.Contains(t, content, "Command line edit")

		// User should be able to run scripts that depend on submodule content
		user.FileWrite(env.ID, "test_script.sh", "#!/bin/bash\necho \"Found $(wc -l < vendor/submodule/submodule.txt) lines in submodule\"\n", "Add test script")
		user.RunCommand(env.ID, "chmod +x test_script.sh", "Make script executable")

		output = user.RunCommand(env.ID, "./test_script.sh", "Run test script")
		assert.Contains(t, output, "Found 1 lines in submodule")
	})
}

// TestSubmoduleWithGitmodulesFile tests that .gitmodules files are handled correctly
func TestSubmoduleWithGitmodulesFile(t *testing.T) {
	t.Parallel()
	WithRepository(t, "submodule-gitmodules", SetupRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Gitmodules Test", "Testing .gitmodules handling")

		// User should be able to read .gitmodules
		gitmodulesContent := user.FileRead(env.ID, ".gitmodules")
		assert.Contains(t, gitmodulesContent, "vendor/submodule")
		assert.Contains(t, gitmodulesContent, "submodule")

		// User should be able to modify .gitmodules
		user.FileWrite(env.ID, ".gitmodules", `[submodule "vendor/submodule"]
	path = vendor/submodule
	url = https://github.com/example/submodule.git
	branch = main
`, "Update .gitmodules")

		// Changes should be visible
		updatedGitmodules := user.FileRead(env.ID, ".gitmodules")
		assert.Contains(t, updatedGitmodules, "branch = main")

		// Submodule files should still be accessible
		content := user.FileRead(env.ID, "vendor/submodule/submodule.txt")
		assert.Contains(t, content, "This is content from the submodule")
	})
}

// TestSubmoduleRealisticWorkflow tests a realistic development workflow with submodules
func TestSubmoduleRealisticWorkflow(t *testing.T) {
	t.Parallel()
	WithRepository(t, "realistic-workflow", SetupRepoWithSubmoduleStructure, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Realistic Workflow", "Testing realistic development workflow")

		// Developer reads existing submodule code
		helperContent := user.FileRead(env.ID, "vendor/submodule/lib/helper.go")
		assert.Contains(t, helperContent, "func Helper()")

		// Developer modifies main code to use submodule
		user.FileWrite(env.ID, "main.go", `package main

import (
	"fmt"
	"github.com/example/submodule/lib"
)

func main() {
	fmt.Println("Main app starting...")
	result := lib.Helper()
	fmt.Println("Helper returned:", result)
	fmt.Println("Main app finished.")
}`, "Update main to use submodule")

		// Developer adds configuration to submodule
		user.FileWrite(env.ID, "vendor/submodule/config.yaml", `database:
  host: localhost
  port: 5432
  name: testdb
logging:
  level: info
  file: app.log`, "Add configuration to submodule")

		// Developer creates a script that processes submodule files
		user.FileWrite(env.ID, "process_config.sh", `#!/bin/bash
echo "Processing configuration..."
if [ -f "vendor/submodule/config.yaml" ]; then
    echo "Config file found:"
    cat vendor/submodule/config.yaml
else
    echo "Config file not found!"
    exit 1
fi
echo "Processing complete."`, "Add config processing script")

		user.RunCommand(env.ID, "chmod +x process_config.sh", "Make script executable")

		// Developer runs the script
		output := user.RunCommand(env.ID, "./process_config.sh", "Run config processing")
		assert.Contains(t, output, "Processing configuration...")
		assert.Contains(t, output, "Config file found:")
		assert.Contains(t, output, "database:")
		assert.Contains(t, output, "Processing complete.")

		// Developer runs tests that depend on submodule
		user.FileWrite(env.ID, "test.sh", `#!/bin/bash
echo "Running tests..."
if [ -f "vendor/submodule/lib/helper.go" ]; then
    echo "Helper library found - tests can run"
    echo "Testing helper function..."
    # Simulate test output
    echo "✓ TestHelper - PASS"
    echo "✓ TestConfig - PASS"
    echo "All tests passed!"
else
    echo "Helper library not found - tests cannot run"
    exit 1
fi`, "Add test script")

		user.RunCommand(env.ID, "chmod +x test.sh", "Make test script executable")

		output = user.RunCommand(env.ID, "./test.sh", "Run tests")
		assert.Contains(t, output, "Running tests...")
		assert.Contains(t, output, "Helper library found")
		assert.Contains(t, output, "All tests passed!")

		// All files should still be accessible after all these operations
		finalMainContent := user.FileRead(env.ID, "main.go")
		assert.Contains(t, finalMainContent, "lib.Helper()")

		finalConfigContent := user.FileRead(env.ID, "vendor/submodule/config.yaml")
		assert.Contains(t, finalConfigContent, "database:")

		finalHelperContent := user.FileRead(env.ID, "vendor/submodule/lib/helper.go")
		assert.Contains(t, finalHelperContent, "func Helper()")
	})
}

// TestRealSubmoduleWorkflow tests with a real submodule (requires network)
func TestRealSubmoduleWorkflow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	t.Parallel()
	WithRepository(t, "real-submodule", SetupRepoWithRealSubmodule, func(t *testing.T, repo *repository.Repository, user *UserActions) {
		// Create environment
		env := user.CreateEnvironment("Real Submodule Test", "Testing with real submodule")

		// User should be able to see real submodule files
		output := user.RunCommand(env.ID, "ls -la vendor/hello-world/", "List real submodule directory")
		assert.Contains(t, output, "README")

		// User should be able to read real submodule files
		readmeContent := user.FileRead(env.ID, "vendor/hello-world/README")
		assert.Contains(t, readmeContent, "Hello World")

		// User should be able to modify files in real submodule
		user.FileWrite(env.ID, "vendor/hello-world/custom.txt", "Custom addition", "Add custom file")

		// Changes should be visible
		customContent := user.FileRead(env.ID, "vendor/hello-world/custom.txt")
		assert.Contains(t, customContent, "Custom addition")

		// User should be able to work with .gitmodules
		gitmodulesContent := user.FileRead(env.ID, ".gitmodules")
		assert.Contains(t, gitmodulesContent, "vendor/hello-world")
		assert.Contains(t, gitmodulesContent, "Hello-World")
	})
}
