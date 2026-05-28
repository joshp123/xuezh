package rpc

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func (h *Handler) StartReview(ctx context.Context, req *connect.Request[xuezhv1.StartReviewRequest]) (*connect.Response[xuezhv1.StartReviewResponse], error) {
	result, err := h.app.StartReview(int(req.Msg.GetLimit()), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	recallItems, err := reviewItemMessages(result.RecallItems)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pronunciationItems, err := reviewItemMessages(result.PronunciationItems)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	generatedAt, err := timestampFromISO(result.GeneratedAt)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&xuezhv1.StartReviewResponse{
		RecallItems:        recallItems,
		PronunciationItems: pronunciationItems,
		GeneratedAt:        generatedAt,
	}), nil
}

func (h *Handler) GradeReview(ctx context.Context, req *connect.Request[xuezhv1.GradeReviewRequest]) (*connect.Response[xuezhv1.GradeReviewResponse], error) {
	result, err := h.app.GradeReview(service.GradeReviewOptions{
		ItemID:             req.Msg.GetItemId(),
		RecallGrade:        intPtrFromProto(req.Msg.Recall),
		PronunciationGrade: intPtrFromProto(req.Msg.Pronunciation),
		NextDue:            timestampString(req.Msg.GetNextDue()),
		Rule:               req.Msg.GetRule(),
	}, time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := gradeReviewMessage(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func (h *Handler) BuryReview(ctx context.Context, req *connect.Request[xuezhv1.BuryReviewRequest]) (*connect.Response[xuezhv1.BuryReviewResponse], error) {
	result, err := h.app.BuryReview(req.Msg.GetItemId(), req.Msg.GetReason(), time.Now().UTC())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	nextDue, err := timestampFromISO(result.NextDue)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&xuezhv1.BuryReviewResponse{ItemId: result.ItemID, Reason: result.Reason, NextDue: nextDue}), nil
}

func reviewItemMessages(items []service.ReviewItem) ([]*xuezhv1.ReviewItem, error) {
	result := make([]*xuezhv1.ReviewItem, 0, len(items))
	for _, item := range items {
		dueAt, err := timestampFromISO(item.DueAt)
		if err != nil {
			return nil, err
		}
		result = append(result, &xuezhv1.ReviewItem{ItemId: item.ItemID, DueAt: dueAt, ReviewType: item.ReviewType})
	}
	return result, nil
}

func gradeReviewMessage(result service.GradeReviewResult) (*xuezhv1.GradeReviewResponse, error) {
	recallNextDue, err := timestampFromOptionalISO(result.RecallNextDue)
	if err != nil {
		return nil, err
	}
	pronunciationNextDue, err := timestampFromOptionalISO(result.PronunciationNextDue)
	if err != nil {
		return nil, err
	}
	return &xuezhv1.GradeReviewResponse{
		ItemId:                   result.ItemID,
		RecallGrade:              int32Ptr(result.RecallGrade),
		RecallNextDue:            recallNextDue,
		RecallRuleApplied:        result.RecallRuleApplied,
		PronunciationGrade:       int32Ptr(result.PronunciationGrade),
		PronunciationNextDue:     pronunciationNextDue,
		PronunciationRuleApplied: result.PronunciationRuleApplied,
	}, nil
}

func intPtrFromProto(value *int32) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}

func int32Ptr(value *int) *int32 {
	if value == nil {
		return nil
	}
	out := int32(*value)
	return &out
}

func timestampString(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return clock.FormatISO(value.AsTime())
}
