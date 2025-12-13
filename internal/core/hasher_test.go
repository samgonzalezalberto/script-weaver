package core

import (
	"testing"
)

// TestComputeHash_IdenticalInputsProduceSameHash verifies tdd.md#Test-1:
// "Given identical task definition, identical input file contents,
// identical environment variables: The computed Task Hash MUST be identical."
func TestComputeHash_IdenticalInputsProduceSameHash(t *testing.T) {
	hasher := NewTaskHasher()

	input := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/a/file1.txt", Content: []byte("content1")},
				{Path: "/a/file2.txt", Content: []byte("content2")},
			},
		},
		Command:    "echo hello",
		Env:        map[string]string{"FOO": "bar", "BAZ": "qux"},
		Outputs:    []string{"output.txt"},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input)
	hash2 := hasher.ComputeHash(input)

	if hash1 != hash2 {
		t.Errorf("identical inputs produced different hashes: %s != %s", hash1, hash2)
	}
}

// TestComputeHash_ContentChangeInvalidatesHash verifies tdd.md#Test-3:
// "Given a single input file content change: The Task Hash MUST change."
func TestComputeHash_ContentChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/a/file.txt", Content: []byte("original content")},
			},
		},
		Command:    "echo hello",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/a/file.txt", Content: []byte("modified content")},
			},
		},
		Command:    "echo hello",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("content change did not invalidate hash")
	}
}

// TestComputeHash_EnvChangeInvalidatesHash verifies tdd.md#Test-4:
// "Given a change to any declared environment variable: The Task Hash MUST change."
func TestComputeHash_EnvChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	baseInput := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/a/file.txt", Content: []byte("content")},
			},
		},
		Command:    "echo hello",
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	// Test: changing env value
	input1 := baseInput
	input1.Env = map[string]string{"KEY": "value1"}

	input2 := baseInput
	input2.Env = map[string]string{"KEY": "value2"}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("env value change did not invalidate hash")
	}

	// Test: adding new env variable
	input3 := baseInput
	input3.Env = map[string]string{"KEY": "value1", "NEW": "var"}

	hash3 := hasher.ComputeHash(input3)

	if hash1 == hash3 {
		t.Error("adding env variable did not invalidate hash")
	}

	// Test: changing env key
	input4 := baseInput
	input4.Env = map[string]string{"DIFFERENT_KEY": "value1"}

	hash4 := hasher.ComputeHash(input4)

	if hash1 == hash4 {
		t.Error("env key change did not invalidate hash")
	}
}

// TestComputeHash_CommandChangeInvalidatesHash verifies command is part of hash.
func TestComputeHash_CommandChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "echo hello",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "echo world",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("command change did not invalidate hash")
	}
}

// TestComputeHash_OutputsChangeInvalidatesHash verifies declared outputs affect hash.
func TestComputeHash_OutputsChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{"output1.txt"},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{"output2.txt"},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("outputs change did not invalidate hash")
	}
}

// TestComputeHash_WorkingDirChangeInvalidatesHash verifies working directory affects hash.
func TestComputeHash_WorkingDirChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work/project1",
	}

	input2 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work/project2",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("working directory change did not invalidate hash")
	}
}

// TestComputeHash_EnvOrderDoesNotAffectHash verifies env vars are sorted.
func TestComputeHash_EnvOrderDoesNotAffectHash(t *testing.T) {
	hasher := NewTaskHasher()

	// Create two inputs with same env vars added in different order
	// Go maps are unordered, so we verify sorting works correctly
	input1 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{"AAA": "1", "ZZZ": "2", "MMM": "3"},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{"ZZZ": "2", "MMM": "3", "AAA": "1"},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 != hash2 {
		t.Error("same env vars in different order produced different hashes")
	}
}

// TestComputeHash_OutputsOrderDoesNotAffectHash verifies outputs are sorted.
func TestComputeHash_OutputsOrderDoesNotAffectHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{"aaa.txt", "zzz.txt", "mmm.txt"},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{"zzz.txt", "mmm.txt", "aaa.txt"},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 != hash2 {
		t.Error("same outputs in different order produced different hashes")
	}
}

// TestComputeHash_InputPathChangeInvalidatesHash verifies path is part of identity.
func TestComputeHash_InputPathChangeInvalidatesHash(t *testing.T) {
	hasher := NewTaskHasher()

	input1 := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/path/a.txt", Content: []byte("content")},
			},
		},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	input2 := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/path/b.txt", Content: []byte("content")},
			},
		},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	hash1 := hasher.ComputeHash(input1)
	hash2 := hasher.ComputeHash(input2)

	if hash1 == hash2 {
		t.Error("input path change did not invalidate hash")
	}
}

// TestComputeHash_NilInputsHandled verifies nil InputSet is handled.
func TestComputeHash_NilInputsHandled(t *testing.T) {
	hasher := NewTaskHasher()

	input := HashInput{
		Inputs:     nil,
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	// Should not panic
	hash := hasher.ComputeHash(input)

	if hash == "" {
		t.Error("nil inputs produced empty hash")
	}
}

// TestComputeHash_EmptyInputsHandled verifies empty InputSet is handled.
func TestComputeHash_EmptyInputsHandled(t *testing.T) {
	hasher := NewTaskHasher()

	input := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "build",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/work",
	}

	hash := hasher.ComputeHash(input)

	if hash == "" {
		t.Error("empty inputs produced empty hash")
	}
}

// TestComputeHash_Deterministic verifies multiple runs produce same hash.
func TestComputeHash_Deterministic(t *testing.T) {
	hasher := NewTaskHasher()

	input := HashInput{
		Inputs: &InputSet{
			Inputs: []Input{
				{Path: "/z.txt", Content: []byte("z")},
				{Path: "/a.txt", Content: []byte("a")},
				{Path: "/m.txt", Content: []byte("m")},
			},
		},
		Command:    "complex command with args",
		Env:        map[string]string{"PATH": "/bin", "HOME": "/home/user"},
		Outputs:    []string{"out1", "out2", "out3"},
		WorkingDir: "/some/path",
	}

	// Compute hash many times
	hashes := make([]TaskHash, 100)
	for i := 0; i < 100; i++ {
		hashes[i] = hasher.ComputeHash(input)
	}

	// All must be identical
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			t.Errorf("iteration %d produced different hash: %s != %s", i, hashes[i], hashes[0])
		}
	}
}

// TestComputeHash_HashFormat verifies hash is hex-encoded SHA256.
func TestComputeHash_HashFormat(t *testing.T) {
	hasher := NewTaskHasher()

	input := HashInput{
		Inputs:     &InputSet{Inputs: []Input{}},
		Command:    "test",
		Env:        map[string]string{},
		Outputs:    []string{},
		WorkingDir: "/",
	}

	hash := hasher.ComputeHash(input)

	// SHA256 produces 32 bytes = 64 hex characters
	if len(hash) != 64 {
		t.Errorf("expected 64 character hash, got %d", len(hash))
	}

	// Verify all characters are valid hex
	for _, c := range hash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("invalid hex character in hash: %c", c)
		}
	}
}
