package service

import (
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/reports"
	"github.com/joshp123/xuezh/internal/xuezh/srs"
)

type DueReportItem struct {
	ItemID string
	DueAt  string
}

type DueReportResult struct {
	Items    []DueReportItem
	Limit    int
	MaxBytes int
}

func (App) ReportHSK(level, window string, maxItems, maxBytes int, includeChars bool) (reports.ReportResult, error) {
	return reports.BuildHSKReport(level, window, maxItems, maxBytes, includeChars)
}

func (App) ReportMastery(itemType, window string, maxItems, maxBytes int) (reports.ReportResult, error) {
	return reports.BuildMasteryReport(itemType, window, maxItems, maxBytes)
}

func (App) ReportDue(limit int, maxBytes int, now time.Time) (DueReportResult, error) {
	items, err := srs.ListDueItems(limit, now, "recall")
	if err != nil {
		return DueReportResult{}, err
	}
	result := DueReportResult{Items: make([]DueReportItem, 0, len(items)), Limit: limit, MaxBytes: maxBytes}
	for _, item := range items {
		result.Items = append(result.Items, DueReportItem{ItemID: item.ItemID, DueAt: item.DueAt})
	}
	return result, nil
}
