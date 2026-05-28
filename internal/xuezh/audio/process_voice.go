package audio

import (
	"fmt"
	"strings"

	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

func ProcessVoice(inPath, refText, backend string) (ProcessVoiceResult, error) {
	if backend != "local" && backend != "azure.speech" {
		return ProcessVoiceResult{}, fmt.Errorf("Unsupported backend: %s", backend)
	}
	normalized, err := ConvertAudio(inPath, mustArtifactPath("normalized-input", "wav"), "wav", "ffmpeg", "normalized_input")
	if err != nil {
		return ProcessVoiceResult{}, err
	}
	normalizedPath, err := paths.ResolveInWorkspace(normalized.Artifacts[0].Path)
	if err != nil {
		return ProcessVoiceResult{}, err
	}
	var assessment map[string]any
	var transcript map[string]any
	var transcriptArtifacts []envelope.Artifact
	if backend == "local" {
		sttResult, err := STTAudio(normalizedPath, "whisper")
		if err != nil {
			return ProcessVoiceResult{}, err
		}
		transcriptMap, ok := sttResult.Data["transcript"].(map[string]any)
		if !ok {
			transcriptMap = map[string]any{}
		}
		transcript = transcriptMap
		transcriptText, _ := transcript["text"].(string)
		assessment = assessFromTranscript(refText, transcriptText)
		transcriptArtifacts = sttResult.Artifacts
	} else {
		azureAssessment, azureTranscript, rawJSON, err := azurePronunciationAssess(refText, normalizedPath)
		if err != nil {
			return ProcessVoiceResult{}, err
		}
		assessment = azureAssessment
		transcript = azureTranscript
		transcriptArtifact, err := writeJSONArtifact(azureTranscript, "transcript", "transcript")
		if err != nil {
			return ProcessVoiceResult{}, err
		}
		rawArtifact, err := writeJSONArtifact(rawJSON, "azure_response", "azure-response")
		if err != nil {
			return ProcessVoiceResult{}, err
		}
		transcriptArtifacts = []envelope.Artifact{transcriptArtifact, rawArtifact}
	}
	assessmentArtifact, err := writeJSONArtifact(assessment, "assessment", "assessment")
	if err != nil {
		return ProcessVoiceResult{}, err
	}
	feedback, err := TTSAudio(refText, "XiaoxiaoNeural", mustArtifactPath("feedback-voice", "ogg"), "edge-tts", "feedback_voice_note")
	if err != nil {
		return ProcessVoiceResult{}, err
	}
	artifacts := []envelope.Artifact{}
	artifacts = append(artifacts, normalized.Artifacts...)
	artifacts = append(artifacts, transcriptArtifacts...)
	artifacts = append(artifacts, assessmentArtifact)
	artifacts = append(artifacts, feedback.Artifacts...)
	artifactsIndex := map[string]any{}
	for _, artifact := range artifacts {
		artifactsIndex[artifact.Purpose] = artifact.Path
	}
	assessmentInline, transcriptInline, inlineTruncated := inlinePronunciationPayload(assessment, transcript, artifactsIndex)
	summary := map[string]any{"assessment": assessment, "artifacts_index": artifactsIndex}
	features := []string{"assessment", "tts", "stt", "convert"}
	if backend == "azure.speech" {
		features = []string{"assessment", "tts", "convert", "azure.speech"}
	}
	data := map[string]any{
		"ref_text":        refText,
		"backend":         map[string]any{"id": backend, "features": features},
		"artifacts_index": artifactsIndex,
		"assessment":      assessmentInline,
		"transcript":      transcriptInline,
	}
	limits := map[string]any{}
	if inlineTruncated {
		limits = map[string]any{"inline_bytes_max": inlineDetailMaxBytes()}
	}
	return ProcessVoiceResult{Data: data, Artifacts: artifacts, Truncated: inlineTruncated, Limits: limits, Summary: summary}, nil
}

func assessFromTranscript(refText, transcriptText string) map[string]any {
	refNorm := normalizeText(refText)
	transNorm := normalizeText(transcriptText)
	return map[string]any{
		"ref_text":        refText,
		"transcript_text": transcriptText,
		"exact_match":     refNorm == transNorm,
		"note":            "local_v0_placeholder",
	}
}

func normalizeText(text string) string {
	parts := strings.Fields(text)
	return strings.ToLower(strings.Join(parts, " "))
}
