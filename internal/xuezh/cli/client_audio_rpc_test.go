package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
)

func TestClientBackedAudioCommandsUseRPCAndLocalDeliveryFiles(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	stub := clientRPCStub{
		ttsRequests:          make(chan *xuezhv1.SynthesizeSpeechRequest, 1),
		processVoiceRequests: make(chan *xuezhv1.ProcessVoiceRequest, 1),
	}
	server := newClientRPCServer(t, stub)
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	outPath := filepath.Join(t.TempDir(), "tts.ogg")
	tts := runClientCommandForTest(t, []string{"audio", "tts", "--text", "你好", "--voice", "XiaoxiaoNeural", "--out", outPath, "--backend", "ignored-client-backend", "--json"})
	ttsReq := <-stub.ttsRequests
	if ttsReq.GetText() != "你好" || ttsReq.GetVoice() != "XiaoxiaoNeural" || ttsReq.GetOutputFormat() != "ogg" {
		t.Fatalf("tts request = %+v", ttsReq)
	}
	if data, err := os.ReadFile(outPath); err != nil || string(data) != "remote-tts" {
		t.Fatalf("tts delivery file data=%q err=%v", data, err)
	}
	if tts.Command != "audio.tts" || tts.Data["delivery_path"] != outPath || tts.Data["text"] != "你好" {
		t.Fatalf("tts envelope = %#v", tts)
	}

	voicePath := filepath.Join(t.TempDir(), "voice.ogg")
	if err := os.WriteFile(voicePath, []byte("local-voice"), 0o644); err != nil {
		t.Fatal(err)
	}
	processed := runClientCommandForTest(t, []string{"audio", "process-voice", "--in", voicePath, "--ref-text", "你好", "--json"})
	processReq := <-stub.processVoiceRequests
	if string(processReq.GetAudio()) != "local-voice" || processReq.GetFilename() != "voice.ogg" || processReq.GetRefText() != "你好" {
		t.Fatalf("process voice request = %+v", processReq)
	}
	if processed.Command != "audio.process-voice" || processed.Data["attempt_id"] != "attempt_remote" || processed.Data["ref_text"] != "你好" {
		t.Fatalf("process voice envelope = %#v", processed)
	}
	if len(processed.Artifacts) != 2 {
		t.Fatalf("process voice artifacts = %#v", processed.Artifacts)
	}
	localFeedback, ok := processed.Artifacts[1]["path"].(string)
	if !ok || filepath.Dir(localFeedback) != filepath.Join(cacheHome, "xuezh", "audio") {
		t.Fatalf("feedback delivery artifact = %#v", processed.Artifacts)
	}
	if data, err := os.ReadFile(localFeedback); err != nil || string(data) != "remote-feedback" {
		t.Fatalf("feedback delivery file data=%q err=%v", data, err)
	}
}

func (s clientRPCStub) SynthesizeSpeech(_ context.Context, req *connect.Request[xuezhv1.SynthesizeSpeechRequest]) (*connect.Response[xuezhv1.SynthesizeSpeechResponse], error) {
	if s.ttsRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("tts request channel missing"))
	}
	s.ttsRequests <- req.Msg
	bytes := int64(10)
	return connect.NewResponse(&xuezhv1.SynthesizeSpeechResponse{
		Text:    req.Msg.GetText(),
		Voice:   "zh-CN-XiaoxiaoNeural",
		Backend: &xuezhv1.BackendInfo{Id: "edge-tts", Features: []string{"tts"}},
		Artifacts: []*xuezhv1.ServerArtifact{{
			Path:    "artifacts/tts.ogg",
			Mime:    "audio/ogg",
			Purpose: "tts_audio",
			Bytes:   &bytes,
		}},
		Audio: &xuezhv1.InlineFile{Data: []byte("remote-tts"), Mime: "audio/ogg", Filename: "tts.ogg"},
	}), nil
}

func (s clientRPCStub) ProcessVoice(_ context.Context, req *connect.Request[xuezhv1.ProcessVoiceRequest]) (*connect.Response[xuezhv1.ProcessVoiceResponse], error) {
	if s.processVoiceRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("process voice request channel missing"))
	}
	s.processVoiceRequests <- req.Msg
	assessment, err := structpb.NewStruct(map[string]any{"exact_match": true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	transcript, err := structpb.NewStruct(map[string]any{"text": "你好"})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	limits, err := structpb.NewStruct(map[string]any{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	bytes := int64(15)
	return connect.NewResponse(&xuezhv1.ProcessVoiceResponse{
		AttemptId:  "attempt_remote",
		RefText:    req.Msg.GetRefText(),
		Backend:    &xuezhv1.BackendInfo{Id: "local", Features: []string{"assessment", "tts", "stt", "convert"}},
		Assessment: assessment,
		Transcript: transcript,
		Artifacts: []*xuezhv1.ServerArtifact{{
			Path:    "artifacts/assessment.json",
			Mime:    "application/json",
			Purpose: "assessment",
			Bytes:   &bytes,
		}},
		FeedbackAudio: &xuezhv1.InlineFile{Data: []byte("remote-feedback"), Mime: "audio/ogg", Filename: "feedback.ogg"},
		Limits:        limits,
	}), nil
}
