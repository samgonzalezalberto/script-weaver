package trace

import (
	"crypto/sha256"
	"encoding/hex"
)

// ComputeTraceHash computes the deterministic TraceHash of a canonical trace encoding.
//
// Requirements (sprint-03 trace-engine):
//   - Must cover the canonical sorted order of events (not insertion order).
//   - Must be stable across architectures/compilers.
//
// This function assumes the input bytes are already a canonical encoding (e.g., from ExecutionTrace.CanonicalJSON()).
//
// Hash function:
//   - sha256 over the canonical bytes, hex-encoded.
//
// Note: While sha256 is cryptographic, it is explicitly allowed by the sprint prompt and is widely standardized.
func ComputeTraceHash(canonicalEncoding []byte) string {
	if len(canonicalEncoding) == 0 {
		return ""
	}
	sum := sha256.Sum256(canonicalEncoding)
	return hex.EncodeToString(sum[:])
}
