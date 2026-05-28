package cli

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
)

func TestClientBackedReviewCommandsUseRPC(t *testing.T) {
	stub := clientRPCStub{
		startRequests: make(chan *xuezhv1.StartReviewRequest, 1),
		gradeRequests: make(chan *xuezhv1.GradeReviewRequest, 2),
		buryRequests:  make(chan *xuezhv1.BuryReviewRequest, 1),
	}
	server := newClientRPCServer(t, stub)
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	start := runClientCommandForTest(t, []string{"review", "start", "--limit", "2", "--json"})
	if req := <-stub.startRequests; req.GetLimit() != 2 {
		t.Fatalf("start request = %+v", req)
	}
	if start.Command != "review.start" || start.Data["generated_at"] != "2026-05-28T09:10:11Z" {
		t.Fatalf("start envelope = %#v", start)
	}
	if items, ok := start.Data["recall_items"].([]any); !ok || len(items) != 1 {
		t.Fatalf("start recall items = %#v", start.Data["recall_items"])
	}

	nextDue := "2026-05-29T09:10:11Z"
	grade := runClientCommandForTest(t, []string{"review", "grade", "--item", "hc:1", "--recall", "4", "--pronunciation", "3", "--next-due", nextDue, "--rule", "manual", "--json"})
	gradeReq := <-stub.gradeRequests
	if gradeReq.GetItemId() != "hc:1" || gradeReq.GetRecall() != 4 || gradeReq.GetPronunciation() != 3 || gradeReq.GetRule() != "manual" {
		t.Fatalf("grade request = %+v", gradeReq)
	}
	if gradeReq.GetNextDue().AsTime().UTC().Format(time.RFC3339) != nextDue {
		t.Fatalf("grade next_due = %s", gradeReq.GetNextDue().AsTime())
	}
	if grade.Data["recall_grade"] != float64(4) || grade.Data["pronunciation_grade"] != float64(3) || grade.Data["recall_next_due"] != nextDue {
		t.Fatalf("grade envelope = %#v", grade)
	}

	legacy := runClientCommandForTest(t, []string{"review", "grade", "--item", "hc:1", "--grade", "4", "--json"})
	legacyReq := <-stub.gradeRequests
	if legacyReq.GetRecall() != 4 || legacyReq.Pronunciation != nil {
		t.Fatalf("legacy grade request = %+v", legacyReq)
	}
	if legacy.Data["grade"] != float64(4) || legacy.Data["next_due"] != nextDue {
		t.Fatalf("legacy grade envelope = %#v", legacy)
	}

	bury := runClientCommandForTest(t, []string{"review", "bury", "--item", "hc:1", "--reason", "too_easy", "--json"})
	buryReq := <-stub.buryRequests
	if buryReq.GetItemId() != "hc:1" || buryReq.GetReason() != "too_easy" {
		t.Fatalf("bury request = %+v", buryReq)
	}
	if bury.Command != "review.bury" || bury.Data["item"] != "hc:1" || bury.Data["reason"] != "too_easy" {
		t.Fatalf("bury envelope = %#v", bury)
	}
}

func newClientRPCServer(t *testing.T, stub clientRPCStub) *httptest.Server {
	t.Helper()
	return httptest.NewServer(newClientRPCMux(t, stub))
}

func (s clientRPCStub) StartReview(_ context.Context, req *connect.Request[xuezhv1.StartReviewRequest]) (*connect.Response[xuezhv1.StartReviewResponse], error) {
	if s.startRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("start request channel missing"))
	}
	s.startRequests <- req.Msg
	return connect.NewResponse(&xuezhv1.StartReviewResponse{
		RecallItems: []*xuezhv1.ReviewItem{{
			ItemId:     "hc:1",
			DueAt:      timestamppb.New(time.Date(2026, 5, 27, 9, 10, 11, 0, time.UTC)),
			ReviewType: "recall",
		}},
		PronunciationItems: []*xuezhv1.ReviewItem{{
			ItemId:     "hc:1",
			DueAt:      timestamppb.New(time.Date(2026, 5, 27, 9, 10, 12, 0, time.UTC)),
			ReviewType: "pronunciation",
		}},
		GeneratedAt: timestamppb.New(time.Date(2026, 5, 28, 9, 10, 11, 0, time.UTC)),
	}), nil
}

func (s clientRPCStub) GradeReview(_ context.Context, req *connect.Request[xuezhv1.GradeReviewRequest]) (*connect.Response[xuezhv1.GradeReviewResponse], error) {
	if s.gradeRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("grade request channel missing"))
	}
	s.gradeRequests <- req.Msg
	rule := "manual"
	nextDue := timestamppb.New(time.Date(2026, 5, 29, 9, 10, 11, 0, time.UTC))
	return connect.NewResponse(&xuezhv1.GradeReviewResponse{
		ItemId:                   req.Msg.GetItemId(),
		RecallGrade:              req.Msg.Recall,
		RecallNextDue:            nextDue,
		RecallRuleApplied:        &rule,
		PronunciationGrade:       req.Msg.Pronunciation,
		PronunciationNextDue:     nextDue,
		PronunciationRuleApplied: &rule,
	}), nil
}

func (s clientRPCStub) BuryReview(_ context.Context, req *connect.Request[xuezhv1.BuryReviewRequest]) (*connect.Response[xuezhv1.BuryReviewResponse], error) {
	if s.buryRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("bury request channel missing"))
	}
	s.buryRequests <- req.Msg
	return connect.NewResponse(&xuezhv1.BuryReviewResponse{
		ItemId:  req.Msg.GetItemId(),
		Reason:  req.Msg.GetReason(),
		NextDue: timestamppb.New(time.Date(2026, 5, 29, 9, 10, 11, 0, time.UTC)),
	}), nil
}
