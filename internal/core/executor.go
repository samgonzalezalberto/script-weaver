// Package core defines the domain models for deterministic task execution.
package core

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"syscall"
)

// ExecutionResult contains the results of a task execution.
//
// From spec.md Cache Behavior, cached data includes:
//   - stdout
//   - stderr
//   - exit code
//   - artifacts defined by outputs
//
// From spec.md Failure Behavior:
//   - Failed executions (non-zero exit code) are cacheable.
type ExecutionResult struct {
	// Stdout is the captured standard output.
	Stdout []byte

	// Stderr is the captured standard error.
	Stderr []byte

	// ExitCode is the process exit code.
	// 0 indicates success, non-zero indicates failure.
	ExitCode int

	// Hash is the TaskHash that was used for this execution.
	Hash TaskHash
}

// Executor runs tasks in an isolated, deterministic environment.
//
// From spec.md Deterministic Guarantees:
//   - Environment Determinism: Only explicitly declared environment variables are visible.
//   - Execution Determinism: Tasks execute in an isolated, controlled environment.
//
// From tdd.md Test 5:
//   - An environment variable not listed in env — the task MUST NOT observe it.
type Executor struct {
	// WorkingDir is the directory where tasks are executed.
	WorkingDir string
}

// NewExecutor creates a new Executor with the given working directory.
func NewExecutor(workingDir string) *Executor {
	return &Executor{WorkingDir: workingDir}
}

// Execute runs the given task with strict environment isolation.
//
// Environment Isolation (CRITICAL):
//   - ONLY variables declared in task.Env are visible to the command.
//   - Host environment variables (HOME, USER, PATH, etc.) are NOT passed through.
//   - If PATH is not in env, the task sees no PATH.
//
// This is an ALLOWLIST approach: the environment starts empty and only
// declared variables are added.
func (e *Executor) Execute(ctx context.Context, task *Task, hash TaskHash) (*ExecutionResult, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}

	if task.Run == "" {
		return nil, fmt.Errorf("task.Run is empty")
	}

	// Create command
	// Using "sh -c" to interpret the command string as a shell command
	cmd := exec.CommandContext(ctx, "sh", "-c", task.Run)

	// Set working directory
	cmd.Dir = e.WorkingDir

	// CRITICAL: Build environment from ALLOWLIST only
	// Start with EMPTY environment, NOT os.Environ()
	// Only add variables explicitly declared in task.Env
	cmd.Env = buildIsolatedEnv(task.Env)

	// Set process group so we can kill the entire process tree on cancellation
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Wait for completion or context cancellation
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	var err error
	select {
	case <-ctx.Done():
		// Context cancelled - kill the entire process group
		if cmd.Process != nil {
			// Kill the process group (negative PID)
			syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		}
		<-done // Wait for the process to actually exit
		return nil, fmt.Errorf("execution cancelled: %w", ctx.Err())
	case err = <-done:
		// Command completed
	}

	// Determine exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			// Command failed to start (e.g., shell not found)
			return nil, fmt.Errorf("failed to execute command: %w", err)
		}
	}

	return &ExecutionResult{
		Stdout:   stdout.Bytes(),
		Stderr:   stderr.Bytes(),
		ExitCode: exitCode,
		Hash:     hash,
	}, nil
}

// buildIsolatedEnv constructs an isolated environment from the declared variables.
//
// CRITICAL: This uses an ALLOWLIST approach.
//   - The environment starts EMPTY.
//   - Only variables in the env map are added.
//   - NO host variables are passed through.
//
// From spec.md Environment Determinism:
//
//	"Only explicitly declared environment variables are visible."
//
// From tdd.md Test 5:
//
//	"An environment variable not listed in env — the task MUST NOT observe it."
func buildIsolatedEnv(env map[string]string) []string {
	if env == nil || len(env) == 0 {
		// Return empty environment, not nil
		// This ensures the command runs with NO environment variables
		return []string{}
	}

	result := make([]string, 0, len(env))
	for key, value := range env {
		result = append(result, fmt.Sprintf("%s=%s", key, value))
	}

	return result
}
