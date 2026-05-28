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

func runClientDoctor(args []string, serverURL string) int {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	_ = fs.Bool("json", true, "Output JSON envelope")
	if err := fs.Parse(args); err != nil {
		return 1
	}

	checks := []map[string]any{
		{
			"name":    "client.mode",
			"ok":      true,
			"details": map[string]any{"mode": "client-backed", "server_url": serverURL},
		},
	}
	client := xuezhv1connect.NewXuezhServiceClient(http.DefaultClient, serverURL)
	resp, err := client.Doctor(context.Background(), connect.NewRequest(&xuezhv1.DoctorRequest{}))
	if err != nil {
		checks = append(checks, map[string]any{
			"name":    "server.reachable",
			"ok":      false,
			"details": map[string]any{"server_url": serverURL, "error": err.Error()},
		})
		out := envelope.OK("doctor", map[string]any{"checks": checks}, nil, false, nil)
		return emit(out)
	}

	checks = append(checks, map[string]any{
		"name":    "server.reachable",
		"ok":      true,
		"details": map[string]any{"server_url": serverURL},
	})
	checks = append(checks, doctorCheckProtoData(resp.Msg.GetChecks())...)
	out := envelope.OK("doctor", map[string]any{
		"server_version":        resp.Msg.GetServerVersion(),
		"server_workspace_role": resp.Msg.GetWorkspaceRole(),
		"server_workspace_path": resp.Msg.GetWorkspacePath(),
		"checks":                checks,
	}, nil, false, nil)
	return emit(out)
}

func doctorCheckProtoData(checks []*xuezhv1.DoctorCheck) []map[string]any {
	result := make([]map[string]any, 0, len(checks))
	for _, check := range checks {
		result = append(result, map[string]any{
			"name":    check.GetName(),
			"ok":      check.GetOk(),
			"details": check.GetDetails().AsMap(),
		})
	}
	return result
}
