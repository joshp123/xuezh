package webserver

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/cram"
)

func TestMuxServesLearnerStateRPC(t *testing.T) {
	useWebserverRPCTestWorkspace(t)
	if _, err := cram.ImportHelloChinese(cram.ImportOptions{Path: filepath.Join("..", "cram", "testdata", "hellochinese.txt"), AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	server := httptest.NewServer(newMux())
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	resp, err := client.GetLearnerState(context.Background(), connect.NewRequest(&xuezhv1.GetLearnerStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Msg.Cards) != 3 {
		t.Fatalf("unexpected cards: %+v", resp.Msg.Cards)
	}
}

func useWebserverRPCTestWorkspace(t *testing.T) {
	t.Helper()
	workspace := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPath := filepath.Join(configHome, "xuezh", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[workspace]\ndir = \"" + workspace + "\"\n"
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
