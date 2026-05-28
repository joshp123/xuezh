package audio

import (
	"github.com/joshp123/xuezh/internal/xuezh/config"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
)

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
