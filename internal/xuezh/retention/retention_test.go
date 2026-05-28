package retention

import "testing"

func TestLoadConfigIgnoresOldEnvOverrides(t *testing.T) {
	t.Setenv("XUEZH_RETENTION_ARTIFACTS_DAYS", "1")
	t.Setenv("XUEZH_RETENTION_BACKUPS_DAYS", "1")
	t.Setenv("XUEZH_RETENTION_EXPORTS_DAYS", "1")
	t.Setenv("XUEZH_RETENTION_CACHE_DAYS", "1")

	cfg := LoadConfig()
	if cfg.ArtifactsDays != defaults["artifacts"] ||
		cfg.BackupsDays != defaults["backups"] ||
		cfg.ExportsDays != defaults["exports"] ||
		cfg.CacheDays != defaults["cache"] {
		t.Fatalf("retention config used env overrides: %+v", cfg)
	}
}
