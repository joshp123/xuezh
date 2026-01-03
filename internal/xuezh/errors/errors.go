package errors

import "fmt"

var known = map[string]struct{}{
	"BACKEND_FAILED":   {},
	"AUTH_FAILED":      {},
	"INVALID_ARGUMENT": {},
	"NOT_IMPLEMENTED":  {},
	"NOT_FOUND":        {},
	"QUOTA_EXCEEDED":   {},
	"TOOL_MISSING":     {},
}

func AssertKnown(errorType string) error {
	if _, ok := known[errorType]; !ok {
		return fmt.Errorf("unknown error type: %q", errorType)
	}
	return nil
}
