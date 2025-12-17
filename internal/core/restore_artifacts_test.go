package core

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRestoreArtifacts_IncrementalRunRestoresMissingOutput(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "restore-artifacts-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cache := NewMemoryCache()
	runner := NewRunner(tmpDir, cache)

	task := &Task{
		Name:    "A",
		Inputs:  []string{},
		Run:     "printf hello > foo.txt",
		Env:     map[string]string{},
		Outputs: []string{"foo.txt"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First run: executes and caches artifacts.
	res1, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("first run failed: %v", err)
	}
	if res1.FromCache {
		t.Fatalf("expected first run not from cache")
	}

	outPath := filepath.Join(tmpDir, "foo.txt")
	if _, err := os.Stat(outPath); err != nil {
		t.Fatalf("expected foo.txt after first run: %v", err)
	}

	// Delete output to simulate a dirty/missing workspace.
	if err := os.Remove(outPath); err != nil {
		t.Fatalf("failed to delete foo.txt: %v", err)
	}

	// Second run: should be cache hit and must restore foo.txt.
	res2, err := runner.Run(ctx, task)
	if err != nil {
		t.Fatalf("second run failed: %v", err)
	}
	if !res2.FromCache {
		t.Fatalf("expected second run from cache")
	}

	content, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("expected foo.txt restored on cache hit: %v", err)
	}
	if string(content) != "hello" {
		t.Fatalf("unexpected restored content: %q", string(content))
	}
}
