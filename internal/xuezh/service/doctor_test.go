package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppDoctorReportsWorkspaceAndConfigWithoutEnvOverrides(t *testing.T) {
	workspace := t.TempDir()
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyPath, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeServiceConfig(t, "[workspace]\ndir = \""+workspace+"\"\n\n[azure.speech]\nkey_file = \""+keyPath+"\"\nregion = \"westeurope\"\n")
	t.Setenv("XUEZH_WORKSPACE_DIR", filepath.Join(t.TempDir(), "workspace-poison"))
	t.Setenv("XUEZH_AZURE_SPEECH_KEY_FILE", filepath.Join(t.TempDir(), "key-poison"))
	t.Setenv("XUEZH_AZURE_SPEECH_REGION", "env-poison")

	result, err := New().Doctor("server")
	if err != nil {
		t.Fatal(err)
	}
	if result.WorkspaceRole != "server" || result.WorkspacePath != workspace {
		t.Fatalf("unexpected doctor workspace: %+v", result)
	}
	workspaceCheck := doctorCheckByName(result.Checks, "workspace.path")
	if workspaceCheck == nil || !workspaceCheck.OK || workspaceCheck.Details["path"] != workspace {
		t.Fatalf("workspace.path check = %+v", workspaceCheck)
	}
	azureCheck := doctorCheckByName(result.Checks, "azure.speech.config")
	if azureCheck == nil || !azureCheck.OK || azureCheck.Details["config_key"] != true || azureCheck.Details["config_region"] != true {
		t.Fatalf("azure.speech.config check = %+v", azureCheck)
	}
	if doctorCheckByName(result.Checks, "azure.speech.env") != nil {
		t.Fatalf("doctor still exposes old env override check: %+v", result.Checks)
	}
}

func writeServiceConfig(t *testing.T, body string) {
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

func doctorCheckByName(checks []DoctorCheck, name string) *DoctorCheck {
	for i := range checks {
		if checks[i].Name == name {
			return &checks[i]
		}
	}
	return nil
}
