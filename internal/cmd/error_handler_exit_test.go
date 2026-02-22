package cmd

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestExitWithError_Subprocess(t *testing.T) {
	if os.Getenv("TEST_EXIT_WITH_ERROR") == "1" {
		ExitWithError(errors.New("boom"))
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestExitWithError_Subprocess")
	cmd.Env = append(os.Environ(), "TEST_EXIT_WITH_ERROR=1")
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected subprocess to exit with error")
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got %T (%v)", err, err)
	}
	if exitErr.ExitCode() != 1 {
		t.Fatalf("expected exit code 1, got %d", exitErr.ExitCode())
	}
	if !strings.Contains(string(out), "Error: boom") {
		t.Fatalf("expected handled error output, got %q", string(out))
	}
}
