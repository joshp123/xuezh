package errors

import "testing"

func TestClientServerErrorTypesAreKnown(t *testing.T) {
	for _, errorType := range []string{"CONFIG_CONFLICT", "UNSUPPORTED_CLIENT_COMMAND"} {
		if err := AssertKnown(errorType); err != nil {
			t.Fatalf("%s must be a known typed error: %v", errorType, err)
		}
	}
}
