package cmd

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

const archiveMaintenanceInspectionSchemaVersion = "archive-inspection/v1"

// archiveMaintenanceInspectRequest names the exact roots and chamber that a
// read-only archive inspection may observe. ContextPath and TombstonePath are
// optional overrides used by isolated verification; the active archive
// reference otherwise determines whether the live context belongs to this
// chamber.
type archiveMaintenanceInspectRequest struct {
	RepositoryRoot string
	DataRoot       string
	ChamberPath    string
	ContextPath    string
	TombstonePath  string
	RequireContext bool
}

type archiveMaintenanceDigestEvidence struct {
	Path             string `json:"path"`
	Kind             string `json:"kind"`
	Digest           string `json:"digest"`
	Size             int64  `json:"size"`
	SourceReference  bool   `json:"source_reference"`
	ClosureReference bool   `json:"closure_reference,omitempty"`
	Status           string `json:"status"`
}

// archiveMaintenanceInspectionResult is shared by chamber verification,
// context repair preflight, and the expert integrity view. The unexported
// manifest is retained only so a subsequent pure repair planner can validate
// its exact closed target set; rendering uses the explicit evidence fields.
type archiveMaintenanceInspectionResult struct {
	SchemaVersion           string                             `json:"schema_version"`
	OperationID             string                             `json:"operation_id"`
	ChamberPath             string                             `json:"chamber_path"`
	ManifestPath            string                             `json:"manifest_path"`
	ManifestDigest          string                             `json:"manifest_digest,omitempty"`
	BaselineDigest          string                             `json:"baseline_digest"`
	SealDisposition         colony.SealDisposition             `json:"seal_disposition,omitempty"`
	OwnerReason             string                             `json:"owner_reason,omitempty"`
	ManifestVerified        bool                               `json:"manifest_verified"`
	ContentVerified         bool                               `json:"content_verified"`
	CrossReferencesVerified bool                               `json:"cross_references_verified"`
	ContextVerified         bool                               `json:"context_verified"`
	EntriesChecked          int                                `json:"entries_checked"`
	ReferencesChecked       int                                `json:"references_checked"`
	DigestEvidence          []archiveMaintenanceDigestEvidence `json:"digest_evidence"`
	Findings                []maintenanceInspectionFinding     `json:"findings"`
	Evidence                []maintenanceInspectionEvidence    `json:"evidence"`
	Verification            maintenanceInspectionVerification  `json:"verification"`
	IntegrityLine           string                             `json:"integrity_line"`
	StateEffect             colony.LifecycleStateEffect        `json:"state_effect"`
	Valid                   bool                               `json:"valid"`

	manifest colony.ArchiveManifest
}

var chamberCreateCmd = &cobra.Command{
	Use:   "chamber-create",
	Short: "Create a chamber archive entry",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}
		goal, _ := cmd.Flags().GetString("goal")
		milestone, _ := cmd.Flags().GetString("milestone")
		phasesCompleted, _ := cmd.Flags().GetInt("phases-completed")
		totalPhases, _ := cmd.Flags().GetInt("total-phases")

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chambersBase := filepath.Join(aetherRoot, ".aether", "chambers")
		chamberDir, err := safeIdentifierSegment(chambersBase, "chamber name", name)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		if err := os.MkdirAll(chamberDir, 0755); err != nil {
			outputError(2, fmt.Sprintf("failed to create chamber directory: %v", err), nil)
			return nil
		}

		manifest := map[string]interface{}{
			"name":             name,
			"goal":             goal,
			"milestone":        milestone,
			"phases_completed": phasesCompleted,
			"total_phases":     totalPhases,
		}

		manifestData, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			outputError(2, fmt.Sprintf("failed to marshal manifest: %v", err), nil)
			return nil
		}
		manifestData = append(manifestData, '\n')

		manifestPath := filepath.Join(chamberDir, "manifest.json")
		if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
			outputError(2, fmt.Sprintf("failed to write manifest: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"created": true,
			"name":    name,
			"path":    chamberDir,
		})
		return nil
	},
}

var chamberVerifyCmd = &cobra.Command{
	Use:         "chamber-verify",
	Short:       "Verify chamber integrity",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true", "aether.io/store-free": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chambersBase := filepath.Join(aetherRoot, ".aether", "chambers")
		chamberDir, err := safeIdentifierSegment(chambersBase, "chamber name", name)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}

		manifestPath := filepath.Join(chamberDir, "manifest.json")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			outputError(1, fmt.Sprintf("chamber %q not found: %v", name, err), nil)
			return nil
		}

		if !json.Valid(data) {
			outputError(1, fmt.Sprintf("chamber %q has invalid manifest.json", name), nil)
			return nil
		}
		var header struct {
			SchemaVersion string `json:"schema_version"`
		}
		if json.Unmarshal(data, &header) == nil && header.SchemaVersion == colony.LifecycleSchemaVersion {
			dataRoot := filepath.Join(aetherRoot, ".aether", "data")
			if configured := strings.TrimSpace(os.Getenv("COLONY_DATA_DIR")); configured != "" {
				dataRoot = filepath.Clean(configured)
			}
			inspection, inspectErr := inspectArchiveMaintenance(archiveMaintenanceInspectRequest{
				RepositoryRoot: filepath.Clean(aetherRoot),
				DataRoot:       dataRoot,
				ChamberPath:    chamberDir,
			})
			if inspectErr != nil {
				outputError(1, inspectErr.Error(), map[string]interface{}{"name": name, "valid": false, "state_effect": colony.LifecycleStateEffectNone})
				return nil
			}
			outputOK(inspection)
			return nil
		}

		// List files in chamber directory
		entries, err := os.ReadDir(chamberDir)
		if err != nil {
			outputError(1, fmt.Sprintf("failed to read chamber directory: %v", err), nil)
			return nil
		}

		files := make([]string, 0, len(entries))
		for _, e := range entries {
			files = append(files, e.Name())
		}
		sort.Strings(files)

		outputOK(map[string]interface{}{
			"name":  name,
			"valid": true,
			"files": files,
		})
		return nil
	},
}

// inspectArchiveMaintenance performs a causally read-only verification. It
// deliberately never creates a directory, lock, journal, or repair artifact;
// malformed archive bytes are returned as typed findings so the caller can
// show exact refusal evidence.
func inspectArchiveMaintenance(request archiveMaintenanceInspectRequest) (archiveMaintenanceInspectionResult, error) {
	result := archiveMaintenanceInspectionResult{
		SchemaVersion:  archiveMaintenanceInspectionSchemaVersion,
		OperationID:    "archive.inspect",
		StateEffect:    colony.LifecycleStateEffectNone,
		DigestEvidence: []archiveMaintenanceDigestEvidence{},
		Findings:       []maintenanceInspectionFinding{},
		Evidence:       []maintenanceInspectionEvidence{},
	}
	repositoryRoot, err := entombManifestRoot(request.RepositoryRoot, "repository")
	if err != nil {
		return result, err
	}
	chamberPath := filepath.Clean(strings.TrimSpace(request.ChamberPath))
	if chamberPath == "." || !filepath.IsAbs(chamberPath) || chamberPath != strings.TrimSpace(request.ChamberPath) {
		return result, fmt.Errorf("archive inspection: chamber path must be a canonical absolute path")
	}
	chambersRoot := filepath.Join(repositoryRoot, ".aether", "chambers")
	if chamberPath == chambersRoot || !pathIsWithin(chambersRoot, chamberPath) {
		return result, fmt.Errorf("archive inspection: chamber path %q is outside %q", chamberPath, chambersRoot)
	}
	if info, statErr := os.Lstat(chamberPath); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return result, fmt.Errorf("archive inspection: chamber path must be a real directory")
		}
	} else if !os.IsNotExist(statErr) {
		return result, fmt.Errorf("archive inspection: inspect chamber: %w", statErr)
	}

	dataRoot := strings.TrimSpace(request.DataRoot)
	if dataRoot == "" {
		dataRoot = filepath.Join(repositoryRoot, ".aether", "data")
	}
	dataRoot = filepath.Clean(dataRoot)
	if !filepath.IsAbs(dataRoot) || dataRoot != strings.TrimSpace(dataRoot) {
		return result, fmt.Errorf("archive inspection: lifecycle data root must be a canonical absolute path")
	}
	if info, statErr := os.Lstat(dataRoot); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return result, fmt.Errorf("archive inspection: lifecycle data root must be a real directory")
		}
	} else if !os.IsNotExist(statErr) {
		return result, fmt.Errorf("archive inspection: inspect lifecycle data root: %w", statErr)
	}

	result.ChamberPath = chamberPath
	result.ManifestPath = filepath.Join(chamberPath, "manifest.json")
	baseline := []string{}
	manifestBytes, readErr := readEntombManifestFile(chamberPath, "manifest.json", result.ManifestPath)
	if readErr != nil {
		addArchiveInspectionFinding(&result, "manifest_missing", fmt.Sprintf("archive manifest is unavailable: %v", readErr), result.ManifestPath)
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive manifest", Paths: []string{result.ManifestPath}, Checked: 1, Status: "fail"})
		baseline = append(baseline, result.ManifestPath+"="+lifecycleTransactionMissingDigest)
		return finalizeArchiveMaintenanceInspection(result, baseline), nil
	}
	manifestBytesDigest := lifecycleDigest(manifestBytes)
	baseline = append(baseline, result.ManifestPath+"="+manifestBytesDigest)
	if err := decodeLifecycleJSON(manifestBytes, &result.manifest); err != nil {
		addArchiveInspectionFinding(&result, "manifest_contract", fmt.Sprintf("archive manifest JSON is invalid: %v", err), result.ManifestPath)
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive manifest", Paths: []string{result.ManifestPath}, Checked: 1, Status: "fail"})
		return finalizeArchiveMaintenanceInspection(result, baseline), nil
	}
	manifest := result.manifest
	result.ManifestDigest = manifest.ManifestDigest
	result.SealDisposition = manifest.Seal.Disposition
	result.OwnerReason = manifest.Seal.OwnerReason
	if err := manifest.Validate(); err != nil {
		addArchiveInspectionFinding(&result, "manifest_contract", fmt.Sprintf("archive manifest contract is invalid: %v", err), result.ManifestPath)
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive manifest", Paths: []string{result.ManifestPath}, Checked: 1, Status: "fail"})
		return finalizeArchiveMaintenanceInspection(result, baseline), nil
	}
	calculatedManifestDigest := lifecycleDigest(entombManifestDigestBytes(manifest))
	if calculatedManifestDigest != manifest.ManifestDigest {
		addArchiveInspectionFinding(&result, "manifest_digest", fmt.Sprintf("manifest digest mismatch: recorded %s, calculated %s", manifest.ManifestDigest, calculatedManifestDigest), result.ManifestPath)
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive manifest digest", Paths: []string{result.ManifestPath}, Checked: 1, Status: "fail"})
		return finalizeArchiveMaintenanceInspection(result, baseline), nil
	}
	result.ManifestVerified = true
	result.ContentVerified = true
	result.CrossReferencesVerified = true
	result.ContextVerified = true
	result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive manifest digest", Paths: []string{result.ManifestPath}, Checked: 1, Status: "pass"})

	entryContent := make(map[string][]byte, len(manifest.Entries))
	kindCounts := make(map[string]int, len(manifest.Entries))
	for _, entry := range manifest.Entries {
		result.EntriesChecked++
		kindCounts[entry.Kind]++
		entryPath, containErr := entombContainedManifestPath(chamberPath, entry.Path, "published archive")
		evidence := archiveMaintenanceDigestEvidence{Path: entryPath, Kind: entry.Kind, Digest: entry.ArchiveDigest, Size: entry.Size, Status: "pass"}
		if containErr != nil {
			entryPath = filepath.Join(chamberPath, filepath.FromSlash(entry.Path))
			evidence.Path = entryPath
			evidence.Status = "fail"
			result.ContentVerified = false
			addArchiveInspectionFinding(&result, "content_path", containErr.Error(), entryPath)
			result.DigestEvidence = append(result.DigestEvidence, evidence)
			continue
		}
		content, contentErr := readEntombManifestFile(chamberPath, entry.Path, entryPath)
		if contentErr != nil {
			evidence.Status = "fail"
			result.ContentVerified = false
			addArchiveInspectionFinding(&result, "content_missing", fmt.Sprintf("archive content %q is unavailable: %v", entry.Path, contentErr), entryPath)
			baseline = append(baseline, entryPath+"="+lifecycleTransactionMissingDigest)
		} else {
			actualDigest := lifecycleDigest(content)
			baseline = append(baseline, entryPath+"="+actualDigest)
			entryContent[entry.Path] = content
			if int64(len(content)) != entry.Size || actualDigest != entry.ArchiveDigest || entry.SourceDigest != entry.ArchiveDigest {
				evidence.Status = "fail"
				result.ContentVerified = false
				addArchiveInspectionFinding(&result, "content_digest", fmt.Sprintf("archive content %q size/digest does not match the manifest", entry.Path), entryPath)
			}
		}
		evidence.SourceReference = entombManifestReferenceExists(manifest, "source:"+entry.Kind, entry.Path, entry.ArchiveDigest, false)
		if !evidence.SourceReference {
			evidence.Status = "fail"
			result.CrossReferencesVerified = false
		}
		if entombClosureArchiveKinds[entry.Kind] {
			evidence.ClosureReference = entombManifestReferenceExists(manifest, "closure:"+entry.Kind, entry.Path, entry.ArchiveDigest, true)
			if !evidence.ClosureReference {
				evidence.Status = "fail"
				result.CrossReferencesVerified = false
			}
		}
		result.DigestEvidence = append(result.DigestEvidence, evidence)
		status := evidence.Status
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "archive content digest: " + entry.Kind, Paths: []string{entryPath, result.ManifestPath}, Checked: 1, Status: status})
	}

	for _, requiredKind := range entombRequiredArchiveKinds {
		if kindCounts[requiredKind] != 1 {
			result.ContentVerified = false
			addArchiveInspectionFinding(&result, "content_kind", fmt.Sprintf("archive requires exactly one %q entry; found %d", requiredKind, kindCounts[requiredKind]), result.ManifestPath)
		}
	}

	outcome, outcomeOK := inspectArchiveSealOutcome(&result, entryContent)
	if !outcomeOK {
		result.ContentVerified = false
		result.CrossReferencesVerified = false
	} else {
		verifyArchiveGlobalReferences(&result, outcome)
	}
	receipt := colony.LifecycleReceipt{ReceiptID: manifest.Receipt.ID, Transaction: colony.LifecycleTransactionReference{ID: manifest.Transaction.ID}}
	if err := verifyPublishedEntombManifest(chamberPath, manifest, receipt); err != nil {
		code := archiveClosureFindingCode(err, manifest.Seal.Disposition)
		lower := strings.ToLower(err.Error())
		switch {
		case strings.Contains(lower, "cross-reference"), strings.Contains(lower, "seal outcome conflicts"):
			result.CrossReferencesVerified = false
		case strings.Contains(lower, "manifest"):
			result.ManifestVerified = false
		default:
			result.ContentVerified = false
		}
		addArchiveInspectionFinding(&result, code, err.Error(), result.ManifestPath)
	}
	result.ReferencesChecked = len(manifest.CrossReferences)
	inspectArchiveContext(&result, request, repositoryRoot, dataRoot, outcome, outcomeOK, &baseline)
	return finalizeArchiveMaintenanceInspection(result, baseline), nil
}

func inspectArchiveSealOutcome(result *archiveMaintenanceInspectionResult, content map[string][]byte) (colony.SealOutcome, bool) {
	var entry *colony.ArchiveEntry
	for index := range result.manifest.Entries {
		if result.manifest.Entries[index].Kind == "seal_outcome" {
			if entry != nil {
				addArchiveInspectionFinding(result, "seal_outcome", "archive contains duplicate seal outcome entries", result.ManifestPath)
				return colony.SealOutcome{}, false
			}
			entry = &result.manifest.Entries[index]
		}
	}
	if entry == nil {
		addArchiveInspectionFinding(result, "seal_outcome", "archive is missing its seal outcome entry", result.ManifestPath)
		return colony.SealOutcome{}, false
	}
	bytes, ok := content[entry.Path]
	if !ok || lifecycleDigest(bytes) != entry.ArchiveDigest {
		addArchiveInspectionFinding(result, "seal_outcome", "archive seal outcome bytes are not digest verified", filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)))
		return colony.SealOutcome{}, false
	}
	var outcome colony.SealOutcome
	if err := decodeLifecycleJSON(bytes, &outcome); err != nil {
		addArchiveInspectionFinding(result, "seal_outcome", fmt.Sprintf("archive seal outcome JSON is invalid: %v", err), filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)))
		return colony.SealOutcome{}, false
	}
	if err := outcome.Validate(); err != nil || outcome.Transaction.Stage != colony.TransactionStageVerified {
		addArchiveInspectionFinding(result, "seal_outcome", fmt.Sprintf("archive seal outcome is not verified: %v", err), filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)))
		return colony.SealOutcome{}, false
	}
	if outcome.OutcomeID != result.manifest.Seal.OutcomeID || outcome.Transaction.ID != result.manifest.Seal.TransactionID || outcome.Disposition != result.manifest.Seal.Disposition || outcome.OwnerReason != result.manifest.Seal.OwnerReason {
		addArchiveInspectionFinding(result, "seal_disposition", "archive seal outcome conflicts with the manifest closure identity", filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)), result.ManifestPath)
		return colony.SealOutcome{}, false
	}
	return outcome, true
}

func verifyArchiveGlobalReferences(result *archiveMaintenanceInspectionResult, outcome colony.SealOutcome) {
	sealDigest := ""
	for _, entry := range result.manifest.Entries {
		if entry.Kind == "seal_outcome" {
			sealDigest = entry.ArchiveDigest
			break
		}
	}
	required := []struct {
		kind, source, target, digest string
	}{
		{"seal_outcome", result.manifest.ArchiveID, outcome.OutcomeID, sealDigest},
		{"seal_transaction", outcome.OutcomeID, outcome.Transaction.ID, ""},
		{"entomb_transaction", result.manifest.ArchiveID, result.manifest.Transaction.ID, ""},
		{"seal_disposition", outcome.OutcomeID, string(outcome.Disposition), ""},
	}
	for _, reference := range required {
		if archiveReferenceCount(result.manifest, reference.kind, reference.source, reference.target, reference.digest, true) != 1 {
			result.CrossReferencesVerified = false
			addArchiveInspectionFinding(result, "cross_reference", fmt.Sprintf("manifest lacks one exact %s cross-reference", reference.kind), result.ManifestPath)
		}
	}
	for _, entry := range result.manifest.Entries {
		if entry.Kind != "archive_xml" {
			continue
		}
		content, err := readEntombManifestFile(result.ChamberPath, entry.Path, filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)))
		var archive entombClosureArchiveXML
		if err == nil {
			err = xml.Unmarshal(content, &archive)
		}
		if err != nil || archive.SealOutcomeID != outcome.OutcomeID || archive.SealTransaction != outcome.Transaction.ID || archive.Disposition != outcome.Disposition || archive.OwnerReason != outcome.OwnerReason {
			result.ContentVerified = false
			code := "closure_reference"
			if outcome.Disposition == colony.SealDispositionForcedIncomplete {
				code = "forced_marker_missing"
			}
			addArchiveInspectionFinding(result, code, "archive XML seal disposition/owner identity conflicts with the verified seal outcome", filepath.Join(result.ChamberPath, filepath.FromSlash(entry.Path)))
		}
	}
}

func inspectArchiveContext(result *archiveMaintenanceInspectionResult, request archiveMaintenanceInspectRequest, repositoryRoot, dataRoot string, outcome colony.SealOutcome, outcomeOK bool, baseline *[]string) {
	requireContext := request.RequireContext || request.ContextPath != "" || request.TombstonePath != ""
	statePath := filepath.Join(dataRoot, "COLONY_STATE.json")
	if stateBytes, err := readLifecycleEvidenceFile(statePath); err == nil {
		*baseline = append(*baseline, statePath+"="+lifecycleDigest(stateBytes))
		var state colony.ColonyState
		if decodeErr := decodeLifecycleJSON(stateBytes, &state); decodeErr == nil && state.ArchiveReference != nil {
			referencedManifest := filepath.Join(repositoryRoot, filepath.FromSlash(state.ArchiveReference.Path))
			if referencedManifest == result.ManifestPath {
				requireContext = true
				if state.ArchiveReference.ID != result.manifest.ArchiveID || state.ArchiveReference.TransactionID != result.manifest.Transaction.ID || state.ArchiveReference.ManifestDigest != result.manifest.ManifestDigest {
					result.ContextVerified = false
					result.CrossReferencesVerified = false
					addArchiveInspectionFinding(result, "context_reference", "active archive reference conflicts with the chamber manifest", statePath, result.ManifestPath)
				}
			}
		}
	}
	if !requireContext {
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "active chamber context", Paths: []string{}, Checked: 0, Status: "skip"})
		return
	}
	if !outcomeOK {
		result.ContextVerified = false
		return
	}
	contextPath := strings.TrimSpace(request.ContextPath)
	if contextPath == "" {
		contextPath = filepath.Join(repositoryRoot, ".aether", "CONTEXT.md")
	}
	tombstonePath := strings.TrimSpace(request.TombstonePath)
	if tombstonePath == "" {
		tombstonePath = filepath.Join(repositoryRoot, ".aether", "HANDOFF.md")
	}
	for _, document := range []struct {
		label string
		path  string
	}{
		{"context", contextPath},
		{"tombstone", tombstonePath},
	} {
		if !filepath.IsAbs(document.path) || (!pathIsWithin(repositoryRoot, document.path) && document.path != repositoryRoot) {
			result.ContextVerified = false
			addArchiveInspectionFinding(result, "context_path", fmt.Sprintf("%s path is outside the repository", document.label), document.path)
			continue
		}
		content, err := readLifecycleEvidenceFile(document.path)
		if err != nil {
			result.ContextVerified = false
			addArchiveInspectionFinding(result, "context_missing", fmt.Sprintf("%s is unavailable: %v", document.label, err), document.path)
			*baseline = append(*baseline, document.path+"="+lifecycleTransactionMissingDigest)
			result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "active chamber " + document.label, Paths: []string{document.path}, Checked: 1, Status: "fail"})
			continue
		}
		*baseline = append(*baseline, document.path+"="+lifecycleDigest(content))
		status := "pass"
		if err := verifyArchiveContextDocument(document.label, string(content), result, outcome); err != nil {
			result.ContextVerified = false
			status = "fail"
			addArchiveInspectionFinding(result, archiveClosureFindingCode(err, outcome.Disposition), err.Error(), document.path)
		}
		result.Evidence = append(result.Evidence, maintenanceInspectionEvidence{Scope: "active chamber " + document.label, Paths: []string{document.path, result.ManifestPath}, Checked: 1, Status: status})
	}
}

func verifyArchiveContextDocument(label, document string, result *archiveMaintenanceInspectionResult, outcome colony.SealOutcome) error {
	required := []string{result.manifest.ManifestDigest, string(outcome.Disposition), filepath.ToSlash(result.ChamberPath)}
	if label == "tombstone" {
		required = append(required, result.manifest.ArchiveID, result.manifest.Receipt.ID)
	}
	for _, marker := range required {
		if strings.TrimSpace(marker) != "" && !strings.Contains(document, marker) {
			return fmt.Errorf("active %s dropped archive marker %q", label, marker)
		}
	}
	lower := strings.ToLower(document)
	forcedMarker := strings.Contains(lower, "forced_incomplete") || strings.Contains(lower, "completion not verified") || strings.Contains(lower, "forced seal record")
	if outcome.Disposition == colony.SealDispositionForcedIncomplete {
		if !forcedMarker || !strings.Contains(document, outcome.OwnerReason) {
			return fmt.Errorf("active %s dropped the forced-incomplete marker or owner reason", label)
		}
		return nil
	}
	if forcedMarker {
		return fmt.Errorf("active %s contains a forced-incomplete marker that conflicts with the verified seal disposition", label)
	}
	return nil
}

func archiveClosureFindingCode(err error, disposition colony.SealDisposition) string {
	message := strings.ToLower(err.Error())
	if disposition == colony.SealDispositionForcedIncomplete && (strings.Contains(message, "forced") || strings.Contains(message, "owner reason")) {
		return "forced_marker_missing"
	}
	if disposition == colony.SealDispositionVerified && strings.Contains(message, "forced") {
		return "forced_marker_conflict"
	}
	return "closure_reference"
}

func archiveReferenceCount(manifest colony.ArchiveManifest, kind, source, target, digest string, requireSource bool) int {
	count := 0
	for _, reference := range manifest.CrossReferences {
		if reference.Kind != kind || reference.TargetID != target || reference.Digest != digest {
			continue
		}
		if requireSource && reference.SourceID != source {
			continue
		}
		if !requireSource && strings.TrimSpace(reference.SourceID) == "" {
			continue
		}
		count++
	}
	return count
}

func addArchiveInspectionFinding(result *archiveMaintenanceInspectionResult, code, summary string, paths ...string) {
	cleanPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if strings.TrimSpace(path) != "" && !containsString(cleanPaths, path) {
			cleanPaths = append(cleanPaths, path)
		}
	}
	finding := maintenanceInspectionFinding{
		Code: code, Summary: summary, EvidencePaths: cleanPaths,
		RecoveryCommand: "aether maintenance archive-inspect, then use an explicit previewed archive repair",
	}
	if len(cleanPaths) > 0 {
		finding.SourcePath = cleanPaths[0]
	}
	result.Findings = append(result.Findings, finding)
}

func finalizeArchiveMaintenanceInspection(result archiveMaintenanceInspectionResult, baseline []string) archiveMaintenanceInspectionResult {
	sort.Strings(baseline)
	result.BaselineDigest = lifecycleDigest([]byte(strings.Join(baseline, "\n")))
	result.Valid = result.ManifestVerified && result.ContentVerified && result.CrossReferencesVerified && result.ContextVerified && len(result.Findings) == 0
	status := "fail"
	if result.Valid {
		status = "pass"
		result.IntegrityLine = fmt.Sprintf("Chamber/context integrity: verified manifest %s; %d content digests and %d cross-references verified; seal disposition %s.", result.ManifestDigest, result.EntriesChecked, result.ReferencesChecked, result.SealDisposition)
	} else {
		manifest := result.ManifestDigest
		if manifest == "" {
			manifest = "unavailable"
		}
		result.IntegrityLine = fmt.Sprintf("Chamber/context integrity: failed manifest %s; %d content/cross-reference finding(s) block mutation.", manifest, len(result.Findings))
	}
	result.Verification = maintenanceInspectionVerification{Status: status, EvidenceCount: len(result.Evidence), FindingCount: len(result.Findings)}
	return result
}

var chamberListCmd = &cobra.Command{
	Use:   "chamber-list",
	Short: "List all chambers",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		aetherRoot := storage.ResolveAetherRoot(context.Background())
		chambersDir := filepath.Join(aetherRoot, ".aether", "chambers")

		entries, err := os.ReadDir(chambersDir)
		if err != nil {
			if os.IsNotExist(err) {
				outputOK(map[string]interface{}{
					"chambers": []interface{}{},
					"by_scope": emptyChamberScopeGroups(),
					"total":    0,
				})
				return nil
			}
			outputError(1, fmt.Sprintf("failed to read chambers directory: %v", err), nil)
			return nil
		}

		chambers := []interface{}{}
		byScope := emptyChamberScopeGroups()
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(chambersDir, entry.Name(), "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue // skip directories without manifest.json
			}
			var manifest map[string]interface{}
			if err := json.Unmarshal(data, &manifest); err != nil {
				continue // skip invalid manifests
			}
			manifest = manifestWithEffectiveScope(manifest)
			chambers = append(chambers, manifest)
			scope := string(chamberManifestScope(manifest))
			byScope[scope] = append(byScope[scope], manifest)
		}

		outputOK(map[string]interface{}{
			"chambers": chambers,
			"by_scope": byScope,
			"total":    len(chambers),
		})
		return nil
	},
}

var chamberCompareCmd = &cobra.Command{
	Use:   "chamber-compare [name]",
	Short: "Compare chamber archive with current colony state",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}
		name, _ := cmd.Flags().GetString("name")
		if len(args) > 0 && name == "" {
			name = args[0]
		}
		if name == "" {
			outputErrorMessage("chamber name is required (--name or positional arg)")
			return nil
		}

		aetherRoot := storage.ResolveAetherRoot(context.Background())
		manifestPath := filepath.Join(aetherRoot, ".aether", "chambers", name, "manifest.json")

		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			outputOK(map[string]interface{}{
				"chamber": name,
				"error":   fmt.Sprintf("chamber '%s' not found", name),
				"matches": []interface{}{},
				"diffs":   []interface{}{},
			})
			return nil
		}

		var manifest map[string]interface{}
		if err := json.Unmarshal(manifestData, &manifest); err != nil {
			outputOK(map[string]interface{}{
				"chamber": name,
				"error":   fmt.Sprintf("invalid manifest: %v", err),
				"matches": []interface{}{},
				"diffs":   []interface{}{},
			})
			return nil
		}

		var matches []interface{}
		var diffs []interface{}
		totalCompared := 0

		// Load colony state (graceful degradation if not available)
		state, stateErr := loadActiveColonyState()

		// Compare goal
		totalCompared++
		manifestGoal := stringValue(manifest["goal"])
		var currentGoal string
		if stateErr == nil && state.Goal != nil {
			currentGoal = *state.Goal
		}
		if manifestGoal == currentGoal {
			matches = append(matches, map[string]interface{}{"field": "goal", "chamber": manifestGoal, "current": currentGoal})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "goal", "chamber_value": manifestGoal, "current_value": currentGoal})
		}

		// Compare milestone
		totalCompared++
		manifestMilestone := stringValue(manifest["milestone"])
		var currentMilestone string
		if stateErr == nil {
			currentMilestone = state.Milestone
		}
		if manifestMilestone == currentMilestone {
			matches = append(matches, map[string]interface{}{"field": "milestone", "chamber": manifestMilestone, "current": currentMilestone})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "milestone", "chamber_value": manifestMilestone, "current_value": currentMilestone})
		}

		// Compare phases_completed
		totalCompared++
		manifestPhases := toInt(manifest["phases_completed"])
		currentPhases := 0
		if stateErr == nil {
			for _, p := range state.Plan.Phases {
				if p.Status == colony.PhaseCompleted {
					currentPhases++
				}
			}
		}
		if manifestPhases == currentPhases {
			matches = append(matches, map[string]interface{}{"field": "phases_completed", "chamber": manifestPhases, "current": currentPhases})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "phases_completed", "chamber_value": manifestPhases, "current_value": currentPhases})
		}

		// Compare total_phases
		totalCompared++
		manifestTotal := toInt(manifest["total_phases"])
		currentTotal := 0
		if stateErr == nil {
			currentTotal = len(state.Plan.Phases)
		}
		if manifestTotal == currentTotal {
			matches = append(matches, map[string]interface{}{"field": "total_phases", "chamber": manifestTotal, "current": currentTotal})
		} else {
			diffs = append(diffs, map[string]interface{}{"field": "total_phases", "chamber_value": manifestTotal, "current_value": currentTotal})
		}

		if matches == nil {
			matches = []interface{}{}
		}
		if diffs == nil {
			diffs = []interface{}{}
		}

		result := map[string]interface{}{
			"chamber":        name,
			"matches":        matches,
			"diffs":          diffs,
			"total_compared": totalCompared,
		}

		if stateErr != nil {
			result["error"] = "colony state not available"
		}

		outputOK(result)
		return nil
	},
}

func init() {
	chamberCreateCmd.Flags().String("name", "", "Chamber name (required)")
	chamberCreateCmd.Flags().String("goal", "", "Colony goal")
	chamberCreateCmd.Flags().String("milestone", "", "Milestone name")
	chamberCreateCmd.Flags().Int("phases-completed", 0, "Number of phases completed")
	chamberCreateCmd.Flags().Int("total-phases", 0, "Total number of phases")

	chamberVerifyCmd.Flags().String("name", "", "Chamber name (required)")

	chamberCompareCmd.Flags().String("name", "", "Chamber name to compare")

	rootCmd.AddCommand(chamberCreateCmd)
	rootCmd.AddCommand(chamberVerifyCmd)
	rootCmd.AddCommand(chamberListCmd)
	rootCmd.AddCommand(chamberCompareCmd)
}

func chamberManifestScope(manifest map[string]interface{}) colony.ColonyScope {
	scope, err := colony.ParseColonyScope(stringValue(manifest["scope"]))
	if err != nil {
		return colony.ScopeProject
	}
	return scope.Effective()
}

func manifestWithEffectiveScope(manifest map[string]interface{}) map[string]interface{} {
	if manifest == nil {
		return nil
	}
	manifest["scope"] = string(chamberManifestScope(manifest))
	return manifest
}

func emptyChamberScopeGroups() map[string][]interface{} {
	return map[string][]interface{}{
		string(colony.ScopeProject): {},
		string(colony.ScopeMeta):    {},
	}
}

// toInt converts an interface{} to int, handling float64 (JSON numbers) and int types.
func toInt(v interface{}) int {
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	default:
		return 0
	}
}
