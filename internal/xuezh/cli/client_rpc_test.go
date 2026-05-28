package cli

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
)

func TestClientBackedLearnerStateUsesRPCAndEmitsCLIEnvelope(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XUEZH_WORKSPACE_DIR", filepath.Join(t.TempDir(), "workspace-poison"))
	t.Setenv("XUEZH_DB_PATH", filepath.Join(t.TempDir(), "db-poison.sqlite3"))

	server := httptest.NewServer(newClientRPCMux(t, clientRPCStub{}))
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	code, stdout := captureStdout(t, func() int {
		return Run([]string{"learner", "state", "--json"})
	})
	if code != 0 {
		t.Fatalf("learner state exit = %d, stdout=%s", code, stdout)
	}

	var env struct {
		OK      bool           `json:"ok"`
		Command string         `json:"command"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, stdout)
	}
	if !env.OK || env.Command != "learner.state" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
	if env.Data["state_hash"] != "remote-state-hash" || env.Data["generated_at"] != "2026-05-28T09:10:11Z" {
		t.Fatalf("CLI did not translate remote proto learner state: %#v", env.Data)
	}
	cards, ok := env.Data["cards"].([]any)
	if !ok || len(cards) != 1 {
		t.Fatalf("cards = %#v", env.Data["cards"])
	}
	row, ok := cards[0].([]any)
	if !ok || len(row) != 3 || row[0] != "hc:1" || row[1] != "你好" || row[2] != float64(4) {
		t.Fatalf("card row = %#v", cards[0])
	}

	for _, path := range []string{
		filepath.Join(home, ".clawdbot"),
		filepath.Join(home, "Library", "Application Support", "xuezh"),
	} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("client-backed learner state touched workspace path %s: %v", path, err)
		}
	}
}

func TestClientBackedSnapshotUsesRPCReportPayload(t *testing.T) {
	requests := make(chan *xuezhv1.GetSnapshotRequest, 1)
	server := httptest.NewServer(newClientRPCMux(t, clientRPCStub{snapshotRequests: requests}))
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	code, stdout := captureStdout(t, func() int {
		return Run([]string{"snapshot", "--window", "7d", "--due-limit", "3", "--evidence-limit", "4", "--max-bytes", "100", "--json"})
	})
	if code != 0 {
		t.Fatalf("snapshot exit = %d, stdout=%s", code, stdout)
	}
	req := <-requests
	if req.GetWindow() != "7d" || req.GetDueLimit() != 3 || req.GetEvidenceLimit() != 4 || req.GetMaxBytes() != 100 {
		t.Fatalf("snapshot request = %+v", req)
	}

	var env struct {
		OK        bool             `json:"ok"`
		Command   string           `json:"command"`
		Data      map[string]any   `json:"data"`
		Artifacts []map[string]any `json:"artifacts"`
		Truncated bool             `json:"truncated"`
		Limits    map[string]any   `json:"limits"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, stdout)
	}
	if !env.OK || env.Command != "snapshot" {
		t.Fatalf("unexpected envelope: %#v", env)
	}
	if env.Data["window"] != "7d" || env.Data["kind"] != "remote-snapshot" || !env.Truncated {
		t.Fatalf("CLI did not translate remote report payload: %#v", env)
	}
	if len(env.Artifacts) != 1 || env.Artifacts[0]["path"] != "artifacts/snapshot.json" || env.Artifacts[0]["bytes"] != float64(123) {
		t.Fatalf("artifacts = %#v", env.Artifacts)
	}
	if env.Limits["max_bytes"] != float64(100) {
		t.Fatalf("limits = %#v", env.Limits)
	}
}

func TestClientBackedReportCommandsUseRPCReportPayload(t *testing.T) {
	stub := clientRPCStub{
		previewRequests: make(chan *xuezhv1.PreviewSRSRequest, 1),
		hskRequests:     make(chan *xuezhv1.ReportHSKRequest, 1),
		masteryRequests: make(chan *xuezhv1.ReportMasteryRequest, 1),
		dueRequests:     make(chan *xuezhv1.ReportDueRequest, 1),
	}
	server := httptest.NewServer(newClientRPCMux(t, stub))
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	preview := runClientCommandForTest(t, []string{"srs", "preview", "--days", "5", "--json"})
	if req := <-stub.previewRequests; req.GetDays() != 5 {
		t.Fatalf("preview request = %+v", req)
	}
	if preview.Command != "srs.preview" || preview.Data["kind"] != "remote-srs-preview" || preview.Data["days"] != float64(5) {
		t.Fatalf("preview envelope = %#v", preview)
	}

	hsk := runClientCommandForTest(t, []string{"report", "hsk", "--level", "1", "--window", "30d", "--max-items", "9", "--max-bytes", "111", "--include-chars", "--json"})
	if req := <-stub.hskRequests; req.GetLevel() != "1" || req.GetWindow() != "30d" || req.GetMaxItems() != 9 || req.GetMaxBytes() != 111 || !req.GetIncludeChars() {
		t.Fatalf("hsk request = %+v", req)
	}
	if hsk.Command != "report.hsk" || hsk.Data["kind"] != "remote-hsk" || hsk.Data["level"] != "1" {
		t.Fatalf("hsk envelope = %#v", hsk)
	}

	mastery := runClientCommandForTest(t, []string{"report", "mastery", "--item-type", "word", "--window", "90d", "--max-items", "8", "--max-bytes", "222", "--json"})
	if req := <-stub.masteryRequests; req.GetItemType() != "word" || req.GetWindow() != "90d" || req.GetMaxItems() != 8 || req.GetMaxBytes() != 222 {
		t.Fatalf("mastery request = %+v", req)
	}
	if mastery.Command != "report.mastery" || mastery.Data["kind"] != "remote-mastery" || mastery.Data["item_type"] != "word" {
		t.Fatalf("mastery envelope = %#v", mastery)
	}

	due := runClientCommandForTest(t, []string{"report", "due", "--limit", "7", "--max-bytes", "333", "--json"})
	if req := <-stub.dueRequests; req.GetLimit() != 7 || req.GetMaxBytes() != 333 {
		t.Fatalf("due request = %+v", req)
	}
	items, ok := due.Data["items"].([]any)
	if due.Command != "report.due" || due.Data["kind"] != "remote-due" || !ok || len(items) != 1 {
		t.Fatalf("due envelope = %#v", due)
	}
}

func TestClientBackedUnsupportedCommandDoesNotFallThroughLocal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeCLIUserConfig(t, "[client]\nserver_url = \"https://chinese.jjpcodes.com\"\n")

	code, stdout := captureStdout(t, func() int {
		return Run([]string{"db", "init", "--json"})
	})
	if code != 1 {
		t.Fatalf("db init exit = %d, stdout=%s", code, stdout)
	}

	var env struct {
		Error struct {
			Type string `json:"type"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, stdout)
	}
	if env.Error.Type != "UNSUPPORTED_CLIENT_COMMAND" {
		t.Fatalf("error type = %q", env.Error.Type)
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "xuezh")); !os.IsNotExist(err) {
		t.Fatalf("client-backed db init touched local workspace: %v", err)
	}
}

type clientRPCStub struct {
	xuezhv1connect.UnimplementedXuezhServiceHandler
	snapshotRequests     chan *xuezhv1.GetSnapshotRequest
	previewRequests      chan *xuezhv1.PreviewSRSRequest
	hskRequests          chan *xuezhv1.ReportHSKRequest
	masteryRequests      chan *xuezhv1.ReportMasteryRequest
	dueRequests          chan *xuezhv1.ReportDueRequest
	startRequests        chan *xuezhv1.StartReviewRequest
	gradeRequests        chan *xuezhv1.GradeReviewRequest
	buryRequests         chan *xuezhv1.BuryReviewRequest
	logEventRequests     chan *xuezhv1.LogEventRequest
	listEventRequests    chan *xuezhv1.ListEventsRequest
	putContentRequests   chan *xuezhv1.PutContentRequest
	getContentRequests   chan *xuezhv1.GetContentRequest
	ttsRequests          chan *xuezhv1.SynthesizeSpeechRequest
	processVoiceRequests chan *xuezhv1.ProcessVoiceRequest
	doctorRequests       chan *xuezhv1.DoctorRequest
}

type testEnvelope struct {
	OK        bool             `json:"ok"`
	Command   string           `json:"command"`
	Data      map[string]any   `json:"data"`
	Artifacts []map[string]any `json:"artifacts"`
	Truncated bool             `json:"truncated"`
	Limits    map[string]any   `json:"limits"`
}

func runClientCommandForTest(t *testing.T, args []string) testEnvelope {
	t.Helper()
	code, stdout := captureStdout(t, func() int {
		return Run(args)
	})
	if code != 0 {
		t.Fatalf("%v exit = %d, stdout=%s", args, code, stdout)
	}
	var env testEnvelope
	if err := json.Unmarshal([]byte(stdout), &env); err != nil {
		t.Fatalf("invalid JSON envelope: %v\n%s", err, stdout)
	}
	if !env.OK {
		t.Fatalf("%v returned error envelope: %#v", args, env)
	}
	return env
}

func newClientRPCMux(t *testing.T, stub clientRPCStub) http.Handler {
	t.Helper()
	path, handler := xuezhv1connect.NewXuezhServiceHandler(stub)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

func (s clientRPCStub) GetLearnerState(context.Context, *connect.Request[xuezhv1.GetLearnerStateRequest]) (*connect.Response[xuezhv1.LearnerState], error) {
	return connect.NewResponse(&xuezhv1.LearnerState{
		GeneratedAt:  timestamppb.New(time.Date(2026, 5, 28, 9, 10, 11, 0, time.UTC)),
		ChangedAt:    timestamppb.New(time.Date(2026, 5, 28, 9, 10, 12, 0, time.UTC)),
		StateHash:    "remote-state-hash",
		LearnedScore: 4,
		Columns:      []string{"item_id", "hanzi", "score"},
		Cards: []*xuezhv1.LearnerCardRow{{
			Values: []*structpb.Value{
				structpb.NewStringValue("hc:1"),
				structpb.NewStringValue("你好"),
				structpb.NewNumberValue(4),
			},
		}},
	}), nil
}

func (s clientRPCStub) GetSnapshot(_ context.Context, req *connect.Request[xuezhv1.GetSnapshotRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	if s.snapshotRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("snapshot request channel missing"))
	}
	s.snapshotRequests <- req.Msg
	data, err := structpb.NewStruct(map[string]any{
		"window": req.Msg.GetWindow(),
		"kind":   "remote-snapshot",
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	limits, err := structpb.NewStruct(map[string]any{"max_bytes": float64(req.Msg.GetMaxBytes())})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	bytes := int64(123)
	return connect.NewResponse(&xuezhv1.ReportPayload{
		Data:      data,
		Artifacts: []*xuezhv1.ServerArtifact{{Path: "artifacts/snapshot.json", Mime: "application/json", Purpose: "snapshot_spill", Bytes: &bytes}},
		Truncated: true,
		Limits:    limits,
	}), nil
}

func (s clientRPCStub) PreviewSRS(_ context.Context, req *connect.Request[xuezhv1.PreviewSRSRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	if s.previewRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("preview request channel missing"))
	}
	s.previewRequests <- req.Msg
	return newReportPayload(map[string]any{"kind": "remote-srs-preview", "days": float64(req.Msg.GetDays())}, nil)
}

func (s clientRPCStub) ReportHSK(_ context.Context, req *connect.Request[xuezhv1.ReportHSKRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	if s.hskRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("hsk request channel missing"))
	}
	s.hskRequests <- req.Msg
	return newReportPayload(map[string]any{"kind": "remote-hsk", "level": req.Msg.GetLevel()}, map[string]any{"max_bytes": float64(req.Msg.GetMaxBytes())})
}

func (s clientRPCStub) ReportMastery(_ context.Context, req *connect.Request[xuezhv1.ReportMasteryRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	if s.masteryRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("mastery request channel missing"))
	}
	s.masteryRequests <- req.Msg
	return newReportPayload(map[string]any{"kind": "remote-mastery", "item_type": req.Msg.GetItemType()}, map[string]any{"max_bytes": float64(req.Msg.GetMaxBytes())})
}

func (s clientRPCStub) ReportDue(_ context.Context, req *connect.Request[xuezhv1.ReportDueRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	if s.dueRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("due request channel missing"))
	}
	s.dueRequests <- req.Msg
	return newReportPayload(map[string]any{
		"kind":  "remote-due",
		"items": []any{map[string]any{"item_id": "hc:1", "due_at": "2026-05-28T09:10:11Z"}},
	}, map[string]any{"limit": float64(req.Msg.GetLimit()), "max_bytes": float64(req.Msg.GetMaxBytes())})
}

func newReportPayload(data map[string]any, limits map[string]any) (*connect.Response[xuezhv1.ReportPayload], error) {
	dataStruct, err := structpb.NewStruct(data)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if limits == nil {
		limits = map[string]any{}
	}
	limitStruct, err := structpb.NewStruct(limits)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&xuezhv1.ReportPayload{Data: dataStruct, Limits: limitStruct}), nil
}
