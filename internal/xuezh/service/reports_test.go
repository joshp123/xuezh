package service

import (
	"testing"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
)

func TestAppReportDueReturnsRecallDueItems(t *testing.T) {
	useServiceTestWorkspace(t)
	now := time.Date(2026, 4, 26, 12, 0, 0, 0, time.UTC)
	itemID := ids.WordID("路", "lu")
	recallDue := clock.FormatISO(now.Add(-time.Hour))
	recallGrade := 2
	if err := srs.UpsertKnowledge(itemID, &recallDue, &recallGrade, nil, nil, now.Add(-24*time.Hour)); err != nil {
		t.Fatalf("seed knowledge: %v", err)
	}

	result, err := New().ReportDue(10, 200000, now)
	if err != nil {
		t.Fatal(err)
	}
	if result.Limit != 10 || result.MaxBytes != 200000 {
		t.Fatalf("unexpected limits: %+v", result)
	}
	if len(result.Items) != 1 || result.Items[0].ItemID != itemID {
		t.Fatalf("unexpected due items: %+v", result.Items)
	}
}

func TestAppReportHSKAndMasteryReturnFactualPayloads(t *testing.T) {
	useServiceTestWorkspace(t)

	hsk, err := New().ReportHSK("1", "30d", 20, 200000, false)
	if err != nil {
		t.Fatal(err)
	}
	if hsk.Data["level"] != "1" {
		t.Fatalf("unexpected hsk report: %+v", hsk.Data)
	}

	mastery, err := New().ReportMastery("word", "90d", 20, 200000)
	if err != nil {
		t.Fatal(err)
	}
	if mastery.Data["item_type"] != "word" {
		t.Fatalf("unexpected mastery report: %+v", mastery.Data)
	}
}
