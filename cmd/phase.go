package cmd

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

var (
	phaseNumber int
	phaseJSON   bool
)

type lifecyclePhaseSelection string

const (
	lifecyclePhaseSelectionDetail lifecyclePhaseSelection = "detail"
	lifecyclePhaseSelectionList   lifecyclePhaseSelection = "list"
	lifecyclePhaseSelectionAll    lifecyclePhaseSelection = "all"
)

// LifecyclePhaseDetail is the complete focused projection for one accepted
// phase. The first six fields also form the deliberately small --list row;
// detail-only fields are pointers so list mode cannot accidentally grow into
// the full dashboard.
type LifecyclePhaseDetail struct {
	Number       int    `json:"number"`
	Name         string `json:"name"`
	State        string `json:"state"`
	Blocked      bool   `json:"blocked"`
	Verification string `json:"verification"`
	Attempt      string `json:"attempt"`

	TotalPhases     int                                  `json:"total_phases,omitempty"`
	Objective       string                               `json:"objective,omitempty"`
	Dependencies    *LifecycleFact[[]string]             `json:"dependencies,omitempty"`
	Tasks           *LifecycleFact[[]colony.Task]        `json:"tasks,omitempty"`
	SuccessCriteria *LifecycleFact[[]string]             `json:"success_criteria,omitempty"`
	Actors          *LifecycleFact[[]LifecycleActorFact] `json:"actors,omitempty"`
	VerificationLog []colony.LifecycleVerification       `json:"verification_evidence,omitempty"`
	Evidence        []colony.LifecycleEvidence           `json:"evidence"`
	Blockers        []colony.LifecycleIssue              `json:"blockers"`
	OwnerDecisions  []colony.LifecycleDecision           `json:"owner_decisions,omitempty"`
	StateEffect     colony.LifecycleStateEffect          `json:"state_effect"`
}

// LifecyclePhaseResult keeps the shared lifecycle answer at the top level and
// selects only phase-focused material beneath it. The compatibility fields at
// the end preserve the original phase JSON contract while callers migrate to
// the typed nested Phase value.
type LifecyclePhaseResult struct {
	SchemaVersion      string             `json:"schema_version"`
	Command            string             `json:"command"`
	OutcomeKind        colony.OutcomeKind `json:"outcome_kind"`
	ProjectionRevision string             `json:"projection_revision"`
	Selection          string             `json:"selection"`
	Platform           string             `json:"platform"`

	Identity LifecycleFact[LifecycleIdentityFacts] `json:"identity"`
	Goal     LifecycleFact[string]                 `json:"goal"`
	Standing LifecycleFact[string]                 `json:"standing"`

	Phase  *LifecyclePhaseDetail  `json:"phase,omitempty"`
	Phases []LifecyclePhaseDetail `json:"phases,omitempty"`

	Evidence       []colony.LifecycleEvidence     `json:"evidence,omitempty"`
	Verification   []colony.LifecycleVerification `json:"verification,omitempty"`
	Blockers       []colony.LifecycleIssue        `json:"blockers,omitempty"`
	OwnerDecisions []colony.LifecycleDecision     `json:"owner_decisions,omitempty"`
	NextAction     LifecycleProjectedAction       `json:"next_action"`
	Alternatives   []LifecycleActionChoice        `json:"alternatives,omitempty"`
	StateEffect    colony.LifecycleStateEffect    `json:"state_effect"`

	Number      int           `json:"number,omitempty"`
	TotalPhases int           `json:"total_phases,omitempty"`
	Name        string        `json:"name,omitempty"`
	Status      string        `json:"status,omitempty"`
	Description string        `json:"description,omitempty"`
	Tasks       []colony.Task `json:"tasks,omitempty"`
	Completed   int           `json:"completed,omitempty"`
	TaskCount   int           `json:"task_count,omitempty"`
	ProgressPct int           `json:"progress_pct,omitempty"`
}

type lifecyclePhaseErrorDetails struct {
	ProjectionRevision string                      `json:"projection_revision"`
	StateEffect        colony.LifecycleStateEffect `json:"state_effect"`
	NextAction         LifecycleProjectedAction    `json:"next_action"`
	Alternatives       []LifecycleActionChoice     `json:"alternatives,omitempty"`
}

func (details lifecyclePhaseErrorDetails) String() string {
	command := lifecycleStatusActionCommand(details.NextAction)
	if command == "" {
		command = "aether status"
	}
	return fmt.Sprintf("State: unchanged\nNext: %s — %s", command, strings.TrimSpace(details.NextAction.Reason))
}

var phaseCmd = &cobra.Command{
	Use:         "phase",
	Short:       "Display focused phase details",
	Args:        cobra.NoArgs,
	Annotations: map[string]string{"aether.io/read-only": "true"},
	RunE: func(cmd *cobra.Command, args []string) error {
		root := resolveAetherRoot()
		now := time.Now().UTC()
		facts, err := loadLifecycleFacts(root, store, now)
		if err != nil {
			facts = unavailableLifecycleFacts(root, now, err.Error())
		}
		projection := projectLifecycle(facts, LifecycleViewFocused, detectPlatform())
		projection.Command = "phase"

		list, _ := cmd.Flags().GetBool("list")
		all, _ := cmd.Flags().GetBool("all")
		numberWasSet := cmd.Flags().Changed("number")
		if (list && all) || (numberWasSet && (list || all)) {
			outputLifecyclePhaseError("phase selectors --number, --list, and --all are mutually exclusive", projection)
			return nil
		}
		if numberWasSet && phaseNumber <= 0 {
			outputLifecyclePhaseError(fmt.Sprintf("phase %d not found; phase numbers start at 1", phaseNumber), projection)
			return nil
		}

		selection := lifecyclePhaseSelectionDetail
		if list {
			selection = lifecyclePhaseSelectionList
		} else if all {
			selection = lifecyclePhaseSelectionAll
		}
		result, message := buildLifecyclePhaseProjection(facts, projection, selection, phaseNumber, numberWasSet)
		if message != "" {
			outputLifecyclePhaseError(message, projection)
			return nil
		}

		if phaseJSON {
			outputOK(result)
			return nil
		}
		outputWorkflow(result, renderLifecyclePhase(result, lifecycleStatusOutputWidth()))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(phaseCmd)
	phaseCmd.Flags().IntVar(&phaseNumber, "number", 0, "Phase number to display (default: current phase)")
	phaseCmd.Flags().BoolVar(&phaseJSON, "json", false, "Output as JSON")
	phaseCmd.Flags().Bool("list", false, "List every accepted phase in phase-number order")
	phaseCmd.Flags().Bool("all", false, "Show the complete focused projection for every accepted phase")
}

func outputLifecyclePhaseError(message string, projection LifecycleProjection) {
	outputError(1, message, lifecyclePhaseErrorDetails{
		ProjectionRevision: projection.ProjectionRevision,
		StateEffect:        colony.LifecycleStateEffectNone,
		NextAction:         projection.NextAction,
		Alternatives:       append([]LifecycleActionChoice(nil), projection.Alternatives...),
	})
}

func buildLifecyclePhaseProjection(facts LifecycleFacts, projection LifecycleProjection, selection lifecyclePhaseSelection, requested int, numberWasSet bool) (LifecyclePhaseResult, string) {
	result := LifecyclePhaseResult{
		SchemaVersion:      projection.SchemaVersion,
		Command:            "phase",
		OutcomeKind:        projection.OutcomeKind,
		ProjectionRevision: projection.ProjectionRevision,
		Selection:          string(selection),
		Platform:           projection.Platform,
		Identity:           projection.Identity,
		Goal:               projection.Goal,
		Standing:           projection.Standing,
		Evidence:           append([]colony.LifecycleEvidence(nil), projection.Evidence...),
		Verification:       append([]colony.LifecycleVerification(nil), projection.Verification...),
		Blockers:           append([]colony.LifecycleIssue(nil), projection.Blockers...),
		OwnerDecisions:     append([]colony.LifecycleDecision(nil), projection.OwnerDecisions...),
		NextAction:         projection.NextAction,
		Alternatives:       append([]LifecycleActionChoice(nil), projection.Alternatives...),
		StateEffect:        colony.LifecycleStateEffectNone,
	}

	phases := append([]colony.Phase(nil), facts.Progress.Value.Phases...)
	sort.SliceStable(phases, func(i, j int) bool {
		if phases[i].ID == phases[j].ID {
			return phases[i].Name < phases[j].Name
		}
		return phases[i].ID < phases[j].ID
	})
	if len(phases) == 0 {
		return result, fmt.Sprintf("no accepted phases are available (%s lifecycle state)", facts.Progress.Source.Provenance)
	}

	current := projection.Phase.Value.CurrentNumber
	if current < 1 {
		current = phases[0].ID
	}
	if numberWasSet {
		current = requested
	}

	switch selection {
	case lifecyclePhaseSelectionList:
		result.Phases = make([]LifecyclePhaseDetail, 0, len(phases))
		for _, phase := range phases {
			result.Phases = append(result.Phases, lifecyclePhaseSummary(facts, projection, phase, current))
		}
		return result, ""
	case lifecyclePhaseSelectionAll:
		result.Phases = make([]LifecyclePhaseDetail, 0, len(phases))
		for _, phase := range phases {
			result.Phases = append(result.Phases, lifecyclePhaseDetail(facts, projection, phase, current, len(phases)))
		}
		return result, ""
	}

	for _, phase := range phases {
		if phase.ID != current {
			continue
		}
		detail := lifecyclePhaseDetail(facts, projection, phase, current, len(phases))
		result.Phase = &detail
		applyLifecyclePhaseCompatibility(&result, phase, len(phases))
		return result, ""
	}
	return result, fmt.Sprintf("phase %d not found (accepted plan has %d phases)", current, len(phases))
}

func lifecyclePhaseSummary(facts LifecycleFacts, projection LifecycleProjection, phase colony.Phase, current int) LifecyclePhaseDetail {
	blocked := strings.EqualFold(strings.TrimSpace(phase.Status), "failed")
	if phase.ID == current && len(projection.Blockers) > 0 {
		blocked = true
	}
	verification := "unreported"
	if phase.ID == current && len(projection.Verification) > 0 {
		verification = "verified"
		for _, item := range projection.Verification {
			if !item.Passed {
				verification = "blocked"
				blocked = true
				break
			}
		}
	}
	return LifecyclePhaseDetail{
		Number:       phase.ID,
		Name:         strings.TrimSpace(phase.Name),
		State:        strings.TrimSpace(phase.Status),
		Blocked:      blocked,
		Verification: verification,
		Attempt:      lifecyclePhaseAttemptSummary(facts, phase, current),
		StateEffect:  colony.LifecycleStateEffectNone,
	}
}

func lifecyclePhaseDetail(facts LifecycleFacts, projection LifecycleProjection, phase colony.Phase, current, total int) LifecyclePhaseDetail {
	detail := lifecyclePhaseSummary(facts, projection, phase, current)
	detail.TotalPhases = total
	detail.Objective = strings.TrimSpace(phase.Description)

	dependencies := lifecyclePhaseDependencies(phase.Tasks)
	dependencyFact := LifecycleFact[[]string]{Value: dependencies, Source: lifecyclePhaseFieldSource("phase dependencies", facts.Progress.Source, len(dependencies) > 0)}
	tasks := append([]colony.Task(nil), phase.Tasks...)
	taskFact := LifecycleFact[[]colony.Task]{Value: tasks, Source: lifecyclePhaseFieldSource("phase tasks", facts.Progress.Source, len(tasks) > 0)}
	criteria := append([]string(nil), phase.SuccessCriteria...)
	criteriaFact := LifecycleFact[[]string]{Value: criteria, Source: lifecyclePhaseFieldSource("phase success criteria", facts.Progress.Source, len(criteria) > 0)}
	detail.Dependencies = &dependencyFact
	detail.Tasks = &taskFact
	detail.SuccessCriteria = &criteriaFact

	actors := LifecycleFact[[]LifecycleActorFact]{Source: lifecycleUnavailableSource("phase actors", facts.Actors.Source.Path, "actor evidence is not phase-tagged for this recorded phase")}
	if phase.ID == current {
		actors = facts.Actors
		actors.Value = append([]LifecycleActorFact(nil), facts.Actors.Value...)
		actors.Source = lifecycleDerivedSource("phase actors", facts.Actors.Source)
		detail.VerificationLog = append([]colony.LifecycleVerification(nil), projection.Verification...)
		detail.Evidence = append([]colony.LifecycleEvidence(nil), projection.Evidence...)
		detail.Blockers = append([]colony.LifecycleIssue(nil), projection.Blockers...)
		detail.OwnerDecisions = append([]colony.LifecycleDecision(nil), projection.OwnerDecisions...)
	} else {
		for _, blocker := range projection.Blockers {
			if blocker.ID == fmt.Sprintf("phase-%d-failed", phase.ID) {
				detail.Blockers = append(detail.Blockers, blocker)
			}
		}
	}
	detail.Actors = &actors
	return detail
}

func lifecyclePhaseFieldSource(domain string, source LifecycleFactSource, recorded bool) LifecycleFactSource {
	result := lifecycleDerivedSource(domain, source)
	if recorded || result.Provenance != LifecycleFactConfirmed {
		return result
	}
	result.Provenance = LifecycleFactMissing
	result.Diagnostic = domain + " are not recorded in the accepted plan"
	return result
}

func lifecyclePhaseDependencies(tasks []colony.Task) []string {
	seen := map[string]struct{}{}
	for _, task := range tasks {
		for _, dependency := range task.DependsOn {
			dependency = strings.TrimSpace(dependency)
			if dependency != "" {
				seen[dependency] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for dependency := range seen {
		result = append(result, dependency)
	}
	sort.Strings(result)
	return result
}

func lifecyclePhaseAttemptSummary(facts LifecycleFacts, phase colony.Phase, current int) string {
	if phase.ID != current {
		return "unreported"
	}
	active := 0
	for _, actorFact := range facts.Actors.Value {
		if agent.IsLiveSpawnStatus(actorFact.Status) {
			active++
		}
	}
	if active > 0 {
		return fmt.Sprintf("%d active ant(s)", active)
	}
	if facts.Evidence.Value.Receipt != nil || facts.State.Value.BuildStartedAt != nil || len(facts.Actors.Value) > 0 {
		return "recorded"
	}
	return "unreported"
}

func applyLifecyclePhaseCompatibility(result *LifecyclePhaseResult, phase colony.Phase, total int) {
	result.Number = phase.ID
	result.TotalPhases = total
	result.Name = phase.Name
	result.Status = phase.Status
	result.Description = phase.Description
	result.Tasks = append([]colony.Task(nil), phase.Tasks...)
	result.TaskCount = len(phase.Tasks)
	for _, task := range phase.Tasks {
		if task.Status == colony.TaskCompleted {
			result.Completed++
		}
	}
	if result.TaskCount > 0 {
		result.ProgressPct = result.Completed * 100 / result.TaskCount
	}
}

func renderLifecyclePhase(result LifecyclePhaseResult, width int) string {
	lines := []string{}
	switch lifecyclePhaseSelection(result.Selection) {
	case lifecyclePhaseSelectionList:
		lines = append(lines, "━━ 🧱 P H A S E   L I S T ━━")
	case lifecyclePhaseSelectionAll:
		lines = append(lines, "━━ 🧱 A L L   P H A S E S ━━")
	default:
		number := 0
		if result.Phase != nil {
			number = result.Phase.Number
		}
		lines = append(lines, fmt.Sprintf("━━ 🧱 P H A S E   %d ━━", number))
	}
	lines = append(lines, "", "Colony")
	lines = append(lines, "Name: "+emptyFallback(strings.TrimSpace(result.Identity.Value.Name), "Unnamed colony"))
	lines = append(lines, "Goal: "+emptyFallback(strings.TrimSpace(result.Goal.Value), "Not recorded"))
	lines = append(lines, "Standing: "+emptyFallback(strings.TrimSpace(result.Standing.Value), "UNKNOWN"))

	switch lifecyclePhaseSelection(result.Selection) {
	case lifecyclePhaseSelectionList:
		lines = append(lines, "", "Accepted phases")
		for _, phase := range result.Phases {
			lines = append(lines, fmt.Sprintf("Phase %d: %s [%s] | blocked: %t | verification: %s | attempt: %s",
				phase.Number, emptyFallback(phase.Name, "Unnamed phase"), emptyFallback(phase.State, "unknown"), phase.Blocked, phase.Verification, phase.Attempt))
		}
	case lifecyclePhaseSelectionAll:
		for _, phase := range result.Phases {
			lines = append(lines, "")
			lines = append(lines, renderLifecyclePhaseDetailLines(phase)...)
		}
	default:
		if result.Phase != nil {
			lines = append(lines, "")
			lines = append(lines, renderLifecyclePhaseDetailLines(*result.Phase)...)
		}
	}
	lines = append(lines, "", "Next Up")
	lines = append(lines, "Command: "+emptyFallback(lifecycleStatusActionCommand(result.NextAction), "Not available"))
	lines = append(lines, "Reason: "+emptyFallback(strings.TrimSpace(result.NextAction.Reason), "Not recorded"))
	for _, choice := range result.NextAction.Choices {
		lines = append(lines, "Choice: "+lifecycleStatusChoiceLine(choice))
	}
	for _, alternative := range result.Alternatives {
		lines = append(lines, "Alternative: "+lifecycleStatusChoiceLine(alternative))
	}
	return lifecycleStatusJoin(lines, width, false)
}

func renderLifecyclePhaseDetailLines(detail LifecyclePhaseDetail) []string {
	lines := []string{fmt.Sprintf("Phase %d/%d: %s [%s]", detail.Number, detail.TotalPhases, emptyFallback(detail.Name, "Unnamed phase"), emptyFallback(detail.State, "unknown"))}
	lines = append(lines, "Objective", emptyFallback(detail.Objective, "Not recorded"))
	lines = append(lines, "Dependencies")
	if detail.Dependencies == nil || len(detail.Dependencies.Value) == 0 {
		lines = append(lines, "None recorded")
	} else {
		for _, dependency := range detail.Dependencies.Value {
			lines = append(lines, "• "+dependency)
		}
	}
	lines = append(lines, "Tasks")
	if detail.Tasks == nil || len(detail.Tasks.Value) == 0 {
		lines = append(lines, "No tasks recorded")
	} else {
		completed := 0
		for _, task := range detail.Tasks.Value {
			if task.Status == colony.TaskCompleted {
				completed++
			}
		}
		lines = append(lines, fmt.Sprintf("Progress: %d/%d tasks complete (%d%%)", completed, len(detail.Tasks.Value), completed*100/len(detail.Tasks.Value)))
		for _, task := range detail.Tasks.Value {
			id := "unnumbered"
			if task.ID != nil && strings.TrimSpace(*task.ID) != "" {
				id = strings.TrimSpace(*task.ID)
			}
			line := fmt.Sprintf("• %s [%s]: %s", id, emptyFallback(strings.TrimSpace(task.Status), "unknown"), emptyFallback(strings.TrimSpace(task.Goal), "Unnamed task"))
			if len(task.DependsOn) > 0 {
				line += " (depends on " + strings.Join(task.DependsOn, ", ") + ")"
			}
			lines = append(lines, line)
		}
	}
	lines = append(lines, "Success criteria")
	if detail.SuccessCriteria == nil || len(detail.SuccessCriteria.Value) == 0 {
		lines = append(lines, "None recorded")
	} else {
		for _, criterion := range detail.SuccessCriteria.Value {
			lines = append(lines, "• "+criterion)
		}
	}
	lines = append(lines, "Current attempt & ants", "Attempt evidence: "+emptyFallback(detail.Attempt, "unreported"))
	if detail.Actors == nil || len(detail.Actors.Value) == 0 {
		lines = append(lines, "No ants are recorded for this phase")
	} else {
		for _, actorFact := range detail.Actors.Value {
			lines = append(lines, lifecycleStatusActorLine("Ant", actorFact))
		}
	}
	lines = append(lines, "Verification & evidence", "Verification: "+emptyFallback(detail.Verification, "unreported"))
	for _, verification := range detail.VerificationLog {
		status := "Passed"
		if !verification.Passed {
			status = "Failed"
		}
		lines = append(lines, fmt.Sprintf("Gate %s: %s", emptyFallback(strings.TrimSpace(verification.Name), "unnamed"), status))
	}
	for _, evidence := range detail.Evidence {
		lines = append(lines, "Evidence: "+emptyFallback(strings.TrimSpace(evidence.Summary), emptyFallback(strings.TrimSpace(evidence.ID), "unnamed")))
	}
	if len(detail.VerificationLog) == 0 && len(detail.Evidence) == 0 {
		lines = append(lines, "No attempt-bound evidence recorded")
	}
	lines = append(lines, "Blockers")
	if len(detail.Blockers) == 0 && len(detail.OwnerDecisions) == 0 {
		lines = append(lines, "No blockers recorded")
	} else {
		for _, blocker := range detail.Blockers {
			lines = append(lines, lifecycleStatusIssueLine("Blocker", blocker))
		}
		for _, decision := range detail.OwnerDecisions {
			lines = append(lines, fmt.Sprintf("Owner decision %s: %s", emptyFallback(strings.TrimSpace(decision.ID), "unnamed"), emptyFallback(strings.TrimSpace(decision.Summary), "Reason not recorded")))
		}
	}
	return lines
}
