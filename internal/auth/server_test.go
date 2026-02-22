package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewSetupServer(t *testing.T) {
	t.Run("creates server with valid CSRF token", func(t *testing.T) {
		server, err := NewSetupServer("default")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		if server == nil {
			t.Fatal("NewSetupServer() returned nil server")
		}

		if server.csrfToken == "" {
			t.Error("NewSetupServer() created server with empty CSRF token")
		}

		// CSRF token should be 64 hex characters (32 bytes)
		if len(server.csrfToken) != 64 {
			t.Errorf("NewSetupServer() CSRF token length = %d, want 64", len(server.csrfToken))
		}

		if server.profile != "default" {
			t.Errorf("NewSetupServer() profile = %q, want %q", server.profile, "default")
		}

		if server.result == nil {
			t.Error("NewSetupServer() result channel is nil")
		}

		if server.shutdown == nil {
			t.Error("NewSetupServer() shutdown channel is nil")
		}
	})

	t.Run("creates unique CSRF tokens", func(t *testing.T) {
		server1, err := NewSetupServer("profile1")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		server2, err := NewSetupServer("profile2")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		if server1.csrfToken == server2.csrfToken {
			t.Error("NewSetupServer() created duplicate CSRF tokens")
		}
	})

	t.Run("accepts different profile names", func(t *testing.T) {
		profiles := []string{"default", "production", "staging", "test-profile", ""}
		for _, profile := range profiles {
			server, err := NewSetupServer(profile)
			if err != nil {
				t.Errorf("NewSetupServer(%q) error = %v", profile, err)
				continue
			}
			if server.profile != profile {
				t.Errorf("NewSetupServer(%q) profile = %q", profile, server.profile)
			}
		}
	})
}

func TestHandleSetup(t *testing.T) {
	server, err := NewSetupServer("default")
	if err != nil {
		t.Fatalf("NewSetupServer() error = %v", err)
	}

	t.Run("serves setup page on root path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		server.handleSetup(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSetup() status = %d, want %d", rec.Code, http.StatusOK)
		}

		contentType := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/html") {
			t.Errorf("handleSetup() Content-Type = %q, want text/html", contentType)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "Chatwoot CLI Setup") {
			t.Error("handleSetup() response does not contain expected title")
		}

		if !strings.Contains(body, server.csrfToken) {
			t.Error("handleSetup() response does not contain CSRF token")
		}
	})

	t.Run("returns 404 for non-root paths", func(t *testing.T) {
		paths := []string{"/other", "/setup", "/index.html", "/api"}
		for _, path := range paths {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			server.handleSetup(rec, req)

			if rec.Code != http.StatusNotFound {
				t.Errorf("handleSetup(%q) status = %d, want %d", path, rec.Code, http.StatusNotFound)
			}
		}
	})
}

func TestHandleValidate(t *testing.T) {
	t.Run("rejects non-POST methods", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/validate", nil)
			rec := httptest.NewRecorder()

			server.handleValidate(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("handleValidate() with %s status = %d, want %d", method, rec.Code, http.StatusMethodNotAllowed)
			}
		}
	})

	t.Run("rejects requests without CSRF token", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://app.chatwoot.com","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("handleValidate() without CSRF status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects requests with wrong CSRF token", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://app.chatwoot.com","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "wrong-token")
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("handleValidate() with wrong CSRF status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects invalid JSON body", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("handleValidate() with invalid JSON status = %d, want %d", rec.Code, http.StatusBadRequest)
		}

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleValidate() with invalid JSON should return success=false")
		}
	})

	t.Run("rejects localhost URLs (SSRF protection)", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"http://localhost:3000","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleValidate() status = %d, want %d", rec.Code, http.StatusOK)
		}

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleValidate() with localhost should return success=false")
		}

		errMsg, ok := response["error"].(string)
		if !ok || !strings.Contains(errMsg, "Invalid URL") {
			t.Errorf("handleValidate() error = %v, want error containing 'Invalid URL'", response["error"])
		}
	})

	t.Run("rejects private IP ranges (SSRF protection)", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		privateURLs := []string{
			"http://192.168.1.1",
			"http://10.0.0.1",
			"http://172.16.0.1",
		}

		for _, url := range privateURLs {
			body := `{"base_url":"` + url + `","api_token":"token","account_id":1}`
			req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", server.csrfToken)
			rec := httptest.NewRecorder()

			server.handleValidate(rec, req)

			var response map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["success"] != false {
				t.Errorf("handleValidate() with %s should return success=false", url)
			}
		}
	})

	t.Run("normalizes URL by removing trailing slash", func(t *testing.T) {
		server, _ := NewSetupServer("default")

		urlsToTest := []string{
			"http://localhost:8080/",
			"http://localhost:8080",
		}

		for _, url := range urlsToTest {
			body := `{"base_url":"` + url + `","api_token":"token","account_id":1}`
			req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", server.csrfToken)
			rec := httptest.NewRecorder()

			server.handleValidate(rec, req)

			var response map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["success"] != false {
				t.Errorf("URL %q should fail SSRF validation", url)
			}
		}
	})
}

func TestHandleSubmit(t *testing.T) {
	t.Run("rejects non-POST methods", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

		for _, method := range methods {
			req := httptest.NewRequest(method, "/submit", nil)
			rec := httptest.NewRecorder()

			server.handleSubmit(rec, req)

			if rec.Code != http.StatusMethodNotAllowed {
				t.Errorf("handleSubmit() with %s status = %d, want %d", method, rec.Code, http.StatusMethodNotAllowed)
			}
		}
	})

	t.Run("rejects requests without CSRF token", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://app.chatwoot.com","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("handleSubmit() without CSRF status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects requests with wrong CSRF token", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://app.chatwoot.com","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "wrong-token")
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		if rec.Code != http.StatusForbidden {
			t.Errorf("handleSubmit() with wrong CSRF status = %d, want %d", rec.Code, http.StatusForbidden)
		}
	})

	t.Run("rejects invalid JSON body", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader("not json"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("handleSubmit() with invalid JSON status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("rejects localhost URLs (SSRF protection)", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"http://127.0.0.1","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleSubmit() with localhost should return success=false")
		}
	})
}

func TestHandleSuccess(t *testing.T) {
	server, _ := NewSetupServer("default")

	t.Run("serves success page", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
		}

		contentType := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(contentType, "text/html") {
			t.Errorf("handleSuccess() Content-Type = %q, want text/html", contentType)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "You're all set") {
			t.Error("handleSuccess() response does not contain expected content")
		}
	})

	t.Run("includes user info from query params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/success?name=John+Doe&email=john@example.com", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		body := rec.Body.String()
		if !strings.Contains(body, "john@example.com") {
			t.Error("handleSuccess() response does not contain user email")
		}
	})

	t.Run("handles missing query params", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/success", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSuccess() without params status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("escapes HTML in user input", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/success?name=<script>alert(1)</script>&email=test@test.com", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		body := rec.Body.String()
		if strings.Contains(body, "<script>alert(1)</script>") {
			t.Error("handleSuccess() should escape HTML in user input")
		}
	})
}

func TestHandleComplete(t *testing.T) {
	t.Run("signals completion and closes shutdown channel", func(t *testing.T) {
		server, _ := NewSetupServer("default")

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		rec := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			server.handleComplete(rec, req)
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("handleComplete() timed out")
		}

		if rec.Code != http.StatusOK {
			t.Errorf("handleComplete() status = %d, want %d", rec.Code, http.StatusOK)
		}

		select {
		case <-server.shutdown:
		default:
			t.Error("handleComplete() did not close shutdown channel")
		}
	})

	t.Run("sends pending result through channel", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		server.pendingResult = &SetupResult{
			Email: "test@example.com",
		}

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		rec := httptest.NewRecorder()

		go server.handleComplete(rec, req)

		select {
		case result := <-server.result:
			if result.Email != "test@example.com" {
				t.Errorf("handleComplete() result email = %q, want %q", result.Email, "test@example.com")
			}
		case <-time.After(time.Second):
			t.Fatal("handleComplete() did not send result")
		}
	})

	t.Run("closes shutdown even without pending result", func(t *testing.T) {
		server, _ := NewSetupServer("default")

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		rec := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			server.handleComplete(rec, req)
			close(done)
		}()

		select {
		case <-done:
			if rec.Code != http.StatusOK {
				t.Errorf("handleComplete() status = %d, want %d", rec.Code, http.StatusOK)
			}
		case <-time.After(time.Second):
			t.Fatal("handleComplete() timed out")
		}

		select {
		case <-server.shutdown:
		default:
			t.Error("shutdown channel should be closed")
		}
	})
}

func TestWriteJSON(t *testing.T) {
	t.Run("writes JSON with correct content type", func(t *testing.T) {
		rec := httptest.NewRecorder()
		data := map[string]any{"key": "value", "number": 42}

		writeJSON(rec, http.StatusOK, data)

		if rec.Code != http.StatusOK {
			t.Errorf("writeJSON() status = %d, want %d", rec.Code, http.StatusOK)
		}

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("writeJSON() Content-Type = %q, want application/json", contentType)
		}

		var result map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("writeJSON() produced invalid JSON: %v", err)
		}

		if result["key"] != "value" {
			t.Errorf("writeJSON() key = %v, want value", result["key"])
		}
	})

	t.Run("writes different status codes", func(t *testing.T) {
		statusCodes := []int{
			http.StatusOK,
			http.StatusCreated,
			http.StatusBadRequest,
			http.StatusInternalServerError,
		}

		for _, code := range statusCodes {
			rec := httptest.NewRecorder()
			writeJSON(rec, code, map[string]string{"status": "test"})

			if rec.Code != code {
				t.Errorf("writeJSON() status = %d, want %d", rec.Code, code)
			}
		}
	})

	t.Run("handles complex data structures", func(t *testing.T) {
		rec := httptest.NewRecorder()
		data := map[string]any{
			"success": true,
			"user": map[string]any{
				"id":    1,
				"name":  "Test User",
				"email": "test@example.com",
			},
			"tags": []string{"admin", "user"},
		}

		writeJSON(rec, http.StatusOK, data)

		var result map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
			t.Fatalf("writeJSON() produced invalid JSON: %v", err)
		}

		if result["success"] != true {
			t.Error("writeJSON() did not preserve success field")
		}

		user, ok := result["user"].(map[string]any)
		if !ok {
			t.Fatal("writeJSON() did not preserve nested user object")
		}
		if user["name"] != "Test User" {
			t.Errorf("writeJSON() user name = %v, want Test User", user["name"])
		}
	})
}

func TestOpenBrowser(t *testing.T) {
	t.Run("handles valid URL without panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("openBrowser() panicked: %v", r)
			}
		}()

		_ = openBrowser("http://localhost:12345")
	})
}

func TestSetupServerIntegration(t *testing.T) {
	t.Run("full HTTP handler setup", func(t *testing.T) {
		server, err := NewSetupServer("test-profile")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/", server.handleSetup)
		mux.HandleFunc("/validate", server.handleValidate)
		mux.HandleFunc("/submit", server.handleSubmit)
		mux.HandleFunc("/success", server.handleSuccess)
		mux.HandleFunc("/complete", server.handleComplete)

		ts := httptest.NewServer(mux)
		defer ts.Close()

		resp, err := http.Get(ts.URL + "/")
		if err != nil {
			t.Fatalf("GET / error: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET / status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		_ = resp.Body.Close()

		body := bytes.NewBufferString(`{"base_url":"https://app.chatwoot.com","api_token":"test","account_id":1}`)
		req, _ := http.NewRequest(http.MethodPost, ts.URL+"/validate", body)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("POST /validate error: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("POST /validate status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		_ = resp.Body.Close()

		resp, err = http.Get(ts.URL + "/success")
		if err != nil {
			t.Fatalf("GET /success error: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("GET /success status = %d, want %d", resp.StatusCode, http.StatusOK)
		}
		_ = resp.Body.Close()
	})
}

func TestStartContextCancellation(t *testing.T) {
	t.Run("respects context cancellation", func(t *testing.T) {
		server, err := NewSetupServer("default")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		var result *SetupResult
		var startErr error

		go func() {
			result, startErr = server.Start(ctx)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)
		cancel()

		select {
		case <-done:
			if startErr != context.Canceled {
				t.Errorf("Start() error = %v, want context.Canceled", startErr)
			}
			if result != nil {
				t.Error("Start() returned non-nil result after cancellation")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Start() did not respect context cancellation")
		}
	})
}

func TestStartServerLifecycle(t *testing.T) {
	t.Run("result channel receives setup result", func(t *testing.T) {
		server, err := NewSetupServer("result-test")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		ctx := context.Background()

		done := make(chan struct{})
		var startResult *SetupResult

		go func() {
			startResult, _ = server.Start(ctx)
			close(done)
		}()

		time.Sleep(200 * time.Millisecond)

		server.result <- SetupResult{
			Email: "direct@test.com",
		}

		select {
		case <-done:
			if startResult == nil {
				t.Error("Start() returned nil result")
			} else if startResult.Email != "direct@test.com" {
				t.Errorf("Start() result email = %q, want direct@test.com", startResult.Email)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Start() did not return after result sent")
		}
	})

	t.Run("shutdown channel triggers graceful shutdown with pending result", func(t *testing.T) {
		server, err := NewSetupServer("shutdown-test")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		server.pendingResult = &SetupResult{
			Email: "pending@test.com",
		}

		ctx := context.Background()

		done := make(chan struct{})
		var startResult *SetupResult
		var startErr error

		go func() {
			startResult, startErr = server.Start(ctx)
			close(done)
		}()

		time.Sleep(200 * time.Millisecond)

		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		rec := httptest.NewRecorder()
		go server.handleComplete(rec, req)

		select {
		case <-done:
			if startErr != nil {
				t.Errorf("Start() returned error: %v", startErr)
			}
			if startResult == nil {
				t.Error("Start() returned nil result")
			} else if startResult.Email != "pending@test.com" {
				t.Errorf("Start() result email = %q, want pending@test.com", startResult.Email)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Start() did not return after shutdown signal")
		}
	})

	t.Run("shutdown without pending result returns error", func(t *testing.T) {
		server, err := NewSetupServer("shutdown-no-result")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		ctx := context.Background()

		done := make(chan struct{})
		var startResult *SetupResult
		var startErr error

		go func() {
			startResult, startErr = server.Start(ctx)
			close(done)
		}()

		time.Sleep(200 * time.Millisecond)

		close(server.shutdown)

		select {
		case <-done:
			if startErr == nil {
				t.Error("Start() should return error when shutdown without pending result")
			}
			if startResult != nil {
				t.Error("Start() should return nil result when shutdown without pending result")
			}
			if startErr != nil && !strings.Contains(startErr.Error(), "setup cancelled") {
				t.Errorf("Start() error = %v, want 'setup cancelled'", startErr)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Start() did not return after shutdown signal")
		}
	})
}

func TestSetupResult(t *testing.T) {
	t.Run("SetupResult fields", func(t *testing.T) {
		result := SetupResult{
			Email: "test@example.com",
			Error: nil,
		}

		if result.Email != "test@example.com" {
			t.Errorf("SetupResult.Email = %q, want test@example.com", result.Email)
		}

		if result.Error != nil {
			t.Errorf("SetupResult.Error = %v, want nil", result.Error)
		}
	})
}

func TestCSRFProtection(t *testing.T) {
	server, _ := NewSetupServer("default")

	endpoints := []struct {
		path    string
		handler http.HandlerFunc
	}{
		{"/validate", server.handleValidate},
		{"/submit", server.handleSubmit},
	}

	for _, ep := range endpoints {
		t.Run(ep.path+" requires CSRF token", func(t *testing.T) {
			body := `{"base_url":"https://example.com","api_token":"token","account_id":1}`

			req := httptest.NewRequest(http.MethodPost, ep.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()

			ep.handler(rec, req)

			if rec.Code != http.StatusForbidden {
				t.Errorf("%s without CSRF status = %d, want %d", ep.path, rec.Code, http.StatusForbidden)
			}
		})

		t.Run(ep.path+" accepts valid CSRF token", func(t *testing.T) {
			body := `{"base_url":"https://example.com","api_token":"token","account_id":1}`

			req := httptest.NewRequest(http.MethodPost, ep.path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", server.csrfToken)
			rec := httptest.NewRecorder()

			ep.handler(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Errorf("%s with valid CSRF unexpectedly returned 403", ep.path)
			}
		})
	}
}

func TestValidateSSRFProtection(t *testing.T) {
	server, _ := NewSetupServer("default")

	ssrfURLs := []struct {
		url         string
		description string
	}{
		{"http://localhost", "localhost"},
		{"http://127.0.0.1", "IPv4 loopback"},
		{"http://[::1]", "IPv6 loopback"},
		{"http://0.0.0.0", "unspecified IPv4"},
		{"http://10.0.0.1", "private 10.x.x.x"},
		{"http://172.16.0.1", "private 172.16.x.x"},
		{"http://192.168.1.1", "private 192.168.x.x"},
		{"http://169.254.169.254", "AWS metadata"},
		{"http://metadata.google.internal", "GCP metadata"},
	}

	for _, tc := range ssrfURLs {
		t.Run("blocks "+tc.description, func(t *testing.T) {
			body := `{"base_url":"` + tc.url + `","api_token":"token","account_id":1}`
			req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-CSRF-Token", server.csrfToken)
			rec := httptest.NewRecorder()

			server.handleValidate(rec, req)

			var response map[string]any
			if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response["success"] != false {
				t.Errorf("URL %s should be blocked", tc.url)
			}
		})
	}
}

func TestSubmitSSRFProtection(t *testing.T) {
	server, _ := NewSetupServer("default")

	t.Run("rejects cloud metadata endpoints", func(t *testing.T) {
		body := `{"base_url":"http://169.254.169.254","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleSubmit() should block cloud metadata endpoints")
		}

		errMsg, ok := response["error"].(string)
		if !ok || !strings.Contains(errMsg, "cloud metadata") {
			t.Errorf("handleSubmit() error = %v, want error about cloud metadata", response["error"])
		}
	})
}

func TestValidateWithUnreachableHost(t *testing.T) {
	t.Run("handles connection failure to unreachable host", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://this-domain-definitely-does-not-exist-12345.com","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleValidate() with unreachable host should return success=false")
		}

		errMsg, ok := response["error"].(string)
		if !ok || !strings.Contains(errMsg, "Connection failed") {
			t.Errorf("handleValidate() error = %v, want error containing 'Connection failed'", response["error"])
		}
	})

	t.Run("validates URL passes SSRF check before API call", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"https://app.chatwoot.com","api_token":"invalid-token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleValidate() with invalid token should return success=false")
		}

		errMsg, ok := response["error"].(string)
		if !ok {
			t.Fatalf("handleValidate() error is not a string: %v", response["error"])
		}

		if strings.Contains(errMsg, "Invalid URL") {
			t.Error("handleValidate() should not reject valid public URL")
		}
	})
}

func TestSubmitWithUnreachableHost(t *testing.T) {
	t.Run("handles API connection failure", func(t *testing.T) {
		server, _ := NewSetupServer("test")
		body := `{"base_url":"https://this-domain-definitely-does-not-exist-67890.com","api_token":"bad-token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		var response map[string]any
		if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response["success"] != false {
			t.Error("handleSubmit() with unreachable host should return success=false")
		}

		errMsg, ok := response["error"].(string)
		if !ok || !strings.Contains(errMsg, "Connection failed") {
			t.Errorf("handleSubmit() error = %v, want error containing 'Connection failed'", response["error"])
		}
	})
}

func TestJSONResponseFormat(t *testing.T) {
	t.Run("validate endpoint returns JSON", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"http://localhost","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
	})

	t.Run("submit endpoint returns JSON", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		body := `{"base_url":"http://localhost","api_token":"token","account_id":1}`
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		contentType := rec.Header().Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", contentType)
		}
	})

	t.Run("complete endpoint returns JSON", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/complete", nil)
		rec := httptest.NewRecorder()

		done := make(chan struct{})
		go func() {
			server.handleComplete(rec, req)
			close(done)
		}()

		select {
		case <-done:
			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", contentType)
			}
		case <-time.After(time.Second):
			t.Fatal("handleComplete() timed out")
		}
	})
}

func TestHandleSetupAdditionalCases(t *testing.T) {
	server, _ := NewSetupServer("test-profile")

	t.Run("handles POST method on root", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/", nil)
		rec := httptest.NewRecorder()

		server.handleSetup(rec, req)

		// POST to / should still work and return the form
		if rec.Code != http.StatusOK {
			t.Errorf("handleSetup() POST status = %d, want %d", rec.Code, http.StatusOK)
		}
	})

	t.Run("handles HEAD method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodHead, "/", nil)
		rec := httptest.NewRecorder()

		server.handleSetup(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSetup() HEAD status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleSuccessAdditionalCases(t *testing.T) {
	server, _ := NewSetupServer("default")

	t.Run("handles special characters in email", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/success?name=Test&email=test%2Buser%40example.com", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
		}

		body := rec.Body.String()
		// URL-decoded email should appear in the page (might be HTML encoded)
		// Check for either plain or HTML-encoded version
		if !strings.Contains(body, "test+user@example.com") && !strings.Contains(body, "test&#43;user@example.com") {
			t.Errorf("handleSuccess() should handle special characters in email, got: %s", body)
		}
	})

	t.Run("handles very long name", func(t *testing.T) {
		longName := strings.Repeat("a", 1000)
		req := httptest.NewRequest(http.MethodGet, "/success?name="+longName+"&email=test@test.com", nil)
		rec := httptest.NewRecorder()

		server.handleSuccess(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("handleSuccess() status = %d, want %d", rec.Code, http.StatusOK)
		}
	})
}

func TestHandleValidateAdditionalCases(t *testing.T) {
	t.Run("handles OPTIONS method", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodOptions, "/validate", nil)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("handleValidate() OPTIONS status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handles HEAD method", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodHead, "/validate", nil)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("handleValidate() HEAD status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handles empty body", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleValidate(rec, req)

		// Empty body should be treated as invalid JSON
		if rec.Code != http.StatusBadRequest {
			t.Errorf("handleValidate() empty body status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleSubmitAdditionalCases(t *testing.T) {
	t.Run("handles OPTIONS method", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodOptions, "/submit", nil)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		if rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("handleSubmit() OPTIONS status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})

	t.Run("handles empty body", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(""))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		// Empty body should be treated as invalid JSON
		if rec.Code != http.StatusBadRequest {
			t.Errorf("handleSubmit() empty body status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})

	t.Run("handles malformed JSON", func(t *testing.T) {
		server, _ := NewSetupServer("default")
		req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(`{"base_url": `))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", server.csrfToken)
		rec := httptest.NewRecorder()

		server.handleSubmit(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("handleSubmit() malformed JSON status = %d, want %d", rec.Code, http.StatusBadRequest)
		}
	})
}

func TestStartWithShutdownChannelAlreadyClosed(t *testing.T) {
	t.Run("handles pre-closed shutdown channel", func(t *testing.T) {
		server, err := NewSetupServer("preclosed")
		if err != nil {
			t.Fatalf("NewSetupServer() error = %v", err)
		}

		// Pre-close the shutdown channel
		close(server.shutdown)

		ctx := context.Background()

		done := make(chan struct{})
		var startResult *SetupResult

		go func() {
			startResult, _ = server.Start(ctx)
			close(done)
		}()

		select {
		case <-done:
			// Should return quickly since shutdown is already closed
			if startResult != nil && startResult.Email != "" {
				t.Error("Start() should not return successful result when shutdown is pre-closed")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Start() did not return after pre-closed shutdown channel")
		}
	})
}
