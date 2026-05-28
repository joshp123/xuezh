package audio

import (
	"fmt"
	"strings"
)

var supportedFormats = map[string]struct{}{"wav": {}, "ogg": {}, "mp3": {}}

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
