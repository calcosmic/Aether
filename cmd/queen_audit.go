package cmd

import (
	"fmt"
	"strings"
	"time"
)

// QueenAuditFile consolidates queen decisions from multiple sources into a single
// historical record. Per D-05: created after phase completion.
// Per D-07: READ-ONLY consolidation -- source files are never modified.
type QueenAuditFile struct {
	Phase       int               `json:"phase"`
	GeneratedAt string            `json:"generated_at"`
	Decisions   []QueenAuditEntry `json:"decisions"`
}

// QueenAuditEntry represents a single autonomous decision by the queen.
// Per D-06: schema includes timestamp, decision_type, input_finding, action_taken, rationale.
type QueenAuditEntry struct {
	Timestamp    string `json:"timestamp"`
	DecisionType string `json:"decision_type"` // gate_evaluate, recovery_action, wave_advance, auto_resolve, fixer_dispatch, escalation
	InputFinding string `json:"input_finding"`
	ActionTaken  string `json:"action_taken"`
	Rationale    string `json:"rationale"`
}

// consolidateQueenAudit reads three source files and produces a single audit.
// Per D-08: reads from existing files using their known schemas.
// Per D-07: source files are never modified.
// Missing source files are treated as empty data -- no error for missing files.
func consolidateQueenAudit(phaseNum int) QueenAuditFile {
	audit := QueenAuditFile{
		Phase:       phaseNum,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Source 1: queen-state-{N}.json (gate decisions from Phase 97)
	if qs, err := queenStateRead(phaseNum); err == nil {
		for _, d := range qs.Decisions {
			audit.Decisions = append(audit.Decisions, QueenAuditEntry{
				Timestamp:    qs.GeneratedAt,
				DecisionType: "gate_evaluate",
				InputFinding: fmt.Sprintf("Gate %s: status=%s, tier=%s", d.GateName, d.Status, d.ClassificationTier),
				ActionTaken:  d.QueenRecommendation,
				Rationale:    d.Rationale,
			})
		}
		for _, e := range qs.EscalationLog {
			audit.Decisions = append(audit.Decisions, QueenAuditEntry{
				Timestamp:    e.Timestamp,
				DecisionType: "escalation",
				InputFinding: fmt.Sprintf("Circuit breaker tripped: %v", e.BreakerTripped),
				ActionTaken:  e.EscalationAction,
				Rationale:    e.Rationale,
			})
		}
	}

	// Source 2: recovery-log-{N}.json (recovery actions from Phase 96)
	if rl, err := recoveryLogReadPhase(phaseNum); err == nil {
		for _, entry := range rl.Entries {
			decisionType := "recovery_action"
			if entry.ActionTaken == "escalate" {
				decisionType = "escalation"
			} else if strings.Contains(entry.ActionTaken, "fixer") {
				decisionType = "fixer_dispatch"
			} else if strings.Contains(entry.ActionTaken, "retry") {
				decisionType = "auto_resolve"
			}
			audit.Decisions = append(audit.Decisions, QueenAuditEntry{
				Timestamp:    entry.Timestamp,
				DecisionType: decisionType,
				InputFinding: fmt.Sprintf("Worker %s (task %s): %s", entry.Failure.WorkerName, entry.Failure.TaskID, entry.Failure.ErrorMessage),
				ActionTaken:  entry.ActionTaken,
				Rationale:    entry.Detail,
			})
		}
	}

	// Source 3: wave-summary-{N}.json (wave lifecycle from Phase 98)
	if ws, err := readWaveSummary(phaseNum); err == nil {
		for _, wave := range ws.Waves {
			for _, r := range wave.Recovered {
				audit.Decisions = append(audit.Decisions, QueenAuditEntry{
					Timestamp:    ws.CompletedAt,
					DecisionType: "wave_advance",
					InputFinding: fmt.Sprintf("Wave %d recovery: worker %s", wave.Wave, r.WorkerName),
					ActionTaken:  fmt.Sprintf("recovered via %s", r.Method),
					Rationale:    r.Detail,
				})
			}
			if wave.Escalated > 0 {
				audit.Decisions = append(audit.Decisions, QueenAuditEntry{
					Timestamp:    ws.CompletedAt,
					DecisionType: "escalation",
					InputFinding: fmt.Sprintf("Wave %d: %d escalated workers", wave.Wave, wave.Escalated),
					ActionTaken:  "escalate_to_human",
					Rationale:    "Unrecovered failures after recovery budget exhausted",
				})
			}
		}
	}

	return audit
}

// writeAuditFile persists the audit file to queen-audit-{phaseNum}.json.
func writeAuditFile(phaseNum int, audit QueenAuditFile) error {
	rel := fmt.Sprintf("queen-audit-%d.json", phaseNum)
	return store.SaveJSON(rel, audit)
}

// readAuditFile reads the audit file from queen-audit-{phaseNum}.json.
func readAuditFile(phaseNum int) (QueenAuditFile, error) {
	var audit QueenAuditFile
	rel := fmt.Sprintf("queen-audit-%d.json", phaseNum)
	err := store.LoadJSON(rel, &audit)
	return audit, err
}
