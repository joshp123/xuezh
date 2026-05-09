package webserver

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFallbackEntrypointAssetUsesNewestMissingJS(t *testing.T) {
	dist := t.TempDir()
	assetDir := filepath.Join(dist, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldAsset := filepath.Join(assetDir, "index-old.js")
	newAsset := filepath.Join(assetDir, "index-new.js")
	writeTestAsset(t, oldAsset, time.Unix(10, 0))
	writeTestAsset(t, newAsset, time.Unix(20, 0))

	got, ok := fallbackEntrypointAsset(dist, "/assets/index-missing.js")
	if !ok {
		t.Fatal("expected fallback asset")
	}
	if got != newAsset {
		t.Fatalf("expected newest asset %q, got %q", newAsset, got)
	}
}

func TestFallbackEntrypointAssetLeavesExistingAssetAlone(t *testing.T) {
	dist := t.TempDir()
	assetDir := filepath.Join(dist, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestAsset(t, filepath.Join(assetDir, "index-current.css"), time.Unix(20, 0))

	if got, ok := fallbackEntrypointAsset(dist, "/assets/index-current.css"); ok {
		t.Fatalf("existing assets should be served normally, got fallback %q", got)
	}
}

func TestFallbackEntrypointAssetIgnoresNonEntrypointAssets(t *testing.T) {
	dist := t.TempDir()
	assetDir := filepath.Join(dist, "assets")
	if err := os.MkdirAll(assetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestAsset(t, filepath.Join(assetDir, "index-current.js"), time.Unix(20, 0))

	if got, ok := fallbackEntrypointAsset(dist, "/assets/DesignSystemPage-old.js"); ok {
		t.Fatalf("non-entry assets should not fallback, got %q", got)
	}
}

func writeTestAsset(t *testing.T, path string, modTime time.Time) {
	t.Helper()
	if err := os.WriteFile(path, []byte("asset"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}
