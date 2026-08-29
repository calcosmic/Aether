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

// applicationHistoryPhaseKey is the extra key recordInstinctApplicationsForPhase
// writes alongside instinct-apply's existing "timestamp"/"success" pair --
// it is what makes a repeated phase-end consolidation idempotent per phase.
const applicationHistoryPhaseKey = "phase"

// recordInstinctApplicationsForPhase reads instinct-deliveries.json for
// phaseID and, for each delivered instinct id that is still present and
// non-archived in instincts.json, appends one ApplicationHistory entry
// matching instinct-apply's shape exactly (cmd/internal_cmds.go) plus the
// extra "phase" key that makes this idempotent per phase. Reaching this
// function means the phase durably advanced and its gates passed --
// runPhaseEndConsolidation's own doc comment already states that contract,
// so success is recorded as true on that basis; this is not a second
// pass/fail signal. Returns the number of applications newly recorded.
// Never returns an error -- a write failure is warned to stderr.
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
			inst.ApplicationHistory = append(inst.ApplicationHistory, map[string]interface{}{
				"timestamp":                now,
				"success":                  true,
				applicationHistoryPhaseKey: phaseID,
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

// instinctAlreadyAppliedForPhase reports whether history already carries an
// entry whose "phase" key equals phaseID -- what makes a repeated phase-end
// consolidation add no further entries for that phase.
func instinctAlreadyAppliedForPhase(history []interface{}, phaseID int) bool {
	for _, raw := range history {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		p, ok := item[applicationHistoryPhaseKey]
		if !ok {
			continue
		}
		switch v := p.(type) {
		case float64:
			if int(v) == phaseID {
				return true
			}
		case int:
			if v == phaseID {
				return true
			}
		}
	}
	return false
}
