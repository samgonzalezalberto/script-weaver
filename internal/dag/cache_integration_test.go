package dag

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"scriptweaver/internal/core"
)

func TestExecutorSerial_CacheHit_DoesNotReexecuteAndRestoresArtifacts(t *testing.T) {
	workDir := t.TempDir()

	cache := core.NewMemoryCache()
	coreRunner := core.NewRunner(workDir, cache)
	cacheRunner, err := NewCacheAwareRunner(coreRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	g, err := NewTaskGraph(
		[]core.Task{
			{
				Name:    "A",
				Run:     "if [ -e ran_once ]; then echo already 1>&2; exit 9; fi; : > ran_once; printf 'artifact-v1' > a.txt",
				Outputs: []string{"a.txt"},
			},
		},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exec1, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.FinalState["A"] != TaskCompleted {
		t.Fatalf("expected A completed, got %s", res1.FinalState["A"])
	}

	baseline, err := os.ReadFile(filepath.Join(workDir, "a.txt"))
	if err != nil {
		t.Fatalf("reading baseline artifact: %v", err)
	}

	// Remove the artifact; cache replay must restore it without re-executing the task.
	if err := os.Remove(filepath.Join(workDir, "a.txt")); err != nil {
		t.Fatalf("removing artifact: %v", err)
	}

	exec2, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res2, err := exec2.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.FinalState["A"] != TaskCached {
		t.Fatalf("expected A cached, got %s", res2.FinalState["A"])
	}

	restored, err := os.ReadFile(filepath.Join(workDir, "a.txt"))
	if err != nil {
		t.Fatalf("reading restored artifact: %v", err)
	}
	if !bytes.Equal(restored, baseline) {
		t.Fatalf("artifact mismatch after replay: got %q want %q", restored, baseline)
	}
}

func TestExecutorSerial_CacheMixedHitMiss_PartialRestorationDeterministic(t *testing.T) {
	workDir := t.TempDir()

	cache := core.NewMemoryCache()
	coreRunner := core.NewRunner(workDir, cache)
	cacheRunner, err := NewCacheAwareRunner(coreRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	inPath := filepath.Join(workDir, "in.txt")
	if err := os.WriteFile(inPath, []byte("v1"), 0o600); err != nil {
		t.Fatalf("writing input: %v", err)
	}

	g, err := NewTaskGraph(
		[]core.Task{
			{
				Name:    "A",
				Inputs:  []string{"in.txt"},
				Run:     "IFS= read -r x < in.txt; printf '%s' \"$x\" > a.txt",
				Outputs: []string{"a.txt"},
			},
			{
				Name:    "B",
				Inputs:  []string{"a.txt"},
				Run:     "IFS= read -r x < a.txt; printf '%sB' \"$x\" > b.txt",
				Outputs: []string{"b.txt"},
			},
			{
				Name:    "C",
				Run:     "if [ -e ran_C ]; then echo ran-twice 1>&2; exit 9; fi; : > ran_C; printf 'C' > c.txt",
				Outputs: []string{"c.txt"},
			},
		},
		[]Edge{{From: "A", To: "B"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Run 1: all cache misses.
	exec1, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res1, err := exec1.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res1.FinalState["A"] != TaskCompleted || res1.FinalState["B"] != TaskCompleted || res1.FinalState["C"] != TaskCompleted {
		t.Fatalf("expected all completed on first run, got: %v", res1.FinalState)
	}

	a1, err := os.ReadFile(filepath.Join(workDir, "a.txt"))
	if err != nil {
		t.Fatalf("reading a.txt: %v", err)
	}
	b1, err := os.ReadFile(filepath.Join(workDir, "b.txt"))
	if err != nil {
		t.Fatalf("reading b.txt: %v", err)
	}
	c1, err := os.ReadFile(filepath.Join(workDir, "c.txt"))
	if err != nil {
		t.Fatalf("reading c.txt: %v", err)
	}

	// Run 2: delete artifacts; all should be replayed from cache.
	for _, p := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.Remove(filepath.Join(workDir, p)); err != nil {
			t.Fatalf("removing %s: %v", p, err)
		}
	}

	exec2, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res2, err := exec2.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2.FinalState["A"] != TaskCached || res2.FinalState["B"] != TaskCached || res2.FinalState["C"] != TaskCached {
		t.Fatalf("expected all cached on second run, got: %v", res2.FinalState)
	}

	a2, _ := os.ReadFile(filepath.Join(workDir, "a.txt"))
	b2, _ := os.ReadFile(filepath.Join(workDir, "b.txt"))
	c2, _ := os.ReadFile(filepath.Join(workDir, "c.txt"))
	if !bytes.Equal(a2, a1) || !bytes.Equal(b2, b1) || !bytes.Equal(c2, c1) {
		t.Fatalf("artifacts not bit-identical after replay")
	}

	// Run 3: change input for A, delete artifacts. A and B must re-execute; C must be replayed.
	if err := os.WriteFile(inPath, []byte("v2"), 0o600); err != nil {
		t.Fatalf("writing input: %v", err)
	}
	for _, p := range []string{"a.txt", "b.txt", "c.txt"} {
		if err := os.Remove(filepath.Join(workDir, p)); err != nil {
			t.Fatalf("removing %s: %v", p, err)
		}
	}

	exec3, err := NewExecutor(g, cacheRunner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	res3, err := exec3.RunSerial(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if res3.FinalState["A"] != TaskCompleted {
		t.Fatalf("expected A completed (miss), got %s", res3.FinalState["A"])
	}
	if res3.FinalState["B"] != TaskCompleted {
		t.Fatalf("expected B completed (miss), got %s", res3.FinalState["B"])
	}
	if res3.FinalState["C"] != TaskCached {
		t.Fatalf("expected C cached (hit), got %s", res3.FinalState["C"])
	}

	a3, _ := os.ReadFile(filepath.Join(workDir, "a.txt"))
	b3, _ := os.ReadFile(filepath.Join(workDir, "b.txt"))
	c3, _ := os.ReadFile(filepath.Join(workDir, "c.txt"))
	if string(a3) != "v2" {
		t.Fatalf("unexpected A output: %q", a3)
	}
	if string(b3) != "v2B" {
		t.Fatalf("unexpected B output: %q", b3)
	}
	if !bytes.Equal(c3, c1) {
		t.Fatalf("C output mismatch after partial restoration")
	}
}
