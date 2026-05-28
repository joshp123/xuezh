package cli

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
)

func runClientEventLog(args []string, serverURL string) int {
	fs := flag.NewFlagSet("event log", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	eventType := fs.String("type", "", "exposure|review|pronunciation_attempt|content_served")
	modality := fs.String("modality", "", "reading|listening|speaking|typing|mixed")
	items := fs.String("items", "", "comma-separated item ids")
	itemsFile := fs.String("items-file", "", "file with item ids")
	contextValue := fs.String("context", "", "context")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	parsed, err := parseClientEventItems(*items, *itemsFile)
	if err != nil {
		return emitTypedError("event.log", "INVALID_ARGUMENT", err.Error(), map[string]any{"type": *eventType, "modality": *modality})
	}
	var contextPtr *string
	if *contextValue != "" {
		contextPtr = contextValue
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.LogEvent(context.Background(), connect.NewRequest(&xuezhv1.LogEventRequest{
		EventType: *eventType,
		Modality:  *modality,
		Items:     parsed,
		Context:   contextPtr,
	}))
	if err != nil {
		return emitError("event.log", err)
	}
	out := envelope.OK("event.log", eventRecordProtoData(resp.Msg), nil, false, nil)
	return emit(out)
}

func parseClientEventItems(items, itemsFile string) ([]string, error) {
	parsed := []string{}
	if items != "" {
		for _, part := range strings.Split(items, ",") {
			item := strings.TrimSpace(part)
			if item != "" {
				parsed = append(parsed, item)
			}
		}
	}
	if itemsFile != "" {
		data, err := os.ReadFile(expandLocalPath(itemsFile))
		if err != nil {
			return nil, err
		}
		for _, line := range strings.Split(string(data), "\n") {
			item := strings.TrimSpace(line)
			if item != "" {
				parsed = append(parsed, item)
			}
		}
	}
	for _, item := range parsed {
		if !ids.IsItemID(item) {
			return nil, fmt.Errorf("invalid item id: %s", item)
		}
	}
	return parsed, nil
}

func runClientEventList(args []string, serverURL string) int {
	fs := flag.NewFlagSet("event list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	since := fs.String("since", "7d", "since")
	limit := fs.Int("limit", 200, "limit")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.ListEvents(context.Background(), connect.NewRequest(&xuezhv1.ListEventsRequest{Since: *since, Limit: int32(*limit)}))
	if err != nil {
		return emitError("event.list", err)
	}
	eventsPayload := []map[string]any{}
	for _, event := range resp.Msg.GetEvents() {
		eventsPayload = append(eventsPayload, eventRecordProtoData(event))
	}
	out := envelope.OK(
		"event.list",
		map[string]any{"events": eventsPayload},
		nil,
		false,
		map[string]any{"limit": *limit, "since": *since},
	)
	return emit(out)
}

func eventRecordProtoData(event *xuezhv1.EventRecord) map[string]any {
	return map[string]any{
		"event_id":   event.GetEventId(),
		"event_type": event.GetEventType(),
		"ts":         protoTime(event.GetTs().AsTime()),
		"modality":   event.GetModality(),
		"items":      event.GetItems(),
		"context":    event.Context,
	}
}
