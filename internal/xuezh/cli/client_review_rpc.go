package cli

import (
	"context"
	"flag"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
)

func runClientReviewStart(args []string, serverURL string) int {
	fs := flag.NewFlagSet("review start", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 10, "limit")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.StartReview(context.Background(), connect.NewRequest(&xuezhv1.StartReviewRequest{Limit: int32(*limit)}))
	if err != nil {
		return emitError("review.start", err)
	}
	recallPayload := reviewItemsProtoData(resp.Msg.GetRecallItems())
	pronPayload := reviewItemsProtoData(resp.Msg.GetPronunciationItems())
	out := envelope.OK("review.start", map[string]any{
		"items":               recallPayload,
		"recall_items":        recallPayload,
		"pronunciation_items": pronPayload,
		"generated_at":        protoTime(resp.Msg.GetGeneratedAt().AsTime()),
	}, nil, false, map[string]any{"limit": *limit})
	return emit(out)
}

func runClientReviewGrade(args []string, serverURL string) int {
	fs := flag.NewFlagSet("review grade", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	item := fs.String("item", "", "item id")
	grade := fs.String("grade", "", "legacy recall grade 0-5")
	recall := fs.String("recall", "", "recall grade 0-5")
	pronunciation := fs.String("pronunciation", "", "pronunciation grade 0-5")
	nextDue := fs.String("next-due", "", "explicit next due timestamp")
	rule := fs.String("rule", "", "sm2|leitner")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *item == "" {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "item is required", map[string]any{"item": *item})
	}
	if *grade != "" && (*recall != "" || *pronunciation != "") {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "use --grade alone or --recall/--pronunciation, not both", map[string]any{"item": *item})
	}
	if *grade == "" && *recall == "" && *pronunciation == "" {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "provide --grade or --recall/--pronunciation", map[string]any{"item": *item})
	}
	recallGrade, err := parseOptionalGrade(*recall)
	if err != nil {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid recall grade", map[string]any{"item": *item})
	}
	pronunciationGrade, err := parseOptionalGrade(*pronunciation)
	if err != nil {
		return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid pronunciation grade", map[string]any{"item": *item})
	}
	legacyGrade := false
	if *grade != "" {
		value, err := parseOptionalGrade(*grade)
		if err != nil {
			return emitTypedError("review.grade", "INVALID_ARGUMENT", "invalid grade", map[string]any{"item": *item})
		}
		recallGrade = value
		legacyGrade = true
	}
	nextDueTimestamp, err := protoTimestampFromFlag(*nextDue)
	if err != nil {
		return emitError("review.grade", err)
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.GradeReview(context.Background(), connect.NewRequest(&xuezhv1.GradeReviewRequest{
		ItemId:        *item,
		Recall:        int32PtrFromInt(recallGrade),
		Pronunciation: int32PtrFromInt(pronunciationGrade),
		NextDue:       nextDueTimestamp,
		Rule:          *rule,
	}))
	if err != nil {
		return emitError("review.grade", err)
	}
	out := envelope.OK("review.grade", gradeReviewProtoData(resp.Msg, legacyGrade), nil, false, nil)
	return emit(out)
}

func runClientReviewBury(args []string, serverURL string) int {
	fs := flag.NewFlagSet("review bury", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	item := fs.String("item", "", "item id")
	reason := fs.String("reason", "manual", "reason")
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *item == "" {
		return emitTypedError("review.bury", "INVALID_ARGUMENT", "item is required", map[string]any{"item": *item})
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.BuryReview(context.Background(), connect.NewRequest(&xuezhv1.BuryReviewRequest{ItemId: *item, Reason: *reason}))
	if err != nil {
		return emitError("review.bury", err)
	}
	out := envelope.OK("review.bury", map[string]any{
		"item":     resp.Msg.GetItemId(),
		"reason":   resp.Msg.GetReason(),
		"next_due": protoTime(resp.Msg.GetNextDue().AsTime()),
	}, nil, false, nil)
	return emit(out)
}

func reviewItemsProtoData(items []*xuezhv1.ReviewItem) []map[string]any {
	payload := []map[string]any{}
	for _, item := range items {
		payload = append(payload, map[string]any{
			"item_id":     item.GetItemId(),
			"due_at":      protoTime(item.GetDueAt().AsTime()),
			"review_type": item.GetReviewType(),
		})
	}
	return payload
}

func gradeReviewProtoData(result *xuezhv1.GradeReviewResponse, includeLegacyGrade bool) map[string]any {
	data := map[string]any{"item": result.GetItemId()}
	if result.RecallGrade != nil {
		data["recall_grade"] = int(result.GetRecallGrade())
		data["recall_next_due"] = protoTime(result.GetRecallNextDue().AsTime())
		data["recall_rule_applied"] = result.GetRecallRuleApplied()
	}
	if result.PronunciationGrade != nil {
		data["pronunciation_grade"] = int(result.GetPronunciationGrade())
		data["pronunciation_next_due"] = protoTime(result.GetPronunciationNextDue().AsTime())
		data["pronunciation_rule_applied"] = result.GetPronunciationRuleApplied()
	}
	if includeLegacyGrade && result.RecallGrade != nil {
		data["grade"] = int(result.GetRecallGrade())
		data["next_due"] = protoTime(result.GetRecallNextDue().AsTime())
		data["rule_applied"] = result.GetRecallRuleApplied()
	}
	return data
}

func parseOptionalGrade(raw string) (*int, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func int32PtrFromInt(value *int) *int32 {
	if value == nil {
		return nil
	}
	out := int32(*value)
	return &out
}

func protoTimestampFromFlag(raw string) (*timestamppb.Timestamp, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := clock.ParseUTCISO(raw)
	if err != nil {
		return nil, err
	}
	return timestamppb.New(value), nil
}
