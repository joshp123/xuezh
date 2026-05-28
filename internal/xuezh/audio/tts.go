package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
	"github.com/joshp123/xuezh/internal/xuezh/process"
)

var voiceAliases = map[string]string{
	"XiaoxiaoNeural": "zh-CN-XiaoxiaoNeural",
}

func buildTTSCommand(text, voice, rate, outPath string) []string {
	cmd := []string{"edge-tts", "--text", text, "--voice", voice}
	if strings.TrimSpace(rate) != "" {
		cmd = append(cmd, "--rate="+strings.TrimSpace(rate))
	}
	return append(cmd, "--write-media", outPath)
}

func TTSAudio(text, voice, outPath, backend, purpose string) (AudioResult, error) {
	return TTSAudioWithRate(text, voice, "", outPath, backend, purpose)
}

func TTSAudioWithRate(text, voice, rate, outPath, backend, purpose string) (AudioResult, error) {
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
	cmd := buildTTSCommand(text, resolvedVoice, rate, tempPath)
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
	data := map[string]any{"text": text, "voice": resolvedVoice, "rate": strings.TrimSpace(rate), "out": artifact.Path, "backend": map[string]any{"id": backend, "features": []string{"tts"}}}
	return AudioResult{Data: data, Artifacts: []envelope.Artifact{artifact}}, nil
}
