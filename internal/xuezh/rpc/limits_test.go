package rpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func TestHandlerRejectsOversizeRequestsBeforeBackendWork(t *testing.T) {
	useRPCTestWorkspace(t)
	client := newRPCTestClient(t)

	_, err := client.ProcessVoice(context.Background(), connect.NewRequest(&xuezhv1.ProcessVoiceRequest{
		Audio:    make([]byte, processVoiceAudioMaxBytes+1),
		Filename: "voice.ogg",
		Mime:     "audio/ogg",
		RefText:  "你好",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("process voice error = %v, code=%s", err, connect.CodeOf(err))
	}

	_, err = client.SynthesizeSpeech(context.Background(), connect.NewRequest(&xuezhv1.SynthesizeSpeechRequest{
		Text:         strings.Repeat("你", ttsTextMaxRunes+1),
		Voice:        "XiaoxiaoNeural",
		OutputFormat: "ogg",
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("tts error = %v, code=%s", err, connect.CodeOf(err))
	}

	_, err = client.PutContent(context.Background(), connect.NewRequest(&xuezhv1.PutContentRequest{
		Type:     "story",
		Key:      "too-big",
		Filename: "story.txt",
		Content:  make([]byte, contentBytesMax+1),
	}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("content error = %v, code=%s", err, connect.CodeOf(err))
	}
}

func TestHandlerRejectsOversizeInlineAudio(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tool scripts are POSIX shell")
	}
	useRPCTestWorkspaceWithConfig(t, "\n[audio]\ntts_backend = \"edge-tts\"\n")
	addFakeOversizeTTSTools(t)
	client := newRPCTestClient(t)

	_, err := client.SynthesizeSpeech(context.Background(), connect.NewRequest(&xuezhv1.SynthesizeSpeechRequest{
		Text:         "你好",
		Voice:        "XiaoxiaoNeural",
		OutputFormat: "ogg",
	}))
	if connect.CodeOf(err) != connect.CodeResourceExhausted {
		t.Fatalf("tts oversize error = %v, code=%s", err, connect.CodeOf(err))
	}
}

func newRPCTestClient(t *testing.T) xuezhv1connect.XuezhServiceClient {
	t.Helper()
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
}

func addFakeOversizeTTSTools(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "edge-tts"), `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --write-media) shift; out="$1" ;;
  esac
  shift
done
printf 'fake-edge-tts' > "$out"
`)
	writeExecutable(t, filepath.Join(bin, "ffmpeg"), `#!/bin/sh
out=""
for arg do
  out="$arg"
done
dd if=/dev/zero of="$out" bs=1048576 count=6 >/dev/null 2>&1
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}
