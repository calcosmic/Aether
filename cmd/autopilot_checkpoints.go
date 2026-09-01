package cmd

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
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
// point: call it immediately after a successful build result. The result's
// claims_path only locates runtime-owned persisted claims; worker booleans
// and prose are deliberately ignored.
func materializeVisualCheckpointFromBuildResult(root string, phaseID int, buildResult map[string]interface{}) ([]autopilotCheckpointReference, error) {
	claimsPath, _ := buildResult["claims_path"].(string)
	claimsPath = strings.TrimSpace(claimsPath)
	if claimsPath == "" {
		return nil, fmt.Errorf("successful build result is missing claims_path")
	}
	return materializeVisualCheckpointFromClaimsPath(root, phaseID, claimsPath)
}

func materializeVisualCheckpointFromClaimsPath(root string, phaseID int, claimsPath string) ([]autopilotCheckpointReference, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	if phaseID <= 0 {
		return nil, fmt.Errorf("phase must be positive")
	}
	rel, err := checkpointClaimsStorePath(claimsPath)
	if err != nil {
		return nil, err
	}
	var claims codexBuildClaims
	if err := store.LoadJSON(rel, &claims); err != nil {
		return nil, fmt.Errorf("load build claims %q: %w", rel, err)
	}
	if claims.BuildPhase != phaseID {
		return nil, fmt.Errorf("build claims phase %d does not match requested phase %d", claims.BuildPhase, phaseID)
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
	}, phaseID, "visual")
	if err != nil {
		return nil, err
	}
	if decision.Resolved {
		return nil, nil
	}
	return []autopilotCheckpointReference{checkpointReference(decision)}, nil
}

func checkpointClaimsStorePath(claimsPath string) (string, error) {
	path := filepath.Clean(filepath.FromSlash(strings.TrimSpace(claimsPath)))
	if strings.TrimSpace(claimsPath) == "" || path == "." {
		return "", fmt.Errorf("claims path is empty")
	}
	if filepath.IsAbs(path) {
		base := filepath.Clean(store.BasePath())
		rel, err := filepath.Rel(base, path)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return "", fmt.Errorf("claims path %q is outside the colony data directory", claimsPath)
		}
		path = rel
	} else {
		prefix := filepath.Join(".aether", "data") + string(filepath.Separator)
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
		}
	}
	if path == ".." || strings.HasPrefix(path, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("claims path %q escapes the colony data directory", claimsPath)
	}
	return filepath.ToSlash(path), nil
}

func materializeRuntimeVerificationCheckpoints(phaseID int, criteria []codexCriterionVerification) ([]autopilotCheckpointReference, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
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
		}, phaseID, strings.Join([]string{criterion.TaskID, criterion.Criterion}, "\x00"))
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

func upsertAutopilotCheckpoint(candidate PendingDecision, phaseID int, subject string) (PendingDecision, bool, error) {
	if store == nil {
		return PendingDecision{}, false, fmt.Errorf("no store initialized")
	}
	if !isAutopilotCheckpointType(candidate.Type) {
		return PendingDecision{}, false, fmt.Errorf("unsupported checkpoint type %q", candidate.Type)
	}
	if phaseID <= 0 {
		return PendingDecision{}, false, fmt.Errorf("phase must be positive")
	}
	scope := loadCurrentPendingDecisionScope()
	key := stableAutopilotCheckpointKey(candidate.Type, phaseID, subject, scope)
	phase := phaseID
	candidate.Phase = &phase
	candidate.CheckpointKey = key
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
			existing.Evidence = boundedCheckpointEvidence(uniqueSortedStrings(append(existing.Evidence, candidate.Evidence...)))
			existing.SourcePaths = boundedCheckpointEvidence(uniqueSortedStrings(append(existing.SourcePaths, candidate.SourcePaths...)))
			if existing.Criterion == "" {
				existing.Criterion = candidate.Criterion
			}
			if existing.TaskID == "" {
				existing.TaskID = candidate.TaskID
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
			decision.CheckpointKey == "" ||
			decision.ID != stableAutopilotCheckpointID(decision.CheckpointKey) ||
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
	if store == nil {
		return nil, fmt.Errorf("load checkpoint seal blockers: no store initialized")
	}
	var preview PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &preview); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("load checkpoint seal blockers from %s: %w", pendingDecisionsFile, err)
	}
	scope := pendingDecisionScopeFromState(state)
	hasCurrentCheckpoint := false
	for _, decision := range preview.Decisions {
		if !decision.Resolved && isAutopilotCheckpointType(decision.Type) && pendingDecisionMatchesScope(decision, scope) {
			hasCurrentCheckpoint = true
			break
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
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		for i := range file.Decisions {
			decision := &file.Decisions[i]
			if decision.Resolved || !isAutopilotCheckpointType(decision.Type) || !pendingDecisionMatchesScope(*decision, scope) {
				continue
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
			Description:     fmt.Sprintf("Phase %d checkpoint %s (%s): %s. Run: %s", phaseID, decision.ID, decision.Type, detail, command),
			Phase:           decision.Phase,
			Source:          "autopilot_checkpoint",
			CreatedAt:       decision.CreatedAt,
			RecoveryCommand: command,
		})
	}
	return blockers, nil
}
