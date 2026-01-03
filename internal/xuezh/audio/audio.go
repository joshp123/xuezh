package audio

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid/v2"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/config"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

var supportedFormats = map[string]struct{}{"wav": {}, "ogg": {}, "mp3": {}}

var voiceAliases = map[string]string{
	"XiaoxiaoNeural": "zh-CN-XiaoxiaoNeural",
}

var attemptEntropy = ulid.Monotonic(rand.Reader, 0)

type AudioResult struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
}

type SttResult struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
	Truncated bool
	Limits    map[string]any
}

type ProcessVoiceResult struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
	Truncated bool
	Limits    map[string]any
}

func mimeForFormat(format string) (string, error) {
	switch strings.ToLower(format) {
	case "wav":
		return "audio/wav", nil
	case "ogg":
		return "audio/ogg", nil
	case "mp3":
		return "audio/mpeg", nil
	default:
		return "", fmt.Errorf("Unsupported audio format: %s", format)
	}
}

func buildConvertCommand(inPath, outPath, format string) ([]string, error) {
	format = strings.ToLower(format)
	if _, ok := supportedFormats[format]; !ok {
		return nil, fmt.Errorf("Unsupported audio format: %s", format)
	}
	cmd := []string{"ffmpeg", "-y", "-hide_banner", "-loglevel", "error", "-i", inPath}
	switch format {
	case "wav":
		cmd = append(cmd, "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le")
	case "ogg":
		cmd = append(cmd, "-ac", "1", "-ar", "48000", "-c:a", "libopus", "-b:a", "24k")
	case "mp3":
		cmd = append(cmd, "-ac", "1", "-ar", "44100", "-c:a", "libmp3lame", "-b:a", "64k")
	}
	cmd = append(cmd, outPath)
	return cmd, nil
}

func buildTTSCommand(text, voice, outPath string) []string {
	return []string{"edge-tts", "--text", text, "--voice", voice, "--write-media", outPath}
}

func buildSTTCommand(inPath, outDir string) []string {
	return []string{"whisper", inPath, "--model", "tiny", "--output_format", "json", "--output_dir", outDir, "--language", "zh", "--task", "transcribe"}
}

func artifactFor(path string, format string, purpose string) (envelope.Artifact, error) {
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return envelope.Artifact{}, err
	}
	rel, err := relativeTo(workspace, path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	mime, err := mimeForFormat(format)
	if err != nil {
		return envelope.Artifact{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	return envelope.Artifact{Path: rel, MIME: mime, Purpose: purpose, Bytes: intPtr(int(info.Size()))}, nil
}

func artifactPath(prefix, ext string, now time.Time) (string, error) {
	root, err := paths.EnsureWorkspace()
	if err != nil {
		return "", err
	}
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.%s", prefix, now.UTC().Format("20060102T150405Z"), ext)
	return filepath.Join(dayPath, filename), nil
}

func ConvertAudio(inPath, outPath, format, backend, purpose string) (AudioResult, error) {
	if backend != "ffmpeg" {
		return AudioResult{}, fmt.Errorf("Unsupported backend: %s", backend)
	}
	inputPath := expandHome(inPath)
	if _, err := os.Stat(inputPath); err != nil {
		return AudioResult{}, fmt.Errorf("Input file not found: %s", inputPath)
	}
	resolvedOut, err := paths.ResolveInWorkspace(outPath)
	if err != nil {
		return AudioResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOut), 0o755); err != nil {
		return AudioResult{}, err
	}
	if _, err := process.EnsureTool("ffmpeg"); err != nil {
		return AudioResult{}, err
	}
	cmd, err := buildConvertCommand(inputPath, resolvedOut, format)
	if err != nil {
		return AudioResult{}, err
	}
	if _, err := process.RunChecked(cmd); err != nil {
		return AudioResult{}, err
	}
	artifact, err := artifactFor(resolvedOut, format, purpose)
	if err != nil {
		return AudioResult{}, err
	}
	data := map[string]any{"in": inputPath, "out": artifact.Path, "format": format, "backend": map[string]any{"id": backend, "features": []string{"convert"}}}
	return AudioResult{Data: data, Artifacts: []envelope.Artifact{artifact}}, nil
}

func TTSAudio(text, voice, outPath, backend, purpose string) (AudioResult, error) {
	if backend != "edge-tts" {
		return AudioResult{}, fmt.Errorf("Unsupported backend: %s", backend)
	}
	resolvedVoice := voiceAliases[voice]
	if resolvedVoice == "" {
		resolvedVoice = voice
	}
	resolvedOut, err := paths.ResolveInWorkspace(outPath)
	if err != nil {
		return AudioResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(resolvedOut), 0o755); err != nil {
		return AudioResult{}, err
	}
	if _, err := process.EnsureTool("edge-tts"); err != nil {
		return AudioResult{}, err
	}
	if _, err := process.EnsureTool("ffmpeg"); err != nil {
		return AudioResult{}, err
	}
	tempPath := filepath.Join(filepath.Dir(resolvedOut), ".tts-"+uuid.New().String()+".mp3")
	cmd := buildTTSCommand(text, resolvedVoice, tempPath)
	if _, err := process.RunChecked(cmd); err != nil {
		return AudioResult{}, err
	}
	defer func() {
		_ = os.Remove(tempPath)
	}()
	fmtOut := strings.TrimPrefix(strings.ToLower(filepath.Ext(resolvedOut)), ".")
	if fmtOut == "" {
		fmtOut = "ogg"
	}
	if _, ok := supportedFormats[fmtOut]; !ok {
		fmtOut = "ogg"
	}
	convertCmd, err := buildConvertCommand(tempPath, resolvedOut, fmtOut)
	if err != nil {
		return AudioResult{}, err
	}
	if _, err := process.RunChecked(convertCmd); err != nil {
		return AudioResult{}, err
	}
	artifact, err := artifactFor(resolvedOut, fmtOut, purpose)
	if err != nil {
		return AudioResult{}, err
	}
	data := map[string]any{"text": text, "voice": resolvedVoice, "out": artifact.Path, "backend": map[string]any{"id": backend, "features": []string{"tts"}}}
	return AudioResult{Data: data, Artifacts: []envelope.Artifact{artifact}}, nil
}

func STTAudio(inPath, backend string) (SttResult, error) {
	if backend != "whisper" {
		return SttResult{}, fmt.Errorf("Unsupported backend: %s", backend)
	}
	inputPath := expandHome(inPath)
	if _, err := os.Stat(inputPath); err != nil {
		return SttResult{}, fmt.Errorf("Input file not found: %s", inputPath)
	}
	if _, err := process.EnsureTool("whisper"); err != nil {
		return SttResult{}, err
	}
	now, err := clock.NowUTC()
	if err != nil {
		return SttResult{}, err
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return SttResult{}, err
	}
	tempDir := filepath.Join(workspace, "artifacts", ".stt-"+uuid.New().String())
	if err := os.MkdirAll(tempDir, 0o755); err != nil {
		return SttResult{}, err
	}
	defer func() {
		_ = os.RemoveAll(tempDir)
	}()
	cmd := buildSTTCommand(inputPath, tempDir)
	if _, err := process.RunChecked(cmd); err != nil {
		return SttResult{}, err
	}
	outputJSON := filepath.Join(tempDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))+".json")
	rawBytes, err := os.ReadFile(outputJSON)
	if err != nil {
		return SttResult{}, err
	}
	var raw map[string]any
	if err := json.Unmarshal(rawBytes, &raw); err != nil {
		return SttResult{}, err
	}
	transcript := extractTranscript(raw)
	transcriptPath, err := artifactPath("stt-"+strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)), "json", now)
	if err != nil {
		return SttResult{}, err
	}
	content, err := jsonio.Dumps(transcript)
	if err != nil {
		return SttResult{}, err
	}
	if err := os.WriteFile(transcriptPath, []byte(content), 0o644); err != nil {
		return SttResult{}, err
	}
	rel, err := relativeTo(workspace, transcriptPath)
	if err != nil {
		return SttResult{}, err
	}
	stat, err := os.Stat(transcriptPath)
	if err != nil {
		return SttResult{}, err
	}
	artifact := envelope.Artifact{Path: rel, MIME: "application/json", Purpose: "transcript", Bytes: intPtr(int(stat.Size()))}
	data := map[string]any{"in": inputPath, "backend": map[string]any{"id": backend, "features": []string{"stt"}}, "transcript": transcript}
	return SttResult{Data: data, Artifacts: []envelope.Artifact{artifact}, Truncated: false, Limits: map[string]any{}}, nil
}

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
	_ = storePronunciationAttempt(backend, artifacts, summary)
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
	return ProcessVoiceResult{Data: data, Artifacts: artifacts, Truncated: inlineTruncated, Limits: limits}, nil
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

func extractTranscript(raw map[string]any) map[string]any {
	text, _ := raw["text"].(string)
	segments := []map[string]any{}
	if rawSegments, ok := raw["segments"].([]any); ok {
		for _, segment := range rawSegments {
			segMap, ok := segment.(map[string]any)
			if !ok {
				continue
			}
			segText := strings.TrimSpace(fmt.Sprintf("%v", segMap["text"]))
			segments = append(segments, map[string]any{
				"start": segMap["start"],
				"end":   segMap["end"],
				"text":  segText,
			})
		}
	}
	transcript := map[string]any{"text": strings.TrimSpace(text), "segments": segments}
	if lang, ok := raw["language"].(string); ok && lang != "" {
		transcript["language"] = lang
	}
	return transcript
}

func inlinePronunciationPayload(assessment, transcript, artifactsIndex map[string]any) (map[string]any, map[string]any, bool) {
	maxBytes := inlineDetailMaxBytes()
	assessmentInline, transcriptInline := dedupeWordDetail(assessment, transcript)
	detailBytes := payloadBytes(assessmentInline, transcriptInline)
	if detailBytes <= maxBytes {
		return assessmentInline, transcriptInline, false
	}
	assessmentSummary := summarizeDetail(assessmentInline)
	transcriptSummary := summarizeDetail(transcriptInline)
	summaryBytes := payloadBytes(assessmentSummary, transcriptSummary)
	if summaryBytes <= maxBytes {
		return assessmentSummary, transcriptSummary, true
	}
	previewLen := 2000
	if text, ok := transcriptInline["text"].(string); ok {
		if len(text) < previewLen {
			previewLen = len(text)
		}
	} else {
		previewLen = 0
	}
	assessmentMin := minimalAssessment(assessmentInline, artifactsIndex)
	transcriptMin := minimalTranscript(transcriptInline, artifactsIndex, previewLen)
	minimalBytes := payloadBytes(assessmentMin, transcriptMin)
	if minimalBytes <= maxBytes {
		return assessmentMin, transcriptMin, true
	}
	transcriptMin = minimalTranscript(transcriptInline, artifactsIndex, 0)
	return assessmentMin, transcriptMin, true
}

func inlineDetailMaxBytes() int {
	if value, ok, _ := config.GetValue("audio", "inline_max_bytes"); ok {
		switch v := value.(type) {
		case int:
			if v > 0 {
				return v
			}
		case int64:
			if v > 0 {
				return int(v)
			}
		case float64:
			if v > 0 {
				return int(v)
			}
		}
	}
	if envValue := strings.TrimSpace(os.Getenv("XUEZH_AUDIO_INLINE_MAX_BYTES")); envValue != "" {
		if parsed, err := strconv.Atoi(envValue); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 200000
}

func payloadBytes(assessment, transcript map[string]any) int {
	payload := map[string]any{"assessment": assessment, "transcript": transcript}
	encoded, err := jsonio.Dumps(payload)
	if err != nil {
		return 0
	}
	return len([]byte(encoded))
}

func dedupeWordDetail(assessment, transcript map[string]any) (map[string]any, map[string]any) {
	if _, ok := assessment["words"]; ok {
		if _, ok := transcript["words"]; ok {
			trimmed := map[string]any{}
			for key, value := range transcript {
				if key == "words" {
					continue
				}
				trimmed[key] = value
			}
			return assessment, trimmed
		}
	}
	return assessment, transcript
}

func summarizeDetail(payload map[string]any) map[string]any {
	summary := map[string]any{}
	for key, value := range payload {
		if key == "words" || key == "segments" {
			continue
		}
		summary[key] = value
	}
	return summary
}

func minimalAssessment(assessment, artifactsIndex map[string]any) map[string]any {
	minimal := map[string]any{}
	if overall, ok := assessment["overall"].(map[string]any); ok && len(overall) > 0 {
		minimal["overall"] = overall
	}
	if value, ok := assessment["exact_match"]; ok {
		minimal["exact_match"] = value
	}
	if value, ok := assessment["note"]; ok {
		minimal["note"] = value
	}
	if spill, ok := artifactsIndex["assessment"]; ok {
		minimal["spill_artifact"] = spill
	}
	return minimal
}

func minimalTranscript(transcript, artifactsIndex map[string]any, previewLen int) map[string]any {
	minimal := map[string]any{}
	if text, ok := transcript["text"].(string); ok && previewLen > 0 {
		if previewLen > len(text) {
			previewLen = len(text)
		}
		minimal["text_preview"] = text[:previewLen]
		minimal["text_truncated"] = len(text) > previewLen
	}
	if spill, ok := artifactsIndex["transcript"]; ok {
		minimal["spill_artifact"] = spill
	}
	return minimal
}

func storePronunciationAttempt(backend string, artifacts []envelope.Artifact, summary map[string]any) error {
	dbPath, err := db.InitDB()
	if err != nil {
		return err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer conn.Close()
	now, err := clock.NowUTC()
	if err != nil {
		return err
	}
	attemptID := ulid.MustNew(ulid.Timestamp(time.Now()), attemptEntropy).String()
	artifactsJSON, err := jsonio.Marshal(artifacts)
	if err != nil {
		return err
	}
	summaryJSON, err := jsonio.Marshal(summary)
	if err != nil {
		return err
	}
	_, err = conn.Exec(
		`INSERT INTO pronunciation_attempts (id, item_id, ts, backend_id, artifacts_json, summary_json)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		attemptID,
		nil,
		clock.FormatISO(now),
		backend,
		artifactsJSON,
		summaryJSON,
	)
	return err
}

func writeJSONArtifact(payload map[string]any, purpose, prefix string) (envelope.Artifact, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return envelope.Artifact{}, err
	}
	path, err := artifactPath(prefix, "json", now)
	if err != nil {
		return envelope.Artifact{}, err
	}
	content, err := jsonio.Dumps(payload)
	if err != nil {
		return envelope.Artifact{}, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return envelope.Artifact{}, err
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return envelope.Artifact{}, err
	}
	rel, err := relativeTo(workspace, path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	return envelope.Artifact{Path: rel, MIME: "application/json", Purpose: purpose, Bytes: intPtr(int(stat.Size()))}, nil
}

func mustArtifactPath(prefix, ext string) string {
	now, err := clock.NowUTC()
	if err != nil {
		now = time.Now().UTC()
	}
	root, _ := paths.EnsureWorkspace()
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	_ = os.MkdirAll(dayPath, 0o755)
	filename := fmt.Sprintf("%s-%s.%s", prefix, now.Format("20060102T150405Z"), ext)
	return filepath.Join(dayPath, filename)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			if strings.HasPrefix(path, "~/") {
				return filepath.Join(home, path[2:])
			}
		}
	}
	return path
}

func intPtr(value int) *int {
	return &value
}

func relativeTo(base, target string) (string, error) {
	baseClean := filepath.Clean(base)
	targetClean := filepath.Clean(target)
	if targetClean != baseClean && !strings.HasPrefix(targetClean, baseClean+string(filepath.Separator)) {
		return "", fmt.Errorf("'%s' is not in the subpath of '%s'", target, base)
	}
	return filepath.Rel(base, target)
}
