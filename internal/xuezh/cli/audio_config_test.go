package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAudioBackendUsesConfigBeforeDefault(t *testing.T) {
	writeCLIUserConfig(t, "[audio]\ntts_backend = \"edge-config\"\n")
	t.Setenv("XUEZH_AUDIO_TTS_BACKEND", "env-poison")
	t.Setenv("XUEZH_AUDIO_BACKEND", "env-global-poison")

	got := resolveAudioBackend("", "edge-default", "tts_backend")
	if got != "edge-config" {
		t.Fatalf("backend = %q, want config value", got)
	}
}

func TestResolveAudioBackendIgnoresOldEnvOverrides(t *testing.T) {
	writeCLIUserConfig(t, "")
	t.Setenv("XUEZH_AUDIO_TTS_BACKEND", "env-poison")
	t.Setenv("XUEZH_AUDIO_BACKEND", "env-global-poison")

	got := resolveAudioBackend("", "edge-default", "tts_backend")
	if got != "edge-default" {
		t.Fatalf("backend = %q, want default", got)
	}
}

func writeCLIUserConfig(t *testing.T, body string) {
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
