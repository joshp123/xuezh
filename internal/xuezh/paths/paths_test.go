package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWorkspaceDirReadsConfig(t *testing.T) {
	workspace := t.TempDir()
	writeUserConfig(t, "[workspace]\ndir = \""+workspace+"\"\n")

	got, err := WorkspaceDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != workspace {
		t.Fatalf("workspace = %q, want %q", got, workspace)
	}

	dbPath, err := DBPath()
	if err != nil {
		t.Fatal(err)
	}
	if dbPath != filepath.Join(workspace, "db.sqlite3") {
		t.Fatalf("db path = %q", dbPath)
	}
}

func TestWorkspaceDirIgnoresOldEnvOverride(t *testing.T) {
	home := t.TempDir()
	poison := filepath.Join(t.TempDir(), "poison")
	t.Setenv("HOME", home)
	t.Setenv("XUEZH_WORKSPACE_DIR", poison)
	t.Setenv("XUEZH_DB_PATH", filepath.Join(poison, "poison.sqlite3"))
	writeUserConfig(t, "")

	got, err := WorkspaceDir()
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(home, defaultWorkspace)
	if got != want {
		t.Fatalf("workspace = %q, want %q", got, want)
	}
}

func writeUserConfig(t *testing.T, body string) {
	t.Helper()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	path := filepath.Join(configHome, "xuezh", "config.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
