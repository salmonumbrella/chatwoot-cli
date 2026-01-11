package api

import "context"

// Requester captures the internal request surface used by resource helpers.
// It is intentionally small to keep API slices loosely coupled.
type Requester interface {
	do(ctx context.Context, method, url string, body any, result any) error
	doRaw(ctx context.Context, method, url string, body any) ([]byte, error)
	accountPath(path string) string
	platformPath(path string) string
	publicPath(path string) string
	PostMultipart(ctx context.Context, path string, fields map[string]string, files map[string][]byte, result any) error
}
