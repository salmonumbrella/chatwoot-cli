package cache_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/cache"
)

func TestStore_PutAndGet(t *testing.T) {
	dir := t.TempDir()
	s := cache.NewStore(dir, "inboxes", "https://example.com", 1)

	type item struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	items := []item{{ID: 1, Name: "Support"}, {ID: 2, Name: "Sales"}}
	s.Put(items)

	var got []item
	ok := s.Get(&got)
	if !ok {
		t.Fatal("expected cache hit")
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got))
	}
	if got[0].Name != "Support" || got[1].Name != "Sales" {
		t.Fatalf("unexpected items: %+v", got)
	}
}

func TestStore_ExpiredTTL(t *testing.T) {
	dir := t.TempDir()
	s := cache.NewStoreWithTTL(dir, "inboxes", "https://example.com", 1, 1*time.Millisecond)

	s.Put([]string{"a"})
	time.Sleep(5 * time.Millisecond)

	var got []string
	ok := s.Get(&got)
	if ok {
		t.Fatal("expected cache miss after TTL expiry")
	}
}

func TestStore_MissOnEmpty(t *testing.T) {
	dir := t.TempDir()
	s := cache.NewStore(dir, "inboxes", "https://example.com", 1)

	var got []string
	ok := s.Get(&got)
	if ok {
		t.Fatal("expected cache miss on empty store")
	}
}

func TestStore_Clear(t *testing.T) {
	dir := t.TempDir()
	s := cache.NewStore(dir, "inboxes", "https://example.com", 1)

	s.Put([]string{"a"})
	s.Clear()

	var got []string
	ok := s.Get(&got)
	if ok {
		t.Fatal("expected cache miss after clear")
	}
}

func TestStore_DifferentAccounts(t *testing.T) {
	dir := t.TempDir()
	s1 := cache.NewStore(dir, "inboxes", "https://example.com", 1)
	s2 := cache.NewStore(dir, "inboxes", "https://example.com", 2)

	s1.Put([]string{"account1"})
	s2.Put([]string{"account2"})

	var got1, got2 []string
	s1.Get(&got1)
	s2.Get(&got2)

	if got1[0] != "account1" || got2[0] != "account2" {
		t.Fatal("accounts should have separate caches")
	}
}

func TestClearAll(t *testing.T) {
	dir := t.TempDir()
	s1 := cache.NewStore(dir, "inboxes", "https://example.com", 1)
	s2 := cache.NewStore(dir, "labels", "https://example.com", 1)

	s1.Put([]string{"a"})
	s2.Put([]string{"b"})

	cache.ClearAll(dir)

	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	if len(files) != 0 {
		t.Fatalf("expected no cache files after ClearAll, got %d", len(files))
	}
}

func TestStore_DisabledByEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CHATWOOT_NO_CACHE", "1")

	s := cache.NewStore(dir, "inboxes", "https://example.com", 1)
	s.Put([]string{"a"})

	var got []string
	ok := s.Get(&got)
	if ok {
		t.Fatal("expected cache miss when disabled via env")
	}

	// Verify no file was written
	files, _ := os.ReadDir(dir)
	if len(files) != 0 {
		t.Fatal("expected no files written when cache disabled")
	}
}
