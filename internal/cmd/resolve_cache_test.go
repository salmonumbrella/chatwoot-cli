package cmd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/chatwoot/chatwoot-cli/internal/api"
)

func TestResolveInboxID_FuzzyMatch(t *testing.T) {
	t.Setenv("CHATWOOT_TESTING", "1") // allow localhost base URL

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/accounts/1/inboxes" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payload":[{"id":1,"name":"Support Inbox","channel_type":"web"},{"id":2,"name":"Sales Inbox","channel_type":"email"}]}`))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_CACHE_DIR", cacheDir)

	client := api.New(srv.URL, "test-token", 1)
	ctx := context.Background()

	id, err := resolveInboxID(ctx, client, "supp")
	if err != nil {
		t.Fatalf("expected fuzzy match, got error: %v", err)
	}
	if id != 1 {
		t.Fatalf("expected inbox ID 1, got %d", id)
	}
}

func TestResolveInboxID_UsesCache(t *testing.T) {
	t.Setenv("CHATWOOT_TESTING", "1") // allow localhost base URL

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/accounts/1/inboxes" {
			http.NotFound(w, r)
			return
		}
		callCount++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"payload":[{"id":1,"name":"Support Inbox","channel_type":"web"}]}`))
	}))
	defer srv.Close()

	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_CACHE_DIR", cacheDir)

	client := api.New(srv.URL, "test-token", 1)
	ctx := context.Background()

	_, _ = resolveInboxID(ctx, client, "support") // populate cache
	_, _ = resolveInboxID(ctx, client, "support") // should use cache

	if callCount != 1 {
		t.Fatalf("expected 1 API call (cached), got %d", callCount)
	}
}

func TestResolveInboxID_NumericIDSkipsCache(t *testing.T) {
	cacheDir := t.TempDir()
	t.Setenv("CHATWOOT_CACHE_DIR", cacheDir)

	client := api.New("https://example.com", "test-token", 1)
	ctx := context.Background()

	id, err := resolveInboxID(ctx, client, "42")
	if err != nil {
		t.Fatal(err)
	}
	if id != 42 {
		t.Fatalf("expected 42, got %d", id)
	}

	files, _ := os.ReadDir(cacheDir)
	if len(files) != 0 {
		t.Fatal("expected no cache files for numeric ID")
	}
}
