package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"testing"
)

func TestRun_Success(t *testing.T) {
	origExec := executeCmd
	origMap := mapExitCode
	t.Cleanup(func() {
		executeCmd = origExec
		mapExitCode = origMap
	})

	var gotArgs []string
	executeCmd = func(_ context.Context, args []string) error {
		gotArgs = append([]string(nil), args...)
		return nil
	}
	mapExitCode = func(_ error) int {
		t.Fatal("mapExitCode should not be called on success")
		return 99
	}

	code := run([]string{"version", "--output", "json"})
	if code != 0 {
		t.Fatalf("run() code = %d, want 0", code)
	}

	want := []string{"version", "--output", "json"}
	if len(gotArgs) != len(want) {
		t.Fatalf("args len = %d, want %d", len(gotArgs), len(want))
	}
	for i := range want {
		if gotArgs[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q", i, gotArgs[i], want[i])
		}
	}
}

func TestRun_GenericErrorUsesMappedExitCode(t *testing.T) {
	origExec := executeCmd
	origMap := mapExitCode
	t.Cleanup(func() {
		executeCmd = origExec
		mapExitCode = origMap
	})

	executeErr := errors.New("boom")
	executeCmd = func(_ context.Context, _ []string) error {
		return executeErr
	}

	called := false
	mapExitCode = func(err error) int {
		called = true
		if !errors.Is(err, executeErr) {
			t.Fatalf("mapExitCode got err %v, want %v", err, executeErr)
		}
		return 23
	}

	code := run([]string{"status"})
	if code != 23 {
		t.Fatalf("run() code = %d, want 23", code)
	}
	if !called {
		t.Fatal("expected mapExitCode to be called")
	}
}

func TestRun_ExitErrorUsesProcessExitCode(t *testing.T) {
	origExec := executeCmd
	origMap := mapExitCode
	t.Cleanup(func() {
		executeCmd = origExec
		mapExitCode = origMap
	})

	exitErr := createExitError(t, 7)
	executeCmd = func(_ context.Context, _ []string) error {
		return exitErr
	}

	mapExitCode = func(_ error) int {
		t.Fatal("mapExitCode should not be called for ExitError")
		return 99
	}

	code := run([]string{"status"})
	if code != 7 {
		t.Fatalf("run() code = %d, want 7", code)
	}
}

func TestMain_UsesTerminateWithRunCode(t *testing.T) {
	origExec := executeCmd
	origMap := mapExitCode
	origTerminate := terminate
	origArgs := os.Args
	t.Cleanup(func() {
		executeCmd = origExec
		mapExitCode = origMap
		terminate = origTerminate
		os.Args = origArgs
	})

	var gotArgs []string
	executeCmd = func(_ context.Context, args []string) error {
		gotArgs = append([]string(nil), args...)
		return errors.New("boom")
	}
	mapExitCode = func(_ error) int { return 13 }

	called := false
	gotCode := 0
	terminate = func(code int) {
		called = true
		gotCode = code
	}

	os.Args = []string{"chatwoot", "status", "--output", "json"}
	main()

	if !called {
		t.Fatal("expected terminate to be called")
	}
	if gotCode != 13 {
		t.Fatalf("terminate code = %d, want 13", gotCode)
	}

	wantArgs := []string{"status", "--output", "json"}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args len = %d, want %d", len(gotArgs), len(wantArgs))
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("args[%d] = %q, want %q", i, gotArgs[i], wantArgs[i])
		}
	}
}

func createExitError(t *testing.T, code int) *exec.ExitError {
	t.Helper()
	cmd := exec.Command("sh", "-c", "exit "+strconv.Itoa(code))
	err := cmd.Run()
	if err == nil {
		t.Fatal("expected command to fail")
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T", err)
	}
	if exitErr.ExitCode() != code {
		t.Fatalf("exit code = %d, want %d", exitErr.ExitCode(), code)
	}
	return exitErr
}
