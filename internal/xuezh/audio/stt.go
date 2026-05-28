package audio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

func buildSTTCommand(inPath, outDir string) []string {
	return []string{"whisper", inPath, "--model", "tiny", "--output_format", "json", "--output_dir", outDir, "--language", "zh", "--task", "transcribe"}
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
