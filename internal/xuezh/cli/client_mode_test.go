package cli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/joshp123/xuezh/internal/xuezh/config"
)

func TestClientCommandMatrixMatchesContract(t *testing.T) {
	contractIDs := readContractCommandIDs(t)
	classified := make([]string, 0, len(clientCommandModes))
	for commandID := range clientCommandModes {
		classified = append(classified, commandID)
	}
	sort.Strings(classified)

	if !equalStrings(contractIDs, classified) {
		t.Fatalf("client command matrix drift\ncontract:   %v\nclassified: %v", contractIDs, classified)
	}
}

func TestClientCommandModes(t *testing.T) {
	tests := []struct {
		commandID string
		want      clientCommandMode
	}{
		{"version", clientCommandLocal},
		{"learner.state", clientCommandRPC},
		{"audio.tts", clientCommandRPC},
		{"audio.process-voice", clientCommandRPC},
		{"content.cache.put", clientCommandRPC},
		{"db.init", clientCommandUnsupported},
		{"audio.convert", clientCommandUnsupported},
		{"web.serve", clientCommandUnsupported},
	}

	for _, tt := range tests {
		got, ok := clientCommandModeForID(tt.commandID)
		if !ok {
			t.Fatalf("%s is not classified", tt.commandID)
		}
		if got != tt.want {
			t.Fatalf("%s mode = %s, want %s", tt.commandID, got, tt.want)
		}
	}
}

func TestUnsupportedClientCommandDoesNotCreateWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XUEZH_WORKSPACE_DIR", "")
	t.Setenv("XUEZH_DB_PATH", "")

	code, stdout := captureStdout(t, func() int {
		return emitUnsupportedClientCommand("db.init", "https://chinese.jjpcodes.com")
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	var env struct {
		OK    bool `json:"ok"`
		Error struct {
			Type    string         `json:"type"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, stdout)
	}
	if env.OK {
		t.Fatal("unsupported client command returned ok envelope")
	}
	if env.Error.Type != "UNSUPPORTED_CLIENT_COMMAND" {
		t.Fatalf("error type = %q", env.Error.Type)
	}
	if env.Error.Details["server_url"] != "https://chinese.jjpcodes.com" {
		t.Fatalf("server_url detail = %#v", env.Error.Details["server_url"])
	}

	for _, path := range []string{
		filepath.Join(home, ".clawdbot"),
		filepath.Join(home, "Library", "Application Support", "xuezh"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("unsupported client command touched workspace path %s: %v", path, err)
		}
	}
}

func TestEmitErrorMapsConfigConflict(t *testing.T) {
	code, stdout := captureStdout(t, func() int {
		return emitError("doctor", config.ConfigConflictError{HostPath: "/etc/xuezh/config.toml", UserPath: "/home/me/.config/xuezh/config.toml"})
	})
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}

	var env struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatal(err)
	}
	if env.Error.Type != "CONFIG_CONFLICT" {
		t.Fatalf("error type = %q", env.Error.Type)
	}
}

func readContractCommandIDs(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "specs", "cli", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		Commands []struct {
			ID string `json:"id"`
		} `json:"commands"`
	}
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(contract.Commands))
	for _, command := range contract.Commands {
		ids = append(ids, command.ID)
	}
	sort.Strings(ids)
	return ids
}

func captureStdout(t *testing.T, fn func() int) (int, string) {
	t.Helper()
	orig := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	defer func() { os.Stdout = orig }()

	code := fn()
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, reader); err != nil {
		t.Fatal(err)
	}
	return code, buf.String()
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
