package cli

import (
	"context"
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func runClientContentCachePut(args []string, serverURL string) int {
	fs := flag.NewFlagSet("content cache put", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	contentType := fs.String("type", "", "story|dialogue|exercise")
	key := fs.String("key", "", "key")
	inPath := fs.String("in", "", "input path")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	filename, data, err := readCLIInputFile(*inPath)
	if err != nil {
		return emitTypedError("content.cache.put", "INVALID_ARGUMENT", err.Error(), map[string]any{"type": *contentType, "key": *key, "in": *inPath})
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.PutContent(context.Background(), connect.NewRequest(&xuezhv1.PutContentRequest{
		Type:     *contentType,
		Key:      *key,
		Filename: filename,
		Content:  data,
	}))
	if err != nil {
		return emitError("content.cache.put", err)
	}
	out := envelope.OK("content.cache.put", contentRecordProtoData(resp.Msg), reportArtifacts(resp.Msg.GetArtifacts()), false, nil)
	return emit(out)
}

func runClientContentCacheGet(args []string, serverURL string) int {
	fs := flag.NewFlagSet("content cache get", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	contentType := fs.String("type", "", "story|dialogue|exercise")
	key := fs.String("key", "", "key")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.GetContent(context.Background(), connect.NewRequest(&xuezhv1.GetContentRequest{Type: *contentType, Key: *key}))
	if err != nil {
		return emitError("content.cache.get", err)
	}
	artifacts, err := materializeContentDelivery(resp.Msg.GetRecord(), resp.Msg.GetContent())
	if err != nil {
		return emitError("content.cache.get", err)
	}
	out := envelope.OK("content.cache.get", contentRecordProtoData(resp.Msg.GetRecord()), artifacts, false, nil)
	return emit(out)
}

func contentRecordProtoData(record *xuezhv1.ContentRecord) map[string]any {
	return map[string]any{
		"type":       record.GetType(),
		"key":        record.GetKey(),
		"content_id": record.GetContentId(),
	}
}

func materializeContentDelivery(record *xuezhv1.ContentRecord, data []byte) ([]envelope.Artifact, error) {
	artifacts := record.GetArtifacts()
	if len(artifacts) == 0 {
		return []envelope.Artifact{}, nil
	}
	result := reportArtifacts(artifacts)
	if len(data) == 0 {
		return result, nil
	}
	localPath, err := clientCachePath(artifacts[0].GetPath())
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return nil, err
	}
	if err := os.WriteFile(localPath, data, 0o644); err != nil {
		return nil, err
	}
	result[0].Path = localPath
	return result, nil
}

func clientCachePath(remotePath string) (string, error) {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".cache")
	}
	rel := filepath.Clean(remotePath)
	if filepath.IsAbs(rel) || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		rel = filepath.Join("content", filepath.Base(remotePath))
	}
	return filepath.Join(base, "xuezh", rel), nil
}
