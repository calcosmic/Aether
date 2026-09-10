package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Phase 201 plan 08 (Task 3, CAP-071) -- attempt-bound claims and
// verification artifact paths. Before this file, "last-build-claims.json"
// and "verification.json" were shared, colony-wide/phase-wide filenames:
// every new build attempt's claims overwrote the last one's, so two
// attempts of the same phase (an original run and its retry, for example)
// could never both keep their own evidence on disk at once. This file
// derives one canonical path per attempt for each artifact kind, and a
// compatibility reader that still finds a colony's pre-existing legacy
// root-level file -- validated by the identical rules an attempt-bound read
// uses, never a separate, more lenient path.

// attemptArtifactKindClaims and attemptArtifactKindVerification are the two
// artifact kinds this file binds to an exact attempt.
const (
	attemptArtifactKindClaims       = "claims"
	attemptArtifactKindVerification = "verification"
)

// legacyAttemptArtifactNames maps each artifact kind to the shared,
// root-level filename colonies written before attempt-bound paths existed.
// A legacy colony's evidence must remain readable -- see
// readAttemptBoundArtifact.
//
// attemptArtifactKindTelemetry (cmd/job_telemetry.go, plan 201-12) is
// registered here too, even though it has no genuine legacy colony-wide
// file -- job timing never existed before this plan. Its "legacy" name is
// simply a root-level filename that will never be found on a real colony,
// which lets it reuse this exact validation path rather than a second one
// (CLAUDE.md: "Two copies of a security boundary is one copy too many").
var legacyAttemptArtifactNames = map[string]string{
	attemptArtifactKindClaims:       "last-build-claims.json",
	attemptArtifactKindVerification: "verification.json",
	attemptArtifactKindTelemetry:    "job-telemetry.json",
}

// attemptBoundArtifactPath derives the one canonical store-relative path for
// an attempt's claims or verification artifact:
// "attempts/<attemptID>/<kind>.json". attemptID must be a bare identifier
// (no path separators, no "." or ".." segment) -- it is untrusted input
// read back off a build attempt record, and a crafted identifier must never
// be able to walk the resulting path outside the attempt's own directory.
// The candidate is additionally validated for containment within the
// colony's data root by resolving symlinks on the deepest EXISTING ancestor
// and rejecting anything that resolves outside it -- reusing
// validateSpendContainedPath's containment discipline
// (cmd/spend_session_capture.go) rather than a second copy of that rule
// (CLAUDE.md: "Two copies of a security boundary is one copy too many").
func attemptBoundArtifactPath(attemptID, kind string) (string, error) {
	attemptID = strings.TrimSpace(attemptID)
	kind = strings.TrimSpace(kind)
	if attemptID == "" {
		return "", fmt.Errorf("attempt artifact path: attempt id is empty")
	}
	if _, known := legacyAttemptArtifactNames[kind]; !known {
		return "", fmt.Errorf("attempt artifact path: unknown artifact kind %q", kind)
	}
	if strings.ContainsAny(attemptID, "/\\") || attemptID != filepath.Base(filepath.Clean(attemptID)) ||
		attemptID == "." || attemptID == ".." || strings.ContainsRune(attemptID, 0) {
		return "", fmt.Errorf("attempt artifact path: attempt id %q must be a bare identifier, not a path", attemptID)
	}

	rel := filepath.ToSlash(filepath.Join("attempts", attemptID, kind+".json"))
	canonical, err := canonicalBuildAttemptDataPath(rel)
	if err != nil {
		return "", fmt.Errorf("attempt artifact path: %w", err)
	}
	if store == nil {
		return "", fmt.Errorf("attempt artifact path: store is not initialized")
	}
	root := store.BasePath()
	candidateAbs := filepath.Join(root, filepath.FromSlash(canonical))
	if _, err := validateSpendContainedPath(root, candidateAbs, "attempt artifact path"); err != nil {
		return "", err
	}
	return canonical, nil
}

// writeAttemptBoundArtifact writes value as kind's artifact for attemptID at
// its attempt-bound path. It never touches the legacy root-level filename --
// existing pre-migration files are left exactly where they are (CAP-071);
// this is purely additive.
func writeAttemptBoundArtifact(attemptID, kind string, value any) error {
	if store == nil {
		return fmt.Errorf("attempt artifact: store is not initialized")
	}
	rel, err := attemptBoundArtifactPath(attemptID, kind)
	if err != nil {
		return err
	}
	return store.SaveJSON(rel, value)
}

// readAttemptBoundArtifact reads kind's artifact for attemptID. It first
// tries the attempt-bound path; when that file does not exist -- a colony
// written before this change -- it falls back to the shared legacy
// root-level filename for that kind. Both branches decode through the
// identical store.LoadJSON call: there is no separate, more lenient legacy
// path. A malformed legacy file is refused with the same error an equally
// malformed attempt-bound file would produce, and a missing or unreadable
// legacy file is refused rather than silently treated as an empty, safe
// value.
func readAttemptBoundArtifact(attemptID, kind string, out any) error {
	if store == nil {
		return fmt.Errorf("attempt artifact: store is not initialized")
	}
	rel, err := attemptBoundArtifactPath(attemptID, kind)
	if err != nil {
		return err
	}
	loadErr := store.LoadJSON(rel, out)
	if loadErr == nil {
		return nil
	}
	if !errors.Is(loadErr, os.ErrNotExist) {
		return fmt.Errorf("attempt artifact %q: %w", rel, loadErr)
	}
	legacyName, known := legacyAttemptArtifactNames[strings.TrimSpace(kind)]
	if !known {
		return fmt.Errorf("attempt artifact: unknown artifact kind %q", kind)
	}
	if err := store.LoadJSON(legacyName, out); err != nil {
		return fmt.Errorf("attempt artifact %q (legacy %q): %w", rel, legacyName, err)
	}
	return nil
}
