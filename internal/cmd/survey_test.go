package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
)

func TestSurveyGetCommand(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/survey/responses/uuid-123", jsonResponse(200, `{
			"rating": 5,
			"feedback_message": "Great service!"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"survey", "get", "uuid-123"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("survey get failed: %v", err)
	}

	if !strings.Contains(output, "Rating: 5") {
		t.Errorf("output missing rating: %s", output)
	}
	if !strings.Contains(output, "Great service!") {
		t.Errorf("output missing feedback message: %s", output)
	}
}

func TestSurveyGetCommand_JSON(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/survey/responses/uuid-123", jsonResponse(200, `{
			"rating": 5,
			"feedback_message": "Great service!"
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"survey", "get", "uuid-123", "-o", "json"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("survey get failed: %v", err)
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(output), &response); err != nil {
		t.Errorf("output is not valid JSON: %v, output: %s", err, output)
	}

	if response["rating"] != float64(5) {
		t.Errorf("expected rating 5, got %v", response["rating"])
	}
}

func TestSurveyGetCommand_NoFeedback(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/survey/responses/uuid-456", jsonResponse(200, `{
			"rating": 3,
			"feedback_message": ""
		}`))

	setupTestEnvWithHandler(t, handler)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Execute(context.Background(), []string{"survey", "get", "uuid-456"})

	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if err != nil {
		t.Errorf("survey get failed: %v", err)
	}

	if !strings.Contains(output, "Rating: 3") {
		t.Errorf("output missing rating: %s", output)
	}
	// Should NOT show "Feedback:" when feedback_message is empty
	if strings.Contains(output, "Feedback:") {
		t.Errorf("output should not show Feedback when empty: %s", output)
	}
}

func TestSurveyGetCommand_APIError(t *testing.T) {
	handler := newRouteHandler().
		On("GET", "/survey/responses/uuid-404", jsonResponse(404, `{"error": "Not found"}`))

	setupTestEnvWithHandler(t, handler)

	err := Execute(context.Background(), []string{"survey", "get", "uuid-404"})
	if err == nil {
		t.Error("expected error for API failure")
	}
}
