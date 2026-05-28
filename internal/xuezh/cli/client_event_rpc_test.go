package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
)

func TestClientBackedEventCommandsUseRPC(t *testing.T) {
	stub := clientRPCStub{
		logEventRequests:  make(chan *xuezhv1.LogEventRequest, 1),
		listEventRequests: make(chan *xuezhv1.ListEventsRequest, 1),
	}
	server := newClientRPCServer(t, stub)
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	itemsPath := filepath.Join(t.TempDir(), "items.txt")
	if err := os.WriteFile(itemsPath, []byte("w_000000000001\nw_000000000002\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logged := runClientCommandForTest(t, []string{"event", "log", "--type", "exposure", "--modality", "reading", "--items-file", itemsPath, "--context", "lesson", "--json"})
	logReq := <-stub.logEventRequests
	if logReq.GetEventType() != "exposure" || logReq.GetModality() != "reading" || logReq.GetContext() != "lesson" {
		t.Fatalf("log request = %+v", logReq)
	}
	if len(logReq.GetItems()) != 2 || logReq.GetItems()[0] != "w_000000000001" || logReq.GetItems()[1] != "w_000000000002" {
		t.Fatalf("log items = %+v", logReq.GetItems())
	}
	if logged.Command != "event.log" || logged.Data["event_id"] != "evt_remote" || logged.Data["context"] != "lesson" {
		t.Fatalf("event.log envelope = %#v", logged)
	}

	listed := runClientCommandForTest(t, []string{"event", "list", "--since", "14d", "--limit", "3", "--json"})
	listReq := <-stub.listEventRequests
	if listReq.GetSince() != "14d" || listReq.GetLimit() != 3 {
		t.Fatalf("list request = %+v", listReq)
	}
	events, ok := listed.Data["events"].([]any)
	if listed.Command != "event.list" || !ok || len(events) != 1 || listed.Limits["since"] != "14d" || listed.Limits["limit"] != float64(3) {
		t.Fatalf("event.list envelope = %#v", listed)
	}
}

func (s clientRPCStub) LogEvent(_ context.Context, req *connect.Request[xuezhv1.LogEventRequest]) (*connect.Response[xuezhv1.EventRecord], error) {
	if s.logEventRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("log event request channel missing"))
	}
	s.logEventRequests <- req.Msg
	contextValue := req.Msg.Context
	return connect.NewResponse(&xuezhv1.EventRecord{
		EventId:   "evt_remote",
		EventType: req.Msg.GetEventType(),
		Ts:        timestamppb.New(time.Date(2026, 5, 28, 9, 10, 11, 0, time.UTC)),
		Modality:  req.Msg.GetModality(),
		Items:     req.Msg.GetItems(),
		Context:   contextValue,
	}), nil
}

func (s clientRPCStub) ListEvents(_ context.Context, req *connect.Request[xuezhv1.ListEventsRequest]) (*connect.Response[xuezhv1.ListEventsResponse], error) {
	if s.listEventRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("list event request channel missing"))
	}
	s.listEventRequests <- req.Msg
	contextValue := "lesson"
	return connect.NewResponse(&xuezhv1.ListEventsResponse{Events: []*xuezhv1.EventRecord{{
		EventId:   "evt_remote",
		EventType: "exposure",
		Ts:        timestamppb.New(time.Date(2026, 5, 28, 9, 10, 11, 0, time.UTC)),
		Modality:  "reading",
		Items:     []string{"w_000000000001"},
		Context:   &contextValue,
	}}}), nil
}
