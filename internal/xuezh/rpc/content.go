package rpc

import (
	"context"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func (h *Handler) PutContent(ctx context.Context, req *connect.Request[xuezhv1.PutContentRequest]) (*connect.Response[xuezhv1.ContentRecord], error) {
	if len(req.Msg.GetContent()) > contentBytesMax {
		return nil, limitError(connect.CodeResourceExhausted, "content", len(req.Msg.GetContent()), contentBytesMax)
	}
	result, err := h.app.PutContent(service.PutContentOptions{
		ContentType: req.Msg.GetType(),
		Key:         req.Msg.GetKey(),
		Filename:    req.Msg.GetFilename(),
		Data:        req.Msg.GetContent(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	return connect.NewResponse(contentRecordMessage(result.Data, result.Artifacts)), nil
}

func (h *Handler) GetContent(ctx context.Context, req *connect.Request[xuezhv1.GetContentRequest]) (*connect.Response[xuezhv1.GetContentResponse], error) {
	result, data, err := h.app.GetContentBytes(req.Msg.GetType(), req.Msg.GetKey())
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if len(data) > contentBytesMax {
		return nil, limitError(connect.CodeResourceExhausted, "content", len(data), contentBytesMax)
	}
	return connect.NewResponse(&xuezhv1.GetContentResponse{
		Record:  contentRecordMessage(result.Data, result.Artifacts),
		Content: data,
	}), nil
}

func contentRecordMessage(data map[string]any, artifacts []envelope.Artifact) *xuezhv1.ContentRecord {
	return &xuezhv1.ContentRecord{
		Type:      stringField(data, "type"),
		Key:       stringField(data, "key"),
		ContentId: stringField(data, "content_id"),
		Artifacts: artifactMessages(artifacts),
	}
}
