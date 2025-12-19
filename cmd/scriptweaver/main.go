package main

import (
	"errors"
	"fmt"
	"os"
	"context"

	"scriptweaver/internal/cli"
)

// main is a deterministic boundary: it canonicalizes all CLI inputs into a
// CLIInvocation before any engine logic is invoked.
func main() {
	inv, err := cli.ParseInvocation(os.Args[1:])
	if err != nil {
		var invErr *cli.InvocationError
		if errors.As(err, &invErr) {
			fmt.Fprintln(os.Stderr, invErr.Message)
			os.Exit(invErr.ExitCode)
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(cli.ExitInternalError)
	}

	result, execErr := cli.Execute(context.Background(), inv)
	if execErr != nil {
		fmt.Fprintln(os.Stderr, execErr)
	}
	os.Exit(result.ExitCode)
}