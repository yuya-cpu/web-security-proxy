package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yuya-cpu/web-security-proxy/internal/config"
)

func TestLoad_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	err := os.WriteFile(path, []byte("server:\n  port: 9000\n"), 0o644)
	require.NoError(t, err)

	cfg, err := config.Load(path)
	require.NoError(t, err)

	assert.Equal(t, 9000, cfg.Server.Port)
	assert.Equal(t, 8888, cfg.Proxy.Port)
	assert.Equal(t, "./data/proxy.db", cfg.Database.Path)
	assert.Equal(t, "127.0.0.1:9000", cfg.Server.Addr())
}
