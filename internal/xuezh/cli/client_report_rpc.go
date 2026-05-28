package cli

import (
	"context"
	"flag"
	"net/http"
	"os"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func runClientSnapshot(args []string, serverURL string) int {
	fs := flag.NewFlagSet("snapshot", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	window := fs.String("window", "30d", "window")
	dueLimit := fs.Int("due-limit", 80, "due limit")
	evidenceLimit := fs.Int("evidence-limit", 200, "evidence limit")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.GetSnapshot(context.Background(), connect.NewRequest(&xuezhv1.GetSnapshotRequest{
		Window:        *window,
		DueLimit:      int32(*dueLimit),
		EvidenceLimit: int32(*evidenceLimit),
		MaxBytes:      int32(*maxBytes),
	}))
	if err != nil {
		return emitError("snapshot", err)
	}
	return emitReportPayload("snapshot", resp.Msg)
}

func runClientSRSPreview(args []string, serverURL string) int {
	fs := flag.NewFlagSet("srs preview", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	days := fs.Int("days", 14, "days")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.PreviewSRS(context.Background(), connect.NewRequest(&xuezhv1.PreviewSRSRequest{Days: int32(*days)}))
	if err != nil {
		return emitError("srs.preview", err)
	}
	return emitReportPayload("srs.preview", resp.Msg)
}

func runClientReportHSK(args []string, serverURL string) int {
	fs := flag.NewFlagSet("report hsk", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	level := fs.String("level", "", "HSK level")
	window := fs.String("window", "30d", "window")
	maxItems := fs.Int("max-items", 200, "max items")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	includeChars := fs.Bool("include-chars", false, "include chars")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *level == "" {
		return emitTypedError("report.hsk", "INVALID_ARGUMENT", "level is required", map[string]any{"level": *level})
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.ReportHSK(context.Background(), connect.NewRequest(&xuezhv1.ReportHSKRequest{
		Level:        *level,
		Window:       *window,
		MaxItems:     int32(*maxItems),
		MaxBytes:     int32(*maxBytes),
		IncludeChars: *includeChars,
	}))
	if err != nil {
		return emitError("report.hsk", err)
	}
	return emitReportPayload("report.hsk", resp.Msg)
}

func runClientReportMastery(args []string, serverURL string) int {
	fs := flag.NewFlagSet("report mastery", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	itemType := fs.String("item-type", "word", "word|character|grammar")
	window := fs.String("window", "90d", "window")
	maxItems := fs.Int("max-items", 200, "max items")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.ReportMastery(context.Background(), connect.NewRequest(&xuezhv1.ReportMasteryRequest{
		ItemType: *itemType,
		Window:   *window,
		MaxItems: int32(*maxItems),
		MaxBytes: int32(*maxBytes),
	}))
	if err != nil {
		return emitError("report.mastery", err)
	}
	return emitReportPayload("report.mastery", resp.Msg)
}

func runClientReportDue(args []string, serverURL string) int {
	fs := flag.NewFlagSet("report due", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 50, "limit")
	maxBytes := fs.Int("max-bytes", 200000, "max bytes")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.ReportDue(context.Background(), connect.NewRequest(&xuezhv1.ReportDueRequest{
		Limit:    int32(*limit),
		MaxBytes: int32(*maxBytes),
	}))
	if err != nil {
		return emitError("report.due", err)
	}
	return emitReportPayload("report.due", resp.Msg)
}

func emitReportPayload(command string, payload *xuezhv1.ReportPayload) int {
	out := envelope.OK(command, payload.GetData().AsMap(), reportArtifacts(payload.GetArtifacts()), payload.GetTruncated(), payload.GetLimits().AsMap())
	return emit(out)
}

func reportArtifacts(artifacts []*xuezhv1.ServerArtifact) []envelope.Artifact {
	result := make([]envelope.Artifact, 0, len(artifacts))
	for _, artifact := range artifacts {
		out := envelope.Artifact{Path: artifact.GetPath(), MIME: artifact.GetMime(), Purpose: artifact.GetPurpose()}
		if artifact.Bytes != nil {
			bytes := int(artifact.GetBytes())
			out.Bytes = &bytes
		}
		result = append(result, out)
	}
	return result
}
