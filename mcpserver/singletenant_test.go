package mcpserver

import (
	"testing"
)

func TestSingleTenantEnvironmentStorage(t *testing.T) {
	// Test setting and getting environment ID
	testEnvID := "test-env-id"
	testEnvSource := "/test/source/path"

	// Test individual setters and getters
	setCurrentEnvironmentID(testEnvID)
	setCurrentEnvironmentSource(testEnvSource)

	retrievedID, err := getCurrentEnvironmentID()
	if err != nil {
		t.Fatalf("Expected no error getting environment ID, got: %v", err)
	}
	if retrievedID != testEnvID {
		t.Fatalf("Expected environment ID %s, got: %s", testEnvID, retrievedID)
	}

	retrievedSource, err := getCurrentEnvironmentSource()
	if err != nil {
		t.Fatalf("Expected no error getting environment source, got: %v", err)
	}
	if retrievedSource != testEnvSource {
		t.Fatalf("Expected environment source %s, got: %s", testEnvSource, retrievedSource)
	}

	// Test combined setter
	newEnvID := "new-env-id"
	newEnvSource := "/new/source/path"
	setCurrentEnvironment(newEnvID, newEnvSource)

	retrievedID, err = getCurrentEnvironmentID()
	if err != nil {
		t.Fatalf("Expected no error getting environment ID after combined set, got: %v", err)
	}
	if retrievedID != newEnvID {
		t.Fatalf("Expected environment ID %s after combined set, got: %s", newEnvID, retrievedID)
	}

	retrievedSource, err = getCurrentEnvironmentSource()
	if err != nil {
		t.Fatalf("Expected no error getting environment source after combined set, got: %v", err)
	}
	if retrievedSource != newEnvSource {
		t.Fatalf("Expected environment source %s after combined set, got: %s", newEnvSource, retrievedSource)
	}

	// Clear state for other tests
	setCurrentEnvironment("", "")
}

func TestSingleTenantEnvironmentStorageEmpty(t *testing.T) {
	// Clear state
	setCurrentEnvironment("", "")

	// Test error when no environment is set
	_, err := getCurrentEnvironmentID()
	if err == nil {
		t.Fatal("Expected error when no environment ID is set")
	}

	_, err = getCurrentEnvironmentSource()
	if err == nil {
		t.Fatal("Expected error when no environment source is set")
	}
}
