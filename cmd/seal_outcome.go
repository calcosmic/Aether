package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/colony"
)

// SealOutcome is the durable lifecycle outcome shared by the seal preflight,
// transaction, receipt, and renderer. Keeping this as the colony type prevents
// a command-only force flag from becoming a second source of completion truth.
type SealOutcome = colony.SealOutcome

// SealCaller names the authority asking for a seal. Only a human owner using
// the direct command may request the destructive forced-incomplete branch.
type SealCaller string

const (
	SealCallerDirectOwner SealCaller = "direct_owner"
	SealCallerWrapper     SealCaller = "wrapper"
	SealCallerFinalizer   SealCaller = "finalizer"
	SealCallerAutopilot   SealCaller = "autopilot"
	SealCallerWorker      SealCaller = "worker"
	SealCallerRecovery    SealCaller = "recovery"
)

type SealPreflightRequest struct {
	Caller SealCaller `json:"caller"`
	Force  bool       `json:"force,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

type SealOwnerCheckpoint struct {
	ID       string `json:"id"`
	Summary  string `json:"summary"`
	Resolved bool   `json:"resolved"`
}

type SealPreservedContent struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Scope  string `json:"scope"`
}

type SealUnresolvedItem struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	EvidenceIDs []string `json:"evidence_ids,omitempty"`
}

// SealPreflight is a pure decision over one immutable LifecycleFacts snapshot.
// It is deliberately richer than a bool: the same enumerated evidence drives
// confirmation, transaction staging, receipts, and the final truthful screen.
type SealPreflight struct {
	Eligible           bool                       `json:"eligible"`
	Caller             SealCaller                 `json:"caller"`
	Disposition        colony.SealDisposition     `json:"disposition"`
	OutcomeKind        colony.OutcomeKind         `json:"outcome_kind"`
	CompletedPhaseIDs  []int                      `json:"completed_phase_ids,omitempty"`
	IncompletePhaseIDs []int                      `json:"incomplete_phase_ids,omitempty"`
	CompletedTaskIDs   []string                   `json:"completed_task_ids,omitempty"`
	IncompleteTaskIDs  []string                   `json:"incomplete_task_ids,omitempty"`
	PassedGates        []colony.GateResultEntry   `json:"passed_gates,omitempty"`
	FailedGates        []colony.GateResultEntry   `json:"failed_gates,omitempty"`
	SkippedGates       []colony.GateResultEntry   `json:"skipped_gates,omitempty"`
	Evidence           []colony.LifecycleEvidence `json:"evidence,omitempty"`
	MissingEvidence    []colony.LifecycleIssue    `json:"missing_evidence,omitempty"`
	OwnerCheckpoints   []SealOwnerCheckpoint      `json:"owner_checkpoints,omitempty"`
	ResidualRisks      []colony.LifecycleIssue    `json:"residual_risks,omitempty"`
	UnresolvedItems    []SealUnresolvedItem       `json:"unresolved_items,omitempty"`
	PreservedContents  []SealPreservedContent     `json:"preserved_contents"`
	OwnerReason        string                     `json:"owner_reason,omitempty"`
	Rollback           *colony.LifecycleRollback  `json:"rollback,omitempty"`
	PrimaryNext        string                     `json:"primary_next"`
	OptionalNext       string                     `json:"optional_next,omitempty"`
	ProjectionRevision string                     `json:"projection_revision,omitempty"`
	FactsCapturedAt    string                     `json:"facts_captured_at,omitempty"`
}

// BuildSealPreflight decides completion truth before any durable effect. It
// returns the complete refused preflight alongside an error so callers can
// render every unresolved item instead of collapsing the refusal to a count.
func BuildSealPreflight(facts LifecycleFacts, request SealPreflightRequest) (SealPreflight, error) {
	request.Reason = strings.TrimSpace(request.Reason)
	if request.Caller == "" {
		request.Caller = SealCallerDirectOwner
	}

	preflight := SealPreflight{
		Caller:             request.Caller,
		Disposition:        colony.SealDispositionVerified,
		OutcomeKind:        colony.OutcomeKindVerifiedCompletion,
		PreservedContents:  sealPreservedContents(),
		PrimaryNext:        "aether status",
		OptionalNext:       "aether entomb",
		ProjectionRevision: LifecycleProjectionRevision,
	}
	if !facts.CapturedAt.IsZero() {
		preflight.FactsCapturedAt = facts.CapturedAt.UTC().Format("2006-01-02T15:04:05Z")
	}

	if request.Force {
		preflight.Disposition = colony.SealDispositionForcedIncomplete
		preflight.OutcomeKind = colony.OutcomeKindForcedIncompleteClosure
		preflight.OwnerReason = request.Reason
	}

	preflight.collectWork(facts.State.Value.Plan.Phases)
	preflight.collectGates(facts.Verification.Value.Gates)
	preflight.collectEvidence(facts)
	preflight.collectOwnerCheckpoints(facts.Blockers.Value)
	preflight.collectConsistency(facts)
	if len(facts.State.Value.Plan.Phases) == 0 {
		preflight.addUnresolved("missing_plan", "plan", "no colony plan exists to verify", []string{"fact:state"})
	}
	preflight.sort()

	if request.Force {
		if request.Caller != SealCallerDirectOwner {
			return preflight, fmt.Errorf("forced-incomplete seal requires a direct owner invocation; %s callers cannot construct or relay --force", request.Caller)
		}
		if request.Reason == "" {
			return preflight, fmt.Errorf("forced-incomplete seal requires a nonblank --reason from the direct owner")
		}
		preflight.Eligible = true
		preflight.Rollback = &colony.LifecycleRollback{
			CheckpointID: "pre-seal-owner-checkpoint",
			Evidence: []colony.LifecycleEvidence{{
				ID:      "seal-force-authority",
				Kind:    "owner_override",
				Source:  "aether seal --force --reason",
				Summary: request.Reason,
			}},
		}
		return preflight, preflight.Validate()
	}

	if len(preflight.UnresolvedItems) > 0 {
		return preflight, fmt.Errorf("normal seal requires verified completion; unresolved: %s", preflight.unresolvedSummary())
	}
	preflight.Eligible = true
	return preflight, preflight.Validate()
}

func (p SealPreflight) Validate() error {
	if !p.Disposition.Valid() {
		return fmt.Errorf("invalid seal disposition %q", p.Disposition)
	}
	if !p.OutcomeKind.Valid() {
		return fmt.Errorf("invalid seal outcome kind %q", p.OutcomeKind)
	}
	if p.PrimaryNext != "aether status" {
		return fmt.Errorf("sealed colony must keep aether status as the primary next action")
	}
	if p.OptionalNext != "aether entomb" {
		return fmt.Errorf("entomb must remain an optional owner action")
	}
	if !p.Eligible {
		return fmt.Errorf("seal preflight is not eligible")
	}
	switch p.Disposition {
	case colony.SealDispositionVerified:
		if p.OutcomeKind != colony.OutcomeKindVerifiedCompletion {
			return fmt.Errorf("verified seal has outcome kind %q", p.OutcomeKind)
		}
		if len(p.UnresolvedItems) > 0 || len(p.IncompletePhaseIDs) > 0 || len(p.IncompleteTaskIDs) > 0 || len(p.FailedGates) > 0 || len(p.SkippedGates) > 0 || len(p.MissingEvidence) > 0 {
			return fmt.Errorf("verified seal contains unresolved completion evidence")
		}
		if p.OwnerReason != "" || p.Rollback != nil {
			return fmt.Errorf("verified seal contains forced-incomplete authority fields")
		}
	case colony.SealDispositionForcedIncomplete:
		if p.OutcomeKind != colony.OutcomeKindForcedIncompleteClosure {
			return fmt.Errorf("forced-incomplete seal has outcome kind %q", p.OutcomeKind)
		}
		if p.Caller != SealCallerDirectOwner || strings.TrimSpace(p.OwnerReason) == "" {
			return fmt.Errorf("forced-incomplete seal requires direct owner authority and reason")
		}
		if len(p.UnresolvedItems) == 0 {
			return fmt.Errorf("forced-incomplete seal must enumerate unresolved evidence")
		}
		if p.Rollback == nil {
			return fmt.Errorf("forced-incomplete seal requires rollback evidence")
		}
		if err := p.Rollback.Validate(); err != nil {
			return fmt.Errorf("forced-incomplete rollback: %w", err)
		}
	}
	return nil
}

func (p *SealPreflight) collectWork(phases []colony.Phase) {
	for _, phase := range phases {
		if phase.Status == colony.PhaseCompleted {
			p.CompletedPhaseIDs = append(p.CompletedPhaseIDs, phase.ID)
		} else {
			p.IncompletePhaseIDs = append(p.IncompletePhaseIDs, phase.ID)
			p.addUnresolved("phase", fmt.Sprintf("phase-%d", phase.ID), fmt.Sprintf("phase %d (%s) is %s", phase.ID, strings.TrimSpace(phase.Name), phase.Status), nil)
		}
		for index, task := range phase.Tasks {
			id := fmt.Sprintf("%d.%d", phase.ID, index+1)
			if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
				id = strings.TrimSpace(*task.ID)
			}
			if task.Status == colony.TaskCompleted {
				p.CompletedTaskIDs = append(p.CompletedTaskIDs, id)
			} else {
				p.IncompleteTaskIDs = append(p.IncompleteTaskIDs, id)
				p.addUnresolved("task", id, fmt.Sprintf("task %s is %s", id, task.Status), nil)
			}
		}
	}
}

func (p *SealPreflight) collectGates(gates []colony.GateResultEntry) {
	for _, gate := range gates {
		detail := strings.ToLower(gate.Name + " " + gate.Detail)
		switch {
		case gate.Passed:
			p.PassedGates = append(p.PassedGates, gate)
		case strings.Contains(detail, "skip"):
			p.SkippedGates = append(p.SkippedGates, gate)
			p.addUnresolved("skipped_gate", gate.Name, fmt.Sprintf("gate %s was skipped: %s", gate.Name, strings.TrimSpace(gate.Detail)), []string{"gate:" + gate.Name})
		default:
			p.FailedGates = append(p.FailedGates, gate)
			p.addUnresolved("failed_gate", gate.Name, fmt.Sprintf("gate %s failed: %s", gate.Name, strings.TrimSpace(gate.Detail)), []string{"gate:" + gate.Name})
		}
	}
}

func (p *SealPreflight) collectEvidence(facts LifecycleFacts) {
	required := []LifecycleFactSource{
		facts.State.Source,
		facts.Progress.Source,
		facts.Verification.Source,
		facts.Evidence.Source,
	}
	if facts.Blockers.Source.Provenance == LifecycleFactMalformed || facts.Blockers.Source.Provenance == LifecycleFactUnavailable {
		required = append(required, facts.Blockers.Source)
	}
	for _, source := range required {
		id := "fact:" + strings.TrimSpace(source.Domain)
		if source.Provenance == LifecycleFactConfirmed {
			p.Evidence = append(p.Evidence, colony.LifecycleEvidence{ID: id, Kind: "lifecycle_fact", Source: source.Path, Summary: source.Domain + " facts confirmed"})
			continue
		}
		diagnostic := strings.TrimSpace(source.Diagnostic)
		if diagnostic == "" {
			diagnostic = fmt.Sprintf("source is %s", source.Provenance)
		}
		issue := colony.LifecycleIssue{ID: id, Summary: fmt.Sprintf("%s evidence is %s: %s", source.Domain, source.Provenance, diagnostic), EvidenceIDs: []string{id}}
		p.MissingEvidence = append(p.MissingEvidence, issue)
		p.addUnresolved("missing_evidence", id, issue.Summary, issue.EvidenceIDs)
	}
	for _, artifact := range facts.Verification.Value.Artifacts {
		artifact = strings.TrimSpace(artifact)
		if artifact == "" {
			continue
		}
		p.Evidence = append(p.Evidence, colony.LifecycleEvidence{ID: "artifact:" + artifact, Kind: "verification_artifact", Source: artifact, Summary: "recorded verification artifact"})
	}
}

func (p *SealPreflight) collectOwnerCheckpoints(flags []colony.FlagEntry) {
	for _, flag := range flags {
		flagType := strings.ToLower(strings.TrimSpace(flag.Type))
		checkpoint := SealOwnerCheckpoint{ID: strings.TrimSpace(flag.ID), Summary: strings.TrimSpace(flag.Description), Resolved: flag.Resolved}
		if flagType != "issue" {
			p.OwnerCheckpoints = append(p.OwnerCheckpoints, checkpoint)
		}
		if flag.Resolved {
			continue
		}
		id := checkpoint.ID
		if id == "" {
			id = "unresolved-owner-checkpoint"
		}
		issue := colony.LifecycleIssue{ID: id, Summary: checkpoint.Summary, EvidenceIDs: []string{"owner_checkpoint:" + id}}
		p.ResidualRisks = append(p.ResidualRisks, issue)
		if flagType == "issue" {
			continue
		}
		kind := "owner_checkpoint"
		if flagType == "blocker" {
			kind = "blocker"
		}
		p.addUnresolved(kind, id, checkpoint.Summary, issue.EvidenceIDs)
	}
}

func (p *SealPreflight) collectConsistency(facts LifecycleFacts) {
	state := facts.State.Value
	progress := facts.Progress.Value
	consistent := state.CurrentPhase == progress.CurrentPhase && sealPhasesEqual(state.Plan.Phases, progress.Phases)
	if consistent {
		p.Evidence = append(p.Evidence, colony.LifecycleEvidence{ID: "facts:consistent", Kind: "consistency", Source: facts.State.Source.Path + "," + facts.Progress.Source.Path, Summary: "state and progress facts agree"})
		return
	}
	issue := colony.LifecycleIssue{ID: "facts:conflict", Summary: "state and progress lifecycle facts conflict", EvidenceIDs: []string{"fact:state", "fact:progress"}}
	p.MissingEvidence = append(p.MissingEvidence, issue)
	p.addUnresolved("conflicting_evidence", issue.ID, issue.Summary, issue.EvidenceIDs)
}

func sealPhasesEqual(left, right []colony.Phase) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].ID != right[i].ID || left[i].Status != right[i].Status || len(left[i].Tasks) != len(right[i].Tasks) {
			return false
		}
		for j := range left[i].Tasks {
			leftID, rightID := "", ""
			if left[i].Tasks[j].ID != nil {
				leftID = strings.TrimSpace(*left[i].Tasks[j].ID)
			}
			if right[i].Tasks[j].ID != nil {
				rightID = strings.TrimSpace(*right[i].Tasks[j].ID)
			}
			if leftID != rightID || left[i].Tasks[j].Status != right[i].Tasks[j].Status {
				return false
			}
		}
	}
	return true
}

func (p *SealPreflight) addUnresolved(kind, id, summary string, evidenceIDs []string) {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		summary = id
	}
	p.UnresolvedItems = append(p.UnresolvedItems, SealUnresolvedItem{Kind: kind, ID: id, Summary: summary, EvidenceIDs: evidenceIDs})
}

func (p *SealPreflight) sort() {
	sort.Ints(p.CompletedPhaseIDs)
	sort.Ints(p.IncompletePhaseIDs)
	sort.Strings(p.CompletedTaskIDs)
	sort.Strings(p.IncompleteTaskIDs)
	sort.SliceStable(p.PassedGates, func(i, j int) bool { return p.PassedGates[i].Name < p.PassedGates[j].Name })
	sort.SliceStable(p.FailedGates, func(i, j int) bool { return p.FailedGates[i].Name < p.FailedGates[j].Name })
	sort.SliceStable(p.SkippedGates, func(i, j int) bool { return p.SkippedGates[i].Name < p.SkippedGates[j].Name })
	sort.SliceStable(p.Evidence, func(i, j int) bool { return p.Evidence[i].ID < p.Evidence[j].ID })
	sort.SliceStable(p.MissingEvidence, func(i, j int) bool { return p.MissingEvidence[i].ID < p.MissingEvidence[j].ID })
	sort.SliceStable(p.OwnerCheckpoints, func(i, j int) bool { return p.OwnerCheckpoints[i].ID < p.OwnerCheckpoints[j].ID })
	sort.SliceStable(p.ResidualRisks, func(i, j int) bool { return p.ResidualRisks[i].ID < p.ResidualRisks[j].ID })
	sort.SliceStable(p.UnresolvedItems, func(i, j int) bool {
		if p.UnresolvedItems[i].Kind == p.UnresolvedItems[j].Kind {
			return p.UnresolvedItems[i].ID < p.UnresolvedItems[j].ID
		}
		return p.UnresolvedItems[i].Kind < p.UnresolvedItems[j].Kind
	})
}

func (p SealPreflight) unresolvedSummary() string {
	items := make([]string, 0, len(p.UnresolvedItems))
	for _, item := range p.UnresolvedItems {
		items = append(items, formatSealUnresolvedItem(item))
	}
	return strings.Join(items, "; ")
}

func formatSealUnresolvedItem(item SealUnresolvedItem) string {
	kind := strings.TrimSpace(item.Kind)
	id := strings.TrimSpace(item.ID)
	summary := strings.TrimSpace(item.Summary)
	if summary == "" {
		summary = id
	}
	if kind == "" && id == "" {
		return summary
	}
	label := kind
	if label == "" {
		label = id
	} else if id != "" {
		label += ":" + id
	}
	return fmt.Sprintf("[%s] %s", label, summary)
}

func sealPreservedContents() []SealPreservedContent {
	return []SealPreservedContent{
		{ID: "active_state", Source: ".aether/data/COLONY_STATE.json", Scope: "project"},
		{ID: "crowned_record", Source: ".aether/CROWNED-ANTHILL.md", Scope: "project"},
		{ID: "findings", Source: ".aether/data/review-findings.json", Scope: "project"},
		{ID: "learnings", Source: ".aether/data/COLONY_STATE.json", Scope: "colony"},
		{ID: "lifecycle_evidence", Source: ".aether/data/lifecycle", Scope: "project"},
		{ID: "owner_checkpoints", Source: ".aether/data/pending-decisions.json", Scope: "project"},
		{ID: "rollback", Source: ".aether/data/lifecycle", Scope: "project"},
		{ID: "signals", Source: ".aether/data/pheromones.json", Scope: "project"},
		{ID: "wisdom", Source: ".aether/QUEEN.md", Scope: "colony"},
	}
}

// SealConfirmationCopy is the byte-for-byte UI contract. The normal branch
// names verification; the forced branch names both incompleteness and the
// fact that the override does not verify completion.
func SealConfirmationCopy(preflight SealPreflight) string {
	if preflight.Disposition == colony.SealDispositionForcedIncomplete {
		return fmt.Sprintf("Force-seal this incomplete colony with %d unresolved item(s)? This records an owner override; it does not verify completion. [y/N]", len(preflight.UnresolvedItems))
	}
	return "Seal this verified colony and write its Crowned Anthill record? [y/N]"
}
