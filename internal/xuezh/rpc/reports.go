package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func (h *Handler) GetSnapshot(ctx context.Context, req *connect.Request[xuezhv1.GetSnapshotRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	result, err := h.app.Snapshot(req.Msg.GetWindow(), int(req.Msg.GetDueLimit()), int(req.Msg.GetEvidenceLimit()), int(req.Msg.GetMaxBytes()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := reportPayloadMessage(result.Data, result.Artifacts, result.Truncated, result.Limits)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) PreviewSRS(ctx context.Context, req *connect.Request[xuezhv1.PreviewSRSRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	result, err := h.app.PreviewSRS(int(req.Msg.GetDays()), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := reportPayloadMessage(map[string]any{"days": result.Days, "forecast": result.Forecast}, nil, false, nil)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) ReportHSK(ctx context.Context, req *connect.Request[xuezhv1.ReportHSKRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	result, err := h.app.ReportHSK(req.Msg.GetLevel(), req.Msg.GetWindow(), int(req.Msg.GetMaxItems()), int(req.Msg.GetMaxBytes()), req.Msg.GetIncludeChars())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := reportPayloadMessage(result.Data, result.Artifacts, result.Truncated, result.Limits)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) ReportMastery(ctx context.Context, req *connect.Request[xuezhv1.ReportMasteryRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	result, err := h.app.ReportMastery(req.Msg.GetItemType(), req.Msg.GetWindow(), int(req.Msg.GetMaxItems()), int(req.Msg.GetMaxBytes()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := reportPayloadMessage(result.Data, result.Artifacts, result.Truncated, result.Limits)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) ReportDue(ctx context.Context, req *connect.Request[xuezhv1.ReportDueRequest]) (*connect.Response[xuezhv1.ReportPayload], error) {
	result, err := h.app.ReportDue(int(req.Msg.GetLimit()), int(req.Msg.GetMaxBytes()), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := reportPayloadMessage(reportDueData(result), nil, false, map[string]any{"limit": result.Limit, "max_bytes": result.MaxBytes})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func reportDueData(result service.DueReportResult) map[string]any {
	items := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, map[string]any{"item_id": item.ItemID, "due_at": item.DueAt})
	}
	return map[string]any{"items": items}
}
