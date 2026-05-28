package cli

import (
	"time"
)

func protoTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}
