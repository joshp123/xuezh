package cli

import (
	"context"
	"flag"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func runClientAudioTTS(args []string, serverURL string) int {
	fs := flag.NewFlagSet("audio tts", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	text := fs.String("text", "", "text")
	voice := fs.String("voice", "XiaoxiaoNeural", "voice")
	outPath := fs.String("out", "", "output path")
	_ = fs.String("backend", "", "backend")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *text == "" || *outPath == "" {
		return emitTypedError("audio.tts", "INVALID_ARGUMENT", "text and out are required", map[string]any{"text": *text, "out": *outPath})
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.SynthesizeSpeech(context.Background(), connect.NewRequest(&xuezhv1.SynthesizeSpeechRequest{
		Text:         *text,
		Voice:        *voice,
		OutputFormat: outputFormatForPath(*outPath),
	}))
	if err != nil {
		return emitError("audio.tts", err)
	}
	localOut := expandLocalPath(*outPath)
	if err := writeDeliveryFile(localOut, resp.Msg.GetAudio().GetData()); err != nil {
		return emitError("audio.tts", err)
	}
	data := map[string]any{
		"text":          resp.Msg.GetText(),
		"voice":         resp.Msg.GetVoice(),
		"backend":       backendInfoData(resp.Msg.GetBackend()),
		"delivery_path": localOut,
	}
	out := envelope.OK("audio.tts", data, reportArtifacts(resp.Msg.GetArtifacts()), false, nil)
	return emit(out)
}

func runClientAudioProcessVoice(args []string, serverURL string) int {
	fs := flag.NewFlagSet("audio process-voice", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	inPath := fs.String("in", "", "input path")
	refText := fs.String("ref-text", "", "reference text")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *inPath == "" || *refText == "" {
		return emitTypedError("audio.process-voice", "INVALID_ARGUMENT", "in and ref-text are required", map[string]any{"in": *inPath, "ref_text": *refText})
	}
	filename, data, err := readCLIInputFile(*inPath)
	if err != nil {
		return emitTypedError("audio.process-voice", "INVALID_ARGUMENT", err.Error(), map[string]any{"in": *inPath, "ref_text": *refText})
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.ProcessVoice(context.Background(), connect.NewRequest(&xuezhv1.ProcessVoiceRequest{
		Audio:    data,
		Filename: filename,
		Mime:     mimeForFilename(filename),
		RefText:  *refText,
	}))
	if err != nil {
		return emitError("audio.process-voice", err)
	}
	artifacts, err := processVoiceArtifacts(resp.Msg)
	if err != nil {
		return emitError("audio.process-voice", err)
	}
	payload := map[string]any{
		"attempt_id":      resp.Msg.GetAttemptId(),
		"ref_text":        resp.Msg.GetRefText(),
		"backend":         backendInfoData(resp.Msg.GetBackend()),
		"artifacts_index": artifactIndex(artifacts),
		"assessment":      resp.Msg.GetAssessment().AsMap(),
		"transcript":      resp.Msg.GetTranscript().AsMap(),
	}
	out := envelope.OK("audio.process-voice", payload, artifacts, resp.Msg.GetTruncated(), resp.Msg.GetLimits().AsMap())
	return emit(out)
}

func processVoiceArtifacts(resp *xuezhv1.ProcessVoiceResponse) ([]envelope.Artifact, error) {
	artifacts := reportArtifacts(resp.GetArtifacts())
	feedback := resp.GetFeedbackAudio()
	if len(feedback.GetData()) == 0 {
		return artifacts, nil
	}
	localPath, err := audioDeliveryCachePath(feedback.GetFilename())
	if err != nil {
		return nil, err
	}
	if err := writeDeliveryFile(localPath, feedback.GetData()); err != nil {
		return nil, err
	}
	bytes := len(feedback.GetData())
	artifacts = append(artifacts, envelope.Artifact{Path: localPath, MIME: feedback.GetMime(), Purpose: "feedback_voice_note_delivery", Bytes: &bytes})
	return artifacts, nil
}

func writeDeliveryFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func audioDeliveryCachePath(filename string) (string, error) {
	if strings.TrimSpace(filename) == "" {
		filename = "audio.ogg"
	}
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	return filepath.Join(base, "xuezh", "audio", filepath.Base(filename)), nil
}

func outputFormatForPath(path string) string {
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	if format == "wav" || format == "mp3" || format == "ogg" {
		return format
	}
	return "ogg"
}

func mimeForFilename(filename string) string {
	mimeType := mime.TypeByExtension(filepath.Ext(filename))
	if mimeType == "" {
		return "application/octet-stream"
	}
	if i := strings.Index(mimeType, ";"); i != -1 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	return mimeType
}

func backendInfoData(backend *xuezhv1.BackendInfo) map[string]any {
	return map[string]any{"id": backend.GetId(), "features": backend.GetFeatures()}
}

func artifactIndex(artifacts []envelope.Artifact) map[string]any {
	index := map[string]any{}
	for _, artifact := range artifacts {
		index[artifact.Purpose] = artifact.Path
	}
	return index
}
