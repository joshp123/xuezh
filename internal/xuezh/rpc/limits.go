package rpc

import (
	"fmt"

	"connectrpc.com/connect"
)

const (
	processVoiceAudioMaxBytes = 25 * 1024 * 1024
	inlineAudioMaxBytes       = 5 * 1024 * 1024
	contentBytesMax           = 1024 * 1024
	ttsTextMaxRunes           = 1000
)

func limitError(code connect.Code, field string, got, max int) error {
	return connect.NewError(code, fmt.Errorf("%s exceeds limit: got %d bytes, max %d bytes", field, got, max))
}
