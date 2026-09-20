package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
	"github.com/spf13/cobra"
)

var stateCheckpointCmd = &cobra.Command{
	Use:   "state-checkpoint",
	Short: "Save current COLONY_STATE.json as a named checkpoint",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		name := mustGetString(cmd, "name")
		if name == "" {
			return nil
		}

		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		checkpointPath := filepath.Join("checkpoints", name+".json")
		if err := store.SaveJSON(checkpointPath, state); err != nil {
			outputError(2, fmt.Sprintf("failed to save checkpoint: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"checkpoint": name,
			"path":       checkpointPath,
		})
		return nil
	},
}

var stateWriteCmd = &cobra.Command{
	Use:   "state-write [json-blob]",
	Short: "Direct write to COLONY_STATE.json (bypasses transition validation)",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		// Positional JSON mode: replace entire state file
		if len(args) > 0 {
			if !json.Valid([]byte(args[0])) {
				outputError(1, "positional argument must be valid JSON", nil)
				return nil
			}
			// Reject if --field/--value flags also provided
			field := mustGetString(cmd, "field")
			if field != "" {
				outputError(1, "cannot use both positional JSON and --field/--value flags", nil)
				return nil
			}
			if err := store.AtomicWrite("COLONY_STATE.json", []byte(args[0])); err != nil {
				outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
				return nil
			}
			outputOK(map[string]interface{}{
				"updated":  true,
				"replaced": true,
			})
			return nil
		}

		field := mustGetString(cmd, "field")
		if field == "" {
			return nil
		}
		value := mustGetString(cmd, "value")
		if value == "" {
			return nil
		}

		// Load raw COLONY_STATE.json as map for arbitrary field setting
		data, err := store.ReadFile("COLONY_STATE.json")
		if err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}

		var m map[string]interface{}
		if err := json.Unmarshal(data, &m); err != nil {
			outputError(1, fmt.Sprintf("failed to parse COLONY_STATE.json: %v", err), nil)
			return nil
		}

		m[field] = value

		if err := store.SaveJSON("COLONY_STATE.json", m); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}

		outputOK(map[string]interface{}{
			"updated": true,
			"field":   field,
			"value":   value,
		})
		return nil
	},
}

const phaseInsertInputExample = `aether insert-phase "problem to stabilise"`

type phaseInsertPromptSession interface {
	Interactive() bool
	Ask(question string) (string, error)
}

type cobraPhaseInsertPromptSession struct {
	interactive bool
	reader      *bufio.Reader
	writer      io.Writer
}

func (p *cobraPhaseInsertPromptSession) Interactive() bool {
	return p.interactive
}

func (p *cobraPhaseInsertPromptSession) Ask(question string) (string, error) {
	if _, err := fmt.Fprintf(p.writer, "%s\n> ", question); err != nil {
		return "", err
	}
	answer, err := p.reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return "", err
	}
	return strings.TrimSpace(answer), nil
}

// phaseInsertPromptSessionFactory is injectable so command tests never depend
// on the test process's terminal. Production input still goes through Cobra's
// inherited input stream rather than reading os.Stdin directly. TTY detection
// reuses the visual runtime's existing terminal-file check.
var phaseInsertPromptSessionFactory = func(cmd *cobra.Command) phaseInsertPromptSession {
	input := cmd.InOrStdin()
	file, _ := input.(*os.File)
	return &cobraPhaseInsertPromptSession{
		interactive: file != nil && isTerminalWriter(file),
		reader:      bufio.NewReader(input),
		writer:      stderr,
	}
}

// persistCorrectiveSwarmRecovery is the durable write seam for the typed
// recovery event. Keeping it injectable lets command tests prove that a failed
// history write aborts the still-locked COLONY_STATE.json mutation.
var persistCorrectiveSwarmRecovery = func(s *storage.Store, record swarmResultRecord) error {
	_, err := canonicalCorrectiveSwarmRecovery(s, record)
	return err
}

func canonicalCorrectiveSwarmRecovery(s *storage.Store, record swarmResultRecord) (swarmResultRecord, error) {
	if s == nil {
		return record, fmt.Errorf("save swarm result: no store initialized")
	}
	record.SwarmID = strings.TrimSpace(record.SwarmID)
	record.Target = strings.TrimSpace(record.Target)
	record.Status = strings.ToLower(strings.TrimSpace(record.Status))
	record.CompletedAt = strings.TrimSpace(record.CompletedAt)
	if _, err := validateDurableSwarmID(s, record.SwarmID); err != nil {
		return record, fmt.Errorf("save swarm result: %w", err)
	}
	record.TargetFingerprint = swarmTargetFingerprint(record.Target)
	if record.TargetFingerprint == "" {
		return record, fmt.Errorf("save swarm result: target is required")
	}
	if record.Status == "" {
		return record, fmt.Errorf("save swarm result: status is required")
	}
	if err := validateSwarmRecoveryRecord(record, nil); err != nil {
		return record, fmt.Errorf("save swarm result: %w", err)
	}
	if _, err := time.Parse(time.RFC3339Nano, record.CompletedAt); err != nil {
		return record, fmt.Errorf("save swarm result: invalid completed_at: %w", err)
	}
	return record, nil
}

type phaseInsertResolveInput struct {
	PositionalIssue                   string
	PromptIssue                       string
	PromptOutcome                     string
	PromptConstraints                 string
	ExplicitName                      string
	ExplicitDescription               string
	ExplicitConstraints               string
	ExplicitAfter                     int
	AfterWasExplicit                  bool
	SpecificationItemID               string
	SpecificationRevisionPrerequisite string
	ExpectedBasePlanRevisionID        string
}

type resolvedPhaseInsertRequest struct {
	After       int
	Name        string
	Description string
	Constraints string
}

// resolvePhaseInsertRequest is the single pure merge boundary for explicit,
// shorthand, and prompted input. The mutation below it only ever receives one
// validated request. Explicit flags win; otherwise one issue sentence supplies
// safe defaults, and prompt answers use that same derivation path.
func resolvePhaseInsertRequest(input phaseInsertResolveInput, currentPhase int) (resolvedPhaseInsertRequest, error) {
	issue := strings.TrimSpace(firstNonEmpty(input.PositionalIssue, input.PromptIssue))
	name := strings.TrimSpace(input.ExplicitName)
	description := strings.TrimSpace(input.ExplicitDescription)
	constraints := strings.TrimSpace(firstNonEmpty(input.ExplicitConstraints, input.PromptConstraints))
	if strings.EqualFold(constraints, "none") {
		constraints = ""
	}

	if name == "" && issue != "" {
		name = derivePhaseInsertName(issue)
	}
	if name == "" {
		return resolvedPhaseInsertRequest{}, fmt.Errorf("flag --name is required")
	}

	if description == "" {
		switch {
		case strings.TrimSpace(input.PositionalIssue) != "":
			description = strings.TrimSpace(input.PositionalIssue)
		case issue != "":
			description = issue
			if outcome := strings.TrimSpace(input.PromptOutcome); outcome != "" && outcome != issue {
				description += "\n\nDesired outcome: " + outcome
			}
		}
	}
	if description == "" {
		return resolvedPhaseInsertRequest{}, fmt.Errorf("flag --description is required")
	}
	if constraints != "" {
		description += "\n\nConstraints: " + constraints
	}

	after := input.ExplicitAfter
	if !input.AfterWasExplicit {
		if currentPhase <= 0 {
			return resolvedPhaseInsertRequest{}, fmt.Errorf("current phase is unavailable; pass --after explicitly")
		}
		after = currentPhase
	}

	return resolvedPhaseInsertRequest{
		After:       after,
		Name:        name,
		Description: description,
		Constraints: constraints,
	}, nil
}

func derivePhaseInsertName(issue string) string {
	allWords := strings.FieldsFunc(strings.TrimSpace(issue), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	if len(allWords) == 0 {
		return ""
	}

	stopWords := map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "are": {}, "be": {}, "for": {},
		"in": {}, "is": {}, "of": {}, "on": {}, "or": {}, "the": {},
		"this": {}, "to": {}, "with": {},
	}
	useful := make([]string, 0, 6)
	for _, word := range allWords {
		if _, stop := stopWords[strings.ToLower(word)]; stop {
			continue
		}
		useful = append(useful, word)
		if len(useful) == 6 {
			break
		}
	}
	if len(useful) == 0 {
		useful = allWords
		if len(useful) > 6 {
			useful = useful[:6]
		}
	}

	name := "Stabilize " + strings.Join(useful, " ")
	for len(name) > 80 && len(useful) > 1 {
		useful = useful[:len(useful)-1]
		name = "Stabilize " + strings.Join(useful, " ")
	}
	if len(name) <= 80 {
		return name
	}

	// A single unusually long token is truncated on a rune boundary. This is
	// only for the derived label; explicit automation keeps its old contract.
	const prefix = "Stabilize "
	remaining := 80 - len(prefix)
	var b strings.Builder
	for _, r := range useful[0] {
		if b.Len()+len(string(r)) > remaining {
			break
		}
		b.WriteRune(r)
	}
	return prefix + b.String()
}

func phaseInsertHasAnyInput(cmd *cobra.Command, args []string) bool {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		return true
	}
	for _, flag := range []string{"after", "name", "description", "constraints", "spec-item", "spec-revision", "base-plan-revision"} {
		if cmd.Flags().Changed(flag) {
			return true
		}
	}
	return false
}

func collectPhaseInsertPromptAnswers(session phaseInsertPromptSession) (phaseInsertResolveInput, bool) {
	if !session.Interactive() {
		return phaseInsertResolveInput{}, false
	}

	questions := []string{
		"What is not working and needs a corrective phase?",
		"What should the inserted phase accomplish?",
		"Any hard constraints to enforce while fixing this? (or say \"none\")",
	}
	answers := make([]string, 0, len(questions))
	for i, question := range questions {
		answer, err := session.Ask(question)
		if err != nil || (i < 2 && strings.TrimSpace(answer) == "") {
			return phaseInsertResolveInput{}, false
		}
		answers = append(answers, strings.TrimSpace(answer))
	}
	return phaseInsertResolveInput{
		PromptIssue:       answers[0],
		PromptOutcome:     answers[1],
		PromptConstraints: answers[2],
	}, true
}

func outputPhaseInsertInputRequired() {
	result := map[string]interface{}{
		"status":  "input_required",
		"example": phaseInsertInputExample,
		"message": "Describe the problem in one sentence, or pass --after, --name, and --description for automation.",
	}
	visual := fmt.Sprintf("Input required. Describe the problem in one sentence.\n\nRun exactly:\n  %s\n", phaseInsertInputExample)
	outputWorkflow(result, visual)
}

func renderPhaseInsertVisual(result map[string]interface{}) string {
	var b strings.Builder
	if candidateCreated, _ := result["candidate_created"].(bool); candidateCreated {
		b.WriteString(renderBanner(commandEmoji("insert-phase"), "Corrective Phase Candidate"))
		b.WriteString(visualDividerStr())
		fmt.Fprintf(&b, "Proposed Phase %d — %s\n", intValue(result["phase_id"]), stringValue(result["name"]))
		fmt.Fprintf(&b, "   └── active plan unchanged; candidate %s awaits owner review\n", stringValue(result["candidate_id"]))
		fmt.Fprintf(&b, "\nReview: %s\n", stringValue(result["review_command"]))
		fmt.Fprintf(&b, "Accept exactly: %s\n", stringValue(result["acceptance_command"]))
		return b.String()
	}
	b.WriteString(renderBanner(commandEmoji("insert-phase"), "Corrective Phase Inserted"))
	b.WriteString(visualDividerStr())
	fmt.Fprintf(&b, "Inserted Phase %d — %s\n", intValue(result["phase_id"]), stringValue(result["name"]))
	fmt.Fprintf(&b, "   └── after Phase %d\n", intValue(result["after"]))
	if constraints := stringValue(result["constraints"]); constraints != "" {
		fmt.Fprintf(&b, "   └── constraints: %s\n", constraints)
	}
	return b.String()
}

var phaseInsertCmd = &cobra.Command{
	Use:     "phase-insert [issue]",
	Short:   "Insert a corrective phase into the active plan",
	Args:    cobra.MaximumNArgs(1),
	Aliases: []string{"insert-phase"},
	RunE:    runPhaseInsertCommand,
}

func runPhaseInsertCommand(cmd *cobra.Command, args []string) error {
	return withPlanningMutationSession(resolveAetherRootPath(), "phase-insert", func(session *planningMutationSession) error {
		return runPhaseInsertCommandInSession(session, cmd, args)
	})
}

func runPhaseInsertCommandInSession(session *planningMutationSession, cmd *cobra.Command, args []string) error {
	if store == nil {
		outputErrorMessage("no store initialized")
		return nil
	}

	input := phaseInsertResolveInput{
		PositionalIssue:                   optionalArg(args, 0),
		ExplicitName:                      mustGetStringCompatOptional(cmd, "name"),
		ExplicitDescription:               mustGetStringCompatOptional(cmd, "description"),
		ExplicitConstraints:               mustGetStringCompatOptional(cmd, "constraints"),
		ExplicitAfter:                     mustGetInt(cmd, "after"),
		AfterWasExplicit:                  cmd.Flags().Changed("after"),
		SpecificationItemID:               mustGetStringCompatOptional(cmd, "spec-item"),
		SpecificationRevisionPrerequisite: mustGetStringCompatOptional(cmd, "spec-revision"),
		ExpectedBasePlanRevisionID:        mustGetStringCompatOptional(cmd, "base-plan-revision"),
	}

	if !phaseInsertHasAnyInput(cmd, args) {
		promptInput, ok := collectPhaseInsertPromptAnswers(phaseInsertPromptSessionFactory(cmd))
		if !ok {
			outputPhaseInsertInputRequired()
			return nil
		}
		input.PromptIssue = promptInput.PromptIssue
		input.PromptOutcome = promptInput.PromptOutcome
		input.PromptConstraints = promptInput.PromptConstraints
	}

	// Current-schema plans are immutable. A manual insertion becomes a
	// reviewable candidate; the legacy direct mutation below remains only
	// for explicitly legacy/unbound colonies that have no accepted revision.
	initialState, err := loadSpecificationColonyStateInSession(session)
	if err != nil {
		outputError(1, "COLONY_STATE.json not found", nil)
		return renderedErrorExit(1)
	}
	if initialState.Plan.AcceptancePolicy == colony.PlanAcceptanceExplicitOwner || planHasCurrentAuthority(initialState.Plan) {
		request, err := resolvePhaseInsertRequest(input, initialState.CurrentPhase)
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		candidate, err := createPhaseInsertCandidateInSession(session, phaseInsertCandidateRequest{
			After: request.After, Name: request.Name, Description: request.Description, Constraints: request.Constraints,
			SpecificationItemID:               input.SpecificationItemID,
			SpecificationRevisionPrerequisite: input.SpecificationRevisionPrerequisite,
			ExpectedBasePlanRevisionID:        input.ExpectedBasePlanRevisionID,
		})
		if err != nil {
			outputError(1, err.Error(), nil)
			return renderedErrorExit(1)
		}
		result := map[string]interface{}{
			"inserted": false, "candidate_created": true,
			"candidate_id": candidate.Candidate.ID, "candidate_status": candidate.Candidate.Status,
			"phase_id": candidate.InsertedPhase.ID, "phase_semantic_id": candidate.InsertedPhase.SemanticID,
			"after": request.After, "name": request.Name, "description": request.Description, "constraints": request.Constraints,
			"base_plan_revision_id":     candidate.Candidate.BasePlanRevisionID,
			"specification_revision_id": candidate.Candidate.SpecificationRevisionID,
			"affected_semantic_ids":     append([]string(nil), candidate.Candidate.Proposal.AffectedSemanticIDs...),
			"preserved_semantic_ids":    append([]string(nil), candidate.Candidate.Proposal.PreservedSemanticIDs...),
			"review_command":            candidate.ReviewCommand, "acceptance_command": candidate.AcceptCommand,
		}
		outputWorkflow(result, renderPhaseInsertVisual(result))
		return nil
	}

	var (
		request         resolvedPhaseInsertRequest
		insertedID      int
		recoveryEventID string
		recoveryRecord  *swarmResultRecord
		mutationErr     error
	)
	state := initialState
	mutationErr = func() error {
		request, mutationErr = resolvePhaseInsertRequest(input, state.CurrentPhase)
		if mutationErr != nil {
			return mutationErr
		}
		if request.After < 0 || request.After > len(state.Plan.Phases) {
			mutationErr = fmt.Errorf("invalid after index %d (plan has %d phases)", request.After, len(state.Plan.Phases))
			return mutationErr
		}

		// Only the positional issue emitted by the swarm refusal can
		// authorize recovery. Explicit descriptions, prompt answers, and
		// arbitrary inserts continue to insert normally but never reset
		// swarm history.
		beforePlan := state.Plan
		beforePlan.Phases = clonePhases(state.Plan.Phases)
		var (
			recoveryHistory    *swarmStrikeHistory
			recoveryEscalation *colony.FlagEntry
		)
		if recoveryTarget := strings.TrimSpace(input.PositionalIssue); recoveryTarget != "" {
			history, historyErr := evaluateSwarmStrikeHistoryAgainstPlan(store, recoveryTarget, &beforePlan)
			if historyErr != nil {
				mutationErr = fmt.Errorf("verify swarm recovery eligibility: %w", historyErr)
				return mutationErr
			}
			if history.StrikeCount >= 3 && history.TargetFingerprint == swarmTargetFingerprint(recoveryTarget) {
				escalation, ok := activeSwarmEscalationForTarget(store, recoveryTarget)
				if !ok {
					mutationErr = fmt.Errorf("stage swarm recovery: active same-target escalation flag is unavailable")
					return mutationErr
				}
				recoveryHistory = &history
				recoveryEscalation = &escalation
			}
		}

		newPhase := colony.Phase{
			Name:        request.Name,
			Description: request.Description,
			Status:      colony.PhasePending,
			Tasks:       []colony.Task{},
		}
		previousPhaseCount := len(state.Plan.Phases)

		// Insert after the specified index (0-based).
		insertAt := request.After
		state.Plan.Phases = append(state.Plan.Phases[:insertAt], append([]colony.Phase{newPhase}, state.Plan.Phases[insertAt:]...)...)

		// Renumber so phase.ID == index+1 holds after every insert.
		// Production call sites index phases by ordinal (phaseNum-1); a
		// mid-slice insert carrying max+1 would leave orders like [1,3,2]
		// and misroute build and continue after the insertion point.
		oldToNew := make(map[int]int, previousPhaseCount)
		for i := range state.Plan.Phases {
			if state.Plan.Phases[i].ID > 0 {
				oldToNew[state.Plan.Phases[i].ID] = i + 1
			}
			state.Plan.Phases[i].ID = i + 1
		}
		insertedID = insertAt + 1
		if mapped, ok := oldToNew[state.CurrentPhase]; ok && state.CurrentPhase > 0 {
			state.CurrentPhase = mapped
		}

		if shouldReopenInsertedPhase(state, insertAt, previousPhaseCount) {
			state.State = colony.StateREADY
			state.CurrentPhase = insertedID
			state.Plan.Phases[insertAt].Status = colony.PhaseReady
		}

		if recoveryHistory != nil && recoveryEscalation != nil {
			record, recoveryErr := buildSwarmRecoveryRecord(
				input.PositionalIssue,
				*recoveryHistory,
				*recoveryEscalation,
				beforePlan,
				state.Plan,
				state.Plan.Phases[insertAt],
				time.Now().UTC(),
			)
			if recoveryErr != nil {
				mutationErr = fmt.Errorf("stage swarm recovery: %w", recoveryErr)
				return mutationErr
			}
			if recoveryErr := persistCorrectiveSwarmRecovery(store, record); recoveryErr != nil {
				mutationErr = fmt.Errorf("persist swarm recovery: %w", recoveryErr)
				return mutationErr
			}
			canonicalRecord, recoveryErr := canonicalCorrectiveSwarmRecovery(store, record)
			if recoveryErr != nil {
				mutationErr = fmt.Errorf("persist swarm recovery: %w", recoveryErr)
				return mutationErr
			}
			recoveryRecord = &canonicalRecord
			recoveryEventID = record.SwarmID
		}
		return nil
	}()
	if mutationErr != nil {
		outputError(1, mutationErr.Error(), nil)
		return renderedErrorExit(1)
	}
	stateBytes, err := marshalSpecificationState(state)
	if err != nil {
		outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
		return renderedErrorExit(2)
	}
	targets := []planningSessionTarget{{Root: lifecycleTransactionRootData, Path: "COLONY_STATE.json", Content: stateBytes}}
	if recoveryRecord != nil {
		recordBytes, marshalErr := json.MarshalIndent(*recoveryRecord, "", "  ")
		if marshalErr != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", marshalErr), nil)
			return renderedErrorExit(2)
		}
		targets = append(targets, planningSessionTarget{
			Root:    lifecycleTransactionRootData,
			Path:    filepath.ToSlash(filepath.Join("swarms", recoveryRecord.SwarmID, "result.json")),
			Content: append(recordBytes, '\n'),
		})
	}
	if err := commitPlanningSessionTargets(session, "phase-insert-legacy", "phase-insert-legacy", targets); err != nil {
		outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
		return renderedErrorExit(2)
	}

	result := map[string]interface{}{
		"inserted":    true,
		"phase_id":    insertedID,
		"after":       request.After,
		"name":        request.Name,
		"description": request.Description,
		"constraints": request.Constraints,
	}
	if recoveryEventID != "" {
		result["swarm_recovery_event"] = recoveryEventID
	}
	outputWorkflow(result, renderPhaseInsertVisual(result))
	return nil
}

func shouldReopenInsertedPhase(state colony.ColonyState, insertAt, previousPhaseCount int) bool {
	if state.State != colony.StateCOMPLETED {
		return false
	}
	if strings.TrimSpace(state.Milestone) == "Crowned Anthill" {
		return false
	}
	return insertAt == previousPhaseCount
}

var validateOracleStateCmd = &cobra.Command{
	Use:   "validate-oracle-state",
	Short: "Validate oracle-specific state structure",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if store == nil {
			outputErrorMessage("no store initialized")
			return nil
		}

		issues := []string{}
		files := map[string]bool{}

		// Path validation: oracle state must be under .aether/data/oracle/
		statePath := oracleStatePath()
		if !strings.HasPrefix(statePath, filepath.Join(".aether", "data", "oracle")) {
			issues = append(issues, fmt.Sprintf("oracle state path %q is outside .aether/data/oracle/", statePath))
		}

		// Check oracle/state.json
		stateData, err := store.ReadFile("oracle/state.json")
		if err != nil {
			files["state.json"] = false
			issues = append(issues, "oracle/state.json not found")
		} else if !json.Valid(stateData) {
			files["state.json"] = false
			issues = append(issues, "oracle/state.json is not valid JSON")
		} else {
			files["state.json"] = true
		}

		// Check oracle/plan.json
		planData, err := store.ReadFile("oracle/plan.json")
		if err != nil {
			files["plan.json"] = false
			issues = append(issues, "oracle/plan.json not found")
		} else if !json.Valid(planData) {
			files["plan.json"] = false
			issues = append(issues, "oracle/plan.json is not valid JSON")
		} else {
			files["plan.json"] = true
		}

		valid := len(issues) == 0

		outputOK(map[string]interface{}{
			"valid":  valid,
			"files":  files,
			"issues": issues,
		})
		return nil
	},
}

func init() {
	stateCheckpointCmd.Flags().String("name", "", "Checkpoint name (required)")

	stateWriteCmd.Flags().String("field", "", "Field to set (required)")
	stateWriteCmd.Flags().String("value", "", "Value to set (required)")

	phaseInsertCmd.Flags().Int("after", 0, "Insert after this phase index (defaults to the current phase)")
	phaseInsertCmd.Flags().String("name", "", "Phase name (derived from the issue when omitted)")
	phaseInsertCmd.Flags().String("description", "", "Phase description (defaults to the issue)")
	phaseInsertCmd.Flags().String("constraints", "", "Hard constraints retained in the phase description")
	phaseInsertCmd.Flags().String("spec-item", "", "Approved specification item that covers the inserted phase")
	phaseInsertCmd.Flags().String("spec-revision", "", "Approved successor specification revision prerequisite for the inserted phase")
	phaseInsertCmd.Flags().String("base-plan-revision", "", "Expected active plan revision; stale values are refused")

	rootCmd.AddCommand(stateCheckpointCmd)
	rootCmd.AddCommand(stateWriteCmd)
	rootCmd.AddCommand(phaseInsertCmd)
	rootCmd.AddCommand(validateOracleStateCmd)
}
