package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorDoesNotReportOldXuezhEnvOverrides(t *testing.T) {
	workspace := t.TempDir()
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyPath, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeCLIUserConfig(t, "[workspace]\ndir = \""+workspace+"\"\n\n[azure.speech]\nkey_file = \""+keyPath+"\"\nregion = \"westeurope\"\n")
	t.Setenv("XUEZH_WORKSPACE_DIR", filepath.Join(t.TempDir(), "workspace-poison"))
	t.Setenv("XUEZH_DB_PATH", filepath.Join(t.TempDir(), "db-poison.sqlite3"))
	t.Setenv("XUEZH_AZURE_SPEECH_KEY_FILE", filepath.Join(t.TempDir(), "key-poison"))
	t.Setenv("XUEZH_AZURE_SPEECH_REGION", "env-poison")
	t.Setenv("XUEZH_AZURE_SPEECH_REGION_FILE", filepath.Join(t.TempDir(), "region-poison"))

	code, stdout := captureStdout(t, func() int {
		return runDoctor([]string{"--json"})
	})
	if code != 0 {
		t.Fatalf("doctor exit = %d, stdout=%s", code, stdout)
	}
	if strings.Contains(stdout, "XUEZH_") || strings.Contains(stdout, "override") {
		t.Fatalf("doctor reported old env override fields: %s", stdout)
	}

	var env struct {
		Data struct {
			Checks []struct {
				Name string `json:"name"`
			} `json:"checks"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatal(err)
	}
	for _, check := range env.Data.Checks {
		if check.Name == "azure.speech.env" {
			t.Fatalf("doctor still exposes env check name: %s", stdout)
		}
	}
}
