package main_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestMain(m *testing.M) {
	// Create a short root; on Windows prefer C:\Temp
	root := shortRoot("cu-stdio")
	cache := filepath.Join(root, "cache")
	tmp := filepath.Join(root, "tmp")
	app := filepath.Join(root, "appdata")
	lapp := filepath.Join(root, "localappdata")
	_ = os.MkdirAll(cache, 0o755)
	_ = os.MkdirAll(tmp, 0o755)
	_ = os.MkdirAll(app, 0o755)
	_ = os.MkdirAll(lapp, 0o755)

	// Shared app/data home for all child servers
	_ = os.Setenv("XDG_CACHE_HOME", cache)
	if runtime.GOOS == "windows" {
		_ = os.Setenv("TEMP", tmp)
		_ = os.Setenv("TMP", tmp)
		_ = os.Setenv("APPDATA", app)
		_ = os.Setenv("LOCALAPPDATA", lapp)
	} else {
		_ = os.Setenv("TMPDIR", tmp)
	}

	// Make Git allow long paths without touching system/global config
	gitcfg := filepath.Join(root, "gitconfig")
	_ = os.WriteFile(gitcfg, []byte("[core]\nlongpaths = true\n"), 0o644)
	_ = os.Setenv("GIT_CONFIG_GLOBAL", gitcfg)

	if os.Getenv("TEST_VERBOSE") != "" {
		slog.Info("stdio test cache configured",
			"XDG_CACHE_HOME", cache,
			"TEMP/TMP/TMPDIR", tmp,
			"APPDATA", app)
	}

	code := m.Run()
	_ = os.RemoveAll(root)
	os.Exit(code)
}

func shortRoot(prefix string) string {
	if runtime.GOOS != "windows" {
		dir, _ := os.MkdirTemp("", prefix+"-")
		return dir
	}
	base := `C:\Temp`
	_ = os.MkdirAll(base, 0o755)
	dir, _ := os.MkdirTemp(base, prefix+"-")
	return dir
}
