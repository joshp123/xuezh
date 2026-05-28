package audio

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/jsonio"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

func artifactFor(path string, format string, purpose string) (envelope.Artifact, error) {
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return envelope.Artifact{}, err
	}
	rel, err := relativeTo(workspace, path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	mime, err := mimeForFormat(format)
	if err != nil {
		return envelope.Artifact{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	return envelope.Artifact{Path: rel, MIME: mime, Purpose: purpose, Bytes: intPtr(int(info.Size()))}, nil
}

func artifactPath(prefix, ext string, now time.Time) (string, error) {
	root, err := paths.EnsureWorkspace()
	if err != nil {
		return "", err
	}
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	if err := os.MkdirAll(dayPath, 0o755); err != nil {
		return "", err
	}
	filename := fmt.Sprintf("%s-%s.%s", prefix, now.UTC().Format("20060102T150405Z"), ext)
	return filepath.Join(dayPath, filename), nil
}

func writeJSONArtifact(payload map[string]any, purpose, prefix string) (envelope.Artifact, error) {
	now, err := clock.NowUTC()
	if err != nil {
		return envelope.Artifact{}, err
	}
	path, err := artifactPath(prefix, "json", now)
	if err != nil {
		return envelope.Artifact{}, err
	}
	content, err := jsonio.Dumps(payload)
	if err != nil {
		return envelope.Artifact{}, err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return envelope.Artifact{}, err
	}
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return envelope.Artifact{}, err
	}
	rel, err := relativeTo(workspace, path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	return envelope.Artifact{Path: rel, MIME: "application/json", Purpose: purpose, Bytes: intPtr(int(stat.Size()))}, nil
}

func mustArtifactPath(prefix, ext string) string {
	now, err := clock.NowUTC()
	if err != nil {
		now = time.Now().UTC()
	}
	root, _ := paths.EnsureWorkspace()
	dayPath := filepath.Join(root, "artifacts", now.Format("2006"), now.Format("01"), now.Format("02"))
	_ = os.MkdirAll(dayPath, 0o755)
	filename := fmt.Sprintf("%s-%s.%s", prefix, now.Format("20060102T150405Z"), ext)
	return filepath.Join(dayPath, filename)
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			if strings.HasPrefix(path, "~/") {
				return filepath.Join(home, path[2:])
			}
		}
	}
	return path
}

func intPtr(value int) *int {
	return &value
}

func relativeTo(base, target string) (string, error) {
	baseClean := cleanExistingPath(base)
	targetClean := cleanExistingPath(target)
	if targetClean != baseClean && !strings.HasPrefix(targetClean, baseClean+string(filepath.Separator)) {
		return "", fmt.Errorf("'%s' is not in the subpath of '%s'", target, base)
	}
	return filepath.Rel(baseClean, targetClean)
}

func cleanExistingPath(path string) string {
	abs, err := filepath.Abs(path)
	if err == nil {
		path = abs
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	}
	return filepath.Clean(path)
}
