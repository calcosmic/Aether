package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// swarmResultRecord is the durable source of truth for retry history. Strike
// state is derived from these terminal records instead of a second mutable
// counter that can drift from what the swarm actually did.
type swarmResultRecord struct {
	SwarmID           string                 `json:"swarm_id"`
	Target            string                 `json:"target"`
	TargetFingerprint string                 `json:"target_fingerprint"`
	Status            string                 `json:"status"`
	Recovery          *swarmRecoveryMetadata `json:"recovery,omitempty"`
	RootCause         string                 `json:"root_cause,omitempty"`
	Solution          string                 `json:"solution,omitempty"`
	Recommendation    string                 `json:"recommendation,omitempty"`
	Workers           []swarmWorkerExecution `json:"workers,omitempty"`
	Files             []string               `json:"files,omitempty"`
	Tests             []string               `json:"tests,omitempty"`
	Blockers          []string               `json:"blockers,omitempty"`
	CompletedAt       string                 `json:"completed_at"`
	DispatchMode      string                 `json:"dispatch_mode,omitempty"`
}

const swarmRecoveryAuthorizationSource = "corrective-phase-insert"

// swarmRecoveryMetadata makes a corrective phase insertion part of the same
// append-only history as ordinary swarm outcomes. It is recovery context, not
// a successful swarm result: no worker, file, or test evidence belongs on it.
type swarmRecoveryMetadata struct {
	AuthorizationSource                string `json:"authorization_source"`
	TargetFingerprint                  string `json:"target_fingerprint"`
	EscalationFlagID                   string `json:"escalation_flag_id"`
	BeforePlanDefinitionHash           string `json:"before_plan_definition_hash"`
	AfterPlanDefinitionHash            string `json:"after_plan_definition_hash"`
	InsertedPhaseID                    int    `json:"inserted_phase_id"`
	InsertedPhaseDefinitionFingerprint string `json:"inserted_phase_definition_fingerprint"`
}

type swarmRecoveryEvidence struct {
	SwarmID                            string `json:"swarm_id"`
	CompletedAt                        string `json:"completed_at"`
	TargetFingerprint                  string `json:"target_fingerprint"`
	EscalationFlagID                   string `json:"escalation_flag_id"`
	AfterPlanDefinitionHash            string `json:"after_plan_definition_hash"`
	InsertedPhaseID                    int    `json:"inserted_phase_id"`
	InsertedPhaseDefinitionFingerprint string `json:"inserted_phase_definition_fingerprint"`
}

type swarmStrikeEvidence struct {
	SwarmID     string `json:"swarm_id"`
	Status      string `json:"status"`
	CompletedAt string `json:"completed_at"`
}

type swarmStrikeHistory struct {
	TargetFingerprint string                 `json:"target_fingerprint"`
	StrikeCount       int                    `json:"strike_count"`
	NextAttempt       int                    `json:"next_attempt"`
	Evidence          []swarmStrikeEvidence  `json:"evidence,omitempty"`
	LatestRecovery    *swarmRecoveryEvidence `json:"latest_recovery,omitempty"`
}

type datedSwarmResult struct {
	record      swarmResultRecord
	completedAt time.Time
}

func swarmTargetFingerprint(target string) string {
	normalized := normalizeSwarmTarget(target)
	if normalized == "" {
		return ""
	}
	digest := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(digest[:])
}

func normalizeSwarmTarget(target string) string {
	runes := []rune(strings.TrimSpace(target))
	for len(runes) > 0 && isSwarmTargetBoundary(runes[0]) {
		runes = runes[1:]
	}
	for len(runes) > 0 && isSwarmTargetBoundary(runes[len(runes)-1]) {
		runes = runes[:len(runes)-1]
	}
	if len(runes) == 0 {
		return ""
	}
	return strings.ToLower(strings.Join(strings.Fields(string(runes)), " "))
}

func isSwarmTargetBoundary(r rune) bool {
	if unicode.IsSpace(r) {
		return true
	}
	switch r {
	case '.', ',', '!', '?', ':', ';', '"', '\'', '`', '(', ')', '[', ']', '{', '}', '“', '”', '‘', '’', '…':
		return true
	default:
		return false
	}
}

func saveSwarmResultRecord(s *storage.Store, record swarmResultRecord) error {
	if s == nil {
		return fmt.Errorf("save swarm result: no store initialized")
	}
	record.SwarmID = strings.TrimSpace(record.SwarmID)
	record.Target = strings.TrimSpace(record.Target)
	record.Status = strings.ToLower(strings.TrimSpace(record.Status))
	record.CompletedAt = strings.TrimSpace(record.CompletedAt)
	if _, err := safeIdentifierSegment(filepath.Join(s.BasePath(), "swarms"), "swarm id", record.SwarmID); err != nil {
		return fmt.Errorf("save swarm result: %w", err)
	}
	if !isValidSwarmResultID(record.SwarmID) {
		return fmt.Errorf("save swarm result: swarm id %q does not match the durable result convention", record.SwarmID)
	}
	record.TargetFingerprint = swarmTargetFingerprint(record.Target)
	if record.TargetFingerprint == "" {
		return fmt.Errorf("save swarm result: target is required")
	}
	if record.Status == "" {
		return fmt.Errorf("save swarm result: status is required")
	}
	if err := validateSwarmRecoveryRecord(record, nil); err != nil {
		return fmt.Errorf("save swarm result: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, record.CompletedAt); err != nil {
		return fmt.Errorf("save swarm result: invalid completed_at: %w", err)
	}
	path := filepath.ToSlash(filepath.Join("swarms", record.SwarmID, "result.json"))
	if err := s.SaveJSON(path, record); err != nil {
		return fmt.Errorf("save swarm result: %w", err)
	}
	return nil
}

func evaluateSwarmStrikeHistory(s *storage.Store, target string) (swarmStrikeHistory, error) {
	fingerprint := swarmTargetFingerprint(target)
	history := swarmStrikeHistory{
		TargetFingerprint: fingerprint,
		NextAttempt:       1,
	}
	if s == nil {
		return history, fmt.Errorf("evaluate swarm strike history: no store initialized")
	}
	if fingerprint == "" {
		return history, fmt.Errorf("evaluate swarm strike history: target is required")
	}

	swarmsDir := filepath.Join(s.BasePath(), "swarms")
	entries, err := os.ReadDir(swarmsDir)
	if os.IsNotExist(err) {
		return history, nil
	}
	if err != nil {
		return history, fmt.Errorf("evaluate swarm strike history: read swarms: %w", err)
	}

	livePlan, livePlanOK := loadSwarmRecoveryLivePlan(s)
	results := make([]datedSwarmResult, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		resultPath := filepath.Join(swarmsDir, entry.Name(), "result.json")
		info, statErr := os.Lstat(resultPath)
		if statErr != nil || !info.Mode().IsRegular() {
			continue
		}
		data, readErr := os.ReadFile(resultPath)
		if readErr != nil {
			continue
		}
		var record swarmResultRecord
		if json.Unmarshal(data, &record) != nil {
			continue
		}
		record.SwarmID = strings.TrimSpace(record.SwarmID)
		record.Target = strings.TrimSpace(record.Target)
		record.Status = strings.ToLower(strings.TrimSpace(record.Status))
		record.CompletedAt = strings.TrimSpace(record.CompletedAt)
		if record.SwarmID == "" || record.SwarmID != entry.Name() {
			continue
		}
		if !isValidSwarmResultID(record.SwarmID) {
			continue
		}
		recordFingerprint := swarmTargetFingerprint(record.Target)
		if recordFingerprint == "" {
			continue
		}
		storedFingerprint := strings.TrimSpace(record.TargetFingerprint)
		if storedFingerprint != "" && storedFingerprint != recordFingerprint {
			continue
		}
		if recordFingerprint != fingerprint {
			continue
		}
		if record.Status != "completed" && record.Status != "failed" && record.Status != "blocked" && record.Status != "recovered" {
			continue
		}
		completedAt, parseErr := time.Parse(time.RFC3339Nano, record.CompletedAt)
		if parseErr != nil {
			continue
		}
		results = append(results, datedSwarmResult{record: record, completedAt: completedAt})
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].completedAt.Equal(results[j].completedAt) {
			return results[i].record.SwarmID < results[j].record.SwarmID
		}
		return results[i].completedAt.Before(results[j].completedAt)
	})
	for _, result := range results {
		if result.record.Status == "completed" {
			history.Evidence = nil
			continue
		}
		if result.record.Status == "recovered" {
			if len(history.Evidence) < 3 || !livePlanOK {
				continue
			}
			if err := validateSwarmRecoveryRecord(result.record, &livePlan); err != nil {
				continue
			}
			history.Evidence = nil
			history.LatestRecovery = swarmRecoveryEvidenceFromRecord(result.record)
			continue
		}
		history.Evidence = append(history.Evidence, swarmStrikeEvidence{
			SwarmID:     result.record.SwarmID,
			Status:      result.record.Status,
			CompletedAt: result.record.CompletedAt,
		})
	}
	history.StrikeCount = len(history.Evidence)
	history.NextAttempt = history.StrikeCount + 1
	return history, nil
}

func isValidSwarmResultID(id string) bool {
	if !strings.HasPrefix(id, "swarm-") || len(id) > 128 {
		return false
	}
	for _, r := range strings.TrimPrefix(id, "swarm-") {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			continue
		}
		return false
	}
	return len(id) > len("swarm-")
}

func newSwarmRunID(now time.Time) string {
	return fmt.Sprintf("swarm-%d", now.UTC().UnixNano())
}

func newSwarmRecoveryID(now time.Time) string {
	return fmt.Sprintf("swarm-recovery-%d", now.UTC().UnixNano())
}

// buildSwarmRecoveryRecord creates the only typed reset event accepted by the
// history evaluator. The caller must present the active stable escalation and
// the exact before/after plans from one corrective phase insertion.
func buildSwarmRecoveryRecord(
	target string,
	history swarmStrikeHistory,
	escalation colony.FlagEntry,
	before colony.Plan,
	after colony.Plan,
	inserted colony.Phase,
	recoveredAt time.Time,
) (swarmResultRecord, error) {
	normalizedTarget := normalizeSwarmTarget(target)
	targetFingerprint := swarmTargetFingerprint(normalizedTarget)
	if targetFingerprint == "" {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: target is required")
	}
	if history.TargetFingerprint != targetFingerprint || history.StrikeCount < 3 || len(history.Evidence) < 3 {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: target does not have an active three-strike history")
	}
	expectedFlagID := swarmEscalationFlagID(targetFingerprint)
	if escalation.ID != expectedFlagID ||
		escalation.Type != "blocker" ||
		escalation.Source != "escalation" ||
		escalation.Resolved {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: matching active swarm escalation is required")
	}
	if recoveredAt.IsZero() {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: recovery time is required")
	}
	lastStrikeAt, err := time.Parse(time.RFC3339Nano, history.Evidence[len(history.Evidence)-1].CompletedAt)
	if err != nil || !recoveredAt.After(lastStrikeAt) {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: recovery must follow the active strike evidence")
	}

	beforeHash, err := planDefinitionHash(before.Phases)
	if err != nil {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: hash plan before insertion: %w", err)
	}
	afterHash, err := planDefinitionHash(after.Phases)
	if err != nil {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: hash plan after insertion: %w", err)
	}
	if beforeHash == afterHash {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: corrective insertion did not change the plan definition")
	}
	insertedFingerprint, err := swarmPhaseDefinitionFingerprint(inserted)
	if err != nil {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: hash inserted phase: %w", err)
	}
	if !swarmPlanContainsPhaseDefinition(after, inserted.ID, insertedFingerprint) {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: inserted phase is absent from the post-insert plan")
	}
	if swarmPlanContainsPhaseDefinition(before, inserted.ID, insertedFingerprint) {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: corrective phase already existed before insertion")
	}

	record := swarmResultRecord{
		SwarmID:           newSwarmRecoveryID(recoveredAt),
		Target:            normalizedTarget,
		TargetFingerprint: targetFingerprint,
		Status:            "recovered",
		CompletedAt:       recoveredAt.UTC().Format(time.RFC3339Nano),
		Recovery: &swarmRecoveryMetadata{
			AuthorizationSource:                swarmRecoveryAuthorizationSource,
			TargetFingerprint:                  targetFingerprint,
			EscalationFlagID:                   expectedFlagID,
			BeforePlanDefinitionHash:           beforeHash,
			AfterPlanDefinitionHash:            afterHash,
			InsertedPhaseID:                    inserted.ID,
			InsertedPhaseDefinitionFingerprint: insertedFingerprint,
		},
	}
	if err := validateSwarmRecoveryRecord(record, &after); err != nil {
		return swarmResultRecord{}, fmt.Errorf("build swarm recovery: %w", err)
	}
	return record, nil
}

func validateSwarmRecoveryRecord(record swarmResultRecord, livePlan *colony.Plan) error {
	if record.Status != "recovered" {
		if record.Recovery != nil {
			return fmt.Errorf("recovery metadata requires recovered status")
		}
		return nil
	}
	if record.Recovery == nil {
		return fmt.Errorf("recovered status requires complete recovery metadata")
	}
	if swarmRecoveryHasOutcomeEvidence(record) {
		return fmt.Errorf("recovery event cannot contain swarm outcome evidence")
	}
	recovery := record.Recovery
	targetFingerprint := swarmTargetFingerprint(record.Target)
	if targetFingerprint == "" ||
		record.TargetFingerprint != targetFingerprint ||
		recovery.TargetFingerprint != targetFingerprint {
		return fmt.Errorf("recovery target fingerprint does not match the normalized target")
	}
	if recovery.AuthorizationSource != swarmRecoveryAuthorizationSource {
		return fmt.Errorf("recovery authorization source is invalid")
	}
	if recovery.EscalationFlagID != swarmEscalationFlagID(targetFingerprint) {
		return fmt.Errorf("recovery escalation identity does not match the target")
	}
	if !isSHA256Hex(recovery.BeforePlanDefinitionHash) ||
		!isSHA256Hex(recovery.AfterPlanDefinitionHash) ||
		recovery.BeforePlanDefinitionHash == recovery.AfterPlanDefinitionHash {
		return fmt.Errorf("recovery plan definition hashes are invalid")
	}
	if recovery.InsertedPhaseID <= 0 || !isSHA256Hex(recovery.InsertedPhaseDefinitionFingerprint) {
		return fmt.Errorf("recovery corrective phase identity is incomplete")
	}
	if livePlan == nil {
		return nil
	}
	liveHash, err := planDefinitionHash(livePlan.Phases)
	if err != nil {
		return fmt.Errorf("hash live recovery plan: %w", err)
	}
	if liveHash != recovery.AfterPlanDefinitionHash {
		return fmt.Errorf("recovery plan definition is not committed in live state")
	}
	if !swarmPlanContainsPhaseDefinition(*livePlan, recovery.InsertedPhaseID, recovery.InsertedPhaseDefinitionFingerprint) {
		return fmt.Errorf("recovery corrective phase is absent from live state")
	}
	return nil
}

func swarmRecoveryHasOutcomeEvidence(record swarmResultRecord) bool {
	return strings.TrimSpace(record.RootCause) != "" ||
		strings.TrimSpace(record.Solution) != "" ||
		strings.TrimSpace(record.Recommendation) != "" ||
		len(record.Workers) > 0 ||
		len(record.Files) > 0 ||
		len(record.Tests) > 0 ||
		len(record.Blockers) > 0 ||
		strings.TrimSpace(record.DispatchMode) != ""
}

func isSHA256Hex(value string) bool {
	value = strings.TrimSpace(value)
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func swarmPhaseDefinitionFingerprint(phase colony.Phase) (string, error) {
	return planDefinitionHash([]colony.Phase{phase})
}

func swarmPlanContainsPhaseDefinition(plan colony.Plan, phaseID int, fingerprint string) bool {
	for _, phase := range plan.Phases {
		if phase.ID != phaseID {
			continue
		}
		got, err := swarmPhaseDefinitionFingerprint(phase)
		return err == nil && got == fingerprint
	}
	return false
}

func loadSwarmRecoveryLivePlan(s *storage.Store) (colony.Plan, bool) {
	if s == nil {
		return colony.Plan{}, false
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		return colony.Plan{}, false
	}
	return state.Plan, true
}

func swarmRecoveryEvidenceFromRecord(record swarmResultRecord) *swarmRecoveryEvidence {
	if record.Recovery == nil {
		return nil
	}
	return &swarmRecoveryEvidence{
		SwarmID:                            record.SwarmID,
		CompletedAt:                        record.CompletedAt,
		TargetFingerprint:                  record.Recovery.TargetFingerprint,
		EscalationFlagID:                   record.Recovery.EscalationFlagID,
		AfterPlanDefinitionHash:            record.Recovery.AfterPlanDefinitionHash,
		InsertedPhaseID:                    record.Recovery.InsertedPhaseID,
		InsertedPhaseDefinitionFingerprint: record.Recovery.InsertedPhaseDefinitionFingerprint,
	}
}

// persistSwarmResultOutcome is the single completion seam for both runtime
// lanes. The durable result is written first; escalation is then derived from
// the same history a later pre-dispatch guard will read.
func persistSwarmResultOutcome(s *storage.Store, record swarmResultRecord) (swarmStrikeHistory, error) {
	if err := saveSwarmResultRecord(s, record); err != nil {
		return swarmStrikeHistory{}, err
	}
	history, err := evaluateSwarmStrikeHistory(s, record.Target)
	if err != nil {
		return history, err
	}
	status := strings.ToLower(strings.TrimSpace(record.Status))
	if (status == "failed" || status == "blocked") && history.StrikeCount >= 3 {
		if err := upsertSwarmEscalationFlag(s, record.Target, history); err != nil {
			return history, err
		}
	}
	return history, nil
}

func upsertSwarmEscalationFlag(s *storage.Store, target string, history swarmStrikeHistory) error {
	if s == nil {
		return fmt.Errorf("upsert swarm escalation: no store initialized")
	}
	if history.StrikeCount < 3 || len(history.Evidence) < 3 || history.TargetFingerprint == "" {
		return nil
	}

	description, err := swarmEscalationDescription(target, history)
	if err != nil {
		return fmt.Errorf("upsert swarm escalation: %w", err)
	}
	flagID := swarmEscalationFlagID(history.TargetFingerprint)
	createdAt := history.Evidence[len(history.Evidence)-1].CompletedAt
	flags := colony.FlagsFile{Version: "1.0", Decisions: []colony.FlagEntry{}}
	if loaded, ok := loadFlagsFile(s); ok {
		flags = loaded
		if flags.Decisions == nil {
			flags.Decisions = []colony.FlagEntry{}
		}
	}

	if err := s.UpdateJSONAtomically("pending-decisions.json", &flags, func() error {
		if strings.TrimSpace(flags.Version) == "" {
			flags.Version = "1.0"
		}
		if flags.Decisions == nil {
			flags.Decisions = []colony.FlagEntry{}
		}
		found := false
		for i := range flags.Decisions {
			flag := &flags.Decisions[i]
			if flag.ID != flagID {
				continue
			}
			if found {
				if !flag.Resolved {
					flag.Resolved = true
					flag.ResolvedAt = createdAt
					flag.Resolution = "deduplicated by stable swarm escalation identity"
				}
				continue
			}
			found = true
			flag.Type = "blocker"
			flag.Description = description
			flag.Source = "escalation"
			flag.Phase = nil
			if strings.TrimSpace(flag.CreatedAt) == "" {
				flag.CreatedAt = createdAt
			}
			flag.Resolved = false
			flag.ResolvedAt = ""
			flag.Resolution = ""
			flag.Acknowledged = false
			flag.AcknowledgedAt = ""
			flag.RecoveryCommand = ""
		}
		if !found {
			flags.Decisions = append(flags.Decisions, colony.FlagEntry{
				ID:          flagID,
				Type:        "blocker",
				Description: description,
				Source:      "escalation",
				CreatedAt:   createdAt,
				Resolved:    false,
			})
		}
		return nil
	}); err != nil {
		return fmt.Errorf("write pending decisions: %w", err)
	}
	return nil
}

func swarmEscalationFlagID(fingerprint string) string {
	return "swarm-escalation-" + fingerprint
}

func swarmEscalationDescription(target string, history swarmStrikeHistory) (string, error) {
	evidenceIDs := swarmStrikeEvidenceIDsForRuntime(history.Evidence)
	description := fmt.Sprintf(
		"Swarm stopped after %d consecutive failed or blocked attempts for %q. Evidence: %s. Add a corrective phase before retrying.",
		history.StrikeCount,
		normalizeSwarmTarget(target),
		strings.Join(evidenceIDs, ", "),
	)
	if sanitized, err := colony.SanitizeSignalContent(description); err == nil {
		return sanitized, nil
	}
	fallback := fmt.Sprintf(
		"Swarm target %s stopped after %d consecutive failed or blocked attempts. Evidence: %s. Add a corrective phase before retrying.",
		history.TargetFingerprint,
		history.StrikeCount,
		strings.Join(evidenceIDs, ", "),
	)
	sanitized, err := colony.SanitizeSignalContent(fallback)
	if err != nil {
		return "", fmt.Errorf("sanitize escalation description: %w", err)
	}
	return sanitized, nil
}

func swarmStrikeEvidenceIDsForRuntime(evidence []swarmStrikeEvidence) []string {
	ids := make([]string, 0, len(evidence))
	for _, item := range evidence {
		ids = append(ids, item.SwarmID)
	}
	return ids
}

func swarmInsertPhaseCommand(target string) string {
	value := strings.Join(strings.Fields(strings.TrimSpace(target)), " ")
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`"`, `\"`,
		`$`, `\$`,
		"`", "\\`",
	)
	return `aether insert-phase "` + replacer.Replace(value) + `"`
}

func swarmArchitecturalConcernResult(target string, history swarmStrikeHistory) map[string]interface{} {
	evidenceIDs := swarmStrikeEvidenceIDsForRuntime(history.Evidence)
	next := swarmInsertPhaseCommand(target)
	return map[string]interface{}{
		"mode":                "destroy",
		"status":              "architectural_concern",
		"dispatch_mode":       "refused",
		"autopilot_available": true,
		"target":              strings.TrimSpace(target),
		"strike_count":        history.StrikeCount,
		"next_attempt":        history.NextAttempt,
		"evidence":            history.Evidence,
		"evidence_ids":        evidenceIDs,
		"worker_count":        0,
		"recommendation":      "Repeated swarm failure needs a corrective architectural phase before more workers are dispatched.",
		"next":                next,
		"watch":               false,
	}
}
