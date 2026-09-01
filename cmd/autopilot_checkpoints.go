package cmd

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

const (
	autopilotCheckpointTypeVisual              = "visual-checkpoint"
	autopilotCheckpointTypeRuntimeVerification = "runtime-verification"
	autopilotCheckpointEvidenceLimit           = 20
	autopilotCheckpointEvidenceLength          = 500
)

// autopilotCheckpointReference is the compact, JSON-safe owner-work handle
// returned by build/continue. The durable PendingDecision remains canonical.
type autopilotCheckpointReference struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	Phase           int    `json:"phase,omitempty"`
	Question        string `json:"question"`
	RecoveryCommand string `json:"recovery_command"`
}

// autopilotCheckpointGeneration is the immutable authorization provenance for
// one owner checkpoint. The execution binding names the exact attempt and
// workspace/manifest contract; EvidenceSHA256 commits to the exact persisted
// bytes (visual) or canonical verification projection (runtime). The two
// derived digests are persisted on PendingDecision for audit and row identity.
type autopilotCheckpointGeneration struct {
	AttemptID              string
	ExecutionBinding       codex.ExecutionBinding
	EvidenceSHA256         string
	ExecutionBindingSHA256 string
	WorkGeneration         string
}

type autopilotCheckpointExecutionBindingMaterial struct {
	SchemaVersion        int    `json:"schema_version"`
	RunID                string `json:"run_id"`
	AttemptID            string `json:"attempt_id"`
	ManifestSHA256       string `json:"manifest_sha256"`
	WorkspaceFingerprint string `json:"workspace_fingerprint"`
	ExecutionOwner       string `json:"execution_owner"`
}

type autopilotCheckpointWorkGenerationMaterial struct {
	Version          int                                         `json:"version"`
	AttemptID        string                                      `json:"attempt_id"`
	ExecutionBinding autopilotCheckpointExecutionBindingMaterial `json:"execution_binding"`
	EvidenceSHA256   string                                      `json:"evidence_sha256"`
}

func newAutopilotCheckpointGeneration(attemptID string, binding *codex.ExecutionBinding, evidenceSHA256 string) (autopilotCheckpointGeneration, error) {
	attemptID = strings.TrimSpace(attemptID)
	evidenceSHA256 = strings.ToLower(strings.TrimSpace(evidenceSHA256))
	if !validBuildAttemptID(attemptID) {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation attempt_id %q is invalid", attemptID)
	}
	if binding == nil {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation requires execution_binding")
	}
	canonicalBinding := codex.ExecutionBinding{
		SchemaVersion:        binding.SchemaVersion,
		RunID:                strings.TrimSpace(binding.RunID),
		AttemptID:            strings.TrimSpace(binding.AttemptID),
		ManifestSHA256:       strings.ToLower(strings.TrimSpace(binding.ManifestSHA256)),
		WorkspaceFingerprint: strings.ToLower(strings.TrimSpace(binding.WorkspaceFingerprint)),
		ExecutionOwner:       strings.TrimSpace(binding.ExecutionOwner),
	}
	if err := canonicalBinding.Validate(); err != nil {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation execution binding: %w", err)
	}
	if canonicalBinding.AttemptID != attemptID {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation attempt_id does not match execution binding")
	}
	if !isSHA256Hex(evidenceSHA256) {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation evidence_sha256 must be a SHA-256 digest")
	}
	bindingMaterial := autopilotCheckpointExecutionBindingMaterial{
		SchemaVersion:        canonicalBinding.SchemaVersion,
		RunID:                canonicalBinding.RunID,
		AttemptID:            canonicalBinding.AttemptID,
		ManifestSHA256:       canonicalBinding.ManifestSHA256,
		WorkspaceFingerprint: canonicalBinding.WorkspaceFingerprint,
		ExecutionOwner:       canonicalBinding.ExecutionOwner,
	}
	bindingDigest, err := jsonSHA256(bindingMaterial)
	if err != nil {
		return autopilotCheckpointGeneration{}, fmt.Errorf("hash checkpoint execution binding: %w", err)
	}
	workGeneration, err := jsonSHA256(autopilotCheckpointWorkGenerationMaterial{
		Version:          1,
		AttemptID:        attemptID,
		ExecutionBinding: bindingMaterial,
		EvidenceSHA256:   evidenceSHA256,
	})
	if err != nil {
		return autopilotCheckpointGeneration{}, fmt.Errorf("hash checkpoint work generation: %w", err)
	}
	return autopilotCheckpointGeneration{
		AttemptID:              attemptID,
		ExecutionBinding:       canonicalBinding,
		EvidenceSHA256:         evidenceSHA256,
		ExecutionBindingSHA256: bindingDigest,
		WorkGeneration:         workGeneration,
	}, nil
}

func (generation autopilotCheckpointGeneration) validated() (autopilotCheckpointGeneration, error) {
	normalized, err := newAutopilotCheckpointGeneration(generation.AttemptID, &generation.ExecutionBinding, generation.EvidenceSHA256)
	if err != nil {
		return autopilotCheckpointGeneration{}, err
	}
	if strings.TrimSpace(generation.ExecutionBindingSHA256) != normalized.ExecutionBindingSHA256 ||
		strings.TrimSpace(generation.WorkGeneration) != normalized.WorkGeneration {
		return autopilotCheckpointGeneration{}, fmt.Errorf("checkpoint generation derived digests are inconsistent")
	}
	return normalized, nil
}

func isAutopilotCheckpointType(decisionType string) bool {
	switch strings.TrimSpace(decisionType) {
	case autopilotCheckpointTypeVisual, autopilotCheckpointTypeRuntimeVerification:
		return true
	default:
		return false
	}
}

// isUICheckpointPath is a pure classifier. It accepts only normalized,
// repo-relative product UI files; docs, tests, absolute paths, and traversal
// never create owner work.
func isUICheckpointPath(path string) bool {
	path = filepath.ToSlash(strings.TrimSpace(path))
	if path == "" || filepath.IsAbs(filepath.FromSlash(path)) {
		return false
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	lower := strings.ToLower(clean)
	segments := strings.Split(lower, "/")
	for _, segment := range segments[:max(0, len(segments)-1)] {
		switch segment {
		case "test", "tests", "__tests__", "fixtures", "docs", "documentation":
			return false
		}
	}
	base := filepath.Base(lower)
	if strings.Contains(base, ".test.") || strings.Contains(base, ".spec.") || strings.Contains(base, "_test.") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(base))
	switch ext {
	case ".tsx", ".jsx", ".vue", ".svelte", ".astro", ".html", ".htm", ".css", ".scss", ".sass", ".less", ".styl", ".templ":
		return true
	case ".svg", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".ico":
		return pathHasUISegment(segments)
	case ".ts", ".js", ".swift", ".dart":
		return pathHasUISegment(segments)
	default:
		return false
	}
}

func pathHasUISegment(segments []string) bool {
	for _, segment := range segments {
		switch segment {
		case "ui", "components", "component", "views", "view", "screens", "screen", "pages", "styles", "assets", "public":
			return true
		}
	}
	return false
}

func normalizeUICheckpointClaimPath(root, path string) (string, bool) {
	if !isUICheckpointPath(path) {
		return "", false
	}
	rootAbs, err := filepath.Abs(strings.TrimSpace(root))
	if err != nil || strings.TrimSpace(root) == "" {
		return "", false
	}
	clean := filepath.Clean(filepath.FromSlash(strings.TrimSpace(path)))
	joined := filepath.Join(rootAbs, clean)
	rel, err := filepath.Rel(rootAbs, joined)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// materializeVisualCheckpointFromBuildResult is the Plan 04 integration
// point: call it immediately after a successful build result. Both paths only
// locate runtime-owned durable artifacts; worker booleans and prose are
// deliberately ignored. Validation and hashing finish before the pending
// decision file is touched.
func materializeVisualCheckpointFromBuildResult(root string, phaseID int, buildResult map[string]interface{}) ([]autopilotCheckpointReference, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	if phaseID <= 0 {
		return nil, fmt.Errorf("phase must be positive")
	}
	attemptPath, _ := buildResult["attempt"].(string)
	attemptPath = strings.TrimSpace(attemptPath)
	if attemptPath == "" {
		return nil, fmt.Errorf("successful build result is missing attempt")
	}
	claimsPath, _ := buildResult["claims_path"].(string)
	claimsPath = strings.TrimSpace(claimsPath)
	if claimsPath == "" {
		return nil, fmt.Errorf("successful build result is missing claims_path")
	}
	attemptRel, err := checkpointStorePath(attemptPath, "attempt")
	if err != nil {
		return nil, err
	}
	claimsRel, err := checkpointStorePath(claimsPath, "claims")
	if err != nil {
		return nil, err
	}

	var attempt buildAttemptRecord
	if err := store.LoadJSON(attemptRel, &attempt); err != nil {
		return nil, fmt.Errorf("load build attempt %q: %w", attemptRel, err)
	}
	if err := validateVisualCheckpointAttempt(attemptRel, claimsRel, phaseID, attempt); err != nil {
		return nil, err
	}
	claimsBytes, err := os.ReadFile(filepath.Join(store.BasePath(), filepath.FromSlash(claimsRel)))
	if err != nil {
		return nil, fmt.Errorf("load build claims %q: %w", claimsRel, err)
	}
	var claims codexBuildClaims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, fmt.Errorf("decode build claims %q: %w", claimsRel, err)
	}
	if claims.BuildPhase != phaseID {
		return nil, fmt.Errorf("build claims phase %d does not match requested phase %d", claims.BuildPhase, phaseID)
	}
	evidenceDigest := sha256.Sum256(claimsBytes)
	generation, err := newAutopilotCheckpointGeneration(
		attempt.ID,
		attempt.PlanManifest.ExecutionBinding,
		hex.EncodeToString(evidenceDigest[:]),
	)
	if err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(claims.FilesCreated)+len(claims.FilesModified))
	for _, raw := range append(append([]string{}, claims.FilesCreated...), claims.FilesModified...) {
		if normalized, ok := normalizeUICheckpointClaimPath(root, raw); ok {
			paths = append(paths, normalized)
		}
	}
	paths = boundedCheckpointEvidence(uniqueSortedStrings(paths))
	if len(paths) == 0 {
		return nil, nil
	}

	question := fmt.Sprintf("Phase %d: please visually confirm the user-interface changes look and behave correctly before the project is signed off.", phaseID)
	decision, _, err := upsertAutopilotCheckpoint(PendingDecision{
		Type:        autopilotCheckpointTypeVisual,
		Description: formatClarificationDescription(question, nil),
		Source:      "autopilot-visual-checkpoint",
		SourcePaths: paths,
	}, phaseID, "visual", generation)
	if err != nil {
		return nil, err
	}
	if decision.Resolved {
		return nil, nil
	}
	return []autopilotCheckpointReference{checkpointReference(decision)}, nil
}

func validateVisualCheckpointAttempt(attemptRel, claimsRel string, phaseID int, attempt buildAttemptRecord) error {
	if attempt.SchemaVersion != buildAttemptSchemaVersion || !validBuildAttemptID(attempt.ID) {
		return fmt.Errorf("build attempt %q is invalid", attemptRel)
	}
	if attempt.Phase != phaseID {
		return fmt.Errorf("build attempt phase %d does not match requested phase %d", attempt.Phase, phaseID)
	}
	expectedAttemptRel := buildAttemptPathForID(phaseID, attempt.ID)
	if attemptRel != expectedAttemptRel {
		return fmt.Errorf("build attempt path %q does not match durable attempt %s", attemptRel, attempt.ID)
	}
	if attempt.Status != buildAttemptBuilt {
		return fmt.Errorf("build attempt %s is %s, not built", attempt.ID, strings.TrimSpace(attempt.Status))
	}
	recordClaimsRel, err := checkpointStorePath(attempt.ClaimsPath, "build attempt claims")
	if err != nil {
		return err
	}
	if recordClaimsRel != claimsRel {
		return fmt.Errorf("build result claims path %q does not match build attempt %s claims path %q", claimsRel, attempt.ID, recordClaimsRel)
	}
	manifest := attempt.PlanManifest
	if manifest == nil {
		return fmt.Errorf("build attempt %s is missing its durable plan manifest", attempt.ID)
	}
	if manifest.Phase != phaseID || strings.TrimSpace(manifest.AttemptID) != attempt.ID || strings.TrimSpace(manifest.AttemptPath) != displayDataPath(attemptRel) {
		return fmt.Errorf("build attempt %s manifest identity is inconsistent", attempt.ID)
	}
	manifestClaimsRel, err := checkpointStorePath(manifest.ClaimsPath, "build manifest claims")
	if err != nil {
		return err
	}
	if manifestClaimsRel != claimsRel {
		return fmt.Errorf("build attempt %s manifest claims path %q does not match %q", attempt.ID, manifestClaimsRel, claimsRel)
	}
	if manifest.ExecutionBinding == nil {
		return fmt.Errorf("build attempt %s manifest is missing execution_binding", attempt.ID)
	}
	binding := manifest.ExecutionBinding
	if err := binding.Validate(); err != nil {
		return fmt.Errorf("build attempt %s execution binding: %w", attempt.ID, err)
	}
	manifestDigest, err := buildManifestSHA256(*manifest)
	if err != nil {
		return fmt.Errorf("hash build attempt %s manifest: %w", attempt.ID, err)
	}
	if binding.AttemptID != attempt.ID || binding.RunID != strings.TrimSpace(attempt.RunID) ||
		binding.ManifestSHA256 != strings.TrimSpace(attempt.ManifestSHA256) || binding.ManifestSHA256 != manifestDigest ||
		binding.WorkspaceFingerprint != strings.TrimSpace(attempt.WorkspaceSHA256) ||
		binding.ExecutionOwner != strings.TrimSpace(attempt.ExecutionOwner) ||
		strings.TrimSpace(manifest.ExecutionOwner) != binding.ExecutionOwner {
		return fmt.Errorf("build attempt %s execution binding is inconsistent with its durable record", attempt.ID)
	}
	return nil
}

func checkpointStorePath(value, kind string) (string, error) {
	path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(value)))
	if strings.TrimSpace(value) == "" || path == "." {
		return "", fmt.Errorf("%s path is empty", kind)
	}
	if filepath.IsAbs(path) {
		base := filepath.Clean(store.BasePath())
		rel, err := filepath.Rel(base, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("%s path %q is outside the colony data directory", kind, value)
		}
		path = rel
	} else {
		prefix := filepath.Join(".aether", "data") + string(filepath.Separator)
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
		}
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s path %q escapes the colony data directory", kind, value)
	}
	return filepath.ToSlash(path), nil
}

func checkpointClaimsStorePath(claimsPath string) (string, error) {
	return checkpointStorePath(claimsPath, "claims")
}

func materializeRuntimeVerificationCheckpoints(phaseID int, criteria []codexCriterionVerification, generations ...autopilotCheckpointGeneration) ([]autopilotCheckpointReference, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	if len(generations) > 1 {
		return nil, fmt.Errorf("runtime verification checkpoints accept exactly one work generation")
	}
	var generation autopilotCheckpointGeneration
	var err error
	if len(generations) == 1 {
		generation, err = generations[0].validated()
	} else {
		// Transitional compatibility for pre-Plan-17 callers. Task 2 removes
		// this branch after both continue lanes and every test fixture pass the
		// durable manifest generation explicitly.
		generation, err = legacyRuntimeCheckpointGeneration(phaseID, criteria)
	}
	if err != nil {
		return nil, err
	}
	refs := []autopilotCheckpointReference{}
	seen := map[string]bool{}
	for _, criterion := range criteria {
		if criterion.State != criterionStateNeedsOwnerConfirmation {
			continue
		}
		question := ownerConfirmationQuestionText(phaseID, criterion.TaskID, criterion.Criterion)
		evidence := append([]string{}, criterion.Evidence...)
		if summary := strings.TrimSpace(criterion.Summary); summary != "" {
			evidence = append(evidence, summary)
		}
		evidence = boundedCheckpointEvidence(uniqueSortedStrings(evidence))
		decision, _, err := upsertAutopilotCheckpoint(PendingDecision{
			Type:        autopilotCheckpointTypeRuntimeVerification,
			Description: formatClarificationDescription(question, nil),
			Source:      "autopilot-runtime-verification",
			Criterion:   strings.TrimSpace(criterion.Criterion),
			TaskID:      strings.TrimSpace(criterion.TaskID),
			Evidence:    evidence,
		}, phaseID, strings.Join([]string{criterion.TaskID, criterion.Criterion}, "\x00"), generation)
		if err != nil {
			return nil, err
		}
		if !decision.Resolved && !seen[decision.ID] {
			seen[decision.ID] = true
			refs = append(refs, checkpointReference(decision))
		}
	}
	return refs, nil
}

func legacyRuntimeCheckpointGeneration(phaseID int, criteria []codexCriterionVerification) (autopilotCheckpointGeneration, error) {
	evidenceDigest, err := jsonSHA256(criteria)
	if err != nil {
		return autopilotCheckpointGeneration{}, fmt.Errorf("hash legacy runtime checkpoint evidence: %w", err)
	}
	suffix := fmt.Sprintf("legacy-phase-%d", phaseID)
	binding := codex.ExecutionBinding{
		SchemaVersion:        codex.ExecutionBindingSchemaVersion,
		RunID:                "run-checkpoint-" + suffix,
		AttemptID:            "attempt-checkpoint-" + suffix,
		ManifestSHA256:       sha256HexString("manifest-" + suffix),
		WorkspaceFingerprint: sha256HexString("workspace-" + suffix),
		ExecutionOwner:       "legacy-runtime-checkpoint",
	}
	return newAutopilotCheckpointGeneration(binding.AttemptID, &binding, evidenceDigest)
}

func sha256HexString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func upsertAutopilotCheckpoint(candidate PendingDecision, phaseID int, subject string, generations ...autopilotCheckpointGeneration) (PendingDecision, bool, error) {
	if store == nil {
		return PendingDecision{}, false, fmt.Errorf("no store initialized")
	}
	if !isAutopilotCheckpointType(candidate.Type) {
		return PendingDecision{}, false, fmt.Errorf("unsupported checkpoint type %q", candidate.Type)
	}
	if phaseID <= 0 {
		return PendingDecision{}, false, fmt.Errorf("phase must be positive")
	}
	if len(generations) > 1 {
		return PendingDecision{}, false, fmt.Errorf("checkpoint accepts exactly one work generation")
	}
	var generation autopilotCheckpointGeneration
	var err error
	if len(generations) == 1 {
		generation, err = generations[0].validated()
	} else {
		// Transitional compatibility for direct protected-row fixtures. Task 2
		// migrates those callers and makes this parameter mandatory.
		generation, err = legacyRuntimeCheckpointGeneration(phaseID, []codexCriterionVerification{{
			TaskID: candidate.TaskID, Criterion: candidate.Criterion, Evidence: candidate.Evidence,
		}})
	}
	if err != nil {
		return PendingDecision{}, false, err
	}
	scope := loadCurrentPendingDecisionScope()
	compatibilityKey := stableAutopilotCheckpointKey(candidate.Type, phaseID, subject, scope)
	key := generationAutopilotCheckpointKey(compatibilityKey, generation.WorkGeneration)
	phase := phaseID
	candidate.Phase = &phase
	candidate.CheckpointCompatibilityKey = compatibilityKey
	candidate.CheckpointKey = key
	candidate.CheckpointAttemptID = generation.AttemptID
	candidate.CheckpointExecutionBindingSHA256 = generation.ExecutionBindingSHA256
	candidate.CheckpointEvidenceSHA256 = generation.EvidenceSHA256
	candidate.WorkGeneration = generation.WorkGeneration
	candidate.ID = stableAutopilotCheckpointID(key)
	candidate.Resolved = false
	candidate.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	candidate.Evidence = boundedCheckpointEvidence(candidate.Evidence)
	candidate.SourcePaths = boundedCheckpointEvidence(candidate.SourcePaths)
	stampPendingDecisionScope(&candidate, scope)
	capability, capabilityHash, err := newForcedReviewerWaiverCapability()
	if err != nil {
		return PendingDecision{}, false, fmt.Errorf("issue %s checkpoint capability: %w", candidate.Type, err)
	}

	var file PendingDecisionFile
	result := candidate
	created := false
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		if file.Decisions == nil {
			file.Decisions = []PendingDecision{}
		}
		for i := range file.Decisions {
			existing := &file.Decisions[i]
			if existing.ID != candidate.ID ||
				existing.CheckpointKey != key ||
				existing.Type != candidate.Type ||
				existing.Phase == nil || *existing.Phase != phaseID ||
				!pendingDecisionMatchesScope(*existing, scope) {
				continue
			}
			if existing.CheckpointCompatibilityKey != candidate.CheckpointCompatibilityKey ||
				existing.CheckpointAttemptID != candidate.CheckpointAttemptID ||
				existing.CheckpointExecutionBindingSHA256 != candidate.CheckpointExecutionBindingSHA256 ||
				existing.CheckpointEvidenceSHA256 != candidate.CheckpointEvidenceSHA256 ||
				existing.WorkGeneration != candidate.WorkGeneration {
				return fmt.Errorf("checkpoint row %s has inconsistent work-generation provenance", existing.ID)
			}
			result = *existing
			if !existing.Resolved {
				appendCheckpointCapabilityHash(existing, capabilityHash)
				result = *existing
				result.CheckpointCapability = capability
			}
			return nil
		}
		candidate.CheckpointCapability = capability
		candidate.CheckpointCapabilitySHA256 = capabilityHash
		file.Decisions = append(file.Decisions, candidate)
		result = candidate
		created = true
		return nil
	}); err != nil {
		return PendingDecision{}, false, fmt.Errorf("persist %s checkpoint: %w", candidate.Type, err)
	}
	return result, created, nil
}

func generationAutopilotCheckpointKey(compatibilityKey, workGeneration string) string {
	material := strings.Join([]string{
		"checkpoint-row-v1",
		strings.TrimSpace(compatibilityKey),
		strings.TrimSpace(workGeneration),
	}, "\x00")
	digest := sha256.Sum256([]byte(material))
	return hex.EncodeToString(digest[:])
}

func validatePersistedAutopilotCheckpointGeneration(decision PendingDecision) error {
	if !isAutopilotCheckpointType(decision.Type) {
		return fmt.Errorf("decision %s is not a protected checkpoint", decision.ID)
	}
	if !validBuildAttemptID(decision.CheckpointAttemptID) {
		return fmt.Errorf("checkpoint %s has invalid attempt provenance", decision.ID)
	}
	for name, value := range map[string]string{
		"compatibility key":        decision.CheckpointCompatibilityKey,
		"execution-binding digest": decision.CheckpointExecutionBindingSHA256,
		"evidence digest":          decision.CheckpointEvidenceSHA256,
		"work generation":          decision.WorkGeneration,
	} {
		if !isSHA256Hex(value) {
			return fmt.Errorf("checkpoint %s has invalid %s", decision.ID, name)
		}
	}
	wantKey := generationAutopilotCheckpointKey(decision.CheckpointCompatibilityKey, decision.WorkGeneration)
	if decision.CheckpointKey != wantKey || decision.ID != stableAutopilotCheckpointID(wantKey) {
		return fmt.Errorf("checkpoint %s generation row identity is inconsistent", decision.ID)
	}
	return nil
}

func stableAutopilotCheckpointID(key string) string {
	key = strings.TrimSpace(key)
	if len(key) <= 20 {
		return "cp_" + key
	}
	return "cp_" + key[len(key)-20:]
}

func appendCheckpointCapabilityHash(decision *PendingDecision, capabilityHash string) {
	if decision == nil {
		return
	}
	capabilityHash = strings.TrimSpace(capabilityHash)
	if capabilityHash == "" {
		return
	}
	if strings.TrimSpace(decision.CheckpointCapabilitySHA256) == "" {
		decision.CheckpointCapabilitySHA256 = capabilityHash
		return
	}
	if decision.CheckpointCapabilitySHA256 == capabilityHash {
		return
	}
	for _, existing := range decision.CheckpointCapabilitySHA256s {
		if strings.TrimSpace(existing) == capabilityHash {
			return
		}
	}
	decision.CheckpointCapabilitySHA256s = append(decision.CheckpointCapabilitySHA256s, capabilityHash)
}

func checkpointCapabilityMatches(decision PendingDecision, providedHash string) bool {
	providedHash = strings.TrimSpace(providedHash)
	if providedHash == "" {
		return false
	}
	if subtle.ConstantTimeCompare(
		[]byte(strings.TrimSpace(decision.CheckpointCapabilitySHA256)),
		[]byte(providedHash),
	) == 1 {
		return true
	}
	for _, candidate := range decision.CheckpointCapabilitySHA256s {
		if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(candidate)), []byte(providedHash)) == 1 {
			return true
		}
	}
	return false
}

func autopilotCheckpointResolutionIndex(file PendingDecisionFile, target string, phaseID int, scope pendingDecisionScope, providedHash string) (int, bool) {
	for i := range file.Decisions {
		decision := file.Decisions[i]
		if decision.Resolved ||
			!isAutopilotCheckpointType(decision.Type) ||
			decision.Phase == nil || *decision.Phase != phaseID ||
			!pendingDecisionMatchesScope(decision, scope) ||
			validatePersistedAutopilotCheckpointGeneration(decision) != nil ||
			normalizeDecisionText(checkpointDecisionQuestion(decision)) != target ||
			!checkpointCapabilityMatches(decision, providedHash) {
			continue
		}
		return i, true
	}
	return -1, false
}

// resolveAutopilotCheckpointPendingDecision is the only state-changing
// checkpoint resolver. The read pass rejects bad authorization without a file
// write; the atomic pass repeats every binding check before consuming one row.
func resolveAutopilotCheckpointPendingDecision(question, answer string, phaseID int, capability string) (PendingDecision, bool, error) {
	if store == nil {
		return PendingDecision{}, false, fmt.Errorf("no store initialized")
	}
	target := normalizeDecisionText(question)
	answer = strings.TrimSpace(answer)
	capability = strings.TrimSpace(capability)
	if target == "" || answer == "" || phaseID <= 0 || capability == "" {
		return PendingDecision{}, false, nil
	}
	scope := loadCurrentPendingDecisionScope()
	digest := sha256.Sum256([]byte(capability))
	providedHash := hex.EncodeToString(digest[:])

	var preview PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &preview); err != nil {
		return PendingDecision{}, false, fmt.Errorf("load checkpoint decisions: %w", err)
	}
	if _, ok := autopilotCheckpointResolutionIndex(preview, target, phaseID, scope, providedHash); !ok {
		return PendingDecision{}, false, nil
	}

	var file PendingDecisionFile
	var resolved PendingDecision
	found := false
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		index, ok := autopilotCheckpointResolutionIndex(file, target, phaseID, scope, providedHash)
		if !ok {
			return nil
		}
		decision := &file.Decisions[index]
		decision.Resolved = true
		decision.Resolution = answer
		decision.ResolvedAt = time.Now().UTC().Format(time.RFC3339)
		resolved = *decision
		found = true
		return nil
	}); err != nil {
		return PendingDecision{}, false, fmt.Errorf("resolve checkpoint decision: %w", err)
	}
	return resolved, found, nil
}

// hasCurrentAutopilotCheckpointQuestion reserves persisted, current-scope
// checkpoint questions for the capability-aware resolver. The public command
// uses this read-only check before ordinary clarification handling so an
// authorization failure cannot fall through and append a shadow answer.
func hasCurrentAutopilotCheckpointQuestion(question string) (bool, error) {
	if store == nil {
		return false, fmt.Errorf("no store initialized")
	}
	target := normalizeDecisionText(question)
	if target == "" {
		return false, nil
	}
	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("load checkpoint decisions: %w", err)
	}
	scope := loadCurrentPendingDecisionScope()
	for _, decision := range file.Decisions {
		if !isAutopilotCheckpointType(decision.Type) ||
			!pendingDecisionMatchesScope(decision, scope) ||
			normalizeDecisionText(checkpointDecisionQuestion(decision)) != target {
			continue
		}
		return true, nil
	}
	return false, nil
}

func stableAutopilotCheckpointKey(decisionType string, phaseID int, subject string, scope pendingDecisionScope) string {
	material := strings.Join([]string{
		strings.TrimSpace(decisionType),
		fmt.Sprintf("%d", phaseID),
		normalizeDecisionText(subject),
		strings.TrimSpace(scope.SessionID),
		strings.TrimSpace(scope.GoalHash),
	}, "\x00")
	sum := sha256.Sum256([]byte(material))
	return fmt.Sprintf("%x", sum)
}

func boundedCheckpointEvidence(values []string) []string {
	bounded := make([]string, 0, min(len(values), autopilotCheckpointEvidenceLimit))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if len(value) > autopilotCheckpointEvidenceLength {
			value = value[:autopilotCheckpointEvidenceLength]
		}
		bounded = append(bounded, value)
		if len(bounded) == autopilotCheckpointEvidenceLimit {
			break
		}
	}
	return bounded
}

func checkpointDecisionQuestion(decision PendingDecision) string {
	question, _ := parseClarificationDescription(decision.Description)
	if strings.TrimSpace(question) != "" {
		return strings.TrimSpace(question)
	}
	return strings.TrimSpace(decision.Description)
}

func checkpointDecisionAnswerCommand(decision PendingDecision) string {
	phaseID := 0
	if decision.Phase != nil {
		phaseID = *decision.Phase
	}
	return fmt.Sprintf(
		"aether decision-answer --question %s --answer 'confirmed' --phase %d --checkpoint-capability %s",
		shellQuote(checkpointDecisionQuestion(decision)),
		phaseID,
		shellQuote(decision.CheckpointCapability),
	)
}

func checkpointReference(decision PendingDecision) autopilotCheckpointReference {
	phaseID := 0
	if decision.Phase != nil {
		phaseID = *decision.Phase
	}
	return autopilotCheckpointReference{
		ID:              decision.ID,
		Type:            decision.Type,
		Phase:           phaseID,
		Question:        checkpointDecisionQuestion(decision),
		RecoveryCommand: checkpointDecisionAnswerCommand(decision),
	}
}

func autopilotCheckpointSealBlockers(state colony.ColonyState) ([]colony.FlagEntry, error) {
	return autopilotCheckpointSealBlockersFromStore(store, state)
}

// autopilotCheckpointSealBlockersFromStore keeps the public loader's global
// compatibility while allowing the shared seal gate to use the exact Store it
// was handed. This matters for tests and for any future caller that validates
// a store before installing it as the process-global runtime store.
func autopilotCheckpointSealBlockersFromStore(checkpointStore *storage.Store, state colony.ColonyState) ([]colony.FlagEntry, error) {
	if checkpointStore == nil {
		return nil, fmt.Errorf("load checkpoint seal blockers: no store initialized")
	}
	var preview PendingDecisionFile
	if err := checkpointStore.LoadJSON(pendingDecisionsFile, &preview); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("load checkpoint seal blockers from %s: %w", pendingDecisionsFile, err)
	}
	scope := pendingDecisionScopeFromState(state)
	hasCurrentCheckpoint := false
	for _, decision := range preview.Decisions {
		if !decision.Resolved && isAutopilotCheckpointType(decision.Type) && pendingDecisionMatchesScope(decision, scope) {
			if err := validatePersistedAutopilotCheckpointGeneration(decision); err != nil {
				return nil, fmt.Errorf("load checkpoint seal blockers from %s: %w", pendingDecisionsFile, err)
			}
			hasCurrentCheckpoint = true
		}
	}
	if !hasCurrentCheckpoint {
		return nil, nil
	}

	// Capability hashes for every current checkpoint are persisted in one
	// read-modify-write transaction. Raw capabilities remain attached only to
	// this in-memory copy so a failed write cannot return an authorization
	// command whose hash never became durable.
	var file PendingDecisionFile
	if err := checkpointStore.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		for i := range file.Decisions {
			decision := &file.Decisions[i]
			if decision.Resolved || !isAutopilotCheckpointType(decision.Type) || !pendingDecisionMatchesScope(*decision, scope) {
				continue
			}
			if err := validatePersistedAutopilotCheckpointGeneration(*decision); err != nil {
				return err
			}
			capability, capabilityHash, err := newForcedReviewerWaiverCapability()
			if err != nil {
				return fmt.Errorf("issue checkpoint capability for %s: %w", decision.ID, err)
			}
			appendCheckpointCapabilityHash(decision, capabilityHash)
			decision.CheckpointCapability = capability
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("persist checkpoint seal capabilities in %s: %w", pendingDecisionsFile, err)
	}

	blockers := []colony.FlagEntry{}
	for _, decision := range file.Decisions {
		if decision.Resolved || !isAutopilotCheckpointType(decision.Type) || !pendingDecisionMatchesScope(decision, scope) {
			continue
		}
		phaseID := 0
		if decision.Phase != nil {
			phaseID = *decision.Phase
		}
		detail := "owner verification is still waiting"
		switch decision.Type {
		case autopilotCheckpointTypeVisual:
			detail = "the user-interface changes still need your visual confirmation"
		case autopilotCheckpointTypeRuntimeVerification:
			if criterion := strings.TrimSpace(decision.Criterion); criterion != "" {
				detail = fmt.Sprintf("%q still needs your hands-on confirmation", criterion)
			}
		}
		command := checkpointDecisionAnswerCommand(decision)
		blockers = append(blockers, colony.FlagEntry{
			ID:              decision.ID,
			Type:            "blocker",
			Description:     fmt.Sprintf("Phase %d checkpoint %s (%s): %s.", phaseID, decision.ID, decision.Type, detail),
			Phase:           decision.Phase,
			Source:          "autopilot_checkpoint",
			CreatedAt:       decision.CreatedAt,
			RecoveryCommand: command,
		})
	}
	return blockers, nil
}

// autopilotCheckpointCompatibilityIDsFromStore projects only the stable
// pre-generation identity used to suppress a live legacy owner-confirmation
// blocker. It deliberately does not return row IDs, mutate capabilities, or
// collapse multiple durable rows that share one compatibility identity.
func autopilotCheckpointCompatibilityIDsFromStore(checkpointStore *storage.Store, state colony.ColonyState) (map[string]bool, error) {
	ids := map[string]bool{}
	if checkpointStore == nil {
		return ids, fmt.Errorf("load checkpoint compatibility identities: no store initialized")
	}
	var file PendingDecisionFile
	if err := checkpointStore.LoadJSON(pendingDecisionsFile, &file); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ids, nil
		}
		return ids, fmt.Errorf("load checkpoint compatibility identities from %s: %w", pendingDecisionsFile, err)
	}
	scope := pendingDecisionScopeFromState(state)
	for _, decision := range file.Decisions {
		if decision.Resolved || !isAutopilotCheckpointType(decision.Type) || !pendingDecisionMatchesScope(decision, scope) {
			continue
		}
		if err := validatePersistedAutopilotCheckpointGeneration(decision); err != nil {
			return nil, err
		}
		ids[stableAutopilotCheckpointID(decision.CheckpointCompatibilityKey)] = true
	}
	return ids, nil
}
