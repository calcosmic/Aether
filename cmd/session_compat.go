package cmd

import "github.com/calcosmic/Aether/pkg/storage"

// ensureLegacySessionMirror is retained only so older orientation callers keep
// compiling while their migration branch ages out. Lifecycle readers are
// causally read-only: a missing top-level session is evidence to classify, not
// permission to create one from a guessed legacy candidate. Pause/resume can
// reconstruct a session only inside their declared lifecycle transaction.
func ensureLegacySessionMirror(_ *storage.Store) (bool, error) {
	return false, nil
}
