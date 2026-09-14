package shadow

import "crypto/sha256"

// Baseline is the current policy value, frozen the same way FrozenEvaluator
// is: unexported fields, a constructor, a digest, accessor methods only. A
// comparison may not proceed when the baseline's digest changed during it
// (Compare checks this at the end of every run rather than assuming it,
// see comparison.go).
type Baseline struct {
	value  []byte
	digest [32]byte
}

// NewBaseline builds a Baseline from value, the current policy's own byte
// representation. The digest is computed once, here, with crypto/sha256.
func NewBaseline(value []byte) Baseline {
	stored := make([]byte, len(value))
	copy(stored, value)
	return Baseline{value: stored, digest: sha256.Sum256(value)}
}

// Digest returns a copy of the baseline's identity.
func (b Baseline) Digest() [32]byte {
	return b.digest
}

// Value returns a copy of the baseline's own byte representation -- a copy,
// never the baseline's own backing array, so a caller cannot mutate what
// NewBaseline was built from.
func (b Baseline) Value() []byte {
	v := make([]byte, len(b.value))
	copy(v, b.value)
	return v
}
