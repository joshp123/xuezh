package rpc

import (
	"context"
	"fmt"
	"path/filepath"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func (h *Handler) SynthesizeSpeech(ctx context.Context, req *connect.Request[xuezhv1.SynthesizeSpeechRequest]) (*connect.Response[xuezhv1.SynthesizeSpeechResponse], error) {
	if utf8.RuneCountInString(req.Msg.GetText()) > ttsTextMaxRunes {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("text exceeds limit: got %d code points, max %d", utf8.RuneCountInString(req.Msg.GetText()), ttsTextMaxRunes))
	}
	result, data, err := h.app.SynthesizeSpeech(req.Msg.GetText(), req.Msg.GetVoice(), req.Msg.GetOutputFormat())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(data) > inlineAudioMaxBytes {
		return nil, limitError(connect.CodeResourceExhausted, "audio", len(data), inlineAudioMaxBytes)
	}
	return connect.NewResponse(&xuezhv1.SynthesizeSpeechResponse{
		Text:      stringField(result.Data, "text"),
		Voice:     stringField(result.Data, "voice"),
		Backend:   backendInfoMessage(result.Data),
		Artifacts: artifactMessages(result.Artifacts),
		Audio:     inlineAudioFile(data, result.Artifacts, req.Msg.GetOutputFormat()),
	}), nil
}

func (h *Handler) ProcessVoice(ctx context.Context, req *connect.Request[xuezhv1.ProcessVoiceRequest]) (*connect.Response[xuezhv1.ProcessVoiceResponse], error) {
	if len(req.Msg.GetAudio()) > processVoiceAudioMaxBytes {
		return nil, limitError(connect.CodeInvalidArgument, "audio", len(req.Msg.GetAudio()), processVoiceAudioMaxBytes)
	}
	result, attemptID, feedback, err := h.app.ProcessVoiceBytes(req.Msg.GetAudio(), req.Msg.GetFilename(), req.Msg.GetRefText(), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(feedback) > inlineAudioMaxBytes {
		return nil, limitError(connect.CodeResourceExhausted, "feedback_audio", len(feedback), inlineAudioMaxBytes)
	}
	assessment, _ := result.Data["assessment"].(map[string]any)
	transcript, _ := result.Data["transcript"].(map[string]any)
	assessmentStruct, err := reportStruct(assessment)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	transcriptStruct, err := reportStruct(transcript)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	limitStruct, err := reportStruct(result.Limits)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&xuezhv1.ProcessVoiceResponse{
		AttemptId:     attemptID,
		RefText:       stringField(result.Data, "ref_text"),
		Backend:       backendInfoMessage(result.Data),
		Assessment:    assessmentStruct,
		Transcript:    transcriptStruct,
		Artifacts:     artifactMessages(result.Artifacts),
		FeedbackAudio: inlineAudioFile(feedback, result.Artifacts, "ogg"),
		Truncated:     result.Truncated,
		Limits:        limitStruct,
	}), nil
}

func backendInfoMessage(data map[string]any) *xuezhv1.BackendInfo {
	backend, _ := data["backend"].(map[string]any)
	id, _ := backend["id"].(string)
	features := []string{}
	switch values := backend["features"].(type) {
	case []string:
		features = append(features, values...)
	case []any:
		for _, value := range values {
			if feature, ok := value.(string); ok {
				features = append(features, feature)
			}
		}
	}
	return &xuezhv1.BackendInfo{Id: id, Features: features}
}

func inlineAudioFile(data []byte, artifacts []envelope.Artifact, format string) *xuezhv1.InlineFile {
	if len(data) == 0 {
		return nil
	}
	mime := "audio/ogg"
	filename := "audio.ogg"
	if len(artifacts) > 0 {
		mime = artifacts[len(artifacts)-1].MIME
		filename = filepath.Base(artifacts[len(artifacts)-1].Path)
	}
	if format != "" && filename == "audio.ogg" {
		filename = "audio." + format
	}
	return &xuezhv1.InlineFile{Data: data, Mime: mime, Filename: filename}
}
