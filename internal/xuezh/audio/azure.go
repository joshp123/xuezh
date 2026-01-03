package audio

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/config"
)

type AzureSpeechError struct {
	Kind    string
	Message string
	Details map[string]any
}

func (e AzureSpeechError) Error() string {
	return e.Message
}

func azurePronunciationAssess(refText, wavPath string) (map[string]any, map[string]any, map[string]any, error) {
	key, region, err := azureCredentials()
	if err != nil {
		return nil, nil, nil, err
	}
	payload := map[string]any{
		"ReferenceText": refText,
		"GradingSystem": "HundredMark",
		"Granularity":   "Phoneme",
		"EnableMiscue":  true,
		"Dimension":     "Comprehensive",
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, nil, err
	}
	headerValue := base64.StdEncoding.EncodeToString(payloadJSON)
	body, err := os.ReadFile(wavPath)
	if err != nil {
		return nil, nil, nil, err
	}
	url := fmt.Sprintf(
		"https://%s.stt.speech.microsoft.com/speech/recognition/conversation/cognitiveservices/v1?language=zh-CN&format=detailed",
		region,
	)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, nil, nil, err
	}
	req.Header.Set("Ocp-Apim-Subscription-Key", key)
	req.Header.Set("Pronunciation-Assessment", headerValue)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "audio/wav; codecs=audio/pcm; samplerate=16000")

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, nil, AzureSpeechError{Kind: "backend", Message: "Azure Speech request failed", Details: map[string]any{"error": err.Error()}}
	}
	defer resp.Body.Close()
	respBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		text := strings.TrimSpace(string(respBytes))
		kind := "backend"
		switch resp.StatusCode {
		case http.StatusTooManyRequests:
			kind = "quota"
		case http.StatusUnauthorized, http.StatusForbidden:
			kind = "auth"
		}
		lowered := strings.ToLower(text)
		if kind == "backend" {
			if strings.Contains(lowered, "quota") || strings.Contains(lowered, "limit") || strings.Contains(lowered, "429") {
				kind = "quota"
			}
			if strings.Contains(lowered, "401") || strings.Contains(lowered, "403") || strings.Contains(lowered, "unauthorized") {
				kind = "auth"
			}
		}
		return nil, nil, nil, AzureSpeechError{
			Kind:    kind,
			Message: "Azure Speech request failed",
			Details: map[string]any{"status": resp.StatusCode, "error_details": text},
		}
	}
	var raw map[string]any
	if err := json.Unmarshal(respBytes, &raw); err != nil {
		return nil, nil, nil, AzureSpeechError{Kind: "backend", Message: "Azure Speech response parse failed", Details: map[string]any{"error_details": string(respBytes)}}
	}
	if status, ok := raw["RecognitionStatus"].(string); ok && status != "" && status != "Success" {
		return nil, nil, nil, AzureSpeechError{
			Kind:    "backend",
			Message: fmt.Sprintf("Azure Speech recognition failed (%s)", status),
			Details: map[string]any{"status": status},
		}
	}
	nbest := firstMap(raw["NBest"])
	displayText := firstString(nbest["Display"], nbest["DisplayText"], raw["DisplayText"], raw["Text"])
	words := []map[string]any{}
	if items, ok := nbest["Words"].([]any); ok {
		for _, item := range items {
			wordMap, ok := item.(map[string]any)
			if !ok {
				continue
			}
			pa := map[string]any{}
			if paRaw, ok := wordMap["PronunciationAssessment"].(map[string]any); ok {
				pa = paRaw
			}
			accuracy := pa["AccuracyScore"]
			if accuracy == nil {
				accuracy = wordMap["AccuracyScore"]
			}
			errorType := pa["ErrorType"]
			if errorType == nil {
				errorType = wordMap["ErrorType"]
			}
			phonemes := normalizeAssessmentEntries(wordMap["Phonemes"])
			syllables := normalizeAssessmentEntries(wordMap["Syllables"])
			words = append(words, map[string]any{
				"word":           wordMap["Word"],
				"accuracy_score": accuracy,
				"error_type":     errorType,
				"syllables":      syllables,
				"phonemes":       phonemes,
			})
		}
	}
	overall := map[string]any{
		"accuracy_score":      nil,
		"fluency_score":       nil,
		"completeness_score":  nil,
		"pronunciation_score": nil,
	}
	if paRaw, ok := nbest["PronunciationAssessment"].(map[string]any); ok {
		overall["accuracy_score"] = paRaw["AccuracyScore"]
		overall["fluency_score"] = paRaw["FluencyScore"]
		overall["completeness_score"] = paRaw["CompletenessScore"]
		overall["pronunciation_score"] = paRaw["PronScore"]
		if overall["pronunciation_score"] == nil {
			overall["pronunciation_score"] = paRaw["PronunciationScore"]
		}
	}
	if overall["accuracy_score"] == nil {
		overall["accuracy_score"] = nbest["AccuracyScore"]
	}
	if overall["fluency_score"] == nil {
		overall["fluency_score"] = nbest["FluencyScore"]
	}
	if overall["completeness_score"] == nil {
		overall["completeness_score"] = nbest["CompletenessScore"]
	}
	if overall["pronunciation_score"] == nil {
		overall["pronunciation_score"] = nbest["PronScore"]
		if overall["pronunciation_score"] == nil {
			overall["pronunciation_score"] = nbest["PronunciationScore"]
		}
	}
	transcript := map[string]any{"text": displayText}
	assessment := map[string]any{
		"reference_text":  refText,
		"transcript_text": transcript["text"],
		"overall":         overall,
		"words":           words,
	}
	return assessment, transcript, raw, nil
}

func normalizeAssessmentEntries(value any) any {
	list, ok := value.([]any)
	if !ok {
		return value
	}
	normalized := make([]any, 0, len(list))
	for _, item := range list {
		entry, ok := item.(map[string]any)
		if !ok {
			normalized = append(normalized, item)
			continue
		}
		if _, ok := entry["PronunciationAssessment"]; ok {
			normalized = append(normalized, entry)
			continue
		}
		accuracy, hasAccuracy := entry["AccuracyScore"]
		if !hasAccuracy {
			normalized = append(normalized, entry)
			continue
		}
		next := map[string]any{}
		for key, val := range entry {
			if key == "AccuracyScore" {
				continue
			}
			next[key] = val
		}
		next["PronunciationAssessment"] = map[string]any{"AccuracyScore": accuracy}
		normalized = append(normalized, next)
	}
	return normalized
}

func azureCredentials() (string, string, error) {
	var key, region string
	if section, ok, _ := config.GetValue("azure", "speech"); ok {
		if sectionMap, ok := section.(map[string]any); ok {
			if value, ok := sectionMap["key"].(string); ok && strings.TrimSpace(value) != "" {
				key = strings.TrimSpace(value)
			}
			if value, ok := sectionMap["key_file"].(string); ok && strings.TrimSpace(value) != "" {
				if data, err := os.ReadFile(expandHomePath(value)); err == nil {
					key = strings.TrimSpace(string(data))
				}
			}
			if value, ok := sectionMap["region"].(string); ok && strings.TrimSpace(value) != "" {
				region = strings.TrimSpace(value)
			}
		}
	}
	if key == "" {
		key = strings.TrimSpace(os.Getenv("AZURE_SPEECH_KEY"))
	}
	if region == "" {
		region = strings.TrimSpace(os.Getenv("AZURE_SPEECH_REGION"))
	}
	if key == "" || region == "" {
		missing := []string{}
		if key == "" {
			missing = append(missing, "AZURE_SPEECH_KEY or config.azure.speech.key/key_file")
		}
		if region == "" {
			missing = append(missing, "AZURE_SPEECH_REGION or config.azure.speech.region")
		}
		return "", "", AzureSpeechError{
			Kind:    "auth",
			Message: fmt.Sprintf("Azure Speech credentials missing (%s)", strings.Join(missing, ", ")),
			Details: map[string]any{"missing": missing},
		}
	}
	return key, region, nil
}

func expandHomePath(path string) string {
	if path == "" || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if len(path) > 1 && (path[1] == '/' || path[1] == '\\') {
		return filepath.Join(home, path[2:])
	}
	return path
}

func firstMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if list, ok := value.([]any); ok && len(list) > 0 {
		if m, ok := list[0].(map[string]any); ok {
			return m
		}
	}
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func firstString(values ...any) string {
	for _, value := range values {
		if s, ok := value.(string); ok && s != "" {
			return s
		}
	}
	return ""
}
