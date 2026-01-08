package paths

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const defaultWorkspace = ".clawdbot/workspace/xuezh"

var workspaceSubdirs = []string{"artifacts", "cache", "exports", "backups"}

func WorkspaceDir() (string, error) {
	if override := os.Getenv("XUEZH_WORKSPACE_DIR"); override != "" {
		return expandHome(override)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, defaultWorkspace), nil
}

func DBPath() (string, error) {
	if override := os.Getenv("XUEZH_DB_PATH"); override != "" {
		return expandHome(override)
	}
	root, err := WorkspaceDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "db.sqlite3"), nil
}

func EnsureWorkspace() (string, error) {
	root, err := WorkspaceDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	for _, subdir := range workspaceSubdirs {
		if err := os.MkdirAll(filepath.Join(root, subdir), 0o755); err != nil {
			return "", err
		}
	}
	return root, nil
}

func ResolveInWorkspace(path string) (string, error) {
	root, err := EnsureWorkspace()
	if err != nil {
		return "", err
	}
	if resolvedRoot, err := filepath.EvalSymlinks(root); err == nil {
		root = resolvedRoot
	}
	candidate := path
	if expanded, expErr := expandHome(candidate); expErr == nil {
		candidate = expanded
	}
	var resolved string
	if filepath.IsAbs(candidate) {
		absPath, absErr := filepath.Abs(candidate)
		if absErr != nil {
			return "", absErr
		}
		if info, statErr := os.Stat(absPath); statErr == nil && info.IsDir() {
			if realPath, err := filepath.EvalSymlinks(absPath); err == nil {
				absPath = realPath
			}
		} else {
			parent := filepath.Dir(absPath)
			base := filepath.Base(absPath)
			if realParent, err := filepath.EvalSymlinks(parent); err == nil {
				absPath = filepath.Join(realParent, base)
			}
		}
		resolved = absPath
	} else {
		resolved, err = filepath.Abs(filepath.Join(root, candidate))
	}
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == "" {
		return resolved, nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes workspace")
	}
	return resolved, nil
}

func expandHome(path string) (string, error) {
	if len(path) == 0 {
		return path, nil
	}
	if path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if len(path) == 1 {
		return home, nil
	}
	if path[1] == '/' || path[1] == '\\' {
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}
