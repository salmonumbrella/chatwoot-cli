package main

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/chatwoot/chatwoot-cli/internal/cmd"
)

var (
	executeCmd  = cmd.Execute
	mapExitCode = cmd.ExitCode
	terminate   = os.Exit
)

func run(args []string) int {
	ctx := context.Background()
	if err := executeCmd(ctx, args); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return exitErr.ExitCode()
		}
		return mapExitCode(err)
	}
	return 0
}

func main() {
	terminate(run(os.Args[1:]))
}
