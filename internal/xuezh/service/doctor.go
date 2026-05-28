package service

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/config"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type DoctorResult struct {
	ServerVersion string
	WorkspaceRole string
	WorkspacePath string
	Checks        []DoctorCheck
}

type DoctorCheck struct {
	Name    string
	OK      bool
	Details map[string]any
}

func (App) Doctor(workspaceRole string) (DoctorResult, error) {
	if strings.TrimSpace(workspaceRole) == "" {
		workspaceRole = "local"
	}
	workspace, err := paths.WorkspaceDir()
	if err != nil {
		return DoctorResult{}, err
	}
	checks := []DoctorCheck{workspacePathCheck(workspace)}

	dbPath, err := paths.DBPath()
	if err != nil {
		return DoctorResult{}, err
	}
	checks = append(checks, dbStatusCheck(dbPath))
	for _, tool := range []string{"ffmpeg", "edge-tts", "whisper"} {
		checks = append(checks, toolCheck(tool))
	}
	checks = append(checks, DoctorCheck{
		Name:    "tool.azure-speech-sdk",
		OK:      true,
		Details: map[string]any{"version": "rest"},
	})
	checks = append(checks, azureSpeechConfigCheck())

	return DoctorResult{
		ServerVersion: Version,
		WorkspaceRole: workspaceRole,
		WorkspacePath: workspace,
		Checks:        checks,
	}, nil
}

func workspacePathCheck(workspace string) DoctorCheck {
	_, err := os.Stat(workspace)
	return DoctorCheck{
		Name: "workspace.path",
		OK:   true,
		Details: map[string]any{
			"path":   workspace,
			"exists": err == nil,
		},
	}
}

func dbStatusCheck(dbPath string) DoctorCheck {
	dbDetails := map[string]any{"path": dbPath, "exists": false}
	if _, err := os.Stat(dbPath); err != nil {
		return DoctorCheck{Name: "db.status", OK: false, Details: dbDetails}
	}
	dbDetails["exists"] = true
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		dbDetails["error"] = err.Error()
		return DoctorCheck{Name: "db.status", OK: false, Details: dbDetails}
	}
	defer conn.Close()
	var count int
	if err := conn.QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&count); err != nil {
		dbDetails["error"] = err.Error()
		return DoctorCheck{Name: "db.status", OK: false, Details: dbDetails}
	}
	dbDetails["schema_migrations"] = count
	return DoctorCheck{Name: "db.status", OK: true, Details: dbDetails}
}

func toolCheck(tool string) DoctorCheck {
	path, err := exec.LookPath(tool)
	ok := err == nil && path != ""
	return DoctorCheck{
		Name:    "tool." + tool,
		OK:      ok,
		Details: map[string]any{"path": path},
	}
}

func azureSpeechConfigCheck() DoctorCheck {
	configSection, ok, _ := config.GetValue("azure", "speech")
	var configKey, configRegion string
	if ok {
		if sectionMap, ok := configSection.(map[string]any); ok {
			if value, ok := sectionMap["key"].(string); ok && strings.TrimSpace(value) != "" {
				configKey = value
			}
			if value, ok := sectionMap["key_file"].(string); ok && strings.TrimSpace(value) != "" {
				keyPath := expandConfigPath(value)
				if data, err := os.ReadFile(keyPath); err == nil {
					configKey = strings.TrimSpace(string(data))
				}
			}
			if value, ok := sectionMap["region"].(string); ok && strings.TrimSpace(value) != "" {
				configRegion = value
			}
		}
	}
	configKeyPresent := strings.TrimSpace(configKey) != ""
	configRegionPresent := strings.TrimSpace(configRegion) != ""
	configPath, _ := config.ConfigPath()
	return DoctorCheck{
		Name: "azure.speech.config",
		OK:   configKeyPresent && configRegionPresent,
		Details: map[string]any{
			"config_key":    configKeyPresent,
			"config_region": configRegionPresent,
			"config_path":   configPath,
		},
	}
}

func expandConfigPath(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
