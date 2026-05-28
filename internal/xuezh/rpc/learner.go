package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/cram"
)

func (h *Handler) GetLearnerState(ctx context.Context, req *connect.Request[xuezhv1.GetLearnerStateRequest]) (*connect.Response[xuezhv1.LearnerState], error) {
	state, err := h.app.LearnerState(time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := learnerStateMessage(state)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func learnerStateMessage(state cram.LearnerState) (*xuezhv1.LearnerState, error) {
	generatedAt, err := timestampFromISO(state.GeneratedAt)
	if err != nil {
		return nil, err
	}
	changedAt, err := timestampFromISO(state.ChangedAt)
	if err != nil {
		return nil, err
	}
	rows := make([]*xuezhv1.LearnerCardRow, 0, len(state.Cards))
	for _, card := range state.Cards {
		values := make([]*structpb.Value, 0, len(card))
		for _, value := range card {
			protoValue, err := structpb.NewValue(value)
			if err != nil {
				return nil, err
			}
			values = append(values, protoValue)
		}
		rows = append(rows, &xuezhv1.LearnerCardRow{Values: values})
	}
	return &xuezhv1.LearnerState{
		GeneratedAt:  generatedAt,
		ChangedAt:    changedAt,
		StateHash:    state.StateHash,
		LearnedScore: int32(state.LearnedScore),
		Columns:      append([]string{}, state.Columns...),
		Cards:        rows,
	}, nil
}
