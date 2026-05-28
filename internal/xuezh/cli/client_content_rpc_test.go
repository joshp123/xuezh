package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
)

func TestClientBackedContentCommandsUseRPCAndLocalDeliveryCache(t *testing.T) {
	cacheHome := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cacheHome)
	stub := clientRPCStub{
		putContentRequests: make(chan *xuezhv1.PutContentRequest, 1),
		getContentRequests: make(chan *xuezhv1.GetContentRequest, 1),
	}
	server := newClientRPCServer(t, stub)
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	inPath := filepath.Join(t.TempDir(), "tea.txt")
	if err := os.WriteFile(inPath, []byte("茶很好喝"), 0o644); err != nil {
		t.Fatal(err)
	}
	put := runClientCommandForTest(t, []string{"content", "cache", "put", "--type", "story", "--key", "tea", "--in", inPath, "--json"})
	putReq := <-stub.putContentRequests
	if putReq.GetType() != "story" || putReq.GetKey() != "tea" || putReq.GetFilename() != "tea.txt" || string(putReq.GetContent()) != "茶很好喝" {
		t.Fatalf("put request = %+v", putReq)
	}
	if put.Command != "content.cache.put" || put.Data["content_id"] != "ct_remote" {
		t.Fatalf("put envelope = %#v", put)
	}

	got := runClientCommandForTest(t, []string{"content", "cache", "get", "--type", "story", "--key", "tea", "--json"})
	getReq := <-stub.getContentRequests
	if getReq.GetType() != "story" || getReq.GetKey() != "tea" {
		t.Fatalf("get request = %+v", getReq)
	}
	if got.Command != "content.cache.get" || got.Data["content_id"] != "ct_remote" || len(got.Artifacts) != 1 {
		t.Fatalf("get envelope = %#v", got)
	}
	localPath, ok := got.Artifacts[0]["path"].(string)
	if !ok || !filepath.IsAbs(localPath) {
		t.Fatalf("local artifact path = %#v", got.Artifacts)
	}
	data, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "茶很好喝" || filepath.Dir(localPath) != filepath.Join(cacheHome, "xuezh", "cache", "content", "story") {
		t.Fatalf("unexpected local delivery file %s: %q", localPath, data)
	}
}

func (s clientRPCStub) PutContent(_ context.Context, req *connect.Request[xuezhv1.PutContentRequest]) (*connect.Response[xuezhv1.ContentRecord], error) {
	if s.putContentRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("put content request channel missing"))
	}
	s.putContentRequests <- req.Msg
	return connect.NewResponse(remoteContentRecord()), nil
}

func (s clientRPCStub) GetContent(_ context.Context, req *connect.Request[xuezhv1.GetContentRequest]) (*connect.Response[xuezhv1.GetContentResponse], error) {
	if s.getContentRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("get content request channel missing"))
	}
	s.getContentRequests <- req.Msg
	return connect.NewResponse(&xuezhv1.GetContentResponse{
		Record:  remoteContentRecord(),
		Content: []byte("茶很好喝"),
	}), nil
}

func remoteContentRecord() *xuezhv1.ContentRecord {
	bytes := int64(12)
	return &xuezhv1.ContentRecord{
		Type:      "story",
		Key:       "tea",
		ContentId: "ct_remote",
		Artifacts: []*xuezhv1.ServerArtifact{{
			Path:    "cache/content/story/tea.txt",
			Mime:    "text/plain",
			Purpose: "cached_content",
			Bytes:   &bytes,
		}},
	}
}
