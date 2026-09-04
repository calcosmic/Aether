package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

type entombManifestFixture199 struct {
	input entombArchiveManifestInput
}

func TestEntombManifest199Build(t *testing.T) {
	fixture := newEntombManifestFixture199(t, colony.SealDispositionVerified)

	manifest, canonical, err := buildEntombArchiveManifest(fixture.input)
	if err != nil {
		t.Fatalf("build manifest: %v", err)
	}
	if err := manifest.Validate(); err != nil {
		t.Fatalf("manifest contract: %v", err)
	}
	if len(canonical) == 0 || !bytes.HasSuffix(canonical, []byte("\n")) {
		t.Fatalf("canonical manifest must be non-empty newline-terminated JSON: %q", canonical)
	}
	if manifest.ManifestDigest != lifecycleDigest(entombManifestDigestBytes(manifest)) {
		t.Fatalf("manifest digest %q does not cover canonical manifest bytes", manifest.ManifestDigest)
	}

	wantKinds := map[string]bool{
		"state": false, "crowned_report": false, "archive_xml": false,
		"seal_outcome": false, "findings": false, "learnings": false,
		"signals": false, "owner_checkpoints": false, "rollback": false,
		"retained_memory": false, "tombstone_input": false,
	}
	for _, entry := range manifest.Entries {
		if filepath.IsAbs(entry.Path) || entry.Size <= 0 || entry.SourceDigest == "" || entry.ArchiveDigest != entry.SourceDigest {
			t.Fatalf("entry lacks contained byte proof: %+v", entry)
		}
		if _, ok := wantKinds[entry.Kind]; ok {
			wantKinds[entry.Kind] = true
		}
	}
	for kind, found := range wantKinds {
		if !found {
			t.Errorf("manifest is missing required source kind %q", kind)
		}
	}
	for _, source := range fixture.input.Sources {
		if !entombManifestHasReference(manifest, "source:"+source.Kind, source.SourcePath, source.ArchivePath) {
			t.Errorf("manifest lacks source cross-reference for %s", source.ArchivePath)
		}
	}
}

func TestEntombManifest199Deterministic(t *testing.T) {
	fixture := newEntombManifestFixture199(t, colony.SealDispositionVerified)
	first, firstBytes, err := buildEntombArchiveManifest(fixture.input)
	if err != nil {
		t.Fatal(err)
	}

	reversed := fixture.input
	reversed.Sources = append([]entombArchiveSource(nil), fixture.input.Sources...)
	for left, right := 0, len(reversed.Sources)-1; left < right; left, right = left+1, right-1 {
		reversed.Sources[left], reversed.Sources[right] = reversed.Sources[right], reversed.Sources[left]
	}
	second, secondBytes, err := buildEntombArchiveManifest(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if first.ManifestDigest != second.ManifestDigest || !bytes.Equal(firstBytes, secondBytes) {
		t.Fatalf("unchanged inputs were not canonical\nfirst=%s\nsecond=%s", firstBytes, secondBytes)
	}
}

func TestEntombManifest199Tamper(t *testing.T) {
	fixture := newEntombManifestFixture199(t, colony.SealDispositionVerified)
	manifest, _, err := buildEntombArchiveManifest(fixture.input)
	if err != nil {
		t.Fatal(err)
	}

	statePath := filepath.Join(fixture.input.ArchiveRoot, "COLONY_STATE.json")
	stateBytes, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	stateBytes[len(stateBytes)-2] ^= 1
	if err := os.WriteFile(statePath, stateBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyEntombArchiveManifest(fixture.input, manifest); err == nil || !strings.Contains(err.Error(), "COLONY_STATE.json") || !strings.Contains(err.Error(), "digest") {
		t.Fatalf("one-byte tamper was not named by path and reason: %v", err)
	}

	fixture = newEntombManifestFixture199(t, colony.SealDispositionVerified)
	manifest, _, err = buildEntombArchiveManifest(fixture.input)
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(fixture.input.ArchiveRoot, "seal", "findings.json")
	if err := os.Remove(missing); err != nil {
		t.Fatal(err)
	}
	if err := verifyEntombArchiveManifest(fixture.input, manifest); err == nil || !strings.Contains(err.Error(), "seal/findings.json") || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("missing archive source was not named by path and reason: %v", err)
	}
}

func TestEntombManifest199CrossReferences(t *testing.T) {
	fixture := newEntombManifestFixture199(t, colony.SealDispositionVerified)
	wrong := fixture.input.SealOutcome
	wrong.OutcomeID = "wrong-closure"
	writeEntombManifestFixtureFile199(t, fixture.input.SourceRoot, ".aether/data/COLONY_STATE.json", mustEntombManifestJSON199(t, colony.ColonyState{
		State: colony.StateCOMPLETED, Milestone: "Crowned Anthill", SealOutcome: &wrong,
	}))
	copyEntombManifestFixtureFile199(t, fixture.input.SourceRoot, fixture.input.ArchiveRoot, ".aether/data/COLONY_STATE.json", "COLONY_STATE.json")
	if _, _, err := buildEntombArchiveManifest(fixture.input); err == nil || !strings.Contains(err.Error(), "COLONY_STATE.json") || !strings.Contains(err.Error(), "outcome_id") {
		t.Fatalf("wrong closure identity was not refused with a named source: %v", err)
	}

	fixture = newEntombManifestFixture199(t, colony.SealDispositionVerified)
	duplicate := fixture.input
	duplicate.Sources = append(append([]entombArchiveSource(nil), fixture.input.Sources...), fixture.input.Sources[0])
	if _, _, err := buildEntombArchiveManifest(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate path was not refused: %v", err)
	}

	abs := fixture.input
	abs.Sources = append([]entombArchiveSource(nil), fixture.input.Sources...)
	abs.Sources[0].SourcePath = filepath.Join(fixture.input.SourceRoot, ".aether", "data", "COLONY_STATE.json")
	if _, _, err := buildEntombArchiveManifest(abs); err == nil || !strings.Contains(err.Error(), "relative") {
		t.Fatalf("absolute source path was not refused: %v", err)
	}

	symlinkTarget := filepath.Join(fixture.input.SourceRoot, "outside-state.json")
	if err := os.WriteFile(symlinkTarget, []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	symlinkPath := filepath.Join(fixture.input.SourceRoot, ".aether", "data", "linked-state.json")
	if err := os.Symlink(symlinkTarget, symlinkPath); err == nil {
		symlinked := fixture.input
		symlinked.Sources = append([]entombArchiveSource(nil), fixture.input.Sources...)
		symlinked.Sources[0].SourcePath = ".aether/data/linked-state.json"
		if _, _, err := buildEntombArchiveManifest(symlinked); err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Fatalf("symlink source was not refused: %v", err)
		}
	}
}

func TestEntombManifest199ForcedMarker(t *testing.T) {
	fixture := newEntombManifestFixture199(t, colony.SealDispositionForcedIncomplete)
	manifest, canonical, err := buildEntombArchiveManifest(fixture.input)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Seal.Disposition != colony.SealDispositionForcedIncomplete || manifest.Seal.OwnerReason != fixture.input.SealOutcome.OwnerReason {
		t.Fatalf("forced marker missing from manifest: %+v", manifest.Seal)
	}
	for _, marker := range []string{"forced_incomplete", fixture.input.SealOutcome.OwnerReason} {
		if !bytes.Contains(canonical, []byte(marker)) {
			t.Fatalf("canonical manifest omitted forced marker %q\n%s", marker, canonical)
		}
	}

	report := []byte("# Forced seal record — completion not verified\n")
	writeEntombManifestFixtureFile199(t, fixture.input.SourceRoot, ".aether/CROWNED-ANTHILL.md", report)
	copyEntombManifestFixtureFile199(t, fixture.input.SourceRoot, fixture.input.ArchiveRoot, ".aether/CROWNED-ANTHILL.md", "CROWNED-ANTHILL.md")
	if _, _, err := buildEntombArchiveManifest(fixture.input); err == nil || !strings.Contains(err.Error(), "CROWNED-ANTHILL.md") || !strings.Contains(err.Error(), "owner reason") {
		t.Fatalf("dropped forced marker was not refused with a named reason: %v", err)
	}
}

func newEntombManifestFixture199(t *testing.T, disposition colony.SealDisposition) entombManifestFixture199 {
	t.Helper()
	root := t.TempDir()
	sourceRoot := filepath.Join(root, "source")
	archiveRoot := filepath.Join(root, "archive")
	for _, path := range []string{sourceRoot, archiveRoot} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	outcome := entombManifestSealOutcome199(disposition)
	state := colony.ColonyState{State: colony.StateCOMPLETED, Milestone: "Crowned Anthill", SealOutcome: &outcome}
	closure := map[string]any{
		"schema_version": colony.LifecycleSchemaVersion,
		"transaction_id": outcome.Transaction.ID,
		"outcome_kind":   outcome.OutcomeKind,
		"disposition":    outcome.Disposition,
		"owner_reason":   outcome.OwnerReason,
	}
	rollback := make(map[string]any, len(closure)+1)
	for key, value := range closure {
		rollback[key] = value
	}
	rollback["rollback"] = outcome.Rollback
	report := "# Crowned Anthill\n\nOutcome: " + outcome.OutcomeID + "\nTransaction: " + outcome.Transaction.ID + "\nDisposition: " + string(outcome.Disposition) + "\n"
	if disposition == colony.SealDispositionForcedIncomplete {
		report = "# Forced seal record — completion not verified\n\nOutcome: " + outcome.OutcomeID + "\nTransaction: " + outcome.Transaction.ID + "\nDisposition: forced_incomplete\nOwner reason: " + outcome.OwnerReason + "\n"
	}

	files := map[string][]byte{
		".aether/data/COLONY_STATE.json":         mustEntombManifestJSON199(t, state),
		".aether/CROWNED-ANTHILL.md":             []byte(report),
		".aether/data/seal/outcome.json":         mustEntombManifestJSON199(t, outcome),
		".aether/data/seal/findings.json":        mustEntombManifestJSON199(t, closure),
		".aether/data/seal/learnings.json":       mustEntombManifestJSON199(t, closure),
		".aether/data/pheromones.json":           []byte("{\"version\":\"2.0\",\"signals\":[]}\n"),
		".aether/data/seal/checkpoints.json":     mustEntombManifestJSON199(t, closure),
		".aether/data/seal/rollback.json":        mustEntombManifestJSON199(t, rollback),
		".aether/QUEEN.md":                       []byte("# Retained colony memory\n"),
		".aether/HANDOFF.md":                     []byte("# Tombstone input\n"),
		".aether/data/entomb/colony-archive.xml": []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<colony-archive colony_id=\"" + outcome.OutcomeID + "\" sealed_at=\"2026-09-04T00:00:00Z\" version=\"1.0\"><pheromones version=\"1.0\" count=\"0\"></pheromones><queen-wisdom version=\"1.0\"></queen-wisdom><colony-registry version=\"1.0\"></colony-registry></colony-archive>"),
	}
	for path, content := range files {
		writeEntombManifestFixtureFile199(t, sourceRoot, path, content)
	}

	sources := []entombArchiveSource{
		{SourcePath: ".aether/data/COLONY_STATE.json", ArchivePath: "COLONY_STATE.json", Kind: "state", Required: true},
		{SourcePath: ".aether/CROWNED-ANTHILL.md", ArchivePath: "CROWNED-ANTHILL.md", Kind: "crowned_report", Required: true},
		{SourcePath: ".aether/data/entomb/colony-archive.xml", ArchivePath: "colony-archive.xml", Kind: "archive_xml", Required: true},
		{SourcePath: ".aether/data/seal/outcome.json", ArchivePath: "seal/outcome.json", Kind: "seal_outcome", Required: true},
		{SourcePath: ".aether/data/seal/findings.json", ArchivePath: "seal/findings.json", Kind: "findings", Required: true},
		{SourcePath: ".aether/data/seal/learnings.json", ArchivePath: "seal/learnings.json", Kind: "learnings", Required: true},
		{SourcePath: ".aether/data/pheromones.json", ArchivePath: "pheromones.json", Kind: "signals", Required: true},
		{SourcePath: ".aether/data/seal/checkpoints.json", ArchivePath: "seal/checkpoints.json", Kind: "owner_checkpoints", Required: true},
		{SourcePath: ".aether/data/seal/rollback.json", ArchivePath: "seal/rollback.json", Kind: "rollback", Required: true},
		{SourcePath: ".aether/QUEEN.md", ArchivePath: "QUEEN.md", Kind: "retained_memory", Required: true},
		{SourcePath: ".aether/HANDOFF.md", ArchivePath: "HANDOFF.md", Kind: "tombstone_input", Required: true},
	}
	for _, source := range sources {
		copyEntombManifestFixtureFile199(t, sourceRoot, archiveRoot, source.SourcePath, source.ArchivePath)
	}
	return entombManifestFixture199{input: entombArchiveManifestInput{
		SourceRoot: sourceRoot, ArchiveRoot: archiveRoot, ArchiveID: "archive-199", TransactionID: "entomb-199",
		ProjectionRevision: LifecycleProjectionRevision, SealOutcome: outcome, Sources: sources,
	}}
}

func entombManifestSealOutcome199(disposition colony.SealDisposition) colony.SealOutcome {
	outcome := colony.SealOutcome{
		SchemaVersion: colony.LifecycleSchemaVersion, OutcomeID: "seal-outcome-199", Command: "seal",
		OutcomeKind: colony.OutcomeKindVerifiedCompletion, ProjectionRevision: LifecycleProjectionRevision,
		Disposition: disposition, CompletedPhases: []int{1}, StateEffect: colony.LifecycleStateEffectCommitted,
		Transaction: colony.LifecycleTransactionReference{ID: "seal-transaction-199", Stage: colony.TransactionStageVerified},
		Receipt:     &colony.LifecycleReceiptReference{ID: "seal-receipt-199", Digest: "sha256:seal-receipt"},
		Provenance:  colony.RecoveryProvenanceConfirmed,
	}
	if disposition == colony.SealDispositionForcedIncomplete {
		outcome.OutcomeKind = colony.OutcomeKindForcedIncompleteClosure
		outcome.OwnerReason = "Owner accepted the unresolved release window"
		outcome.IncompletePhases = []int{2}
		outcome.UnresolvedEvidence = []colony.LifecycleEvidence{{ID: "unresolved-phase-2"}}
		outcome.Rollback = &colony.LifecycleRollback{CheckpointID: "pre-seal-checkpoint", Evidence: []colony.LifecycleEvidence{{ID: "rollback-evidence"}}}
	}
	return outcome
}

func writeEntombManifestFixtureFile199(t *testing.T, root, path string, content []byte) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, content, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyEntombManifestFixtureFile199(t *testing.T, sourceRoot, archiveRoot, sourcePath, archivePath string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(sourceRoot, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	writeEntombManifestFixtureFile199(t, archiveRoot, archivePath, content)
}

func mustEntombManifestJSON199(t *testing.T, value any) []byte {
	t.Helper()
	content, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(content, '\n')
}

func entombManifestHasReference(manifest colony.ArchiveManifest, kind, source, target string) bool {
	for _, reference := range manifest.CrossReferences {
		if reference.Kind == kind && reference.SourceID == source && reference.TargetID == target {
			return true
		}
	}
	return false
}
