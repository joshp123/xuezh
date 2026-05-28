package service

import (
	"crypto/rand"
	"database/sql"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/oklog/ulid/v2"

	"github.com/joshp123/xuezh/internal/xuezh/audio"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/config"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

var pronunciationAttemptEntropy = ulid.Monotonic(rand.Reader, 0)

func (App) TTS(text, voice, outPath, backend, purpose string) (audio.AudioResult, error) {
	return audio.TTSAudio(text, voice, outPath, backend, purpose)
}

func (app App) SynthesizeSpeech(text, voice, outputFormat string) (audio.AudioResult, []byte, error) {
	backend := configuredAudioBackend("tts_backend", "edge-tts")
	format := audioFormat(outputFormat)
	result, err := app.TTS(text, voice, filepath.Join("artifacts", "tts-"+uuid.New().String()+"."+format), backend, "tts_audio")
	if err != nil {
		return audio.AudioResult{}, nil, err
	}
	data, err := readFirstArtifact(result.Artifacts)
	if err != nil {
		return audio.AudioResult{}, nil, err
	}
	return result, data, nil
}

func (app App) ProcessVoice(inPath, refText, backend string, now time.Time) (audio.ProcessVoiceResult, string, error) {
	result, err := audio.ProcessVoice(inPath, refText, backend)
	if err != nil {
		return audio.ProcessVoiceResult{}, "", err
	}
	return app.finishProcessVoice(backend, result, now)
}

func (app App) ProcessVoiceBytes(data []byte, filename, refText string, now time.Time) (audio.ProcessVoiceResult, string, []byte, error) {
	uploadPath, uploadArtifact, err := writeVoiceUpload(filename, data, now)
	if err != nil {
		return audio.ProcessVoiceResult{}, "", nil, err
	}
	backend := configuredAudioBackend("process_voice_backend", "azure.speech")
	result, err := audio.ProcessVoice(uploadPath, refText, backend)
	if err != nil {
		return audio.ProcessVoiceResult{}, "", nil, err
	}
	result.Artifacts = append([]envelope.Artifact{uploadArtifact}, result.Artifacts...)
	addArtifactIndex(result.Data, "voice_upload", uploadArtifact.Path)
	addArtifactIndex(result.Summary, "voice_upload", uploadArtifact.Path)
	result, attemptID, err := app.finishProcessVoice(backend, result, now)
	if err != nil {
		return audio.ProcessVoiceResult{}, "", nil, err
	}
	feedback, err := readArtifactByPurpose(result.Artifacts, "feedback_voice_note")
	if err != nil {
		return audio.ProcessVoiceResult{}, "", nil, err
	}
	return result, attemptID, feedback, nil
}

func (app App) finishProcessVoice(backend string, result audio.ProcessVoiceResult, now time.Time) (audio.ProcessVoiceResult, string, error) {
	attemptID, err := app.RecordPronunciationAttempt(backend, result.Artifacts, result.Summary, now)
	if err != nil {
		return audio.ProcessVoiceResult{}, "", err
	}
	result.Data["attempt_id"] = attemptID
	return result, attemptID, nil
}

func configuredAudioBackend(configKey, defaultValue string) string {
	if value, ok, _ := config.GetString("audio", configKey); ok {
		return value
	}
	if value, ok, _ := config.GetString("audio", "backend_global"); ok {
		return value
	}
	return defaultValue
}

func audioFormat(value string) string {
	format := strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), ".")
	if format == "wav" || format == "mp3" || format == "ogg" {
		return format
	}
	return "ogg"
}

func readFirstArtifact(artifacts []envelope.Artifact) ([]byte, error) {
	if len(artifacts) == 0 {
		return nil, os.ErrNotExist
	}
	return readArtifact(artifacts[0])
}

func readArtifactByPurpose(artifacts []envelope.Artifact, purpose string) ([]byte, error) {
	for _, artifact := range artifacts {
		if artifact.Purpose == purpose {
			return readArtifact(artifact)
		}
	}
	return nil, os.ErrNotExist
}

func readArtifact(artifact envelope.Artifact) ([]byte, error) {
	path, err := paths.ResolveInWorkspace(artifact.Path)
	if err != nil {
		return nil, err
	}
	return os.ReadFile(path)
}

func writeVoiceUpload(filename string, data []byte, now time.Time) (string, envelope.Artifact, error) {
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = ".ogg"
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return "", envelope.Artifact{}, err
	}
	dayPath := filepath.Join(workspace, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		return "", envelope.Artifact{}, err
	}
	path := filepath.Join(dayPath, "voice-upload-"+uuid.New().String()+ext)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", envelope.Artifact{}, err
	}
	rel, err := filepath.Rel(workspace, path)
	if err != nil {
		return "", envelope.Artifact{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return "", envelope.Artifact{}, err
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if i := strings.Index(mimeType, ";"); i != -1 {
		mimeType = strings.TrimSpace(mimeType[:i])
	}
	bytes := int(stat.Size())
	return path, envelope.Artifact{Path: rel, MIME: mimeType, Purpose: "voice_upload", Bytes: &bytes}, nil
}

func addArtifactIndex(data map[string]any, purpose string, path string) {
	index, ok := data["artifacts_index"].(map[string]any)
	if !ok {
		index = map[string]any{}
		data["artifacts_index"] = index
	}
	index[purpose] = path
}

func (App) RecordPronunciationAttempt(backend string, artifacts []envelope.Artifact, summary map[string]any, now time.Time) (string, error) {
	dbPath, err := db.InitDB()
	if err != nil {
		return "", err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	attemptID := ulid.MustNew(ulid.Timestamp(now), pronunciationAttemptEntropy).String()
	artifactsJSON, err := jsonio.Marshal(artifacts)
	if err != nil {
		return "", err
	}
	summaryJSON, err := jsonio.Marshal(summary)
	if err != nil {
		return "", err
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
	if err != nil {
		return "", err
	}
	return attemptID, nil
}
