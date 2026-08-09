package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// clientRoots holds the roots provided by the MCP client. Agents frequently
// pass a wrong environment_source; the roots give them a concrete hint about
// which repositories the client actually has open.
//
// This assumes a single client, which may go out the window when we add
// support for streaming http.
var (
	clientRoots   []mcp.Root
	clientRootsMu sync.RWMutex
)

// requestRoots sends a roots/list request to the client and stores the
// result. Errors are logged and swallowed: roots are a hint, never required.
func requestRoots(ctx context.Context, s *server.MCPServer) {
	// The ctx of hooks/notification handlers is canceled as soon as the
	// triggering message is done processing, which would abort the request
	// before the client can answer. Detach cancellation (values like the
	// client session are kept) and bound the wait instead.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 30*time.Second)
	defer cancel()

	result, err := s.RequestRoots(ctx, mcp.ListRootsRequest{})
	if err != nil {
		slog.Info("Failed to request roots from client", "error", err)
		return
	}

	clientRootsMu.Lock()
	clientRoots = result.Roots
	clientRootsMu.Unlock()

	slog.Info("Updated client roots", "count", len(result.Roots))
}

// repoOpenErrorMessage provides helpful error messages when repository
// opening fails, listing the roots the client has open when available.
func repoOpenErrorMessage(source string, originalErr error) error {
	baseMsg := fmt.Sprintf("unable to open repository '%s'", source)

	clientRootsMu.RLock()
	defer clientRootsMu.RUnlock()

	if len(clientRoots) > 0 {
		baseMsg += "\n\nAvailable roots from client:"
		for _, root := range clientRoots {
			uri := strings.TrimPrefix(root.URI, "file://")
			if root.Name != "" {
				baseMsg += fmt.Sprintf("\n  - %s (%s)", uri, root.Name)
			} else {
				baseMsg += fmt.Sprintf("\n  - %s", uri)
			}
		}
		return fmt.Errorf("%s: %w", baseMsg, originalErr)
	}

	// Fallback: suggest common patterns
	baseMsg += "\n\nTry using:\n  - '.' for current directory\n  - An absolute path to your git repository"
	return fmt.Errorf("%s: %w", baseMsg, originalErr)
}
