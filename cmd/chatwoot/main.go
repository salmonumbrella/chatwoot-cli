package main

import (
	"context"
	"os"
	"os/exec"

	"github.com/chatwoot/chatwoot-cli/internal/cmd"
)

func main() {
	ctx := context.Background()
	if err := cmd.Execute(ctx, os.Args[1:]); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(cmd.ExitCode(err))
	}
}
