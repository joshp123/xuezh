package rpc

import (
	"context"

	"connectrpc.com/connect"

	xuezhv1 "github.com/joshp123/xuezh/api/xuezh/v1"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

func (h *Handler) Doctor(ctx context.Context, req *connect.Request[xuezhv1.DoctorRequest]) (*connect.Response[xuezhv1.DoctorResponse], error) {
	result, err := h.app.Doctor("server")
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	msg, err := doctorResponseMessage(result)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(msg), nil
}

func doctorResponseMessage(result service.DoctorResult) (*xuezhv1.DoctorResponse, error) {
	checks := make([]*xuezhv1.DoctorCheck, 0, len(result.Checks))
	for _, check := range result.Checks {
		details, err := reportStruct(check.Details)
		if err != nil {
			return nil, err
		}
		checks = append(checks, &xuezhv1.DoctorCheck{Name: check.Name, Ok: check.OK, Details: details})
	}
	return &xuezhv1.DoctorResponse{
		ServerVersion: result.ServerVersion,
		WorkspaceRole: result.WorkspaceRole,
		WorkspacePath: result.WorkspacePath,
		Checks:        checks,
	}, nil
}
