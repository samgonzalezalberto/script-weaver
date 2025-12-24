package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

const (
	ExitSuccess           = 0
	ExitGraphFailure      = 1
	ExitInvalidInvocation = 2
	ExitConfigError       = 3
	ExitInternalError     = 4
)

type ExecutionMode string

const (
	ExecutionModeClean       ExecutionMode = "clean"
	ExecutionModeIncremental ExecutionMode = "incremental"
	ExecutionModeResumeOnly  ExecutionMode = "resume-only"
)

type TraceConfig struct {
	Enabled bool
	Path    string
}

// CLIInvocation is the fully canonicalized, deterministic description of a run.
//
// All paths are normalized (Clean) and all relative paths are resolved relative
// to WorkDir.
//
// NOTE: WorkDir is required and must be absolute; this prevents any dependency
// on the process current working directory.
type CLIInvocation struct {
	GraphPath      string
	WorkDir        string
	CacheDir       string
	OutputDir      string
	ExecutionMode  ExecutionMode
	Trace          TraceConfig
	OriginalGraph  string
	OriginalCache  string
	OriginalOutput string
	OriginalTrace  string
}

type InvocationError struct {
	ExitCode int
	Message  string
}

func (e *InvocationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func invalidInvocationf(format string, args ...any) error {
	return &InvocationError{ExitCode: ExitInvalidInvocation, Message: fmt.Sprintf(format, args...)}
}

// ParseInvocation parses CLI flags into a canonical CLIInvocation.
//
// Determinism goals:
//   - Does not read env vars.
//   - Does not read/assume the process CWD.
//   - Requires WorkDir to be explicit and absolute.
func ParseInvocation(args []string) (CLIInvocation, error) {
	fs := flag.NewFlagSet("scriptweaver", flag.ContinueOnError)
	fs.SetOutput(io.Discard) // parsing errors are returned, not printed

	var workDir string
	var graphPath string
	var cacheDir string
	var outputDir string
	var tracePath string
	var mode string

	fs.StringVar(&workDir, "workdir", "", "Absolute working directory. Required.")
	fs.StringVar(&graphPath, "graph", "", "Graph source path. Required.")
	fs.StringVar(&cacheDir, "cache-dir", "", "Cache directory. Required.")
	fs.StringVar(&outputDir, "output-dir", "", "Output directory. Required.")
	fs.StringVar(&tracePath, "trace", "", "Trace output path (optional).")
	fs.StringVar(&mode, "mode", string(ExecutionModeIncremental), "Execution mode: clean|incremental|resume-only")

	// We intentionally do not accept environment-derived defaults.
	if err := fs.Parse(args); err != nil {
		// flag package returns errors like: "flag provided but not defined: -x"
		return CLIInvocation{}, invalidInvocationf("%v", err)
	}
	if fs.NArg() != 0 {
		return CLIInvocation{}, invalidInvocationf("unexpected positional arguments: %q", strings.Join(fs.Args(), " "))
	}

	workDir = filepath.Clean(workDir)
	if workDir == "" {
		return CLIInvocation{}, invalidInvocationf("--workdir is required")
	}
	if !filepath.IsAbs(workDir) {
		return CLIInvocation{}, invalidInvocationf("--workdir must be an absolute path (got %q)", workDir)
	}

	if graphPath == "" {
		return CLIInvocation{}, invalidInvocationf("--graph is required")
	}
	if cacheDir == "" {
		return CLIInvocation{}, invalidInvocationf("--cache-dir is required")
	}
	if outputDir == "" {
		return CLIInvocation{}, invalidInvocationf("--output-dir is required")
	}

	parsedMode, err := parseExecutionMode(mode)
	if err != nil {
		return CLIInvocation{}, err
	}

	resolvedGraph, err := resolveUnderWorkDir(workDir, graphPath)
	if err != nil {
		return CLIInvocation{}, err
	}
	resolvedCache, err := resolveUnderWorkDir(workDir, cacheDir)
	if err != nil {
		return CLIInvocation{}, err
	}
	resolvedOutput, err := resolveUnderWorkDir(workDir, outputDir)
	if err != nil {
		return CLIInvocation{}, err
	}

	inv := CLIInvocation{
		WorkDir:        workDir,
		GraphPath:      resolvedGraph,
		CacheDir:       resolvedCache,
		OutputDir:      resolvedOutput,
		ExecutionMode:  parsedMode,
		OriginalGraph:  graphPath,
		OriginalCache:  cacheDir,
		OriginalOutput: outputDir,
		OriginalTrace:  tracePath,
	}

	if strings.TrimSpace(tracePath) != "" {
		resolvedTrace, err := resolveUnderWorkDir(workDir, tracePath)
		if err != nil {
			return CLIInvocation{}, err
		}
		inv.Trace = TraceConfig{Enabled: true, Path: resolvedTrace}
	}

	return inv, nil
}

func parseExecutionMode(raw string) (ExecutionMode, error) {
	n := strings.ToLower(strings.TrimSpace(raw))
	switch ExecutionMode(n) {
	case ExecutionModeClean, ExecutionModeIncremental, ExecutionModeResumeOnly:
		return ExecutionMode(n), nil
	case "":
		return "", invalidInvocationf("--mode is required")
	default:
		return "", invalidInvocationf("invalid --mode %q (expected clean|incremental|resume-only)", raw)
	}
}

func resolveUnderWorkDir(workDir, p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", invalidInvocationf("path must not be empty")
	}
	clean := filepath.Clean(p)
	if clean == "." {
		return "", invalidInvocationf("path must not be '.'")
	}

	// If absolute, accept as-is; it is still deterministic.
	// If relative, resolve under WorkDir.
	if filepath.IsAbs(clean) {
		return clean, nil
	}

	// WorkDir is required to be absolute, so Join does not consult process CWD.
	return filepath.Clean(filepath.Join(workDir, clean)), nil
}

// ExitCode extracts a semantic exit code from a ParseInvocation error.
// If the error is not a known invocation error, it returns ExitInternalError.
func ExitCode(err error) int {
	var invErr *InvocationError
	if errors.As(err, &invErr) && invErr != nil {
		if invErr.ExitCode != 0 {
			return invErr.ExitCode
		}
		return ExitInvalidInvocation
	}
	if err == nil {
		return ExitSuccess
	}
	return ExitInternalError
}
