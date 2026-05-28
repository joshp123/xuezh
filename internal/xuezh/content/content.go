package content

import (
	"database/sql"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/joshp123/xuezh/internal/xuezh/clock"
	"github.com/joshp123/xuezh/internal/xuezh/db"
	"github.com/joshp123/xuezh/internal/xuezh/envelope"
	"github.com/joshp123/xuezh/internal/xuezh/ids"
	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

var allowedContentTypes = map[string]struct{}{
	"story":    {},
	"dialogue": {},
	"exercise": {},
}

type ContentResult struct {
	Data      map[string]any
	Artifacts []envelope.Artifact
}

func contentDestination(contentType, key, suffix string) (string, error) {
	rel := filepath.Join("cache", "content", contentType, key+suffix)
	return paths.ResolveInWorkspace(rel)
}

func artifactFor(path string) (envelope.Artifact, error) {
	workspace, err := paths.EnsureWorkspace()
	if err != nil {
		return envelope.Artifact{}, err
	}
	rel, err := relativeTo(workspace, path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	mimeType := mime.TypeByExtension(filepath.Ext(path))
	if mimeType == "" {
		mimeType = "text/plain"
	}
	if idx := strings.Index(mimeType, ";"); idx != -1 {
		mimeType = strings.TrimSpace(mimeType[:idx])
	}
	stat, err := os.Stat(path)
	if err != nil {
		return envelope.Artifact{}, err
	}
	return envelope.Artifact{Path: rel, MIME: mimeType, Purpose: "cached_content", Bytes: intPtr(int(stat.Size()))}, nil
}

func PutContentBytes(contentType, key, filename string, data []byte) (ContentResult, error) {
	if _, ok := allowedContentTypes[contentType]; !ok {
		return ContentResult{}, errors.New("Unsupported content type: " + contentType)
	}
	suffix := filepath.Ext(filename)
	if suffix == "" {
		suffix = ".txt"
	}
	destPath, err := contentDestination(contentType, key, suffix)
	if err != nil {
		return ContentResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return ContentResult{}, err
	}
	contentID := ids.ContentID(contentType, key)
	dbPath, err := db.InitDB()
	if err != nil {
		return ContentResult{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return ContentResult{}, err
	}
	defer conn.Close()
	var storedRel string
	row := conn.QueryRow("SELECT path FROM generated_content WHERE id = ?", contentID)
	switch err := row.Scan(&storedRel); err {
	case nil:
		resolved, err := paths.ResolveInWorkspace(storedRel)
		if err != nil {
			return ContentResult{}, err
		}
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			if err := writeFile(resolved, data); err != nil {
				return ContentResult{}, err
			}
		}
		destPath = resolved
	case sql.ErrNoRows:
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			if err := writeFile(destPath, data); err != nil {
				return ContentResult{}, err
			}
		}
		workspace, err := paths.EnsureWorkspace()
		if err != nil {
			return ContentResult{}, err
		}
		relPath, err := relativeTo(workspace, destPath)
		if err != nil {
			return ContentResult{}, err
		}
		now, err := clock.NowUTC()
		if err != nil {
			return ContentResult{}, err
		}
		_, err = conn.Exec(
			`INSERT INTO generated_content (id, content_type, content_key, path, created_at)
			 VALUES (?, ?, ?, ?, ?)`,
			contentID, contentType, key, relPath, clock.FormatISO(now),
		)
		if err != nil {
			return ContentResult{}, err
		}
	default:
		return ContentResult{}, err
	}
	artifact, err := artifactFor(destPath)
	if err != nil {
		return ContentResult{}, err
	}
	payload := map[string]any{"type": contentType, "key": key, "content_id": contentID}
	return ContentResult{Data: payload, Artifacts: []envelope.Artifact{artifact}}, nil
}

func GetContent(contentType, key string) (ContentResult, error) {
	if _, ok := allowedContentTypes[contentType]; !ok {
		return ContentResult{}, errors.New("Unsupported content type: " + contentType)
	}
	contentID := ids.ContentID(contentType, key)
	dbPath, err := db.InitDB()
	if err != nil {
		return ContentResult{}, err
	}
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return ContentResult{}, err
	}
	defer conn.Close()
	var storedRel string
	row := conn.QueryRow("SELECT path FROM generated_content WHERE id = ?", contentID)
	if err := row.Scan(&storedRel); err != nil {
		if err == sql.ErrNoRows {
			return ContentResult{}, errors.New("Content not found for key: " + key)
		}
		return ContentResult{}, err
	}
	resolved, err := paths.ResolveInWorkspace(storedRel)
	if err != nil {
		return ContentResult{}, err
	}
	if _, err := os.Stat(resolved); err != nil {
		return ContentResult{}, errors.New("Cached content missing on disk: " + resolved)
	}
	artifact, err := artifactFor(resolved)
	if err != nil {
		return ContentResult{}, err
	}
	data := map[string]any{"type": contentType, "key": key, "content_id": contentID}
	return ContentResult{Data: data, Artifacts: []envelope.Artifact{artifact}}, nil
}

func GetContentBytes(contentType, key string) (ContentResult, []byte, error) {
	result, err := GetContent(contentType, key)
	if err != nil {
		return ContentResult{}, nil, err
	}
	if len(result.Artifacts) == 0 {
		return ContentResult{}, nil, errors.New("Cached content has no artifact")
	}
	path, err := paths.ResolveInWorkspace(result.Artifacts[0].Path)
	if err != nil {
		return ContentResult{}, nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ContentResult{}, nil, err
	}
	return result, data, nil
}

func writeFile(path string, data []byte) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()
	if _, err := out.Write(data); err != nil {
		return err
	}
	return out.Sync()
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

func intPtr(value int) *int {
	return &value
}
