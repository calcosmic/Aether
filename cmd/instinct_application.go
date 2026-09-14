package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// instinctDeliveriesPath is the append-only, phase-keyed ledger of which
// instincts a worker was actually given -- the input FEED-03 was missing.
// Recording here (never inferring from "instincts are always injected") is
// what makes T-198.1-09's negative proof (TestQueenPromotionNeverHappensWithoutRecordedUse)
// possible: a delivery only exists when the instinct's own action text was
// genuinely present in the capsule a worker received.
const instinctDeliveriesPath = "instinct-deliveries.json"

// instinctDelivery is one recorded fact: this instinct's action text was
// present in the capsule a worker received on this phase, on this workflow.
type instinctDelivery struct {
	Phase      int    `json:"phase"`
	InstinctID string `json:"instinct_id"`
	Workflow   string `json:"workflow"`
	RecordedAt string `json:"recorded_at"`
}

// instinctDeliveryFile is the on-disk shape of instinct-deliveries.json.
type instinctDeliveryFile struct {
	Deliveries []instinctDelivery `json:"deliveries"`
}

// recordInstinctDeliveries loads instincts.json and, for every non-archived
// instinct whose trimmed Action is non-empty and is contained in capsule,
// appends one delivery for (phaseID, instinct.ID) -- unless that exact pair
// already exists (T-198.1-10: a repeated check on one phase must not inflate
// the count). Writes go through store.UpdateJSONAtomically so concurrent
// wave workers cannot clobber each other's deliveries. Returns the number of
// NEW deliveries recorded. Never returns an error -- a write failure is
// warned to stderr; bookkeeping must never fail a build.
func recordInstinctDeliveries(phaseID int, workflow, capsule string) int {
	if store == nil || strings.TrimSpace(capsule) == "" {
		return 0
	}

	file := loadInstinctFileOrEmpty(store)
	if len(file.Instincts) == 0 {
		return 0
	}

	candidates := make([]instinctDelivery, 0)
	now := time.Now().UTC().Format(time.RFC3339)
	for _, inst := range file.Instincts {
		if inst.Archived {
			continue
		}
		action := strings.TrimSpace(inst.Action)
		if action == "" {
			continue
		}
		if !strings.Contains(capsule, action) {
			continue
		}
		candidates = append(candidates, instinctDelivery{
			Phase:      phaseID,
			InstinctID: inst.ID,
			Workflow:   workflow,
			RecordedAt: now,
		})
	}
	if len(candidates) == 0 {
		return 0
	}

	newCount := 0
	var df instinctDeliveryFile
	err := store.UpdateJSONAtomically(instinctDeliveriesPath, &df, func() error {
		existing := make(map[string]bool, len(df.Deliveries))
		for _, d := range df.Deliveries {
			existing[deliveryKey(d.Phase, d.InstinctID)] = true
		}
		for _, d := range candidates {
			key := deliveryKey(d.Phase, d.InstinctID)
			if existing[key] {
				continue
			}
			df.Deliveries = append(df.Deliveries, d)
			existing[key] = true
			newCount++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record instinct deliveries: %v\n", err)
		return 0
	}
	return newCount
}

// deliveryKey is the (phase, instinct) dedup key shared by the read and
// write side of recordInstinctDeliveries's idempotency check.
func deliveryKey(phase int, instinctID string) string {
	return fmt.Sprintf("%d|%s", phase, instinctID)
}

// capsuleForDispatch returns dispatch.ContextCapsule when non-empty (the
// wrapper/plan-only lane and the native in-process lane both populate it
// directly on the dispatch). Otherwise it loads the phase's persisted build
// manifest and returns its own ContextCapsule -- the fallback the delegate
// lane's persistExternalBuildHandoffs needs, since the codex.WorkerDispatch
// it constructs for recordDispatchWorkerOutcome carries no capsule of its
// own (cmd/codex_build_finalize.go).
func capsuleForDispatch(dispatch codex.WorkerDispatch) string {
	if strings.TrimSpace(dispatch.ContextCapsule) != "" {
		return dispatch.ContextCapsule
	}
	// loadCodexContinueManifest calls store.LoadJSON directly with no nil
	// guard of its own -- some existing dispatch-path tests (e.g.
	// TestBuildDispatchStartsHeartbeatMonitor) legitimately run with a nil
	// store, and this fallback must degrade to "no capsule available"
	// rather than panic (found while proving 198.1-03's full-suite run).
	if store == nil {
		return ""
	}
	manifest := loadCodexContinueManifest(dispatch.Phase)
	if !manifest.Present {
		return ""
	}
	return manifest.Data.ContextCapsule
}

// recordInstinctApplicationsForPhase reads instinct-deliveries.json for
// phaseID and, for each delivered instinct id that is still present and
// non-archived in instincts.json, appends one typed ApplicationHistory
// entry (colony.InstinctApplicationEntry, SYN-204-05/06) carrying the
// extra "phase" key that makes this idempotent per phase.
//
// The outcome is never inferred from the phase having merely advanced --
// reaching this function proves nothing about whether any PARTICULAR
// instinct helped (SYN-204-05, ruling (c)). It is looked up from the
// evidence-gated credit ledger (cmd/recruitment_credit.go) via
// recruitmentCreditForContribution, keyed by this instinct's ID and the
// deterministic decision identifier phaseApplicationDecisionID declares
// (cmd/application_evidence.go, 204-02-PLAN.md). When a credit record
// exists for (instinct, decision), the entry carries that record's real
// outcome and its record ID. When none exists yet, the entry carries the
// pending outcome and no credit record ID -- a delivery that has not yet
// been measured, never a default or inferred success.
//
// Returns the number of applications newly recorded. Never returns an
// error -- a write failure is warned to stderr.
func recordInstinctApplicationsForPhase(phaseID int) int {
	if store == nil {
		return 0
	}

	var df instinctDeliveryFile
	if err := store.LoadJSON(instinctDeliveriesPath, &df); err != nil {
		return 0
	}
	delivered := make(map[string]bool)
	for _, d := range df.Deliveries {
		if d.Phase == phaseID {
			delivered[d.InstinctID] = true
		}
	}
	if len(delivered) == 0 {
		return 0
	}

	decisionIDs := latestPhaseDecisionIDs(phaseID)

	newApplications := 0
	var file colony.InstinctsFile
	err := store.UpdateJSONAtomically("instincts.json", &file, func() error {
		for i := range file.Instincts {
			inst := &file.Instincts[i]
			if inst.Archived || !delivered[inst.ID] {
				continue
			}
			if instinctAlreadyAppliedForPhase(inst.ApplicationHistory, phaseID) {
				continue
			}
			now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
			inst.Provenance.LastApplied = &now
			inst.Provenance.ApplicationCount++
			outcome, creditRecordID := instinctApplicationOutcome(inst.ID, decisionIDs)
			inst.ApplicationHistory = append(inst.ApplicationHistory, colony.InstinctApplicationEntry{
				Timestamp:      now,
				Phase:          phaseID,
				Outcome:        outcome,
				CreditRecordID: creditRecordID,
			})
			newApplications++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record instinct applications: %v\n", err)
		return 0
	}
	return newApplications
}

// latestPhaseDecisionIDs returns the deterministic decision identifiers
// (phaseApplicationDecisionID, cmd/application_evidence.go) for every
// decision-kind knowledge delta on phaseID's latest durable build attempt,
// or nil when there is none -- the SAME decision IDs
// recordPhaseApplicationCredit derives, so a credit record recorded
// against one of these IDs is the one this function can find.
func latestPhaseDecisionIDs(phaseID int) []string {
	_, latestAttempt, ok := loadLatestBuildAttempt(phaseID)
	if !ok {
		return nil
	}
	var decisionIDs []string
	decisionIndex := 0
	for _, delta := range latestAttempt.KnowledgeDeltas {
		if delta.Kind != buildKnowledgeDeltaKindDecision {
			continue
		}
		decisionIDs = append(decisionIDs, phaseApplicationDecisionID(latestAttempt.ID, decisionIndex))
		decisionIndex++
	}
	return decisionIDs
}

// instinctApplicationOutcome looks up the real credit record for
// (instinctID, decisionID) across every candidate decision ID, returning
// the first one found's outcome and record ID. No decision ID to check, or
// no credit record recorded against any of them yet, returns the pending
// outcome and no credit record ID -- never a default or inferred success
// (SYN-204-05/06).
func instinctApplicationOutcome(instinctID string, decisionIDs []string) (string, string) {
	for _, decisionID := range decisionIDs {
		record, ok, err := recruitmentCreditForContribution(instinctID, decisionID)
		if err != nil || !ok {
			continue
		}
		return string(record.Outcome), record.RecordID
	}
	return string(recruitmentCreditOutcomePending), ""
}

// instinctAlreadyAppliedForPhase reports whether history already carries an
// entry whose Phase equals phaseID -- what makes a repeated phase-end
// consolidation add no further entries for that phase. Reads both shapes
// (Phase is populated by UnmarshalJSON for either the new typed shape or
// the old "phase" key), so a legacy entry recorded for this phase is
// correctly recognized too.
func instinctAlreadyAppliedForPhase(history []colony.InstinctApplicationEntry, phaseID int) bool {
	for _, entry := range history {
		if entry.Phase == phaseID {
			return true
		}
	}
	return false
}
