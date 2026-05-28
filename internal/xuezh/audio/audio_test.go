package audio

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAudioPackageDoesNotImportDatabaseState(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imported := range pkg.Imports {
		if imported == "database/sql" || strings.HasSuffix(imported, "/internal/xuezh/db") {
			t.Fatalf("audio package imports stateful storage package %q", imported)
		}
	}
}

func TestBuildTTSCommandPassesNegativeRateAsSingleArg(t *testing.T) {
	cmd := buildTTSCommand("你是龙大。", "zh-CN-XiaoxiaoNeural", "-23%", "out.mp3")
	for _, arg := range cmd {
		if arg == "--rate" {
			t.Fatalf("negative edge-tts rates must be passed as --rate=-23%%, got separate args: %#v", cmd)
		}
	}
	found := false
	for _, arg := range cmd {
		if arg == "--rate=-23%" {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing rate arg: %#v", cmd)
	}
}

func TestAzureCredentialsIgnoreOldEnvOverrides(t *testing.T) {
	writeAudioUserConfig(t, "")
	t.Setenv("XUEZH_AZURE_SPEECH_KEY_FILE", filepath.Join(t.TempDir(), "key"))
	t.Setenv("XUEZH_AZURE_SPEECH_REGION", "westeurope")
	t.Setenv("XUEZH_AZURE_SPEECH_REGION_FILE", filepath.Join(t.TempDir(), "region"))

	_, _, err := azureCredentials()
	if err == nil || !strings.Contains(err.Error(), "Azure Speech credentials missing") {
		t.Fatalf("expected missing credentials without config, got %v", err)
	}
}

func TestAzureCredentialsReadConfig(t *testing.T) {
	keyPath := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(keyPath, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeAudioUserConfig(t, "[azure.speech]\nkey_file = \""+keyPath+"\"\nregion = \"westeurope\"\n")

	key, region, err := azureCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if key != "secret" || region != "westeurope" {
		t.Fatalf("credentials = %q, %q", key, region)
	}
}

func TestInlineDetailMaxBytesIgnoresOldEnvOverride(t *testing.T) {
	writeAudioUserConfig(t, "")
	t.Setenv("XUEZH_AUDIO_INLINE_MAX_BYTES", "1")

	if got := inlineDetailMaxBytes(); got != 200000 {
		t.Fatalf("inline max bytes = %d, want default", got)
	}
}

func TestInlineDetailMaxBytesReadsConfig(t *testing.T) {
	writeAudioUserConfig(t, "[audio]\ninline_max_bytes = 12345\n")

	if got := inlineDetailMaxBytes(); got != 12345 {
		t.Fatalf("inline max bytes = %d, want config value", got)
	}
}

func writeAudioUserConfig(t *testing.T, body string) {
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
