package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatalf("DefaultDir error: %v", err)
	}
	if dir == "" || !strings.Contains(dir, "chatwoot-cli") {
		t.Fatalf("unexpected default cache dir: %q", dir)
	}
}

func TestIsCacheFilename(t *testing.T) {
	cases := map[string]bool{
		"inboxes_abcdef123456_1.json": true,
		"labels_ABCDEF123456_42.json": true,
		"_abcdef123456_1.json":        false,
		"inboxes_abcdef_1.json":       false,
		"inboxes_abcdef123456_x.json": false,
		"inboxes_abcdef123456_1.txt":  false,
		"inboxes__1.json":             false,
	}
	for name, want := range cases {
		if got := isCacheFilename(name); got != want {
			t.Fatalf("isCacheFilename(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestClearAll_RemovesOnlyCacheFiles(t *testing.T) {
	dir := t.TempDir()
	cacheFile := filepath.Join(dir, "inboxes_abcdef123456_1.json")
	keepFile := filepath.Join(dir, "README.txt")
	subdir := filepath.Join(dir, "sub")
	nestedCache := filepath.Join(subdir, "contacts_abcdef123456_1.json")

	if err := os.WriteFile(cacheFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}
	if err := os.WriteFile(keepFile, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write keep file: %v", err)
	}
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(nestedCache, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write nested file: %v", err)
	}

	ClearAll(dir)

	if _, err := os.Stat(cacheFile); !os.IsNotExist(err) {
		t.Fatalf("expected cache file removed, stat err=%v", err)
	}
	if _, err := os.Stat(keepFile); err != nil {
		t.Fatalf("expected non-cache file kept, err=%v", err)
	}
	if _, err := os.Stat(nestedCache); err != nil {
		t.Fatalf("expected nested file untouched, err=%v", err)
	}
}
