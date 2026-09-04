package cmd

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/exchange"
)

// entombArchiveSource is one explicitly declared source/archive byte pair.
// Both paths are relative to roots supplied by the transaction so callers
// cannot smuggle an unrelated file into a chamber manifest.
type entombArchiveSource struct {
	SourcePath  string
	ArchivePath string
	Kind        string
	Required    bool
}

// entombArchiveManifestInput contains only immutable inputs. Building or
// verifying a manifest deliberately performs no publication or clearing.
type entombArchiveManifestInput struct {
	SourceRoot         string
	ArchiveRoot        string
	ArchiveID          string
	TransactionID      string
	ProjectionRevision string
	SealOutcome        colony.SealOutcome
	Sources            []entombArchiveSource
}

var entombRequiredArchiveKinds = []string{
	"archive_xml",
	"crowned_report",
	"findings",
	"learnings",
	"owner_checkpoints",
	"retained_memory",
	"rollback",
	"seal_outcome",
	"signals",
	"state",
	"tombstone_input",
}

var entombClosureArchiveKinds = map[string]bool{
	"archive_xml":       true,
	"crowned_report":    true,
	"findings":          true,
	"learnings":         true,
	"owner_checkpoints": true,
	"rollback":          true,
	"seal_outcome":      true,
	"signals":           true,
	"state":             true,
}

// buildEntombArchiveManifest hashes the already-staged archive alongside the
// active sealed sources and binds both copies to one closure identity.
func buildEntombArchiveManifest(input entombArchiveManifestInput) (colony.ArchiveManifest, []byte, error) {
	if strings.TrimSpace(input.ArchiveID) == "" {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("archive_id is required")
	}
	if strings.TrimSpace(input.TransactionID) == "" {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("transaction_id is required")
	}
	if strings.TrimSpace(input.ProjectionRevision) == "" {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("projection revision is required")
	}
	if err := input.SealOutcome.Validate(); err != nil {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("seal outcome: %w", err)
	}
	if input.SealOutcome.Transaction.Stage != colony.TransactionStageVerified {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("seal outcome transaction %q is not verified", input.SealOutcome.Transaction.ID)
	}

	sourceRoot, err := entombManifestRoot(input.SourceRoot, "source")
	if err != nil {
		return colony.ArchiveManifest{}, nil, err
	}
	archiveRoot, err := entombManifestRoot(input.ArchiveRoot, "archive")
	if err != nil {
		return colony.ArchiveManifest{}, nil, err
	}

	seenSources := make(map[string]bool, len(input.Sources))
	seenArchives := make(map[string]bool, len(input.Sources))
	seenKinds := make(map[string]int, len(input.Sources))
	entries := make([]colony.ArchiveEntry, 0, len(input.Sources))
	references := make([]colony.ArchiveCrossReference, 0, len(input.Sources)*2+8)
	evidence := make([]colony.LifecycleEvidence, 0, len(input.Sources))

	for _, source := range input.Sources {
		sourcePath, err := entombContainedManifestPath(sourceRoot, source.SourcePath, "source")
		if err != nil {
			return colony.ArchiveManifest{}, nil, err
		}
		archivePath, err := entombContainedManifestPath(archiveRoot, source.ArchivePath, "archive")
		if err != nil {
			return colony.ArchiveManifest{}, nil, err
		}
		kind := strings.TrimSpace(source.Kind)
		if kind == "" {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("archive source %q kind is required", source.SourcePath)
		}
		if seenSources[source.SourcePath] {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("duplicate source path %q", source.SourcePath)
		}
		if seenArchives[source.ArchivePath] {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("duplicate archive path %q", source.ArchivePath)
		}
		seenSources[source.SourcePath] = true
		seenArchives[source.ArchivePath] = true
		seenKinds[kind]++

		sourceBytes, sourceErr := readEntombManifestFile(sourceRoot, source.SourcePath, sourcePath)
		archiveBytes, archiveErr := readEntombManifestFile(archiveRoot, source.ArchivePath, archivePath)
		if sourceErr != nil || archiveErr != nil {
			if !source.Required && errors.Is(sourceErr, os.ErrNotExist) && errors.Is(archiveErr, os.ErrNotExist) {
				continue
			}
			if sourceErr != nil {
				return colony.ArchiveManifest{}, nil, entombManifestReadError("source", source.SourcePath, sourceErr)
			}
			return colony.ArchiveManifest{}, nil, entombManifestReadError("archive", source.ArchivePath, archiveErr)
		}

		sourceDigest := lifecycleDigest(sourceBytes)
		archiveDigest := lifecycleDigest(archiveBytes)
		if sourceDigest != archiveDigest || !bytes.Equal(sourceBytes, archiveBytes) {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("archive %q digest mismatch with source %q", source.ArchivePath, source.SourcePath)
		}
		if err := verifyEntombClosureArtifact(source.SourcePath, kind, sourceBytes, input.SealOutcome); err != nil {
			return colony.ArchiveManifest{}, nil, err
		}

		entries = append(entries, colony.ArchiveEntry{
			Path:          source.ArchivePath,
			Kind:          kind,
			Size:          int64(len(archiveBytes)),
			SourceDigest:  sourceDigest,
			ArchiveDigest: archiveDigest,
			Provenance:    colony.RecoveryProvenanceConfirmed,
		})
		references = append(references, colony.ArchiveCrossReference{
			Kind:     "source:" + kind,
			SourceID: source.SourcePath,
			TargetID: source.ArchivePath,
			Digest:   archiveDigest,
		})
		if entombClosureArchiveKinds[kind] {
			references = append(references, colony.ArchiveCrossReference{
				Kind:     "closure:" + kind,
				SourceID: input.SealOutcome.OutcomeID,
				TargetID: source.ArchivePath,
				Digest:   archiveDigest,
			})
		}
		evidence = append(evidence, colony.LifecycleEvidence{
			ID:     "archive:" + source.ArchivePath,
			Kind:   kind,
			Source: source.SourcePath,
			Digest: archiveDigest,
		})
	}

	for _, kind := range entombRequiredArchiveKinds {
		if seenKinds[kind] == 0 {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("required archive source kind %q is missing", kind)
		}
		if seenKinds[kind] > 1 {
			return colony.ArchiveManifest{}, nil, fmt.Errorf("duplicate archive source kind %q", kind)
		}
	}

	sealDigest := ""
	for _, entry := range entries {
		if entry.Kind == "seal_outcome" {
			sealDigest = entry.ArchiveDigest
			break
		}
	}
	references = append(references,
		colony.ArchiveCrossReference{
			Kind:     "seal_outcome",
			SourceID: input.ArchiveID,
			TargetID: input.SealOutcome.OutcomeID,
			Digest:   sealDigest,
		},
		colony.ArchiveCrossReference{
			Kind:     "seal_transaction",
			SourceID: input.SealOutcome.OutcomeID,
			TargetID: input.SealOutcome.Transaction.ID,
		},
		colony.ArchiveCrossReference{
			Kind:     "entomb_transaction",
			SourceID: input.ArchiveID,
			TargetID: input.TransactionID,
		},
		colony.ArchiveCrossReference{
			Kind:     "seal_disposition",
			SourceID: input.SealOutcome.OutcomeID,
			TargetID: string(input.SealOutcome.Disposition),
		},
	)
	if input.SealOutcome.Rollback != nil {
		references = append(references, colony.ArchiveCrossReference{
			Kind:     "rollback_checkpoint",
			SourceID: input.SealOutcome.OutcomeID,
			TargetID: input.SealOutcome.Rollback.CheckpointID,
		})
		for _, item := range input.SealOutcome.Rollback.Evidence {
			references = append(references, colony.ArchiveCrossReference{
				Kind:     "rollback_evidence",
				SourceID: input.SealOutcome.OutcomeID,
				TargetID: item.ID,
				Digest:   item.Digest,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Path == entries[j].Path {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Path < entries[j].Path
	})
	sort.Slice(references, func(i, j int) bool {
		left := references[i].Kind + "\x00" + references[i].SourceID + "\x00" + references[i].TargetID + "\x00" + references[i].Digest
		right := references[j].Kind + "\x00" + references[j].SourceID + "\x00" + references[j].TargetID + "\x00" + references[j].Digest
		return left < right
	})
	sort.Slice(evidence, func(i, j int) bool { return evidence[i].ID < evidence[j].ID })

	manifest := colony.ArchiveManifest{
		SchemaVersion:      colony.LifecycleSchemaVersion,
		ArchiveID:          input.ArchiveID,
		Command:            "entomb",
		OutcomeKind:        colony.OutcomeKindArchived,
		ProjectionRevision: input.ProjectionRevision,
		Seal: colony.SealOutcomeReference{
			OutcomeID:     input.SealOutcome.OutcomeID,
			Disposition:   input.SealOutcome.Disposition,
			TransactionID: input.SealOutcome.Transaction.ID,
			OwnerReason:   input.SealOutcome.OwnerReason,
		},
		Entries:         entries,
		CrossReferences: references,
		Evidence:        evidence,
		Verification: []colony.LifecycleVerification{{
			Name:        "archive bytes and cross-references",
			Passed:      true,
			EvidenceIDs: entombManifestEvidenceIDs(evidence),
		}},
		StateEffect: colony.LifecycleStateEffectCommitted,
		Transaction: colony.LifecycleTransactionReference{
			ID:    input.TransactionID,
			Stage: colony.TransactionStageVerified,
		},
		Receipt: &colony.LifecycleReceiptReference{
			ID: input.TransactionID + "-archive-receipt",
		},
		Provenance: colony.RecoveryProvenanceConfirmed,
	}
	manifest.ManifestDigest = lifecycleDigest(entombManifestDigestBytes(manifest))
	if err := manifest.Validate(); err != nil {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("archive manifest: %w", err)
	}
	canonical, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return colony.ArchiveManifest{}, nil, fmt.Errorf("marshal archive manifest: %w", err)
	}
	return manifest, append(canonical, '\n'), nil
}

// verifyEntombArchiveManifest proves the manifest still describes the current
// source and staged bytes. Directory existence alone is never sufficient.
func verifyEntombArchiveManifest(input entombArchiveManifestInput, manifest colony.ArchiveManifest) error {
	if err := manifest.Validate(); err != nil {
		return fmt.Errorf("archive manifest contract: %w", err)
	}
	if digest := lifecycleDigest(entombManifestDigestBytes(manifest)); digest != manifest.ManifestDigest {
		return fmt.Errorf("archive manifest digest mismatch: recorded %s, calculated %s", manifest.ManifestDigest, digest)
	}
	expected, expectedBytes, err := buildEntombArchiveManifest(input)
	if err != nil {
		return err
	}
	actualBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal archive manifest for verification: %w", err)
	}
	actualBytes = append(actualBytes, '\n')
	if !bytes.Equal(actualBytes, expectedBytes) || !reflect.DeepEqual(manifest, expected) {
		return fmt.Errorf("archive manifest does not match the deterministic source set and closure cross-references")
	}
	return nil
}

// entombManifestDigestBytes is the canonical non-self-referential digest
// payload. JSON struct field order is stable and all slices are pre-sorted.
func entombManifestDigestBytes(manifest colony.ArchiveManifest) []byte {
	manifest.ManifestDigest = ""
	content, _ := json.Marshal(manifest)
	return content
}

func entombManifestEvidenceIDs(evidence []colony.LifecycleEvidence) []string {
	ids := make([]string, 0, len(evidence))
	for _, item := range evidence {
		ids = append(ids, item.ID)
	}
	return ids
}

func entombManifestRoot(root, label string) (string, error) {
	if strings.TrimSpace(root) == "" {
		return "", fmt.Errorf("%s root is required", label)
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("resolve %s root: %w", label, err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect %s root %q: %w", label, root, err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("%s root %q is a symlink", label, root)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%s root %q is not a directory", label, root)
	}
	return filepath.Clean(absolute), nil
}

func entombContainedManifestPath(root, relative, label string) (string, error) {
	relative = strings.TrimSpace(relative)
	clean := path.Clean(relative)
	if relative == "" || strings.Contains(relative, "\\") || path.IsAbs(relative) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != relative {
		return "", fmt.Errorf("%s path %q must be a canonical contained relative path", label, relative)
	}
	return filepath.Join(root, filepath.FromSlash(clean)), nil
}

func readEntombManifestFile(root, relative, fullPath string) ([]byte, error) {
	components := strings.Split(relative, "/")
	cursor := root
	for _, component := range components {
		cursor = filepath.Join(cursor, component)
		info, err := os.Lstat(cursor)
		if err != nil {
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%q is a symlink", relative)
		}
	}
	return readLifecycleEvidenceFile(fullPath)
}

func entombManifestReadError(side, relative string, err error) error {
	if errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s %q is missing", side, relative)
	}
	return fmt.Errorf("read %s %q: %w", side, relative, err)
}

func verifyEntombClosureArtifact(relative, kind string, content []byte, expected colony.SealOutcome) error {
	switch kind {
	case "state":
		var state colony.ColonyState
		if err := decodeLifecycleJSON(content, &state); err != nil {
			return fmt.Errorf("source %q has invalid state JSON: %w", relative, err)
		}
		if state.State != colony.StateCOMPLETED || strings.TrimSpace(state.Milestone) != "Crowned Anthill" {
			return fmt.Errorf("source %q is not sealed Crowned Anthill state", relative)
		}
		if state.SealOutcome == nil {
			return fmt.Errorf("source %q is missing seal outcome_id", relative)
		}
		if err := verifyEntombSealIdentity(relative, *state.SealOutcome, expected); err != nil {
			return err
		}
	case "seal_outcome":
		var outcome colony.SealOutcome
		if err := decodeLifecycleJSON(content, &outcome); err != nil {
			return fmt.Errorf("source %q has invalid seal outcome JSON: %w", relative, err)
		}
		if err := outcome.Validate(); err != nil {
			return fmt.Errorf("source %q has invalid seal outcome: %w", relative, err)
		}
		if err := verifyEntombSealIdentity(relative, outcome, expected); err != nil {
			return err
		}
	case "findings", "learnings", "owner_checkpoints", "rollback":
		record, err := decodeEntombClosureRecord(content)
		if err != nil {
			return fmt.Errorf("source %q has invalid closure JSON: %w", relative, err)
		}
		if err := verifyEntombClosureRecord(relative, record, expected); err != nil {
			return err
		}
		if kind == "rollback" && expected.Disposition == colony.SealDispositionForcedIncomplete {
			var rollback struct {
				Rollback *colony.LifecycleRollback `json:"rollback"`
			}
			if err := json.Unmarshal(content, &rollback); err != nil || rollback.Rollback == nil || !reflect.DeepEqual(*rollback.Rollback, *expected.Rollback) {
				return fmt.Errorf("source %q rollback checkpoint/evidence does not match seal outcome", relative)
			}
		}
	case "crowned_report":
		if err := verifyEntombCrownedReport(relative, string(content), expected); err != nil {
			return err
		}
	case "archive_xml":
		var archive exchange.ColonyArchiveXML
		if err := xml.Unmarshal(content, &archive); err != nil {
			return fmt.Errorf("source %q has invalid archive XML: %w", relative, err)
		}
		if archive.XMLName.Local != "colony-archive" || strings.TrimSpace(archive.Version) == "" {
			return fmt.Errorf("source %q has invalid colony archive root/version", relative)
		}
		if archive.ColonyID != expected.OutcomeID {
			return fmt.Errorf("source %q colony_id %q does not match seal outcome_id %q", relative, archive.ColonyID, expected.OutcomeID)
		}
		if archive.Pheromones == nil || archive.Wisdom == nil || archive.Registry == nil {
			return fmt.Errorf("source %q is missing required archive XML sections", relative)
		}
	}
	return nil
}

func verifyEntombSealIdentity(relative string, actual, expected colony.SealOutcome) error {
	if actual.OutcomeID != expected.OutcomeID {
		return fmt.Errorf("source %q outcome_id %q does not match %q", relative, actual.OutcomeID, expected.OutcomeID)
	}
	if actual.Transaction.ID != expected.Transaction.ID {
		return fmt.Errorf("source %q transaction_id %q does not match %q", relative, actual.Transaction.ID, expected.Transaction.ID)
	}
	if actual.OutcomeKind != expected.OutcomeKind {
		return fmt.Errorf("source %q outcome_kind %q does not match %q", relative, actual.OutcomeKind, expected.OutcomeKind)
	}
	if actual.Disposition != expected.Disposition {
		return fmt.Errorf("source %q disposition %q does not match %q", relative, actual.Disposition, expected.Disposition)
	}
	if actual.OwnerReason != expected.OwnerReason {
		return fmt.Errorf("source %q owner reason does not match seal outcome", relative)
	}
	return nil
}

type entombClosureRecord struct {
	TransactionID string                 `json:"transaction_id"`
	OutcomeKind   colony.OutcomeKind     `json:"outcome_kind"`
	Disposition   colony.SealDisposition `json:"disposition"`
	OwnerReason   string                 `json:"owner_reason,omitempty"`
}

func decodeEntombClosureRecord(content []byte) (entombClosureRecord, error) {
	var record entombClosureRecord
	if err := json.Unmarshal(content, &record); err != nil {
		return record, err
	}
	if strings.TrimSpace(record.TransactionID) == "" {
		return record, fmt.Errorf("transaction_id is required")
	}
	if !record.OutcomeKind.Valid() {
		return record, fmt.Errorf("outcome_kind is invalid")
	}
	if !record.Disposition.Valid() {
		return record, fmt.Errorf("disposition is invalid")
	}
	return record, nil
}

func verifyEntombClosureRecord(relative string, actual entombClosureRecord, expected colony.SealOutcome) error {
	if actual.TransactionID != expected.Transaction.ID {
		return fmt.Errorf("source %q transaction_id %q does not match seal transaction %q", relative, actual.TransactionID, expected.Transaction.ID)
	}
	if actual.OutcomeKind != expected.OutcomeKind {
		return fmt.Errorf("source %q outcome_kind %q does not match seal outcome %q", relative, actual.OutcomeKind, expected.OutcomeKind)
	}
	if actual.Disposition != expected.Disposition {
		return fmt.Errorf("source %q disposition %q does not match seal disposition %q", relative, actual.Disposition, expected.Disposition)
	}
	if actual.OwnerReason != expected.OwnerReason {
		return fmt.Errorf("source %q owner reason does not match seal outcome", relative)
	}
	return nil
}

func verifyEntombCrownedReport(relative, report string, expected colony.SealOutcome) error {
	lower := strings.ToLower(report)
	if !strings.Contains(lower, "crowned") && !strings.Contains(lower, "forced seal record") {
		return fmt.Errorf("source %q is not a Crowned or forced seal report", relative)
	}
	forcedMarker := strings.Contains(lower, "completion not verified") || strings.Contains(lower, "forced seal record") || strings.Contains(lower, "forced_incomplete")
	if expected.Disposition == colony.SealDispositionForcedIncomplete {
		if !forcedMarker {
			return fmt.Errorf("source %q dropped the forced-incomplete marker", relative)
		}
		if !strings.Contains(report, expected.OwnerReason) {
			return fmt.Errorf("source %q dropped the forced owner reason", relative)
		}
		if !strings.Contains(report, expected.Transaction.ID) {
			return fmt.Errorf("source %q dropped the forced seal transaction_id", relative)
		}
		return nil
	}
	if forcedMarker {
		return fmt.Errorf("source %q disagrees with the verified seal disposition", relative)
	}
	if strings.Contains(report, "Outcome:") && !strings.Contains(report, expected.OutcomeID) {
		return fmt.Errorf("source %q outcome_id does not match seal outcome", relative)
	}
	if strings.Contains(report, "Transaction:") && !strings.Contains(report, expected.Transaction.ID) {
		return fmt.Errorf("source %q transaction_id does not match seal outcome", relative)
	}
	return nil
}
