package dag

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

// computeTaskDefHash hashes only the declarative definition fields required by the
// DAG prompt: inputs, env, run.
//
// Determinism rules:
//   - Inputs are treated as a set for identity and thus sorted.
//   - Env map is sorted by key.
//   - All fields are length-prefixed to avoid ambiguity.
func computeTaskDefHash(inputs []string, env map[string]string, run string) TaskDefHash {
	h := sha256.New()

	writeField := func(data []byte) {
		length := uint64(len(data))
		lengthBytes := []byte{
			byte(length >> 56),
			byte(length >> 48),
			byte(length >> 40),
			byte(length >> 32),
			byte(length >> 24),
			byte(length >> 16),
			byte(length >> 8),
			byte(length),
		}
		h.Write(lengthBytes)
		h.Write(data)
	}

	// Inputs (sorted)
	sortedInputs := make([]string, len(inputs))
	copy(sortedInputs, inputs)
	sort.Strings(sortedInputs)
	writeField([]byte{byte(len(sortedInputs))})
	for _, in := range sortedInputs {
		writeField([]byte(in))
	}

	// Env (sorted)
	envKeys := make([]string, 0, len(env))
	for k := range env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	writeField([]byte{byte(len(envKeys))})
	for _, k := range envKeys {
		writeField([]byte(k))
		writeField([]byte(env[k]))
	}

	// Run
	writeField([]byte(run))

	sum := h.Sum(nil)
	return TaskDefHash(hex.EncodeToString(sum))
}
