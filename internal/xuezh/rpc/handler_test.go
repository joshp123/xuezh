package rpc

import (
	"context"
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/cram"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/service"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
)

func TestHandlerGetLearnerState(t *testing.T) {
	useRPCTestWorkspace(t)
	if _, err := cram.ImportHelloChinese(cram.ImportOptions{Path: filepath.Join("..", "cram", "testdata", "hellochinese.txt"), AudioMode: "none"}); err != nil {
		t.Fatalf("import hellochinese: %v", err)
	}
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	resp, err := client.GetLearnerState(context.Background(), connect.NewRequest(&xuezhv1.GetLearnerStateRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.StateHash == "" || len(resp.Msg.Cards) != 3 {
		t.Fatalf("unexpected learner state: %+v", resp.Msg)
	}
	if len(resp.Msg.Columns) == 0 || len(resp.Msg.Cards[0].Values) != len(resp.Msg.Columns) {
		t.Fatalf("columnar row mismatch: columns=%d values=%d", len(resp.Msg.Columns), len(resp.Msg.Cards[0].Values))
	}
}

func TestHandlerGetSnapshot(t *testing.T) {
	useRPCTestWorkspace(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	resp, err := client.GetSnapshot(context.Background(), connect.NewRequest(&xuezhv1.GetSnapshotRequest{
		Window:        "7d",
		DueLimit:      5,
		EvidenceLimit: 10,
		MaxBytes:      64 * 1024,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetData().GetFields()["window"].GetStringValue() != "7d" || resp.Msg.GetTruncated() {
		t.Fatalf("unexpected snapshot payload: %+v", resp.Msg)
	}
}

func TestHandlerReportPayloadRPCs(t *testing.T) {
	useRPCTestWorkspace(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	preview, err := client.PreviewSRS(context.Background(), connect.NewRequest(&xuezhv1.PreviewSRSRequest{Days: 5}))
	if err != nil {
		t.Fatal(err)
	}
	if preview.Msg.GetData().GetFields()["days"].GetNumberValue() != 5 {
		t.Fatalf("unexpected srs preview payload: %+v", preview.Msg)
	}

	hsk, err := client.ReportHSK(context.Background(), connect.NewRequest(&xuezhv1.ReportHSKRequest{Level: "1", Window: "30d", MaxItems: 10, MaxBytes: 64 * 1024}))
	if err != nil {
		t.Fatal(err)
	}
	if hsk.Msg.GetData().GetFields()["level"].GetStringValue() != "1" {
		t.Fatalf("unexpected hsk payload: %+v", hsk.Msg)
	}

	mastery, err := client.ReportMastery(context.Background(), connect.NewRequest(&xuezhv1.ReportMasteryRequest{ItemType: "word", Window: "30d", MaxItems: 10, MaxBytes: 64 * 1024}))
	if err != nil {
		t.Fatal(err)
	}
	if mastery.Msg.GetData().GetFields()["item_type"].GetStringValue() != "word" {
		t.Fatalf("unexpected mastery payload: %+v", mastery.Msg)
	}

	due, err := client.ReportDue(context.Background(), connect.NewRequest(&xuezhv1.ReportDueRequest{Limit: 10, MaxBytes: 64 * 1024}))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := due.Msg.GetData().GetFields()["items"]; !ok {
		t.Fatalf("unexpected due payload: %+v", due.Msg)
	}
}

func TestHandlerReviewRPCsMutateServerState(t *testing.T) {
	useRPCTestWorkspace(t)
	itemID := ids.WordID("吗", "ma")
	now := time.Now().UTC()
	dueAt := clock.FormatISO(now.Add(-time.Hour))
	grade := 3
	if err := srs.UpsertKnowledge(itemID, &dueAt, &grade, &dueAt, &grade, now.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed due item: %v", err)
	}

	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	start, err := client.StartReview(context.Background(), connect.NewRequest(&xuezhv1.StartReviewRequest{Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(start.Msg.GetRecallItems()) != 1 || start.Msg.GetRecallItems()[0].GetItemId() != itemID {
		t.Fatalf("unexpected start response: %+v", start.Msg)
	}

	recall := int32(4)
	pronunciation := int32(3)
	nextDue := now.Add(48 * time.Hour)
	graded, err := client.GradeReview(context.Background(), connect.NewRequest(&xuezhv1.GradeReviewRequest{
		ItemId:        itemID,
		Recall:        &recall,
		Pronunciation: &pronunciation,
		NextDue:       timestamppb.New(nextDue),
		Rule:          "manual",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if graded.Msg.GetItemId() != itemID || graded.Msg.GetRecallGrade() != 4 || graded.Msg.GetPronunciationGrade() != 3 {
		t.Fatalf("unexpected grade response: %+v", graded.Msg)
	}

	buried, err := client.BuryReview(context.Background(), connect.NewRequest(&xuezhv1.BuryReviewRequest{ItemId: itemID, Reason: "too_easy"}))
	if err != nil {
		t.Fatal(err)
	}
	if buried.Msg.GetItemId() != itemID || buried.Msg.GetReason() != "too_easy" || buried.Msg.GetNextDue() == nil {
		t.Fatalf("unexpected bury response: %+v", buried.Msg)
	}
}

func TestHandlerEventRPCsMutateServerState(t *testing.T) {
	useRPCTestWorkspace(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	contextValue := "lesson"
	logged, err := client.LogEvent(context.Background(), connect.NewRequest(&xuezhv1.LogEventRequest{
		EventType: "exposure",
		Modality:  "reading",
		Items:     []string{"w_000000000001"},
		Context:   &contextValue,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if logged.Msg.GetEventId() == "" || logged.Msg.GetContext() != "lesson" {
		t.Fatalf("unexpected log response: %+v", logged.Msg)
	}

	listed, err := client.ListEvents(context.Background(), connect.NewRequest(&xuezhv1.ListEventsRequest{Since: "30d", Limit: 10}))
	if err != nil {
		t.Fatal(err)
	}
	if len(listed.Msg.GetEvents()) != 1 || listed.Msg.GetEvents()[0].GetEventId() != logged.Msg.GetEventId() {
		t.Fatalf("unexpected list response: %+v", listed.Msg)
	}
}

func TestHandlerContentRPCsUseServerArtifacts(t *testing.T) {
	useRPCTestWorkspace(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	put, err := client.PutContent(context.Background(), connect.NewRequest(&xuezhv1.PutContentRequest{
		Type:     "story",
		Key:      "tea",
		Filename: "tea.txt",
		Content:  []byte("茶很好喝"),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if put.Msg.GetKey() != "tea" || len(put.Msg.GetArtifacts()) != 1 {
		t.Fatalf("unexpected put response: %+v", put.Msg)
	}

	got, err := client.GetContent(context.Background(), connect.NewRequest(&xuezhv1.GetContentRequest{Type: "story", Key: "tea"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.Msg.GetRecord().GetContentId() != put.Msg.GetContentId() || string(got.Msg.GetContent()) != "茶很好喝" {
		t.Fatalf("unexpected get response: %+v content=%q", got.Msg.GetRecord(), got.Msg.GetContent())
	}
}

func TestHandlerAudioRPCsUseServerAudioAndRecordPronunciationAttempt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("fake tool scripts are POSIX shell")
	}
	useRPCTestWorkspaceWithConfig(t, "\n[audio]\nprocess_voice_backend = \"local\"\ntts_backend = \"edge-tts\"\n")
	addFakeAudioTools(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	tts, err := client.SynthesizeSpeech(context.Background(), connect.NewRequest(&xuezhv1.SynthesizeSpeechRequest{
		Text:         "你好",
		Voice:        "XiaoxiaoNeural",
		OutputFormat: "ogg",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if tts.Msg.GetText() != "你好" || len(tts.Msg.GetArtifacts()) != 1 || string(tts.Msg.GetAudio().GetData()) == "" {
		t.Fatalf("unexpected tts response: %+v", tts.Msg)
	}

	processed, err := client.ProcessVoice(context.Background(), connect.NewRequest(&xuezhv1.ProcessVoiceRequest{
		Audio:    []byte("voice"),
		Filename: "voice.ogg",
		Mime:     "audio/ogg",
		RefText:  "你好",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if processed.Msg.GetAttemptId() == "" || processed.Msg.GetAssessment().GetFields()["exact_match"].GetBoolValue() != true || string(processed.Msg.GetFeedbackAudio().GetData()) == "" {
		t.Fatalf("unexpected process voice response: %+v", processed.Msg)
	}
	if countPronunciationAttempts(t) != 1 {
		t.Fatal("process voice did not record exactly one server-side pronunciation attempt")
	}
}

func TestHandlerDoctorRPCReportsServerWorkspace(t *testing.T) {
	useRPCTestWorkspace(t)
	path, handler := NewHandler(service.New())
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	client := xuezhv1connect.NewXuezhServiceClient(server.Client(), server.URL)
	resp, err := client.Doctor(context.Background(), connect.NewRequest(&xuezhv1.DoctorRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Msg.GetWorkspaceRole() != "server" || resp.Msg.GetWorkspacePath() == "" {
		t.Fatalf("unexpected doctor response: %+v", resp.Msg)
	}
	if !hasRPCDoctorCheck(resp.Msg.GetChecks(), "workspace.path") {
		t.Fatalf("doctor checks = %+v", resp.Msg.GetChecks())
	}
}

func hasRPCDoctorCheck(checks []*xuezhv1.DoctorCheck, name string) bool {
	for _, check := range checks {
		if check.GetName() == name {
			return true
		}
	}
	return false
}

func useRPCTestWorkspace(t *testing.T) {
	t.Helper()
	useRPCTestWorkspaceWithConfig(t, "")
}

func useRPCTestWorkspaceWithConfig(t *testing.T, extraConfig string) {
	t.Helper()
	workspace := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	configPath := filepath.Join(configHome, "xuezh", "config.toml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "[workspace]\ndir = \"" + workspace + "\"\n" + extraConfig
	if err := os.WriteFile(configPath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func addFakeAudioTools(t *testing.T) {
	t.Helper()
	bin := t.TempDir()
	writeExecutable(t, filepath.Join(bin, "edge-tts"), `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --write-media) shift; out="$1" ;;
	esac
  shift
done
printf 'fake-edge-tts' > "$out"
`)
	writeExecutable(t, filepath.Join(bin, "ffmpeg"), `#!/bin/sh
out=""
for arg do
  out="$arg"
done
printf 'fake-ffmpeg' > "$out"
`)
	writeExecutable(t, filepath.Join(bin, "whisper"), `#!/bin/sh
in="$1"
outdir=""
while [ "$#" -gt 0 ]; do
  case "$1" in
    --output_dir) shift; outdir="$1" ;;
  esac
  shift
done
base="$(basename "$in")"
base="${base%.*}"
mkdir -p "$outdir"
printf '{"text":"你好","segments":[],"language":"zh"}' > "$outdir/$base.json"
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func countPronunciationAttempts(t *testing.T) int {
	t.Helper()
	dbPath, err := db.InitDB()
	if err != nil {
		t.Fatal(err)
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM pronunciation_attempts").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
