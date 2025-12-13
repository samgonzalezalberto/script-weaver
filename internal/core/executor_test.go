package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestExecute_UndeclaredEnvVarsInvisible verifies tdd.md#Test-5:
// "Given an environment variable not listed in env:
// The task MUST NOT observe it."
func TestExecute_UndeclaredEnvVarsInvisible(t *testing.T) {
	// Set a host environment variable that should NOT be visible
	os.Setenv("SECRET_HOST_VAR", "should_not_see_this")
	defer os.Unsetenv("SECRET_HOST_VAR")

	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	// Task does NOT declare SECRET_HOST_VAR in env
	task := &Task{
		Name:   "test-undeclared-env",
		Inputs: []string{},
		Run:    "echo \"VAR=${SECRET_HOST_VAR:-unset}\"",
		Env:    map[string]string{}, // Empty - no declared vars
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// The variable should be unset/empty, not the host value
	stdout := string(result.Stdout)
	if strings.Contains(stdout, "should_not_see_this") {
		t.Errorf("task observed undeclared host variable: %s", stdout)
	}

	if !strings.Contains(stdout, "VAR=unset") {
		t.Errorf("expected VAR=unset, got: %s", stdout)
	}
}

// TestExecute_OnlyDeclaredEnvVarsVisible verifies env allowlist.
func TestExecute_OnlyDeclaredEnvVarsVisible(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-declared-env",
		Inputs: []string{},
		Run:    "echo \"FOO=$FOO BAR=$BAR\"",
		Env: map[string]string{
			"FOO": "hello",
			"BAR": "world",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)
	if !strings.Contains(stdout, "FOO=hello") {
		t.Errorf("FOO not visible: %s", stdout)
	}
	if !strings.Contains(stdout, "BAR=world") {
		t.Errorf("BAR not visible: %s", stdout)
	}
}

// TestExecute_NoPathMeansNoPath verifies PATH is not passed through from host.
// Note: Some shells have a compiled-in default PATH, but the HOST's PATH
// environment variable should NOT be passed through.
func TestExecute_NoPathMeansNoPath(t *testing.T) {
	// Get the host's actual PATH
	hostPath := os.Getenv("PATH")
	if hostPath == "" {
		t.Skip("HOST PATH not set")
	}

	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	// Task does NOT declare PATH
	task := &Task{
		Name:   "test-no-path",
		Inputs: []string{},
		Run:    "echo \"PATH=${PATH:-no_path}\"",
		Env:    map[string]string{}, // No PATH declared
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)
	// The host's specific PATH should NOT be visible
	// (Shell may have a built-in default, but that's different from host's PATH)
	if strings.Contains(stdout, hostPath) && len(hostPath) > 20 {
		// Only fail if the full host PATH is visible (not just common prefixes)
		t.Errorf("task observed host PATH: got %s, host PATH was %s", stdout, hostPath)
	}
}

// TestExecute_ExplicitPathWorks verifies declared PATH is used.
func TestExecute_ExplicitPathWorks(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-explicit-path",
		Inputs: []string{},
		Run:    "echo \"PATH=$PATH\"",
		Env: map[string]string{
			"PATH": "/custom/bin:/other/bin",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)
	if !strings.Contains(stdout, "/custom/bin:/other/bin") {
		t.Errorf("explicit PATH not visible: %s", stdout)
	}
}

// TestExecute_CapturesStdout verifies stdout is captured.
func TestExecute_CapturesStdout(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-stdout",
		Inputs: []string{},
		Run:    "echo 'hello stdout'",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(string(result.Stdout), "hello stdout") {
		t.Errorf("stdout not captured: %s", result.Stdout)
	}
}

// TestExecute_CapturesStderr verifies stderr is captured.
func TestExecute_CapturesStderr(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-stderr",
		Inputs: []string{},
		Run:    "echo 'hello stderr' >&2",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !strings.Contains(string(result.Stderr), "hello stderr") {
		t.Errorf("stderr not captured: %s", result.Stderr)
	}
}

// TestExecute_CapturesExitCode verifies exit code is captured.
func TestExecute_CapturesExitCode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	// Test successful exit
	task := &Task{
		Name:   "test-exit-success",
		Inputs: []string{},
		Run:    "exit 0",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", result.ExitCode)
	}
}

// TestExecute_CapturesNonZeroExitCode verifies failed exit code is captured.
func TestExecute_CapturesNonZeroExitCode(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-exit-failure",
		Inputs: []string{},
		Run:    "exit 42",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", result.ExitCode)
	}
}

// TestExecute_UsesWorkingDir verifies working directory is set.
func TestExecute_UsesWorkingDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	workDir := filepath.Join(tmpDir, "workdir")
	if err := os.Mkdir(workDir, 0755); err != nil {
		t.Fatalf("failed to create workdir: %v", err)
	}

	executor := NewExecutor(workDir)

	task := &Task{
		Name:   "test-workdir",
		Inputs: []string{},
		Run:    "pwd",
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := strings.TrimSpace(string(result.Stdout))
	if stdout != workDir {
		t.Errorf("expected workdir %s, got %s", workDir, stdout)
	}
}

// TestExecute_NilTaskFails verifies nil task is rejected.
func TestExecute_NilTaskFails(t *testing.T) {
	executor := NewExecutor("/tmp")

	ctx := context.Background()
	_, err := executor.Execute(ctx, nil, TaskHash("test-hash"))

	if err == nil {
		t.Error("expected error for nil task")
	}
}

// TestExecute_EmptyRunFails verifies empty run command is rejected.
func TestExecute_EmptyRunFails(t *testing.T) {
	executor := NewExecutor("/tmp")

	task := &Task{
		Name:   "test-empty-run",
		Inputs: []string{},
		Run:    "",
		Env:    map[string]string{},
	}

	ctx := context.Background()
	_, err := executor.Execute(ctx, task, TaskHash("test-hash"))

	if err == nil {
		t.Error("expected error for empty run command")
	}
}

// TestExecute_HashStoredInResult verifies hash is stored in result.
func TestExecute_HashStoredInResult(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-hash-stored",
		Inputs: []string{},
		Run:    "echo test",
		Env:    map[string]string{},
	}

	expectedHash := TaskHash("expected-hash-value")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, expectedHash)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Hash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, result.Hash)
	}
}

// TestExecute_ContextCancellation verifies context cancellation works.
func TestExecute_ContextCancellation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-cancellation",
		Inputs: []string{},
		Run:    "sleep 60", // Long-running command
		Env:    map[string]string{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	// Cancel immediately after a short delay
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err = executor.Execute(ctx, task, TaskHash("test-hash"))
	elapsed := time.Since(start)

	// Should complete quickly (not wait 60 seconds)
	// The command should be killed, resulting in an error or non-zero exit
	if elapsed > 5*time.Second {
		t.Errorf("context cancellation took too long: %v", elapsed)
	}
	
	// Note: exec.CommandContext kills the process on context cancellation
	// which results in either an error or a signal-based exit code
}

// TestExecute_HomeNotPassedThrough verifies HOME is not passed through.
func TestExecute_HomeNotPassedThrough(t *testing.T) {
	// Ensure HOME is set on host
	if os.Getenv("HOME") == "" {
		t.Skip("HOME not set on host")
	}

	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-no-home",
		Inputs: []string{},
		Run:    "echo \"HOME=${HOME:-not_set}\"",
		Env:    map[string]string{}, // HOME not declared
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)
	hostHome := os.Getenv("HOME")
	if strings.Contains(stdout, hostHome) {
		t.Errorf("task observed host HOME: %s", stdout)
	}
}

// TestExecute_UserNotPassedThrough verifies USER is not passed through.
func TestExecute_UserNotPassedThrough(t *testing.T) {
	if os.Getenv("USER") == "" {
		t.Skip("USER not set on host")
	}

	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	task := &Task{
		Name:   "test-no-user",
		Inputs: []string{},
		Run:    "echo \"USER=${USER:-not_set}\"",
		Env:    map[string]string{}, // USER not declared
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)
	hostUser := os.Getenv("USER")
	if strings.Contains(stdout, hostUser) && hostUser != "not_set" {
		t.Errorf("task observed host USER: %s", stdout)
	}
}

// TestBuildIsolatedEnv_EmptyEnv verifies empty env produces empty slice.
func TestBuildIsolatedEnv_EmptyEnv(t *testing.T) {
	result := buildIsolatedEnv(map[string]string{})
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

// TestBuildIsolatedEnv_NilEnv verifies nil env produces empty slice.
func TestBuildIsolatedEnv_NilEnv(t *testing.T) {
	result := buildIsolatedEnv(nil)
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d elements", len(result))
	}
}

// TestBuildIsolatedEnv_FormatsCorrectly verifies KEY=VALUE format.
func TestBuildIsolatedEnv_FormatsCorrectly(t *testing.T) {
	env := map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	}

	result := buildIsolatedEnv(env)

	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}

	// Check both entries exist (order may vary)
	found := make(map[string]bool)
	for _, entry := range result {
		found[entry] = true
	}

	if !found["FOO=bar"] {
		t.Error("missing FOO=bar")
	}
	if !found["BAZ=qux"] {
		t.Error("missing BAZ=qux")
	}
}

// TestExecute_HostEnvCompletelyIsolated is a comprehensive test that verifies
// the task environment is completely isolated from the host.
func TestExecute_HostEnvCompletelyIsolated(t *testing.T) {
	// Set multiple unique host variables that should NOT be visible
	uniqueVars := map[string]string{
		"SCRIPTWEAVER_TEST_VAR1": "secret_value_1",
		"SCRIPTWEAVER_TEST_VAR2": "secret_value_2",
		"SCRIPTWEAVER_TEST_VAR3": "secret_value_3",
	}

	for k, v := range uniqueVars {
		os.Setenv(k, v)
		defer os.Unsetenv(k)
	}

	tmpDir, err := os.MkdirTemp("", "executor-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	executor := NewExecutor(tmpDir)

	// Task that prints all environment variables
	task := &Task{
		Name:   "test-complete-isolation",
		Inputs: []string{},
		Run:    "env",
		Env: map[string]string{
			"ALLOWED_VAR": "allowed_value",
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := executor.Execute(ctx, task, TaskHash("test-hash"))
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	stdout := string(result.Stdout)

	// Verify NONE of the host-specific test variables are visible
	for k, v := range uniqueVars {
		if strings.Contains(stdout, k) {
			t.Errorf("host variable %s leaked through", k)
		}
		if strings.Contains(stdout, v) {
			t.Errorf("host value %s leaked through", v)
		}
	}

	// Verify the allowed variable IS visible
	if !strings.Contains(stdout, "ALLOWED_VAR=allowed_value") {
		t.Errorf("allowed variable not visible: %s", stdout)
	}
}
