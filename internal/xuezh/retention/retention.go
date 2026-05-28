package retention

import (
	"os"
	"path/filepath"
	"sort"
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

func LoadConfig() Config {
	return Config{
		ArtifactsDays: defaults["artifacts"],
		BackupsDays:   defaults["backups"],
		ExportsDays:   defaults["exports"],
		CacheDays:     defaults["cache"],
	}
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
