// Package mcpserver provides single-tenant mode functionality for MCP servers.
//
// In single-tenant mode, a single MCP server process is assumed to serve only one
// chat session. This allows for optimizations where environment_id parameters can
// be omitted from most tools, with the server maintaining the current environment
// in memory.

package mcpserver

import (
	"fmt"
	"sync"
)

var (
	// currentEnvironmentID stores the current environment ID for single-tenant mode
	// This is per-server-process, not persisted to disk
	currentEnvironmentID string
	// currentEnvironmentSource stores the current environment source for single-tenant mode
	// This is per-server-process, not persisted to disk
	currentEnvironmentSource string
	currentEnvMutex          sync.RWMutex
)

// getCurrentEnvironmentID returns the current environment ID for single-tenant mode
func getCurrentEnvironmentID() (string, error) {
	currentEnvMutex.RLock()
	defer currentEnvMutex.RUnlock()

	if currentEnvironmentID == "" {
		return "", fmt.Errorf("no current environment set. Use environment_create or environment_open first")
	}
	return currentEnvironmentID, nil
}

// getCurrentEnvironmentSource returns the current environment source for single-tenant mode
func getCurrentEnvironmentSource() (string, error) {
	currentEnvMutex.RLock()
	defer currentEnvMutex.RUnlock()

	if currentEnvironmentSource == "" {
		return "", fmt.Errorf("no current environment set. Use environment_create or environment_open first")
	}
	return currentEnvironmentSource, nil
}

// setCurrentEnvironmentID sets the current environment ID for single-tenant mode
func setCurrentEnvironmentID(envID string) {
	currentEnvMutex.Lock()
	defer currentEnvMutex.Unlock()
	currentEnvironmentID = envID
}

// setCurrentEnvironmentSource sets the current environment source for single-tenant mode
func setCurrentEnvironmentSource(envSource string) {
	currentEnvMutex.Lock()
	defer currentEnvMutex.Unlock()
	currentEnvironmentSource = envSource
}

// setCurrentEnvironment sets both the current environment ID and source for single-tenant mode
func setCurrentEnvironment(envID, envSource string) {
	currentEnvMutex.Lock()
	defer currentEnvMutex.Unlock()
	currentEnvironmentID = envID
	currentEnvironmentSource = envSource
}
