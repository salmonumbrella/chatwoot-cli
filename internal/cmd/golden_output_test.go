package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func assertGolden(t *testing.T, name string, got string) {
	t.Helper()

	path := filepath.Join("testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("failed to write golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read golden file: %v", err)
	}
	if string(want) != got {
		t.Fatalf("golden output mismatch for %s (set UPDATE_GOLDEN=1 to update)", name)
	}
}

func TestGoldenLabelsListJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels", jsonResponse(200, `{
			"payload": [
				{"id": 1, "title": "Bug", "description": "Bug reports", "color": "#ff0000", "show_on_sidebar": true},
				{"id": 2, "title": "Feature", "color": "#00ff00", "show_on_sidebar": false}
			]
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "list", "-o", "json"}); err != nil {
			t.Fatalf("labels list failed: %v", err)
		}
	})

	assertGolden(t, "labels_list.json", output)
}

func TestGoldenLabelsGetJSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/api/v1/accounts/1/labels/123", jsonResponse(200, `{
			"id": 123,
			"title": "Important",
			"description": "Important issues",
			"color": "#0000ff",
			"show_on_sidebar": true
		}`))

	setupTestEnvWithHandler(t, handler)

	output := captureStdout(t, func() {
		if err := Execute(context.Background(), []string{"labels", "get", "123", "-o", "json"}); err != nil {
			t.Fatalf("labels get failed: %v", err)
		}
	})

	assertGolden(t, "labels_get.json", output)
}
