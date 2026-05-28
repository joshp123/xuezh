package rpc

import (
	"context"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func (h *Handler) LogEvent(ctx context.Context, req *connect.Request[xuezhv1.LogEventRequest]) (*connect.Response[xuezhv1.EventRecord], error) {
	event, err := h.app.LogEvent(service.LogEventOptions{
		EventType: req.Msg.GetEventType(),
		Modality:  req.Msg.GetModality(),
		Items:     req.Msg.GetItems(),
		Context:   req.Msg.Context,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := eventRecordMessage(event)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) ListEvents(ctx context.Context, req *connect.Request[xuezhv1.ListEventsRequest]) (*connect.Response[xuezhv1.ListEventsResponse], error) {
	events, err := h.app.ListEvents(req.Msg.GetSince(), int(req.Msg.GetLimit()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	messages := make([]*xuezhv1.EventRecord, 0, len(events))
	for _, event := range events {
		msg, err := eventRecordMessage(event)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		messages = append(messages, msg)
	}
	return connect.NewResponse(&xuezhv1.ListEventsResponse{Events: messages}), nil
}

func eventRecordMessage(event service.EventRecord) (*xuezhv1.EventRecord, error) {
	ts, err := timestampFromISO(event.TS)
	if err != nil {
		return nil, err
	}
	return &xuezhv1.EventRecord{
		EventId:   event.EventID,
		EventType: event.EventType,
		Ts:        ts,
		Modality:  event.Modality,
		Items:     append([]string{}, event.Items...),
		Context:   event.Context,
	}, nil
}
