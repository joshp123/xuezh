package audio

import "github.com/joshp123/xuezh/internal/xuezh/envelope"

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
	Summary   map[string]any
}
