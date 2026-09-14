package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
)

const (
	autopilotReplanDecisionType   = "replan"
	autopilotReplanDecisionSource = "autopilot-replan"
	autopilotLessonReferenceLimit = 20
	autopilotLessonContentLimit   = 120
)

// confirmedAutopilotLesson is a compact, evidence-rich projection of one
// learning record. It deliberately keeps the worker and gate provenance used
// to admit the entry, so a replan request can be audited without treating a
// raw observation as a confirmed lesson.
type confirmedAutopilotLesson struct {
	EntryID        string   `json:"entry_id"`
	Content        string   `json:"content"`
	ContentHash    string   `json:"content_hash"`
	Phase          int      `json:"phase"`
	RunID          string   `json:"run_id,omitempty"`
	Workers        []string `json:"workers"`
	GatesPassed    int      `json:"gates_passed"`
	GatesTotal     int      `json:"gates_total"`
	EvidenceAt     string   `json:"evidence_at"`
	PlanRevisionID string   `json:"plan_revision_id"`
	PlanRevisionAt string   `json:"plan_revision_at"`
}

// activePlanLessonBoundary returns the accepted revision identity and the
// timestamp after which learning evidence may influence another replan. A
// revisioned plan must have a parseable active revision; legacy plans use
// Plan.GeneratedAt and a deterministic legacy identity.
func activePlanLessonBoundary(plan colony.Plan) (string, time.Time, error) {
	if revision, ok := activePlanRevision(plan); ok {
		createdAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(revision.CreatedAt))
		if err != nil {
			return "", time.Time{}, fmt.Errorf("active plan revision %q has invalid created_at: %w", revision.ID, err)
		}
		return revision.ID, createdAt.UTC(), nil
	}
	if strings.TrimSpace(plan.ActiveRevisionID) != "" {
		return "", time.Time{}, fmt.Errorf("active plan revision %q is missing from revision history", plan.ActiveRevisionID)
	}
	if plan.GeneratedAt == nil {
		// Some pre-revision colonies omitted generated_at entirely. The zero
		// boundary preserves those colonies' ability to accumulate new typed
		// evidence; modern plans always take the stricter timestamp paths above.
		return activePlanRevisionID(plan), time.Time{}, nil
	}
	return activePlanRevisionID(plan), plan.GeneratedAt.UTC(), nil
}

// confirmedAutopilotLessonsSincePlan is a pure selector over the live plan and
// the real learn.Entry shape written by captureContinueLearning. It performs
// no promotion and reads no raw observation/memory stores.
//
// LEARN-01 (204-02-PLAN.md Task 2, ruling (b)): status admission runs
// through the shared vocabulary (learningVerifiedEntries,
// cmd/learning_status_vocabulary.go) rather than its own inline status
// comparison -- a hypothesis-status entry is refused here exactly as a
// disproven one already was, closing the same gap ruling (b) named at
// cmd/colony_prime_context.go's render filter. Every OTHER admission rule
// below (the phase and evidence-phase floors, the blocked classification,
// the plan-revision boundary, the all-gates-passed requirement, and the
// all-workers-completed requirement) is stronger than a status check, not a
// substitute for it, and is unchanged.
func confirmedAutopilotLessonsSincePlan(plan colony.Plan, entries []learn.Entry) ([]confirmedAutopilotLesson, error) {
	revisionID, boundary, err := activePlanLessonBoundary(plan)
	if err != nil {
		return nil, err
	}

	entries = learningVerifiedEntries(entries)
	candidates := make([]confirmedAutopilotLesson, 0, len(entries))
	for _, entry := range entries {
		content := strings.TrimSpace(entry.Content)
		if entry.Phase <= 0 || entry.Evidence.Phase <= 0 || content == "" || entry.Classification == learn.ClassBlocked {
			continue
		}
		evidenceAt, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(entry.Evidence.Timestamp))
		if err != nil || !evidenceAt.After(boundary) {
			continue
		}
		if len(entry.Evidence.Workers) == 0 || entry.Evidence.GatesTotal <= 0 || entry.Evidence.GatesPassed != entry.Evidence.GatesTotal {
			continue
		}
		workers := make([]string, 0, len(entry.Evidence.Workers))
		workersCompleted := true
		for _, worker := range entry.Evidence.Workers {
			if !strings.EqualFold(strings.TrimSpace(worker.Status), "completed") {
				workersCompleted = false
				break
			}
			name := strings.TrimSpace(worker.Name)
			if name == "" {
				name = strings.TrimSpace(worker.Caste)
			}
			workers = append(workers, name)
		}
		if !workersCompleted {
			continue
		}
		sort.Strings(workers)
		hash := sha256Sum(normalizeDecisionText(content))
		candidates = append(candidates, confirmedAutopilotLesson{
			EntryID:        strings.TrimSpace(entry.ID),
			Content:        content,
			ContentHash:    hash,
			Phase:          entry.Phase,
			RunID:          strings.TrimSpace(entry.Evidence.RunID),
			Workers:        workers,
			GatesPassed:    entry.Evidence.GatesPassed,
			GatesTotal:     entry.Evidence.GatesTotal,
			EvidenceAt:     evidenceAt.UTC().Format(time.RFC3339Nano),
			PlanRevisionID: revisionID,
			PlanRevisionAt: boundary.UTC().Format(time.RFC3339Nano),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].EvidenceAt != candidates[j].EvidenceAt {
			return candidates[i].EvidenceAt < candidates[j].EvidenceAt
		}
		if candidates[i].Phase != candidates[j].Phase {
			return candidates[i].Phase < candidates[j].Phase
		}
		if candidates[i].ContentHash != candidates[j].ContentHash {
			return candidates[i].ContentHash < candidates[j].ContentHash
		}
		return candidates[i].EntryID < candidates[j].EntryID
	})

	lessons := make([]confirmedAutopilotLesson, 0, len(candidates))
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		if _, duplicate := seen[candidate.ContentHash]; duplicate {
			continue
		}
		seen[candidate.ContentHash] = struct{}{}
		lessons = append(lessons, candidate)
	}
	return lessons, nil
}

func loadConfirmedAutopilotLessonsSincePlan(plan colony.Plan) ([]confirmedAutopilotLesson, error) {
	if store == nil {
		return nil, fmt.Errorf("no store initialized")
	}
	entries, err := learn.NewColonyStore(store).List(learn.EntryFilter{})
	if err != nil {
		return nil, fmt.Errorf("load confirmed autopilot lessons: %w", err)
	}
	return confirmedAutopilotLessonsSincePlan(plan, entries)
}

// autopilotReplanCadence is a durable projection of interval progress under
// one accepted plan revision. CompletedSinceRevision is reconstructed from
// plan state, while LastHandledBoundary is reconstructed from every replan
// decision for that revision, including decisions the owner already resolved.
// NextBoundary is the earliest interval boundary not represented by that
// history. DueBoundary and CheckpointPhaseID are populated only after current
// durable state has reached that boundary.
type autopilotReplanCadence struct {
	PlanRevisionID         string
	CompletedSinceRevision int
	LastHandledBoundary    int
	NextBoundary           int
	DueBoundary            int
	CheckpointPhaseID      int
}

// projectAutopilotReplanCadence reconstructs replan interval progress without
// wall-clock or invocation-local input. Modern plans compare the live phase
// statuses with the immutable active revision snapshot, so accepting a new
// revision is the only reset. Plans that predate revision snapshots use the
// deterministic compatibility rule of counting every currently completed
// phase from the beginning of the legacy plan.
func projectAutopilotReplanCadence(plan colony.Plan, decisions []PendingDecision, interval int) (autopilotReplanCadence, error) {
	projection := autopilotReplanCadence{PlanRevisionID: activePlanRevisionID(plan)}
	if len(plan.Phases) == 0 {
		if interval > 0 {
			projection.NextBoundary = interval
		}
		return projection, nil
	}

	baseline := map[int]string{}
	hasRevisionSnapshot := false
	if revision, ok := activePlanRevision(plan); ok {
		if len(revision.Phases) > 0 {
			hasRevisionSnapshot = true
			for _, phase := range revision.Phases {
				if _, duplicate := baseline[phase.ID]; duplicate {
					return autopilotReplanCadence{}, fmt.Errorf("active plan revision %q duplicates phase %d", revision.ID, phase.ID)
				}
				baseline[phase.ID] = phase.Status
			}
		}
	} else if strings.TrimSpace(plan.ActiveRevisionID) != "" {
		return autopilotReplanCadence{}, fmt.Errorf("active plan revision %q is missing from revision history", plan.ActiveRevisionID)
	}

	completionPhaseIDs := make([]int, 0, len(plan.Phases))
	seenCurrent := make(map[int]struct{}, len(plan.Phases))
	for _, phase := range plan.Phases {
		if _, duplicate := seenCurrent[phase.ID]; duplicate {
			return autopilotReplanCadence{}, fmt.Errorf("current plan duplicates phase %d", phase.ID)
		}
		seenCurrent[phase.ID] = struct{}{}
		baselineStatus, existedAtAcceptance := baseline[phase.ID]
		if hasRevisionSnapshot && !existedAtAcceptance {
			return autopilotReplanCadence{}, fmt.Errorf("active plan revision %q has no snapshot for current phase %d", projection.PlanRevisionID, phase.ID)
		}
		if phase.Status != colony.PhaseCompleted {
			continue
		}
		if hasRevisionSnapshot && baselineStatus == colony.PhaseCompleted {
			continue
		}
		completionPhaseIDs = append(completionPhaseIDs, phase.ID)
	}
	projection.CompletedSinceRevision = len(completionPhaseIDs)

	if interval <= 0 {
		return projection, nil
	}
	completionOrdinal := make(map[int]int, len(completionPhaseIDs))
	for index, phaseID := range completionPhaseIDs {
		completionOrdinal[phaseID] = index + 1
	}
	for _, decision := range decisions {
		if decision.Type != autopilotReplanDecisionType || strings.TrimSpace(decision.PlanRevisionID) != projection.PlanRevisionID {
			continue
		}
		checkpointPhase := decision.LatestCheckpointPhase
		if checkpointPhase <= 0 && decision.Phase != nil {
			checkpointPhase = *decision.Phase
		}
		ordinal, found := completionOrdinal[checkpointPhase]
		if !found {
			continue
		}
		handledBoundary := (ordinal / interval) * interval
		if handledBoundary > projection.LastHandledBoundary {
			projection.LastHandledBoundary = handledBoundary
		}
	}

	projection.NextBoundary = interval
	if projection.LastHandledBoundary > 0 {
		projection.NextBoundary = projection.LastHandledBoundary + interval
	}
	if projection.CompletedSinceRevision >= projection.NextBoundary {
		projection.DueBoundary = projection.NextBoundary
		projection.CheckpointPhaseID = completionPhaseIDs[projection.DueBoundary-1]
	}
	return projection, nil
}

func lessonAwareReplanDue(phasesCompleted, interval int, lessons []confirmedAutopilotLesson, bypass bool) bool {
	return !bypass && interval > 0 && phasesCompleted > 0 && phasesCompleted%interval == 0 && len(lessons) > 0
}

// legacyInteractiveReplanDue keeps the pre-revision interactive checkpoint
// recoverable for state files that never recorded any plan boundary at all.
// Headless mode never takes this compatibility branch: unattended replanning
// always requires confirmed evidence.
func legacyInteractiveReplanDue(plan colony.Plan, phasesCompleted, interval int, bypass, headless bool) bool {
	return !headless && !bypass && plan.GeneratedAt == nil && strings.TrimSpace(plan.ActiveRevisionID) == "" && len(plan.Revisions) == 0 && interval > 0 && phasesCompleted > 0 && phasesCompleted%interval == 0
}

func compactAutopilotLessonReferences(lessons []confirmedAutopilotLesson) []string {
	byHash := make(map[string]confirmedAutopilotLesson, len(lessons))
	for _, lesson := range lessons {
		hash := strings.TrimSpace(lesson.ContentHash)
		if hash == "" {
			hash = sha256Sum(normalizeDecisionText(lesson.Content))
		}
		if _, exists := byHash[hash]; !exists {
			lesson.ContentHash = hash
			byHash[hash] = lesson
		}
	}
	hashes := make([]string, 0, len(byHash))
	for hash := range byHash {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	refs := make([]string, 0, min(len(hashes), autopilotLessonReferenceLimit))
	for _, hash := range hashes {
		lesson := byHash[hash]
		content := strings.Join(strings.Fields(strings.TrimSpace(lesson.Content)), " ")
		if len(content) > autopilotLessonContentLimit {
			content = content[:autopilotLessonContentLimit-1] + "…"
		}
		shortHash := hash
		if len(shortHash) > 12 {
			shortHash = shortHash[:12]
		}
		refs = append(refs, fmt.Sprintf("phase %d [%s] %s", lesson.Phase, shortHash, content))
		if len(refs) == autopilotLessonReferenceLimit {
			break
		}
	}
	return refs
}

func replanDecisionDescription(revisionID string, lessonCount, firstCheckpoint, latestCheckpoint int) string {
	return fmt.Sprintf(
		"Replan suggested for plan revision %s: %d unique evidence-confirmed lesson(s) accumulated between phase checkpoints %d and %d. Review with `aether plan`; the active plan has not been changed.",
		revisionID, lessonCount, firstCheckpoint, latestCheckpoint,
	)
}

// upsertAutopilotReplanDecision persists exactly one unresolved replan note for
// the active plan revision. A newly accepted revision closes the prior note;
// repeated checkpoints only expand the existing note's evidence window.
func upsertAutopilotReplanDecision(state colony.ColonyState, checkpointPhase int, lessons []confirmedAutopilotLesson, now time.Time) (PendingDecision, error) {
	if store == nil {
		return PendingDecision{}, fmt.Errorf("no store initialized")
	}
	if checkpointPhase <= 0 {
		return PendingDecision{}, fmt.Errorf("replan checkpoint phase must be positive")
	}
	revisionID, revisionAt, err := activePlanLessonBoundary(state.Plan)
	if err != nil {
		return PendingDecision{}, err
	}
	refs := compactAutopilotLessonReferences(lessons)
	if len(refs) == 0 {
		return PendingDecision{}, fmt.Errorf("replan checkpoint requires at least one confirmed lesson")
	}
	scope := pendingDecisionScopeFromState(state)
	key := stableAutopilotCheckpointKey(autopilotReplanDecisionType, 0, revisionID, scope)
	createdAt := now.UTC().Format(time.RFC3339Nano)
	candidate := PendingDecision{
		ID:                    "rp_" + key[len(key)-20:],
		Type:                  autopilotReplanDecisionType,
		Description:           replanDecisionDescription(revisionID, len(refs), checkpointPhase, checkpointPhase),
		Phase:                 &checkpointPhase,
		Source:                autopilotReplanDecisionSource,
		Resolved:              false,
		CreatedAt:             createdAt,
		CheckpointKey:         key,
		PlanRevisionID:        revisionID,
		PlanRevisionAt:        revisionAt.UTC().Format(time.RFC3339Nano),
		LessonCount:           len(refs),
		FirstCheckpointPhase:  checkpointPhase,
		LatestCheckpointPhase: checkpointPhase,
		LessonReferences:      refs,
	}
	stampPendingDecisionScope(&candidate, scope)

	var file PendingDecisionFile
	result := candidate
	if err := store.UpdateJSONAtomically(pendingDecisionsFile, &file, func() error {
		if file.Decisions == nil {
			file.Decisions = []PendingDecision{}
		}
		for i := range file.Decisions {
			existing := &file.Decisions[i]
			if existing.Type != autopilotReplanDecisionType || existing.Resolved || !pendingDecisionMatchesScope(*existing, scope) {
				continue
			}
			if existing.PlanRevisionID != revisionID {
				existing.Resolved = true
				existing.Resolution = fmt.Sprintf("Superseded by accepted plan revision %s", revisionID)
				existing.ResolvedAt = createdAt
				continue
			}

			existing.CheckpointKey = key
			existing.PlanRevisionAt = candidate.PlanRevisionAt
			existing.LessonReferences = uniqueSortedStrings(append(existing.LessonReferences, refs...))
			if len(existing.LessonReferences) > autopilotLessonReferenceLimit {
				existing.LessonReferences = existing.LessonReferences[:autopilotLessonReferenceLimit]
			}
			existing.LessonCount = len(existing.LessonReferences)
			if existing.FirstCheckpointPhase <= 0 || checkpointPhase < existing.FirstCheckpointPhase {
				existing.FirstCheckpointPhase = checkpointPhase
			}
			if checkpointPhase > existing.LatestCheckpointPhase {
				existing.LatestCheckpointPhase = checkpointPhase
			}
			phase := existing.LatestCheckpointPhase
			existing.Phase = &phase
			existing.Description = replanDecisionDescription(revisionID, existing.LessonCount, existing.FirstCheckpointPhase, existing.LatestCheckpointPhase)
			stampPendingDecisionScope(existing, scope)
			result = *existing
			return nil
		}
		file.Decisions = append(file.Decisions, candidate)
		result = candidate
		return nil
	}); err != nil {
		return PendingDecision{}, fmt.Errorf("persist replan checkpoint: %w", err)
	}
	return result, nil
}
