package main

import (
	"context"
	"os"

	"github.com/chatwoot/chatwoot-cli/internal/cmd"
)

func main() {
	ctx := context.Background()
	if err := cmd.Execute(ctx, os.Args[1:]); err != nil {
		os.Exit(1)
	}
}
