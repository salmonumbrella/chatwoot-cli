package cmd

import (
	"errors"
	"testing"
	"time"

	"github.com/chatwoot/chatwoot-cli/internal/api"
	"github.com/spf13/pflag"
)

func TestExitCodeMapping(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code int
	}{
		{"nil", nil, exitOK},
		{"help", pflag.ErrHelp, exitOK},
		{"auth", &api.AuthError{Reason: "missing"}, exitAuth},
		{"not found", &api.APIError{StatusCode: 404, Body: "not found"}, exitNotFound},
		{"forbidden", &api.APIError{StatusCode: 403, Body: "forbidden"}, exitForbidden},
		{"rate limited", &api.RateLimitError{RetryAfter: time.Second}, exitRateLimited},
		{"server", &api.APIError{StatusCode: 500, Body: "oops"}, exitServer},
		{"usage", errors.New("unknown command \"nope\""), exitUsage},
		{"usage shorthand", errors.New("unknown shorthand flag: 'a' in -a"), exitUsage},
		{"network", errors.New("dial tcp: connection refused"), exitNetwork},
		{"generic", errors.New("boom"), exitGeneric},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ExitCode(tc.err); got != tc.code {
				t.Fatalf("ExitCode(%v) = %d, want %d", tc.err, got, tc.code)
			}
		})
	}
}

func TestExitCode_HandledErrorUsesStoredCode(t *testing.T) {
	err := &handledError{err: errors.New("wrapped"), exitCode: exitNotFound}
	if got := ExitCode(err); got != exitNotFound {
		t.Fatalf("ExitCode(handled) = %d, want %d", got, exitNotFound)
	}
}
