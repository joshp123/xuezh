package service

import (
	"testing"

	"github.com/joshp123/xuezh/internal/xuezh/ids"
)

func TestAppLogAndListEvents(t *testing.T) {
	useServiceTestWorkspace(t)
	itemID := ids.WordID("听", "ting")
	context := "openclaw"

	logged, err := New().LogEvent(LogEventOptions{
		EventType: "exposure",
		Modality:  "listening",
		Items:     []string{itemID},
		Context:   &context,
	})
	if err != nil {
		t.Fatal(err)
	}
	if logged.EventID == "" || logged.Context == nil || *logged.Context != context {
		t.Fatalf("unexpected logged event: %+v", logged)
	}

	listed, err := New().ListEvents("7d", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].EventID != logged.EventID {
		t.Fatalf("unexpected listed events: %+v", listed)
	}
}
