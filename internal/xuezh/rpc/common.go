package rpc

import (
	"encoding/json"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func stringField(data map[string]any, key string) string {
	value, _ := data[key].(string)
	return value
}

func reportPayloadMessage(data map[string]any, artifacts []envelope.Artifact, truncated bool, limits map[string]any) (*xuezhv1.ReportPayload, error) {
	dataStruct, err := reportStruct(data)
	if err != nil {
		return nil, err
	}
	limitStruct, err := reportStruct(limits)
	if err != nil {
		return nil, err
	}
	return &xuezhv1.ReportPayload{
		Data:      dataStruct,
		Artifacts: artifactMessages(artifacts),
		Truncated: truncated,
		Limits:    limitStruct,
	}, nil
}

func artifactMessages(artifacts []envelope.Artifact) []*xuezhv1.ServerArtifact {
	protoArtifacts := make([]*xuezhv1.ServerArtifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		msg := &xuezhv1.ServerArtifact{Path: artifact.Path, Mime: artifact.MIME, Purpose: artifact.Purpose}
		if artifact.Bytes != nil {
			bytes := int64(*artifact.Bytes)
			msg.Bytes = &bytes
		}
		protoArtifacts = append(protoArtifacts, msg)
	}
	return protoArtifacts
}

func reportStruct(data map[string]any) (*structpb.Struct, error) {
	if data == nil {
		data = map[string]any{}
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return structpb.NewStruct(normalized)
}

func timestampFromOptionalISO(value *string) (*timestamppb.Timestamp, error) {
	if value == nil {
		return nil, nil
	}
	return timestampFromISO(*value)
}

func timestampFromISO(value string) (*timestamppb.Timestamp, error) {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp %q: %w", value, err)
	}
	return timestamppb.New(parsed), nil
}
