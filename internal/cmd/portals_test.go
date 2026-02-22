package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestPortalsListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals", jsonResponse(200, `{
			"payload": [
				{"id": 1, "name": "Help Center", "slug": "help", "account_id": 1},
				{"id": 2, "name": "Support", "slug": "support", "account_id": 1}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals list failed: %v", err)
	}

	if !strings.Contains(output, "Help Center") {
		t.Errorf("output missing portal name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "SLUG") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestPortalsListCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals", jsonResponse(200, `{
			"payload": [{"id": 1, "name": "Help Center", "slug": "help", "account_id": 1}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "list", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals list failed: %v", err)
	}

	_ = decodeItems(t, output)
}

func TestPortalsGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help", jsonResponse(200, `{
			"id": 1,
			"name": "Help Center",
			"slug": "help",
			"account_id": 1
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "get", "help"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals get failed: %v", err)
	}

	if !strings.Contains(output, "Help Center") {
		t.Errorf("output missing portal name: %s", output)
	}
}

func TestPortalsGetCommand_InvalidSlug(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{"portals", "get", "invalid@slug!"})
	if err == nil {
		t.Error("expected error for invalid slug")
	}
	if !strings.Contains(err.Error(), "invalid slug") {
		t.Errorf("expected 'invalid slug' error, got: %v", err)
	}
}

func TestPortalsCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/portals", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Portal", "slug": "new", "account_id": 1}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "create",
		"--name", "New Portal",
		"--slug", "new",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals create failed: %v", err)
	}

	if !strings.Contains(output, "Created portal 1: New Portal") {
		t.Errorf("expected success message, got: %s", output)
	}

	portal := receivedBody["portal"].(map[string]any)
	if portal["name"] != "New Portal" {
		t.Errorf("expected name 'New Portal', got %v", portal["name"])
	}
}

func TestPortalsCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "create",
		"--slug", "new",
	})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestPortalsCreateCommand_MissingSlug(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "create",
		"--name", "New Portal",
	})
	if err == nil {
		t.Error("expected error when slug is missing")
	}
	if !strings.Contains(err.Error(), "--slug is required") {
		t.Errorf("expected '--slug is required' error, got: %v", err)
	}
}

func TestPortalsUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/portals/help", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Portal", "slug": "help", "account_id": 1}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "update", "help",
		"--name", "Updated Portal",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals update failed: %v", err)
	}

	if !strings.Contains(output, "Updated portal 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPortalsDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/portals/help", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "delete", "help"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted portal help") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// Articles tests

func TestPortalsArticlesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles", jsonResponse(200, `[
			{"id": 1, "title": "Getting Started", "slug": "getting-started", "status": "published", "views": 100},
			{"id": 2, "title": "FAQ", "slug": "faq", "status": "draft", "views": 0}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "articles", "list", "help"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals articles list failed: %v", err)
	}

	if !strings.Contains(output, "Getting Started") {
		t.Errorf("output missing article title: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "TITLE") || !strings.Contains(output, "STATUS") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestPortalsArticlesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles/1", jsonResponse(200, `{
			"id": 1,
			"title": "Getting Started",
			"slug": "getting-started",
			"status": "published",
			"views": 100,
			"content": "# Welcome\nThis is the getting started guide."
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "articles", "get", "help", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals articles get failed: %v", err)
	}

	if !strings.Contains(output, "Getting Started") {
		t.Errorf("output missing article title: %s", output)
	}
	if !strings.Contains(output, "# Welcome") {
		t.Errorf("output missing content: %s", output)
	}
}

func TestPortalsArticlesGetCommand_AcceptsHashAndPrefixedIDs(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles/1", jsonResponse(200, `{
			"id": 1,
			"title": "Getting Started",
			"slug": "getting-started",
			"status": "published",
			"views": 100,
			"content": "# Welcome\nThis is the getting started guide."
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "articles", "get", "help", "#1"}); err != nil {
			t.Fatalf("portals articles get hash ID failed: %v", err)
		}
	})
	if !strings.Contains(output, "Getting Started") {
		t.Errorf("output missing article title: %s", output)
	}

	output2 := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"portals", "articles", "get", "help", "article:1"}); err != nil {
			t.Fatalf("portals articles get prefixed ID failed: %v", err)
		}
	})
	if !strings.Contains(output2, "Getting Started") {
		t.Errorf("output missing article title: %s", output2)
	}
}

func TestPortalsArticlesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/portals/help/articles", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "New Article", "slug": "new-article", "status": "draft"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "articles", "create", "help",
		"--title", "New Article",
		"--content", "Article content here",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals articles create failed: %v", err)
	}

	if !strings.Contains(output, "Created article 1: New Article") {
		t.Errorf("expected success message, got: %s", output)
	}

	if receivedBody["title"] != "New Article" {
		t.Errorf("expected title 'New Article', got %v", receivedBody["title"])
	}
}

func TestPortalsArticlesCreateCommand_MissingTitle(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "articles", "create", "help",
		"--content", "Some content",
	})
	if err == nil {
		t.Error("expected error when title is missing")
	}
	if !strings.Contains(err.Error(), "--title is required") {
		t.Errorf("expected '--title is required' error, got: %v", err)
	}
}

func TestPortalsArticlesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/portals/help/articles/1", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "title": "Updated Article", "slug": "updated-article"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "articles", "update", "help", "1",
		"--title", "Updated Article",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals articles update failed: %v", err)
	}

	if !strings.Contains(output, "Updated article 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPortalsArticlesUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "articles", "update", "help", "1",
	})
	if err == nil {
		t.Error("expected error when no fields specified")
	}
	if !strings.Contains(err.Error(), "at least one field must be specified") {
		t.Errorf("expected 'at least one field must be specified' error, got: %v", err)
	}
}

func TestPortalsArticlesDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/portals/help/articles/1", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "articles", "delete", "help", "1"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals articles delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted article 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPortalsArticlesReorderCommand_ArticleIDsFromStdin(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/portals/help/articles/reorder", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		})

	setupTestEnvWithHandler(t, handler)

	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = oldStdin })

	go func() {
		_, _ = w.Write([]byte("1\n2\n3\n"))
		_ = w.Close()
	}()

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"portals", "articles", "reorder", "help", "--article-ids", "@-"})
		if err != nil {
			t.Errorf("portals articles reorder failed: %v", err)
		}
	})

	if !strings.Contains(output, "Reordered 3 articles in portal help") {
		t.Errorf("expected success message, got: %s", output)
	}

	idsAny, ok := receivedBody["article_ids"].([]any)
	if !ok || len(idsAny) != 3 {
		t.Fatalf("expected article_ids length 3, got %v", receivedBody["article_ids"])
	}
}

// Search articles tests

func TestPortalsArticlesSearchCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles", func(w http.ResponseWriter, r *http.Request) {
			if q := r.URL.Query().Get("query"); q != "return policy" {
				t.Errorf("expected query 'return policy', got %q", q)
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id": 1, "title": "Return Policy", "slug": "return-policy", "status": "published", "views": 42, "content": "Our return policy allows..."}
			]`))
		})

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"portals", "articles", "search", "help", "return policy"})
		if err != nil {
			t.Fatalf("portals articles search failed: %v", err)
		}
	})

	if !strings.Contains(output, "Return Policy") {
		t.Errorf("output missing article title: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "TITLE") || !strings.Contains(output, "STATUS") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestPortalsArticlesSearchCommand_NoResults(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles", jsonResponse(200, `[]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"portals", "articles", "search", "help", "nonexistent"})
		if err != nil {
			t.Fatalf("portals articles search failed: %v", err)
		}
	})

	if !strings.Contains(output, "No articles found.") {
		t.Errorf("expected 'No articles found.' message, got: %s", output)
	}
}

func TestPortalsArticlesSearchCommand_IncludeBody(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/articles", jsonResponse(200, `[
			{"id": 1, "title": "Return Policy", "slug": "return-policy", "status": "published", "views": 42, "content": "Our return policy allows 30-day returns."}
		]`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		err := Execute(context.Background(), []string{"portals", "articles", "search", "help", "return", "--include-body"})
		if err != nil {
			t.Fatalf("portals articles search --include-body failed: %v", err)
		}
	})

	if !strings.Contains(output, "--- Article 1: Return Policy ---") {
		t.Errorf("output missing article header: %s", output)
	}
	if !strings.Contains(output, "Our return policy allows 30-day returns.") {
		t.Errorf("output missing article content: %s", output)
	}
}

// Categories tests

func TestPortalsCategoriesListCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/categories", jsonResponse(200, `[
			{"id": 1, "name": "FAQ", "slug": "faq", "position": 1},
			{"id": 2, "name": "Guides", "slug": "guides", "position": 2}
		]`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "categories", "list", "help"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals categories list failed: %v", err)
	}

	if !strings.Contains(output, "FAQ") {
		t.Errorf("output missing category name: %s", output)
	}
	if !strings.Contains(output, "ID") || !strings.Contains(output, "NAME") || !strings.Contains(output, "SLUG") {
		t.Errorf("output missing expected headers: %s", output)
	}
}

func TestPortalsCategoriesGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals/help/categories/faq", jsonResponse(200, `{
			"id": 1,
			"name": "FAQ",
			"slug": "faq",
			"position": 1,
			"description": "Frequently asked questions"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "categories", "get", "help", "faq"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals categories get failed: %v", err)
	}

	if !strings.Contains(output, "FAQ") {
		t.Errorf("output missing category name: %s", output)
	}
	if !strings.Contains(output, "Frequently asked questions") {
		t.Errorf("output missing description: %s", output)
	}
}

func TestPortalsCategoriesCreateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("POST", "/api/v1/accounts/1/portals/help/categories", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "New Category", "slug": "new-category", "position": 1}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "categories", "create", "help",
		"--name", "New Category",
		"--slug", "new-category",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals categories create failed: %v", err)
	}

	if !strings.Contains(output, "Created category 1: New Category") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPortalsCategoriesCreateCommand_MissingName(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "categories", "create", "help",
		"--slug", "new",
	})
	if err == nil {
		t.Error("expected error when name is missing")
	}
	if !strings.Contains(err.Error(), "--name is required") {
		t.Errorf("expected '--name is required' error, got: %v", err)
	}
}

func TestPortalsCategoriesUpdateCommand(t *testing.T) {
	var receivedBody map[string]any
	handler := newRouteHandler().
		On("PATCH", "/api/v1/accounts/1/portals/help/categories/faq", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewDecoder(r.Body).Decode(&receivedBody)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id": 1, "name": "Updated Category", "slug": "faq"}`))
		})

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{
		"portals", "categories", "update", "help", "faq",
		"--name", "Updated Category",
	})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals categories update failed: %v", err)
	}

	if !strings.Contains(output, "Updated category 1") {
		t.Errorf("expected success message, got: %s", output)
	}
}

func TestPortalsCategoriesUpdateCommand_NoFields(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "categories", "update", "help", "faq",
	})
	if err == nil {
		t.Error("expected error when no fields specified")
	}
	if !strings.Contains(err.Error(), "at least one field must be specified") {
		t.Errorf("expected 'at least one field must be specified' error, got: %v", err)
	}
}

func TestPortalsCategoriesDeleteCommand(t *testing.T) {
	handler := newRouteHandler().
		On("DELETE", "/api/v1/accounts/1/portals/help/categories/faq", jsonResponse(200, ``))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portals", "categories", "delete", "help", "faq"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portals categories delete failed: %v", err)
	}

	if !strings.Contains(output, "Deleted category faq") {
		t.Errorf("expected success message, got: %s", output)
	}
}

// API error test
func TestPortalsListCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals", jsonResponse(500, `{"error": "Internal Server Error"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"portals", "list"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}

// Portal alias test
func TestPortalsListCommand_PortalAlias(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/portals", jsonResponse(200, `{
			"payload": [{"id": 1, "name": "Help Center", "slug": "help", "account_id": 1}]
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"portal", "list"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("portal list failed: %v", err)
	}

	if !strings.Contains(output, "Help Center") {
		t.Errorf("output missing portal name: %s", output)
	}
}

// Test article with invalid status
func TestPortalsArticlesCreateCommand_InvalidStatus(t *testing.T) {
	t.Setenv("CHATWOOT_BASE_URL", "https://test.chatwoot.com")
	t.Setenv("CHATWOOT_API_TOKEN", "test-token")
	t.Setenv("CHATWOOT_ACCOUNT_ID", "1")

	err := Execute(context.Background(), []string{
		"portals", "articles", "create", "help",
		"--title", "Test",
		"--status", "5",
	})
	if err == nil {
		t.Error("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "invalid status") {
		t.Errorf("expected 'invalid status' error, got: %v", err)
	}
}
