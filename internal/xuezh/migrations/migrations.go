package migrations

import root "github.com/joshp123/xuezh/migrations"

type Migration = root.Migration

func Load() ([]Migration, error) {
	return root.Load()
}
