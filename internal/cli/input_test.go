package cli

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseInvocation_DeterministicStruct(t *testing.T) {
	workDir := t.TempDir()
	args := []string{
		"--workdir", workDir,
		"--graph", "graphs/../graph.json",
		"--cache-dir", "./cache/..//cache",
		"--output-dir", "out/./",
		"--mode", "incremental",
		"--trace", "traces/../trace.json",
	}

	inv1, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	inv2, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(inv1, inv2) {
		t.Fatalf("expected identical invocations, got\n%#v\n%#v", inv1, inv2)
	}

	if inv1.WorkDir != filepath.Clean(workDir) {
		t.Fatalf("workdir not canonicalized: %q", inv1.WorkDir)
	}
	if inv1.GraphPath != filepath.Join(workDir, "graph.json") {
		t.Fatalf("graph path not resolved/canonicalized: %q", inv1.GraphPath)
	}
	if inv1.CacheDir != filepath.Join(workDir, "cache") {
		t.Fatalf("cache dir not resolved/canonicalized: %q", inv1.CacheDir)
	}
	if inv1.OutputDir != filepath.Join(workDir, "out") {
		t.Fatalf("output dir not resolved/canonicalized: %q", inv1.OutputDir)
	}
	if !inv1.Trace.Enabled || inv1.Trace.Path != filepath.Join(workDir, "trace.json") {
		t.Fatalf("trace not resolved/canonicalized: %#v", inv1.Trace)
	}
}

func TestParseInvocation_ResolvesRelativePathsAgainstWorkDir_NotCwd(t *testing.T) {
	workDir := t.TempDir()
	otherCwd := t.TempDir()

	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldCwd) })

	if err := os.Chdir(otherCwd); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	args := []string{
		"--workdir", workDir,
		"--graph", "g.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
	}
	inv, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if inv.GraphPath != filepath.Join(workDir, "g.json") {
		t.Fatalf("expected graph under workdir, got %q", inv.GraphPath)
	}
	if inv.CacheDir != filepath.Join(workDir, "cache") {
		t.Fatalf("expected cache under workdir, got %q", inv.CacheDir)
	}
	if inv.OutputDir != filepath.Join(workDir, "out") {
		t.Fatalf("expected output under workdir, got %q", inv.OutputDir)
	}
}

func TestParseInvocation_IgnoresEnvironmentVariables(t *testing.T) {
	workDir := t.TempDir()
	args := []string{
		"--workdir", workDir,
		"--graph", "g.json",
		"--cache-dir", "cache",
		"--output-dir", "out",
		"--mode", "clean",
	}

	inv1, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("DEBUG", "1")
	t.Setenv("CLICOLOR", "1")
	t.Setenv("SOME_OTHER_VAR", "some value")

	inv2, err := ParseInvocation(args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !reflect.DeepEqual(inv1, inv2) {
		t.Fatalf("expected env vars to not affect parsing, got\n%#v\n%#v", inv1, inv2)
	}
}

func TestParseInvocation_WorkDirIsMandatoryAndAbsolute(t *testing.T) {
	_, err := ParseInvocation([]string{"--graph", "g", "--cache-dir", "c", "--output-dir", "o"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != ExitInvalidInvocation {
		t.Fatalf("expected exit code %d, got %d", ExitInvalidInvocation, ExitCode(err))
	}

	_, err = ParseInvocation([]string{"--workdir", "relative", "--graph", "g", "--cache-dir", "c", "--output-dir", "o"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if ExitCode(err) != ExitInvalidInvocation {
		t.Fatalf("expected exit code %d, got %d", ExitInvalidInvocation, ExitCode(err))
	}
}
