package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCachePathCmd(t *testing.T) {
	t.Run("nonexistent directory", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "missing-cache")
		t.Setenv("CHATWOOT_CACHE_DIR", dir)

		cmd := newCachePathCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("cache path command failed: %v", err)
		}
		text := out.String()
		if !strings.Contains(text, dir) {
			t.Fatalf("expected output to include cache dir %q, got %q", dir, text)
		}
	})

	t.Run("lists json files", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv("CHATWOOT_CACHE_DIR", dir)

		if err := os.WriteFile(filepath.Join(dir, "inboxes_abcdef123456_1.json"), []byte(`{"ok":true}`), 0o644); err != nil {
			t.Fatalf("write json file: %v", err)
		}
		if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0o644); err != nil {
			t.Fatalf("write non-json file: %v", err)
		}

		cmd := newCachePathCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetErr(&bytes.Buffer{})
		if err := cmd.Execute(); err != nil {
			t.Fatalf("cache path command failed: %v", err)
		}

		text := out.String()
		if !strings.Contains(text, "inboxes_abcdef123456_1.json") {
			t.Fatalf("expected cache listing in output, got %q", text)
		}
		if strings.Contains(text, "notes.txt") {
			t.Fatalf("did not expect non-json file listing, got %q", text)
		}
	})
}

func TestCacheClearCmd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CHATWOOT_CACHE_DIR", dir)

	cachePath := filepath.Join(dir, "inboxes_abcdef123456_1.json")
	if err := os.WriteFile(cachePath, []byte(`{"cached":true}`), 0o644); err != nil {
		t.Fatalf("write cache file: %v", err)
	}

	cmd := newCacheClearCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("cache clear command failed: %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("expected cache file removed, stat err=%v", err)
	}
	if !strings.Contains(out.String(), "Cache cleared:") {
		t.Fatalf("expected clear confirmation output, got %q", out.String())
	}
}
