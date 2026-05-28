package cli

import "github.com/joshp123/xuezh/internal/xuezh/service"

func doctorData(result service.DoctorResult) map[string]any {
	return map[string]any{
		"server_version": result.ServerVersion,
		"workspace_role": result.WorkspaceRole,
		"workspace_path": result.WorkspacePath,
		"checks":         doctorCheckData(result.Checks),
	}
}

func doctorCheckData(checks []service.DoctorCheck) []map[string]any {
	result := make([]map[string]any, 0, len(checks))
	for _, check := range checks {
		result = append(result, map[string]any{
			"name":    check.Name,
			"ok":      check.OK,
			"details": check.Details,
		})
	}
	return result
}
