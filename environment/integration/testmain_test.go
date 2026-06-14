package integration

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMain isolates cache/temp per test process to avoid Dagger cache races on Windows.
func TestMain(m *testing.M) {
	// Private sandbox (per process)
	root, err := os.MkdirTemp("", "cu-test-")
	if err != nil {
		panic(err)
	}
	cacheHome := filepath.Join(root, "cache")
	tmpHome := filepath.Join(root, "tmp")
	if err := os.MkdirAll(cacheHome, 0o755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(tmpHome, 0o755); err != nil {
		panic(err)
	}

	// Dagger respects XDG; temp isolation also helps with path length/locks on Windows.
	_ = os.Setenv("XDG_CACHE_HOME", cacheHome)
	if runtime.GOOS == "windows" {
		_ = os.Setenv("TEMP", tmpHome)
		_ = os.Setenv("TMP", tmpHome)
	} else {
		_ = os.Setenv("TMPDIR", tmpHome)
	}

	if os.Getenv("TEST_VERBOSE") != "" {
		slog.Info("using isolated test cache", "XDG_CACHE_HOME", cacheHome, "tmp", tmpHome)
	}

	code := m.Run()

	// Release file handles before cleanup.
	if testDaggerClient != nil {
		_ = testDaggerClient.Close()
	}
	_ = os.RemoveAll(root)

	os.Exit(code)
}
