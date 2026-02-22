package api

import "context"

// PathResolver provides methods for resolving API endpoint paths.
// It abstracts the URL construction logic, allowing services to build
// paths without knowing the base URL or account ID details.
//
// This interface enables testing path resolution independently from
// HTTP execution, and allows services that only need path building
// to depend on a minimal interface.
type PathResolver interface {
	// accountPath returns the full path for account-scoped API endpoints.
	// Example: accountPath("/contacts") -> "/api/v1/accounts/123/contacts"
	accountPath(path string) string

	// platformPath returns the full path for platform API endpoints.
	// Example: platformPath("/users") -> "/platform/api/v1/users"
	platformPath(path string) string

	// publicPath returns the full path for public API endpoints.
	// Example: publicPath("/inboxes/abc/contacts") -> "/public/api/v1/inboxes/abc/contacts"
	publicPath(path string) string
}

// HTTPExecutor provides methods for executing HTTP requests.
// It abstracts the HTTP client logic, handling JSON serialization,
// error handling, retries, and response parsing.
//
// This interface enables testing HTTP execution independently from
// path resolution, and allows mocking of network operations in tests.
type HTTPExecutor interface {
	// do executes an HTTP request with JSON body and response parsing.
	// The body is marshaled to JSON if non-nil, and the response is
	// unmarshaled into result if non-nil.
	do(ctx context.Context, method, url string, body any, result any) error

	// doRaw executes an HTTP request and returns the raw response bytes.
	// Useful when the response format is not JSON or needs custom parsing.
	doRaw(ctx context.Context, method, url string, body any) ([]byte, error)

	// PostMultipart performs a multipart/form-data POST request.
	// Used for file uploads and form submissions with binary data.
	PostMultipart(ctx context.Context, path string, fields map[string]string, files map[string][]byte, result any) error
}

// Requester combines PathResolver and HTTPExecutor to provide
// the complete request surface used by resource helpers.
//
// It is the primary interface that API services depend on, allowing
// them to both construct paths and execute HTTP requests. Services
// that need only a subset of functionality can depend on the smaller
// interfaces (PathResolver or HTTPExecutor) for improved testability.
//
// Example usage in tests:
//
//	// Mock only path resolution
//	type mockPathResolver struct { ... }
//	func (m *mockPathResolver) accountPath(p string) string { return "/mock" + p }
//
//	// Mock only HTTP execution
//	type mockHTTPExecutor struct { ... }
//	func (m *mockHTTPExecutor) do(...) error { return nil }
type Requester interface {
	PathResolver
	HTTPExecutor
}
