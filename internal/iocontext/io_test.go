// internal/iocontext/io_test.go
package iocontext

import (
	"bytes"
	"context"
	"testing"
)

func TestDefaultIO(t *testing.T) {
	io := DefaultIO()
	if io.Out == nil || io.ErrOut == nil || io.In == nil {
		t.Error("DefaultIO should return non-nil streams")
	}
}

func TestWithIO(t *testing.T) {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}

	io := &IO{Out: out, ErrOut: errOut}
	ctx := WithIO(context.Background(), io)

	got := GetIO(ctx)
	if got.Out != out {
		t.Error("GetIO should return the IO set with WithIO")
	}
}

func TestGetIO_DefaultsWhenNotSet(t *testing.T) {
	ctx := context.Background()
	io := GetIO(ctx)
	if io == nil {
		t.Error("GetIO should return default IO when not set")
	}
}
