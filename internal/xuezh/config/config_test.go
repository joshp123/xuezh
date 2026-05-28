package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadRejectsClientAndWorkspaceInSameConfig(t *testing.T) {
	configDir := t.TempDir()
	writeConfig(t, filepath.Join(configDir, "xuezh", "config.toml"), `
[client]
server_url = "https://chinese.jjpcodes.com"

[workspace]
dir = "/var/lib/xuezh"
`)
	t.Setenv("XDG_CONFIG_HOME", configDir)
	withHostConfigPath(t, filepath.Join(t.TempDir(), "missing.toml"))

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "cannot contain both [client] and [workspace]") {
		t.Fatalf("expected mixed client/workspace config error, got %v", err)
	}
}

func TestLoadRejectsHostAndUserConfigConflict(t *testing.T) {
	configDir := t.TempDir()
	writeConfig(t, filepath.Join(configDir, "xuezh", "config.toml"), "[client]\nserver_url = \"https://chinese.jjpcodes.com\"\n")
	hostPath := filepath.Join(t.TempDir(), "config.toml")
	writeConfig(t, hostPath, "[workspace]\ndir = \"/var/lib/xuezh\"\n")
	t.Setenv("XDG_CONFIG_HOME", configDir)
	withHostConfigPath(t, hostPath)

	_, err := Load()
	var conflict ConfigConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected ConfigConflictError, got %T %v", err, err)
	}
}

func TestGetStringReadsUserConfig(t *testing.T) {
	configDir := t.TempDir()
	writeConfig(t, filepath.Join(configDir, "xuezh", "config.toml"), "[workspace]\ndir = \"/tmp/xuezh-test\"\n")
	t.Setenv("XDG_CONFIG_HOME", configDir)
	withHostConfigPath(t, filepath.Join(t.TempDir(), "missing.toml"))

	value, ok, err := GetString("workspace", "dir")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || value != "/tmp/xuezh-test" {
		t.Fatalf("workspace.dir = %q, %v", value, ok)
	}
}

func writeConfig(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func withHostConfigPath(t *testing.T, path string) {
	t.Helper()
	original := hostConfigPath
	hostConfigPath = path
	t.Cleanup(func() { hostConfigPath = original })
}
