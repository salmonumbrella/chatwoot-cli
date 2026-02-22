package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/chatwoot/chatwoot-cli/internal/config"
	"github.com/chatwoot/chatwoot-cli/internal/validation"
)

// SetupResult contains the result of a browser-based setup
type SetupResult struct {
	Account config.Account
	Email   string
	Error   error
}

// SetupServer handles the browser-based authentication flow
type SetupServer struct {
	result        chan SetupResult
	shutdown      chan struct{}
	pendingResult *SetupResult
	csrfToken     string
	profile       string
}

// NewSetupServer creates a new setup server
func NewSetupServer(profile string) (*SetupServer, error) {
	// Generate CSRF token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate CSRF token: %w", err)
	}

	return &SetupServer{
		result:    make(chan SetupResult, 1),
		shutdown:  make(chan struct{}),
		csrfToken: hex.EncodeToString(tokenBytes),
		profile:   profile,
	}, nil
}

// Start starts the setup server and opens the browser
func (s *SetupServer) Start(ctx context.Context) (*SetupResult, error) {
	// Find an available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	port := listener.Addr().(*net.TCPAddr).Port
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleSetup)
	mux.HandleFunc("/validate", s.handleValidate)
	mux.HandleFunc("/submit", s.handleSubmit)
	mux.HandleFunc("/success", s.handleSuccess)
	mux.HandleFunc("/complete", s.handleComplete)

	server := &http.Server{
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// Start server in background
	go func() {
		_ = server.Serve(listener)
	}()

	// Print URL first so user can open manually if needed
	fmt.Printf("Open this URL in your browser to authenticate:\n  %s\n", baseURL)
	fmt.Println("Attempting to open browser automatically...")
	if err := openBrowser(baseURL); err != nil {
		fmt.Fprintf(os.Stderr, "Could not open browser automatically: %v\n", err)
		fmt.Println("Please open the URL manually in your browser.")
	}

	// Wait for result or context cancellation
	select {
	case result := <-s.result:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close() // Force close if graceful shutdown fails
		}
		return &result, nil
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close() // Force close if graceful shutdown fails
		}
		return nil, ctx.Err()
	case <-s.shutdown:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			_ = server.Close() // Force close if graceful shutdown fails
		}
		if s.pendingResult != nil {
			return s.pendingResult, nil
		}
		return nil, fmt.Errorf("setup cancelled")
	}
}

// handleSetup serves the main setup page
func (s *SetupServer) handleSetup(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.New("setup").Parse(setupTemplate)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := map[string]string{
		"CSRFToken": s.csrfToken,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// handleValidate tests credentials without saving
func (s *SetupServer) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify CSRF token
	if r.Header.Get("X-CSRF-Token") != s.csrfToken {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	var req struct {
		BaseURL   string `json:"base_url"`
		APIToken  string `json:"api_token"`
		AccountID int    `json:"account_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Normalize URL
	req.BaseURL = strings.TrimSuffix(req.BaseURL, "/")

	// Validate URL to prevent SSRF attacks
	if err := validation.ValidateChatwootURL(req.BaseURL); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Invalid URL: %v", err),
		})
		return
	}

	// Test the credentials by making an API call
	client := api.New(req.BaseURL, req.APIToken, req.AccountID)
	profile, err := client.Profile().Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"message":    "Connection successful!",
		"user_id":    profile.ID,
		"user_name":  profile.Name,
		"user_email": profile.Email,
	})
}

// handleSubmit saves credentials after validation
func (s *SetupServer) handleSubmit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify CSRF token
	if r.Header.Get("X-CSRF-Token") != s.csrfToken {
		http.Error(w, "Invalid CSRF token", http.StatusForbidden)
		return
	}

	var req struct {
		BaseURL   string `json:"base_url"`
		APIToken  string `json:"api_token"`
		AccountID int    `json:"account_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"success": false,
			"error":   "Invalid request body",
		})
		return
	}

	// Normalize URL
	req.BaseURL = strings.TrimSuffix(req.BaseURL, "/")

	// Validate URL to prevent SSRF attacks
	if err := validation.ValidateChatwootURL(req.BaseURL); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Invalid URL: %v", err),
		})
		return
	}

	// Validate first
	client := api.New(req.BaseURL, req.APIToken, req.AccountID)
	profile, err := client.Profile().Get(r.Context())
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}

	// Save to keychain
	account := config.Account{
		BaseURL:   req.BaseURL,
		APIToken:  req.APIToken,
		AccountID: req.AccountID,
	}

	if err := config.SaveProfile(s.profile, account); err != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"success": false,
			"error":   fmt.Sprintf("Failed to save credentials: %v", err),
		})
		return
	}

	// Store pending result
	s.pendingResult = &SetupResult{
		Account: account,
		Email:   profile.Email,
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"user_name":  profile.Name,
		"user_email": profile.Email,
	})
}

// handleSuccess serves the success page
func (s *SetupServer) handleSuccess(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("success").Parse(successTemplate)
	if err != nil {
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	data := map[string]string{
		"UserName":  r.URL.Query().Get("name"),
		"UserEmail": r.URL.Query().Get("email"),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// handleComplete signals that setup is done
func (s *SetupServer) handleComplete(w http.ResponseWriter, r *http.Request) {
	if s.pendingResult != nil {
		s.result <- *s.pendingResult
	}
	close(s.shutdown)
	writeJSON(w, http.StatusOK, map[string]any{"success": true})
}

// writeJSON writes a JSON response
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// openBrowser opens the URL in the default browser
func openBrowser(url string) error {
	if shouldSkipAutoBrowserOpen() {
		return nil
	}

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

func shouldSkipAutoBrowserOpen() bool {
	// Always skip browser launch when running under `go test`.
	if flag.Lookup("test.v") != nil {
		return true
	}

	// Explicit opt-outs for automation/CI environments.
	noBrowser := strings.TrimSpace(strings.ToLower(os.Getenv("CHATWOOT_NO_BROWSER")))
	if noBrowser == "1" || noBrowser == "true" || noBrowser == "yes" {
		return true
	}

	return os.Getenv("CHATWOOT_TESTING") == "1"
}
