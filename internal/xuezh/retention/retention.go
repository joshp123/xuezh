package retention

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/joshp123/xuezh/internal/xuezh/paths"
)

type Config struct {
	ArtifactsDays int
	BackupsDays   int
	ExportsDays   int
	CacheDays     int
}

var defaults = map[string]int{
	"artifacts": 90,
	"backups":   30,
	"exports":   180,
	"cache":     180,
}

var envKeys = map[string]string{
	"artifacts": "XUEZH_RETENTION_ARTIFACTS_DAYS",
	"backups":   "XUEZH_RETENTION_BACKUPS_DAYS",
	"exports":   "XUEZH_RETENTION_EXPORTS_DAYS",
	"cache":     "XUEZH_RETENTION_CACHE_DAYS",
}

func LoadConfig() Config {
	return Config{
		ArtifactsDays: loadDays("artifacts"),
		BackupsDays:   loadDays("backups"),
		ExportsDays:   loadDays("exports"),
		CacheDays:     loadDays("cache"),
	}
}

func loadDays(key string) int {
	envKey := envKeys[key]
	if raw := os.Getenv(envKey); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			return value
		}
	}
	return defaults[key]
}

func CollectGCCandidates(root string, now time.Time) ([]string, error) {
	config := LoadConfig()
	windows := map[string]int{
		"artifacts": config.ArtifactsDays,
		"backups":   config.BackupsDays,
		"exports":   config.ExportsDays,
		"cache":     config.CacheDays,
	}

	candidates := []string{}
	for subdir, days := range windows {
		cutoff := now.Add(-time.Duration(days) * 24 * time.Hour)
		files, err := filesUnder(root, subdir)
		if err != nil {
			return nil, err
		}
		for _, path := range files {
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			if info.ModTime().UTC().Before(cutoff) {
				resolved, err := paths.ResolveInWorkspace(path)
				if err != nil {
					continue
				}
				candidates = append(candidates, resolved)
			}
		}
	}
	return sortByRelative(root, candidates), nil
}

func filesUnder(root, subdir string) ([]string, error) {
	base := filepath.Join(root, subdir)
	info, err := os.Stat(base)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return []string{}, nil
	}
	pathsOut := []string{}
	err = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		pathsOut = append(pathsOut, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(pathsOut)
	return pathsOut, nil
}

func sortByRelative(root string, candidates []string) []string {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return candidates
	}
	withRel := make([]struct {
		abs string
		rel string
	}, 0, len(candidates))
	for _, path := range candidates {
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			rel = path
		}
		withRel = append(withRel, struct {
			abs string
			rel string
		}{abs: path, rel: rel})
	}
	sort.Slice(withRel, func(i, j int) bool {
		return withRel[i].rel < withRel[j].rel
	})
	out := make([]string, 0, len(withRel))
	for _, item := range withRel {
		out = append(out, item.abs)
	}
	return out
}
