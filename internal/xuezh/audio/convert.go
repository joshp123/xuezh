package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

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
