package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadableNonEmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret")
	if err := os.WriteFile(path, []byte(" value\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if !readableNonEmptyFile(path) {
		t.Fatal("expected non-empty file to be present")
	}
}

func TestReadableNonEmptyFileRejectsBlankAndMissing(t *testing.T) {
	dir := t.TempDir()
	blank := filepath.Join(dir, "blank")
	if err := os.WriteFile(blank, []byte(" \n"), 0600); err != nil {
		t.Fatal(err)
	}
	if readableNonEmptyFile(blank) {
		t.Fatal("expected blank file to be rejected")
	}
	if readableNonEmptyFile(filepath.Join(dir, "missing")) {
		t.Fatal("expected missing file to be rejected")
	}
}
