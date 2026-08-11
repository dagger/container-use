package environment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKVListSetUpdatesExisting(t *testing.T) {
	kv := KVList{"FOO=bar", "BAZ=qux"}
	kv.Set("FOO", "new")
	assert.Equal(t, "new", kv.Get("FOO"))
	assert.Equal(t, []string{"BAZ=qux", "FOO=new"}, []string(kv))
}

func TestKVListUnsetReturnsFound(t *testing.T) {
	kv := KVList{"FOO=bar", "BAZ=qux"}
	assert.True(t, kv.Unset("FOO"))
	assert.False(t, kv.Unset("FOO"))
	assert.Equal(t, []string{"BAZ=qux"}, []string(kv))
}

func TestKVListClear(t *testing.T) {
	kv := KVList{"FOO=bar"}
	kv.Clear()
	assert.Empty(t, kv)
}

func TestKVListKeys(t *testing.T) {
	kv := KVList{"FOO=bar", "=nokey", "BAZ=qux", ""}
	assert.Equal(t, []string{"FOO", "BAZ"}, kv.Keys())
}

func TestKVListGetEmpty(t *testing.T) {
	var kv KVList
	assert.Equal(t, "", kv.Get("missing"))
}

func TestServiceConfigsGet(t *testing.T) {
	sc := ServiceConfigs{
		{Name: "api", Image: "nginx"},
		{Name: "db", Image: "postgres"},
	}
	assert.Equal(t, "nginx", sc.Get("api").Image)
	assert.Nil(t, sc.Get("missing"))
}

func TestEnvironmentConfigCopy(t *testing.T) {
	orig := &EnvironmentConfig{
		BaseImage: "ubuntu:24.04",
		Services:  ServiceConfigs{{Name: "svc", Image: "img"}},
	}
	cp := orig.Copy()
	cp.BaseImage = "alpine"
	cp.Services[0].Image = "changed"

	assert.Equal(t, "ubuntu:24.04", orig.BaseImage)
	assert.Equal(t, "img", orig.Services[0].Image)
	assert.Equal(t, "alpine", cp.BaseImage)
	assert.Equal(t, "changed", cp.Services[0].Image)
}

func TestEnvironmentConfigSave(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.BaseImage = "custom:image"
	require.NoError(t, cfg.Save(dir))

	data, err := os.ReadFile(filepath.Join(dir, configDir, environmentFile))
	require.NoError(t, err)
	assert.Contains(t, string(data), "custom:image")

	info, err := os.Stat(filepath.Join(dir, configDir, environmentFile))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())
}
