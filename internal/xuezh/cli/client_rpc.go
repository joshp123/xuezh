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

func runClientRPC(commandID string, args []string, serverURL string) int {
	switch commandID {
	case "learner.state":
		return runClientLearnerState(args[2:], serverURL)
	case "snapshot":
		return runClientSnapshot(args[1:], serverURL)
	case "review.start":
		return runClientReviewStart(args[2:], serverURL)
	case "review.grade":
		return runClientReviewGrade(args[2:], serverURL)
	case "review.bury":
		return runClientReviewBury(args[2:], serverURL)
	case "event.log":
		return runClientEventLog(args[2:], serverURL)
	case "event.list":
		return runClientEventList(args[2:], serverURL)
	case "content.cache.put":
		return runClientContentCachePut(args[3:], serverURL)
	case "content.cache.get":
		return runClientContentCacheGet(args[3:], serverURL)
	case "doctor":
		return runClientDoctor(args[1:], serverURL)
	case "audio.tts":
		return runClientAudioTTS(args[2:], serverURL)
	case "audio.process-voice":
		return runClientAudioProcessVoice(args[2:], serverURL)
	case "srs.preview":
		return runClientSRSPreview(args[2:], serverURL)
	case "report.hsk":
		return runClientReportHSK(args[2:], serverURL)
	case "report.mastery":
		return runClientReportMastery(args[2:], serverURL)
	case "report.due":
		return runClientReportDue(args[2:], serverURL)
	default:
		return emitUnsupportedClientCommand(commandID, serverURL)
	}
}

func runClientLearnerState(args []string, serverURL string) int {
	fs := flag.NewFlagSet("learner state", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.GetLearnerState(context.Background(), connect.NewRequest(&xuezhv1.GetLearnerStateRequest{}))
	if err != nil {
		return emitError("learner.state", err)
	}
	out := envelope.OK("learner.state", learnerStateProtoData(resp.Msg), nil, false, nil)
	return emit(out)
}

func learnerStateProtoData(state *xuezhv1.LearnerState) map[string]any {
	cards := make([]any, 0, len(state.GetCards()))
	for _, card := range state.GetCards() {
		row := make([]any, 0, len(card.GetValues()))
		for _, value := range card.GetValues() {
			row = append(row, value.AsInterface())
		}
		cards = append(cards, row)
	}
	return map[string]any{
		"generated_at":  protoTime(state.GetGeneratedAt().AsTime()),
		"changed_at":    protoTime(state.GetChangedAt().AsTime()),
		"state_hash":    state.GetStateHash(),
		"learned_score": int(state.GetLearnedScore()),
		"columns":       state.GetColumns(),
		"cards":         cards,
	}
}
