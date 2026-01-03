package envelope

import "github.com/joshp123/xuezh/internal/xuezh/errors"

type Artifact struct {
	Path    string `json:"path"`
	MIME    string `json:"mime"`
	Purpose string `json:"purpose"`
	Bytes   *int   `json:"bytes,omitempty"`
}

type OKEnvelope struct {
	OK        bool           `json:"ok"`
	SchemaVer string         `json:"schema_version"`
	Command   string         `json:"command"`
	Data      map[string]any `json:"data"`
	Artifacts []Artifact     `json:"artifacts"`
	Truncated bool           `json:"truncated"`
	Limits    map[string]any `json:"limits"`
}

type ErrorDetail struct {
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

type ErrorEnvelope struct {
	OK        bool        `json:"ok"`
	SchemaVer string      `json:"schema_version"`
	Command   string      `json:"command"`
	Error     ErrorDetail `json:"error"`
}

func OK(command string, data map[string]any, artifacts []Artifact, truncated bool, limits map[string]any) OKEnvelope {
	if data == nil {
		data = map[string]any{}
	}
	if artifacts == nil {
		artifacts = []Artifact{}
	}
	if limits == nil {
		limits = map[string]any{}
	}
	return OKEnvelope{
		OK:        true,
		SchemaVer: "1",
		Command:   command,
		Data:      data,
		Artifacts: artifacts,
		Truncated: truncated,
		Limits:    limits,
	}
}

func Err(command, errorType, message string, details map[string]any) (ErrorEnvelope, error) {
	if details == nil {
		details = map[string]any{}
	}
	if err := errors.AssertKnown(errorType); err != nil {
		return ErrorEnvelope{}, err
	}
	return ErrorEnvelope{
		OK:        false,
		SchemaVer: "1",
		Command:   command,
		Error: ErrorDetail{
			Type:    errorType,
			Message: message,
			Details: details,
		},
	}, nil
}
