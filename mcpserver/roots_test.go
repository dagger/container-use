package mcpserver

import (
	"errors"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
)

func withClientRoots(t *testing.T, roots []mcp.Root) {
	t.Helper()
	clientRootsMu.Lock()
	old := clientRoots
	clientRoots = roots
	clientRootsMu.Unlock()
	t.Cleanup(func() {
		clientRootsMu.Lock()
		clientRoots = old
		clientRootsMu.Unlock()
	})
}

func TestRepoOpenErrorMessage_NoRoots(t *testing.T) {
	withClientRoots(t, nil)

	err := repoOpenErrorMessage("/nonexistent", errors.New("not a git repo"))

	assert.ErrorIs(t, err, err)
	assert.Contains(t, err.Error(), "unable to open repository '/nonexistent'")
	assert.Contains(t, err.Error(), "'.' for current directory")
	assert.Contains(t, err.Error(), "not a git repo")
}

func TestRepoOpenErrorMessage_WithRoots(t *testing.T) {
	withClientRoots(t, []mcp.Root{
		{URI: "file:///home/user/project", Name: "project"},
		{URI: "file:///home/user/other"},
	})

	err := repoOpenErrorMessage("/nonexistent", errors.New("not a git repo"))

	assert.Contains(t, err.Error(), "Available roots from client:")
	assert.Contains(t, err.Error(), "- /home/user/project (project)")
	assert.Contains(t, err.Error(), "- /home/user/other")
	assert.NotContains(t, err.Error(), "file://")
	assert.Contains(t, err.Error(), "not a git repo")
}

func TestRepoOpenErrorMessage_WrapsOriginal(t *testing.T) {
	withClientRoots(t, nil)
	original := errors.New("root cause")

	err := repoOpenErrorMessage(".", original)

	assert.ErrorIs(t, err, original)
}
