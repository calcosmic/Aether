package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

const (
	AgencyAcknowledgementPending = "Not yet acknowledged"
	AgencyMeasuredEffectPending  = "No measured effect yet"
)

// AgencyWorkEffect is the complete Phase 199 vocabulary for what a durable
// steering write did to work that was already active.
type AgencyWorkEffect string

const (
	AgencyWorkContinuedIndependent AgencyWorkEffect = "continued_independent_work"
	AgencyWorkConflictingJobPaused AgencyWorkEffect = "conflicting_job_paused"
	AgencyWorkNoActive             AgencyWorkEffect = "no_active_work"
)

// AgencyReceiptEvidence contains only recorded evidence that can strengthen a
// signal receipt. An absent field is not inferred from prose or process state.
type AgencyReceiptEvidence struct {
	LifecycleBoundary       *colony.LifecycleEvidence     `json:"lifecycle_boundary,omitempty"`
	LiveDelivery            *colony.LifecycleEvidence     `json:"live_delivery,omitempty"`
	Acknowledgement         *colony.SignalAcknowledgement `json:"acknowledgement,omitempty"`
	AcknowledgementEvidence *colony.LifecycleEvidence     `json:"acknowledgement_evidence,omitempty"`
	ChangedDecision         *colony.LifecycleDecision     `json:"changed_decision,omitempty"`
	EffectEvidence          *colony.LifecycleEvidence     `json:"effect_evidence,omitempty"`
	ActiveJobIDs            []string                      `json:"active_job_ids,omitempty"`
	ConflictingJobID        string                        `json:"conflicting_job_id,omitempty"`
	ConflictEvidence        *colony.LifecycleEvidence     `json:"conflict_evidence,omitempty"`
}

// AgencySignalResult is the shared JSON/visual truth for focus, feedback, and
// redirect. Receipt retains the lifecycle wire contract; the sibling fields
// are its concise owner-facing projection.
type AgencySignalResult struct {
	Created         bool                         `json:"created"`
	Reinforced      bool                         `json:"reinforced"`
	Replaced        bool                         `json:"replaced"`
	Signal          colony.PheromoneSignal       `json:"signal"`
	OwnerWording    string                       `json:"owner_wording"`
	Scope           string                       `json:"scope"`
	Delivery        string                       `json:"delivery"`
	Acknowledgement string                       `json:"acknowledgement"`
	MeasuredEffect  string                       `json:"measured_effect"`
	WorkEffect      AgencyWorkEffect             `json:"work_effect"`
	PausedJobID     string                       `json:"paused_job_id,omitempty"`
	Receipt         colony.SignalDeliveryReceipt `json:"receipt"`
}

// BuildAgencySignalResult projects a stored signal and explicit evidence into
// one receipt. It is pure: callers perform the durable write before invoking
// it, and this function never scans processes, reads files, or changes state.
func BuildAgencySignalResult(signal colony.PheromoneSignal, reinforced bool, facts AgencyReceiptEvidence) (AgencySignalResult, error) {
	signal.ID = strings.TrimSpace(signal.ID)
	signal.Type = strings.ToUpper(strings.TrimSpace(signal.Type))
	if signal.ID == "" {
		return AgencySignalResult{}, fmt.Errorf("durable signal id is required")
	}
	if signal.Type != "FOCUS" && signal.Type != "FEEDBACK" && signal.Type != "REDIRECT" {
		return AgencySignalResult{}, fmt.Errorf("unsupported agency signal type %q", signal.Type)
	}

	stored := colony.LifecycleEvidence{
		ID:      "stored-signal:" + signal.ID,
		Kind:    "durable_signal",
		Source:  "pheromones.json",
		Summary: "The runtime returned the signal after its durable write.",
	}
	receipt := colony.SignalDeliveryReceipt{
		SchemaVersion:      colony.LifecycleSchemaVersion,
		ReceiptID:          "signal-receipt:" + signal.ID,
		Command:            strings.ToLower(signal.Type),
		OutcomeKind:        colony.OutcomeKindCompleted,
		ProjectionRevision: LifecycleProjectionRevision,
		SignalID:           signal.ID,
		Evidence:           []colony.LifecycleEvidence{stored},
		Changes: []colony.LifecycleChange{{
			Target: "pheromones.json#" + signal.ID,
			Action: agencySignalWriteAction(reinforced),
		}},
		StateEffect: colony.LifecycleStateEffectCommitted,
		Transaction: colony.LifecycleTransactionReference{
			ID:    "pheromone-write:" + signal.ID,
			Stage: colony.TransactionStageCommitted,
		},
		Provenance: colony.RecoveryProvenanceConfirmed,
	}

	acknowledgement := AgencyAcknowledgementPending
	if facts.LiveDelivery != nil {
		if facts.Acknowledgement == nil || facts.AcknowledgementEvidence == nil {
			return AgencySignalResult{}, fmt.Errorf("live delivery requires a named acknowledgement and linked evidence")
		}
		if err := requireAgencyEvidence("live delivery", facts.LiveDelivery); err != nil {
			return AgencySignalResult{}, err
		}
		if err := requireAgencyEvidence("acknowledgement", facts.AcknowledgementEvidence); err != nil {
			return AgencySignalResult{}, err
		}
		if err := facts.Acknowledgement.Validate(); err != nil {
			return AgencySignalResult{}, fmt.Errorf("live delivery acknowledgement: %w", err)
		}
		if strings.TrimSpace(facts.Acknowledgement.EvidenceID) != strings.TrimSpace(facts.AcknowledgementEvidence.ID) {
			return AgencySignalResult{}, fmt.Errorf("live delivery requires linked acknowledgement evidence")
		}
		ack := *facts.Acknowledgement
		receipt.Delivery = colony.SignalDeliveryLive
		receipt.Acknowledgement = &ack
		receipt.Evidence = appendAgencyEvidence(receipt.Evidence, *facts.LiveDelivery, *facts.AcknowledgementEvidence)
		acknowledgement = fmt.Sprintf("%s (evidence %s)", ack.ActorID, ack.EvidenceID)
	} else {
		if facts.Acknowledgement != nil || facts.AcknowledgementEvidence != nil {
			return AgencySignalResult{}, fmt.Errorf("acknowledgement requires recorded live delivery")
		}
		if facts.LifecycleBoundary != nil {
			if err := requireAgencyEvidence("lifecycle boundary", facts.LifecycleBoundary); err != nil {
				return AgencySignalResult{}, err
			}
			receipt.Delivery = colony.SignalDeliveryNextSafeBoundary
			receipt.FallbackEvidence = []colony.LifecycleEvidence{*facts.LifecycleBoundary}
			receipt.Evidence = appendAgencyEvidence(receipt.Evidence, *facts.LifecycleBoundary)
		} else {
			receipt.Delivery = colony.SignalDeliveryUnsupported
			receipt.FallbackEvidence = []colony.LifecycleEvidence{stored}
		}
	}

	measuredEffect := AgencyMeasuredEffectPending
	if (facts.ChangedDecision == nil) != (facts.EffectEvidence == nil) {
		return AgencySignalResult{}, fmt.Errorf("measured effect requires a linked changed-decision receipt and effect evidence")
	}
	if facts.ChangedDecision != nil {
		decision := *facts.ChangedDecision
		if strings.TrimSpace(decision.ID) == "" {
			return AgencySignalResult{}, fmt.Errorf("changed decision id is required")
		}
		if err := requireAgencyEvidence("measured effect", facts.EffectEvidence); err != nil {
			return AgencySignalResult{}, err
		}
		if !agencyContainsID(decision.EvidenceIDs, facts.EffectEvidence.ID) {
			return AgencySignalResult{}, fmt.Errorf("measured effect requires linked changed-decision evidence")
		}
		receipt.AffectedDecisionIDs = []string{decision.ID}
		receipt.EffectEvidence = []colony.LifecycleEvidence{*facts.EffectEvidence}
		receipt.Evidence = appendAgencyEvidence(receipt.Evidence, *facts.EffectEvidence)
		receipt.Decisions = []colony.LifecycleDecision{decision}
		measuredEffect = fmt.Sprintf("decision %s (evidence %s)", decision.ID, facts.EffectEvidence.ID)
	}

	workEffect, pausedJobID, err := agencyWorkEffect(signal.Type, facts, &receipt)
	if err != nil {
		return AgencySignalResult{}, err
	}
	if err := receipt.Validate(); err != nil {
		return AgencySignalResult{}, fmt.Errorf("agency signal receipt: %w", err)
	}

	return AgencySignalResult{
		Created:         true,
		Reinforced:      reinforced,
		Replaced:        reinforced,
		Signal:          signal,
		OwnerWording:    strings.TrimSpace(extractText(signal.Content)),
		Scope:           agencySignalScope(signal),
		Delivery:        agencyDeliveryLabel(receipt.Delivery),
		Acknowledgement: acknowledgement,
		MeasuredEffect:  measuredEffect,
		WorkEffect:      workEffect,
		PausedJobID:     pausedJobID,
		Receipt:         receipt,
	}, nil
}

// agencyEvidenceFromTrophallaxisDecision resolves a real ChangedDecision and
// EffectEvidence pair from a trophallaxis packet's own recorded decision
// (cmd/trophallaxis.go, 203-10-PLAN.md) and the recruitment credit record
// recorded for it (cmd/recruitment_credit.go, this plan's Task 1) -- so
// BuildAgencySignalResult's already-correct both-or-neither join (above)
// finally receives genuine recorded data instead of remaining permanently
// starved behind a stale stub constant that has now been deleted.
//
// Read-only: it neither writes recruitment/packets.json nor
// credit/records.json. It reports ok=false (never a fabricated or default
// value) when the named packet has no recorded decision yet, or when no
// credit record naming that decision's own effect evidence exists yet --
// there is genuinely nothing to report, not an inferred credit.
func agencyEvidenceFromTrophallaxisDecision(packetID string) (AgencyReceiptEvidence, bool, error) {
	packetID = strings.TrimSpace(packetID)
	if packetID == "" || store == nil {
		return AgencyReceiptEvidence{}, false, nil
	}

	var packets trophallaxisPacketsFile
	if err := store.LoadJSON(trophallaxisPacketsPath, &packets); err != nil {
		return AgencyReceiptEvidence{}, false, nil
	}
	var decision *colony.LifecycleDecision
	for i := range packets.Entries {
		if packets.Entries[i].PacketID == packetID {
			decision = packets.Entries[i].Decision
			break
		}
	}
	if decision == nil {
		return AgencyReceiptEvidence{}, false, nil
	}

	records, err := recruitmentCreditForDecision(decision.ID)
	if err != nil {
		return AgencyReceiptEvidence{}, false, err
	}
	for _, record := range records {
		if record.Outcome == recruitmentCreditOutcomePending {
			continue
		}
		effectEvidenceID := strings.TrimSpace(record.EffectEvidenceID)
		if effectEvidenceID == "" || !agencyContainsID(decision.EvidenceIDs, effectEvidenceID) {
			continue
		}
		decisionCopy := *decision
		effect := &colony.LifecycleEvidence{
			ID:      effectEvidenceID,
			Kind:    "recruitment_credit_outcome",
			Source:  recruitmentCreditPath,
			Summary: fmt.Sprintf("Recruitment credit outcome %q for contribution %s", record.Outcome, record.ContributionID),
		}
		return AgencyReceiptEvidence{ChangedDecision: &decisionCopy, EffectEvidence: effect}, true, nil
	}
	return AgencyReceiptEvidence{}, false, nil
}

func agencyWorkEffect(signalType string, facts AgencyReceiptEvidence, receipt *colony.SignalDeliveryReceipt) (AgencyWorkEffect, string, error) {
	active := agencyIDs(facts.ActiveJobIDs)
	if len(active) == 0 {
		if strings.TrimSpace(facts.ConflictingJobID) != "" {
			return "", "", fmt.Errorf("conflicting job %q is not an active job", facts.ConflictingJobID)
		}
		return AgencyWorkNoActive, "", nil
	}
	if signalType != "REDIRECT" || strings.TrimSpace(facts.ConflictingJobID) == "" {
		return AgencyWorkContinuedIndependent, "", nil
	}

	jobID := strings.TrimSpace(facts.ConflictingJobID)
	if !agencyContainsID(active, jobID) {
		return "", "", fmt.Errorf("conflicting job %q is not an active job", jobID)
	}
	if err := requireAgencyEvidence("conflicting job", facts.ConflictEvidence); err != nil {
		return "", "", err
	}
	receipt.AffectedJobIDs = []string{jobID}
	receipt.Evidence = appendAgencyEvidence(receipt.Evidence, *facts.ConflictEvidence)
	return AgencyWorkConflictingJobPaused, jobID, nil
}

func agencySignalWriteAction(reinforced bool) string {
	if reinforced {
		return "reinforced"
	}
	return "created"
}

func agencySignalScope(signal colony.PheromoneSignal) string {
	if signal.Scope != nil && signal.Scope.Global {
		return "global"
	}
	if signal.SourcePhase != nil && *signal.SourcePhase > 0 {
		return fmt.Sprintf("phase %d", *signal.SourcePhase)
	}
	return "colony"
}

func agencyDeliveryLabel(delivery colony.SignalDelivery) string {
	return strings.ReplaceAll(string(delivery), "_", " ")
}

func requireAgencyEvidence(label string, evidence *colony.LifecycleEvidence) error {
	if evidence == nil || strings.TrimSpace(evidence.ID) == "" {
		return fmt.Errorf("%s evidence id is required", label)
	}
	return nil
}

func appendAgencyEvidence(existing []colony.LifecycleEvidence, additions ...colony.LifecycleEvidence) []colony.LifecycleEvidence {
	seen := make(map[string]bool, len(existing)+len(additions))
	for _, evidence := range existing {
		seen[strings.TrimSpace(evidence.ID)] = true
	}
	for _, evidence := range additions {
		id := strings.TrimSpace(evidence.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		existing = append(existing, evidence)
	}
	return existing
}

func agencyContainsID(ids []string, target string) bool {
	target = strings.TrimSpace(target)
	for _, id := range ids {
		if strings.TrimSpace(id) == target {
			return true
		}
	}
	return false
}

func agencyIDs(ids []string) []string {
	seen := make(map[string]bool, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

// RenderAgencySignalResult is a focused projection of the same result emitted
// as JSON. It never upgrades or infers a claim while formatting it.
func RenderAgencySignalResult(result AgencySignalResult) string {
	var b strings.Builder
	b.WriteString(renderBanner(commandEmoji(strings.ToLower(result.Signal.Type)), result.Signal.Type+" Signal"))
	b.WriteString(visualDividerStr())
	if result.Reinforced {
		b.WriteString("Existing signal reinforced.\n")
	} else {
		b.WriteString("New signal stored.\n")
	}
	b.WriteString("Signal: " + result.Signal.Type + " — " + result.OwnerWording + "\n")
	b.WriteString("Signal ID: " + result.Signal.ID + "\n")
	b.WriteString("Scope: " + result.Scope + "\n")
	b.WriteString("Delivery: " + result.Delivery + "\n")
	b.WriteString("Acknowledgement: " + result.Acknowledgement + "\n")
	b.WriteString("Effect: " + result.MeasuredEffect + "\n")
	b.WriteString("Work effect: " + agencyWorkEffectLabel(result.WorkEffect, result.PausedJobID) + "\n")
	primary := "Inspect all active colony guidance."
	if candidate, ok := nextActionCandidateFor(candidatePheromones); ok {
		primary = fmt.Sprintf("Run `%s` to inspect all active colony guidance.", candidate.Template)
	}
	alternative := "Inspect the colony to find the next safe lifecycle boundary."
	if candidate, ok := nextActionCandidateFor(candidateStatus); ok {
		alternative = fmt.Sprintf("Run `%s` to see the next safe lifecycle boundary.", candidate.Template)
	}
	b.WriteString(renderNextUp(
		primary,
		alternative,
	))
	return b.String()
}

func agencyWorkEffectLabel(effect AgencyWorkEffect, pausedJobID string) string {
	switch effect {
	case AgencyWorkConflictingJobPaused:
		return "Conflicting job paused: " + strings.TrimSpace(pausedJobID)
	case AgencyWorkNoActive:
		return "No active work"
	default:
		return "Continued independent work"
	}
}

// SwarmInterventionEvidence is deliberately small. A job can be selected only
// by exact durable task ID; checkpoint and verified-result IDs must come from
// later runtime evidence, never from a target's prose.
type SwarmInterventionEvidence struct {
	AffectedJobID            string `json:"affected_job_id,omitempty"`
	CheckpointEvidenceID     string `json:"checkpoint_evidence_id,omitempty"`
	VerifiedResultEvidenceID string `json:"verified_result_evidence_id,omitempty"`
}

// SwarmInterventionContract exposes the localized future control loop without
// pretending typed checkpoint/pause/resume events already reach this
// preflight -- swarmInterventionPreflight (cmd/swarm_cmd.go) never supplies
// CheckpointEvidenceID today, so this stays a read-only localization report
// until a caller starts passing that evidence.
type SwarmInterventionContract struct {
	SchemaVersion            string   `json:"schema_version"`
	AffectedJobID            string   `json:"affected_job_id,omitempty"`
	AffectedCheckpoint       string   `json:"affected_checkpoint,omitempty"`
	DependencyPath           []string `json:"dependency_path,omitempty"`
	IndependentJobIDs        []string `json:"independent_job_ids,omitempty"`
	IndependentWorkPolicy    string   `json:"independent_work_policy"`
	VerifiedResultRequired   bool     `json:"verified_result_required"`
	VerifiedResultEvidenceID string   `json:"verified_result_evidence_id,omitempty"`
	ResumePoint              string   `json:"resume_point"`
	CurrentCapability        string   `json:"current_capability"`
	Localization             string   `json:"localization"`
	Limitation               string   `json:"limitation"`
}

// BuildSwarmInterventionContract resolves exact task/dependency facts from an
// already-loaded lifecycle projection. The projection itself remains unchanged.
func BuildSwarmInterventionContract(projection LifecycleProjection, evidence SwarmInterventionEvidence) (SwarmInterventionContract, error) {
	contract := SwarmInterventionContract{
		SchemaVersion:          colony.LifecycleSchemaVersion,
		IndependentWorkPolicy:  "continue only recorded independent active jobs",
		VerifiedResultRequired: true,
		ResumePoint:            "unavailable until a verified result and typed checkpoint exist",
		CurrentCapability:      "read_only_localization_preflight",
		Localization:           "unsupported",
		Limitation: "No typed checkpoint evidence is supplied to this preflight; it can " +
			"identify the affected job and its dependents, but cannot integrate a " +
			"verified swarm result until checkpoint evidence exists.",
	}
	jobID := strings.TrimSpace(evidence.AffectedJobID)
	if jobID == "" {
		return contract, nil
	}

	tasks := make(map[string]colony.Task)
	active := make(map[string]colony.Task)
	for _, task := range projection.Tasks.Value {
		id := ""
		if task.ID != nil {
			id = strings.TrimSpace(*task.ID)
		}
		if id == "" {
			continue
		}
		tasks[id] = task
		if task.Status == colony.TaskInProgress {
			active[id] = task
		}
	}
	_, ok := active[jobID]
	if !ok {
		return SwarmInterventionContract{}, fmt.Errorf("affected job %q is not present as active lifecycle evidence", jobID)
	}
	contract.AffectedJobID = jobID
	affectedPath := map[string]bool{}
	agencyCollectTaskDependencies(jobID, tasks, affectedPath, map[string]bool{})
	for candidate := range tasks {
		if candidate != jobID && agencyTaskDependsOn(candidate, jobID, tasks, map[string]bool{}) {
			affectedPath[candidate] = true
		}
	}
	delete(affectedPath, jobID)
	for id := range affectedPath {
		contract.DependencyPath = append(contract.DependencyPath, id)
	}
	sort.Strings(contract.DependencyPath)
	for candidate := range active {
		if candidate == jobID || affectedPath[candidate] {
			continue
		}
		contract.IndependentJobIDs = append(contract.IndependentJobIDs, candidate)
	}
	sort.Strings(contract.IndependentJobIDs)
	contract.Localization = "affected_path_identified"

	if checkpoint := strings.TrimSpace(evidence.CheckpointEvidenceID); checkpoint != "" {
		contract.AffectedCheckpoint = checkpoint
	}
	if verified := strings.TrimSpace(evidence.VerifiedResultEvidenceID); verified != "" {
		if contract.AffectedCheckpoint == "" {
			return SwarmInterventionContract{}, fmt.Errorf("verified swarm result cannot be integrated without checkpoint evidence")
		}
		contract.VerifiedResultEvidenceID = verified
		contract.ResumePoint = "after verified finalizer result at checkpoint " + contract.AffectedCheckpoint
	}
	return contract, nil
}

func agencyCollectTaskDependencies(taskID string, tasks map[string]colony.Task, path, visited map[string]bool) {
	if visited[taskID] {
		return
	}
	visited[taskID] = true
	task, ok := tasks[taskID]
	if !ok {
		return
	}
	for _, rawDependency := range task.DependsOn {
		dependency := strings.TrimSpace(rawDependency)
		if dependency == "" {
			continue
		}
		path[dependency] = true
		agencyCollectTaskDependencies(dependency, tasks, path, visited)
	}
}

func agencyTaskDependsOn(taskID, targetID string, tasks map[string]colony.Task, visited map[string]bool) bool {
	if visited[taskID] {
		return false
	}
	visited[taskID] = true
	task, ok := tasks[taskID]
	if !ok {
		return false
	}
	for _, rawDependency := range task.DependsOn {
		dependency := strings.TrimSpace(rawDependency)
		if dependency == targetID {
			return true
		}
		if dependency != "" && agencyTaskDependsOn(dependency, targetID, tasks, visited) {
			return true
		}
	}
	return false
}

// RenderSwarmInterventionContract formats the typed preflight without adding
// live activity, delivery, pause, or resume claims.
func RenderSwarmInterventionContract(contract SwarmInterventionContract) string {
	var b strings.Builder
	b.WriteString("Swarm intervention\n")
	if contract.AffectedJobID == "" {
		b.WriteString("Affected job: Unsupported — no exact active job evidence.\n")
	} else {
		b.WriteString("Affected job: " + contract.AffectedJobID + "\n")
	}
	if len(contract.DependencyPath) > 0 {
		b.WriteString("Dependency path: " + strings.Join(contract.DependencyPath, ", ") + "\n")
	}
	b.WriteString("Independent work: " + contract.IndependentWorkPolicy + "\n")
	b.WriteString("Verified result required: yes\n")
	b.WriteString("Current capability: " + contract.CurrentCapability + "\n")
	b.WriteString("Limitation: " + contract.Limitation + "\n")
	return b.String()
}
