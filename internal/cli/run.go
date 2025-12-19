package cli

import "context"

// Run is a high-level CLI entrypoint suitable for black-box tests.
// It accepts the argument slice (excluding argv[0]) and returns the semantic
// exit code plus any error.
func Run(ctx context.Context, args []string) (CLIResult, error) {
	inv, err := ParseInvocation(args)
	if err != nil {
		return CLIResult{ExitCode: ExitCode(err)}, err
	}
	return Execute(ctx, inv)
}
