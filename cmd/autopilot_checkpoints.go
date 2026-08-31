package cmd

import (
	"crypto/sha256"
	"fmt"
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
	candidate.ID = "cp_" + key[len(key)-20:]
	candidate.Resolved = false
	candidate.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	candidate.Evidence = boundedCheckpointEvidence(candidate.Evidence)
	candidate.SourcePaths = boundedCheckpointEvidence(candidate.SourcePaths)
	stampPendingDecisionScope(&candidate, scope)

	var file PendingDecisionFile
	result := candidate
	created := false
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		if file.Decisions == nil {
			file.Decisions = []PendingDecision{}
		}
		for i := range file.Decisions {
			existing := &file.Decisions[i]
			if existing.CheckpointKey != key || existing.Type != candidate.Type || !pendingDecisionMatchesScope(*existing, scope) {
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
			return nil
		}
		file.Decisions = append(file.Decisions, candidate)
		result = candidate
		created = true
		return nil
	}); err != nil {
		return PendingDecision{}, false, fmt.Errorf("persist %s checkpoint: %w", candidate.Type, err)
	}
	return result, created, nil
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
	return fmt.Sprintf("aether decision-answer --question %s --answer 'confirmed' --phase %d", shellQuote(checkpointDecisionQuestion(decision)), phaseID)
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

func autopilotCheckpointSealBlockers(state colony.ColonyState) []colony.FlagEntry {
	if store == nil {
		return nil
	}
	var file PendingDecisionFile
	if err := store.LoadJSON(pendingDecisionsFile, &file); err != nil {
		return nil
	}
	scope := pendingDecisionScopeFromState(state)
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
	return blockers
}
