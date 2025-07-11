package mcpserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// clientRoots holds the roots provided by the client
// this assumes a single client, which may go out the window when we add support for streaming http.
var (
	clientRoots   []mcp.Root
	clientRootsMu sync.RWMutex
)

// sendRootsListRequest sends a roots/list request to the client
func sendRootsListRequest(ctx context.Context, session server.ClientSession) {
	if session == nil {
		return
	}

	// Send roots/list request as a notification
	// Note: In a proper implementation, this would be a request-response
	// but for now we'll send as notification and handle response separately
	notification := mcp.JSONRPCNotification{
		JSONRPC: "2.0",
		Notification: mcp.Notification{
			Method: "roots/list",
			Params: mcp.NotificationParams{},
		},
	}

	select {
	case session.NotificationChannel() <- notification:
		slog.Info("Requested roots/list from client")
	case <-ctx.Done():
		return
	}
}

// updateClientRoots parses roots from notification params and updates the global clientRoots
func updateClientRoots(notification mcp.JSONRPCNotification) int {
	if notification.Params.AdditionalFields == nil {
		return 0
	}
	rootsData, ok := notification.Params.AdditionalFields["roots"].([]any)
	if !ok {
		return 0
	}

	newRoots := make([]mcp.Root, 0, len(rootsData))
	for _, rootData := range rootsData {
		if rootMap, ok := rootData.(map[string]any); ok {
			root := mcp.Root{}
			if uri, ok := rootMap["uri"].(string); ok {
				root.URI = uri
			}
			if name, ok := rootMap["name"].(string); ok {
				root.Name = name
			}
			newRoots = append(newRoots, root)
		}
	}

	clientRootsMu.Lock()
	clientRoots = newRoots
	clientRootsMu.Unlock()

	return len(newRoots)
}

// repoOpenErrorMessage provides helpful error messages when repository opening fails
func repoOpenErrorMessage(source string, originalErr error) error {
	baseMsg := fmt.Sprintf("failed to open repository '%s'", source)

	// If we have client roots, suggest them
	clientRootsMu.RLock()
	defer clientRootsMu.RUnlock()

	if len(clientRoots) > 0 {
		baseMsg += "\n\nAvailable roots from client:"
		for _, root := range clientRoots {
			uri := root.URI
			uri = strings.TrimPrefix(uri, "file://")
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
