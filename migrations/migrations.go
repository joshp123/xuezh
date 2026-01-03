package migrations

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed *.sql
var migrationFS embed.FS

type Migration struct {
	Version string
	SQL     string
}

func Load() ([]Migration, error) {
	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return nil, err
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		versions = append(versions, entry.Name())
	}
	sort.Strings(versions)
	migrations := make([]Migration, 0, len(versions))
	for _, version := range versions {
		content, err := migrationFS.ReadFile(version)
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{Version: version, SQL: string(content)})
	}
	return migrations, nil
}
