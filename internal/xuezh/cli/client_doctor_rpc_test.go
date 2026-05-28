package cli

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
)

func TestClientBackedDoctorUsesRPCAndDoesNotTouchLocalWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	requests := make(chan *xuezhv1.DoctorRequest, 1)
	server := httptest.NewServer(newClientRPCMux(t, clientRPCStub{doctorRequests: requests}))
	defer server.Close()
	writeCLIUserConfig(t, "[client]\nserver_url = \""+server.URL+"\"\n")

	env := runClientCommandForTest(t, []string{"doctor", "--json"})
	if env.Command != "doctor" {
		t.Fatalf("command = %q", env.Command)
	}
	if _, ok := <-requests; !ok {
		t.Fatal("doctor RPC was not called")
	}
	if env.Data["server_workspace_role"] != "server" || env.Data["server_workspace_path"] != "/var/lib/xuezh" {
		t.Fatalf("doctor did not report remote workspace: %#v", env.Data)
	}
	if !hasDoctorCheck(env.Data["checks"], "client.mode") || !hasDoctorCheck(env.Data["checks"], "server.reachable") || !hasDoctorCheck(env.Data["checks"], "db.status") {
		t.Fatalf("doctor checks = %#v", env.Data["checks"])
	}
	if _, err := os.Stat(filepath.Join(home, ".local", "share", "xuezh")); !os.IsNotExist(err) {
		t.Fatalf("client-backed doctor touched local workspace: %v", err)
	}
}

func hasDoctorCheck(raw any, name string) bool {
	checks, ok := raw.([]any)
	if !ok {
		return false
	}
	for _, item := range checks {
		check, ok := item.(map[string]any)
		if ok && check["name"] == name {
			return true
		}
	}
	return false
}

func (s clientRPCStub) Doctor(_ context.Context, req *connect.Request[xuezhv1.DoctorRequest]) (*connect.Response[xuezhv1.DoctorResponse], error) {
	if s.doctorRequests == nil {
		return nil, connect.NewError(connect.CodeInternal, errors.New("doctor request channel missing"))
	}
	s.doctorRequests <- req.Msg
	details, err := structpb.NewStruct(map[string]any{"path": "/var/lib/xuezh", "exists": true})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&xuezhv1.DoctorResponse{
		ServerVersion: "0.1.0",
		WorkspaceRole: "server",
		WorkspacePath: "/var/lib/xuezh",
		Checks: []*xuezhv1.DoctorCheck{{
			Name:    "db.status",
			Ok:      true,
			Details: details,
		}},
	}), nil
}
