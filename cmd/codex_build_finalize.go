package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/spf13/cobra"
)

type codexExternalBuildCompletion struct {
	DispatchManifest *codexBuildManifest              `json:"dispatch_manifest,omitempty"`
	Manifest         *codexBuildManifest              `json:"manifest,omitempty"`
	Dispatches       []codexExternalBuildWorkerResult `json:"dispatches,omitempty"`
	Results          []codexExternalBuildWorkerResult `json:"results,omitempty"`
	Workers          []codexExternalBuildWorkerResult `json:"workers,omitempty"`
	Claims           *codexBuildClaims                `json:"claims,omitempty"`

	// submittedRaw holds the wrapper's original decoded JSON value
	// (map[string]any / []any / scalars), envelope-unwrapped, exactly as
	// submitted -- before any field this Go struct doesn't declare was
	// silently dropped by encoding/json.Unmarshal. It is set only by
	// loadExternalBuildCompletion. It stays nil for completions constructed
	// in-process (no submitted bytes exist), such as
	// stageBuildAttemptCompletionFromWorkerRuns and every test that builds a
	// codexExternalBuildCompletion literal -- those fall back to a struct
	// round-trip via structuralInput().
	//
	// Deliberately unexported: encoding/json.Marshal skips unexported
	// fields, so packet digests via jsonSHA256 stay byte-identical, and
	// invopop/jsonschema reflection also skips it, so the committed schema
	// does not drift.
	submittedRaw any
}

// structuralInput returns the JSON value structural validation
// (validateCompletionPacketStructure) should inspect: the wrapper's
// submitted, envelope-unwrapped JSON when this completion came from
// loadExternalBuildCompletion, or a struct round-trip via
// completionPacketAsRaw when no submitted bytes exist (completions built
// in-process, e.g. stageBuildAttemptCompletionFromWorkerRuns, or a
// codexExternalBuildCompletion literal constructed directly by a test).
func (c codexExternalBuildCompletion) structuralInput() (any, error) {
	if c.submittedRaw != nil {
		return c.submittedRaw, nil
	}
	return completionPacketAsRaw(c)
}

type codexExternalBuildWorkerResult struct {
	Stage         string `json:"stage,omitempty"`
	Wave          int    `json:"wave,omitempty"`
	ExecutionWave int    `json:"execution_wave,omitempty"`
	Caste         string `json:"caste,omitempty"`
	Name          string `json:"name"`
	AntName       string `json:"ant_name,omitempty"`
	Task          string `json:"task,omitempty"`
	Status        string `json:"status"`
	Summary       string `json:"summary,omitempty"`
	TaskID        string `json:"task_id,omitempty"`
	TaskIndex     int    `json:"task_index,omitempty"`
	// CoveredTaskIDs names every OTHER manifest dispatch this worker's real
	// work actually covered, for the case where the WRAPPER (not the
	// runtime) bundled several manifest-listed dispatches into one worker
	// call the manifest still lists as separate dispatches. This is the
	// WORKER's own claim, submitted via the completion packet -- contrast
	// codexBuildDispatch.CoveredTaskIDs (cmd/codex_build.go), which the
	// RUNTIME writes when it groups a dependent chain into one dispatch
	// before any worker runs (planCoherentJobs). Because this
	// field is worker-supplied, the trust boundary inverts relative to that
	// runtime-written analog: mergeExternalBuildResults validates every
	// entry against the manifest's own dispatches before granting any
	// credit, never trusting the claim blindly (191.1-PATTERNS.md Pattern
	// 6) -- including the CLAIMANT itself (191.1-REVIEW.md CR-01): only a
	// result that is itself a genuine, evidenced success and corresponds to
	// a real dispatch in this manifest may grant credit to another. An entry
	// naming a task ID absent from the manifest, claimed by two different
	// results, or granted by a claimant that is
	// failed/unevidenced/unrecognized, is a distinct, named contract
	// violation, never a silent credit or a silently dropped field.
	CoveredTaskIDs []string `json:"covered_task_ids,omitempty"`
	// TaskReceipts carries optional task-specific completion evidence using
	// the same additive vocabulary as native worker results. CoveredTaskIDs
	// remains assignment scope; later reconciliation decides whether any
	// receipt earns task credit.
	TaskReceipts []codex.TaskReceipt `json:"task_receipts,omitempty"`
	// Disposition qualifies a completed_no_change status (ruling D6): the
	// only recognized value is "verified_existing" — the worker proved the
	// required behavior already exists rather than merely finding nothing to
	// do. One success status with a disposition, never a second terminal
	// state machine. Raw status "verified_existing" is folded into
	// status=completed_no_change + this disposition by the merge path.
	Disposition   string              `json:"disposition,omitempty"`
	DependsOn     []string            `json:"depends_on,omitempty"`
	Outputs       []string            `json:"outputs,omitempty"`
	Blockers      []string            `json:"blockers,omitempty"`
	Duration      float64             `json:"duration,omitempty"`
	ToolCount     int                 `json:"tool_count,omitempty"`
	FilesCreated  []string            `json:"files_created,omitempty"`
	FilesModified []string            `json:"files_modified,omitempty"`
	TestsWritten  []string            `json:"tests_written,omitempty"`
	Handoff       codex.WorkerHandoff `json:"handoff,omitempty"`
}

// effectiveName returns the worker name, falling back to AntName when Name is empty.
func (r codexExternalBuildWorkerResult) effectiveName() string {
	if n := strings.TrimSpace(r.Name); n != "" {
		return n
	}
	return strings.TrimSpace(r.AntName)
}

// completionContractError carries every violation found while validating a
// submitted completion packet (D-05). Returning any violation is an
// unconditional, whole-packet rejection (D-06) -- callers must not touch
// colony state, the attempt journal, or the completion digest binding
// before this error (or a nil/empty violations slice) has been decided.
type completionContractError struct {
	Violations []contractViolation
}

// Error renders a multi-line human summary: a first line naming the
// violation count, followed by one indented line per violation in the form
// "  - <worker>: <field> (<rule>): <message>" (the "<worker>: " prefix is
// omitted when Worker is empty). The machine-readable Violations slice rides
// the error envelope's `details` field separately -- see the build-finalize
// cobra handler below.
func (e *completionContractError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d completion packet violation(s)", len(e.Violations))
	for _, v := range e.Violations {
		b.WriteString("\n  - ")
		if worker := strings.TrimSpace(v.Worker); worker != "" {
			b.WriteString(worker)
			b.WriteString(": ")
		}
		fmt.Fprintf(&b, "%s (%s): %s", v.Field, v.Rule, v.Message)
	}
	return b.String()
}

var buildFinalizeCmd = &cobra.Command{
	Use:   "build-finalize <phase>",
	Short: "Record externally spawned wrapper build workers as the phase build packet",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := parsePositivePhaseArg(args[0])
		if err != nil {
			outputError(1, err.Error(), nil)
			return err
		}
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalBuildCompletion(completionPath)
		if err != nil {
			var contractErr *completionContractError
			if errors.As(err, &contractErr) {
				outputError(1, err.Error(), contractErr.Violations)
			} else {
				outputError(1, err.Error(), nil)
			}
			return err
		}
		result, state, phase, dispatches, err := runCodexBuildFinalize(skillWorkspaceRoot(), phaseNum, completion, false)
		if err != nil {
			var contractErr *completionContractError
			if errors.As(err, &contractErr) {
				outputError(1, err.Error(), contractErr.Violations)
			} else {
				outputError(1, err.Error(), nil)
			}
			return err
		}
		// NEW-03 (195-REVIEW.iter2.md): the same screen the direct lane
		// already shows. WR-05's fix landed only on `aether build`, so the
		// wrapper's own lane -- plan, spawn, then build-finalize, the one the
		// project's guide documents as primary -- still told the owner a
		// half-built phase was ready to be checked and never showed the
		// command that finishes the rest.
		if partial, _ := result["recovery_job"].(bool); partial {
			unfinished, _ := result["unfinished_task_ids"].([]string)
			recoveryCommand, _ := result["recovery_command"].(string)
			outputWorkflow(result, renderBuildPartialCreditVisual(state, phase, unfinished, recoveryCommand))
			return nil
		}
		outputWorkflow(result, renderBuildFinalizeVisual(state, phase, dispatches))
		return nil
	},
}

var buildCompletionStageCmd = &cobra.Command{
	Use:    "build-completion-stage <phase>",
	Short:  "Persist an accepted wrapper completion packet for crash-safe finalization",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		phaseNum, err := parsePositivePhaseArg(args[0])
		if err != nil {
			outputError(1, err.Error(), nil)
			return err
		}
		completionPath, _ := cmd.Flags().GetString("completion-file")
		completion, err := loadExternalBuildCompletion(completionPath)
		if err != nil {
			var contractErr *completionContractError
			if errors.As(err, &contractErr) {
				outputError(1, err.Error(), contractErr.Violations)
			} else {
				outputError(1, err.Error(), nil)
			}
			return err
		}
		manifest := completion.activeManifest()
		if manifest == nil || manifest.Phase != phaseNum {
			err := fmt.Errorf("completion manifest phase does not match requested phase %d", phaseNum)
			outputError(1, err.Error(), nil)
			return err
		}
		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return err
		}
		stageDigest, digestErr := jsonSHA256(completion)
		if digestErr != nil {
			outputError(1, fmt.Sprintf("hash completion packet: %v", digestErr), nil)
			return digestErr
		}
		binding, err := validateBuildAttemptManifestBinding(*manifest, state, isCommittedPartialAttemptReplay(*manifest, stageDigest))
		if err != nil || !binding.Bound {
			if err == nil {
				err = fmt.Errorf("completion manifest is not bound to a durable build attempt")
			}
			outputError(1, err.Error(), nil)
			return err
		}
		durablePath, digest, err := stageBuildAttemptCompletion(binding.Path, completion)
		if err != nil {
			var contractErr *completionContractError
			if errors.As(err, &contractErr) {
				outputError(1, err.Error(), contractErr.Violations)
			} else {
				outputError(1, err.Error(), nil)
			}
			return err
		}
		outputWorkflow(map[string]interface{}{
			"phase":             phaseNum,
			"attempt_id":        manifest.AttemptID,
			"completion_path":   durablePath,
			"completion_sha256": digest,
			"next":              buildFinalizeRecoveryCommand(phaseNum, durablePath),
		}, "")
		return nil
	},
}

func parsePositivePhaseArg(value string) (int, error) {
	phaseNum, err := strconv.Atoi(value)
	if err != nil || phaseNum < 1 {
		return 0, fmt.Errorf("invalid phase %q", value)
	}
	return phaseNum, nil
}

func loadExternalBuildCompletion(path string) (codexExternalBuildCompletion, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return codexExternalBuildCompletion{}, fmt.Errorf("flag --completion-file is required")
	}
	if err := validateFinalizerCompletionFilePath(path); err != nil {
		return codexExternalBuildCompletion{}, err
	}
	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("read completion file: %w", err)
	}

	// Decode the submitted bytes into a generic value FIRST -- this is what
	// structural validation will ultimately see, unmodified by whatever the
	// Go struct below does or does not know about (T-163.1-40).
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return codexExternalBuildCompletion{}, fmt.Errorf("parse completion file: %w", err)
	}

	var completion codexExternalBuildCompletion
	typeErrorTolerated := false
	if err := json.Unmarshal(data, &completion); err != nil {
		var typeErr *json.UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			return codexExternalBuildCompletion{}, fmt.Errorf("parse completion file: %w", err)
		}
		// encoding/json saves only the first type mismatch and keeps
		// decoding the rest of the packet -- tolerate it here so a single
		// wrong-typed field no longer aborts validation before it starts
		// (T-163.1-43); remember it so a manifest-absent packet below gets a
		// real structural violation instead of the bare
		// "must include dispatch_manifest" message.
		typeErrorTolerated = true
	}
	// A tolerated type error on the dispatch_manifest/manifest field itself
	// still leaves activeManifest() non-nil: encoding/json allocates a
	// zero-valued struct behind the pointer before it discovers the value
	// it was given (e.g. a bare string) is not an object, and never rolls
	// that allocation back. So activeManifest() alone cannot tell "no
	// manifest" apart from "manifest field allocated but never actually
	// populated" -- cross-check against raw, which reflects the submitted
	// shape with no such allocation quirk. The cross-check must be
	// key-specific (the key activeManifest() actually selected), not
	// either-key: see manifestSelectionMatchesRaw (WR-163.1-02).
	if completion.activeManifest() != nil && manifestSelectionMatchesRaw(completion, raw) {
		completion.submittedRaw = raw
		return completion, nil
	}

	// Envelope handling: retry the struct decode against `{"result": ...}`
	// exactly as before, tolerating the same class of type error.
	var envelope struct {
		Result codexExternalBuildCompletion `json:"result"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		var typeErr *json.UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			return codexExternalBuildCompletion{}, fmt.Errorf("parse completion envelope: %w", err)
		}
		typeErrorTolerated = true
	}

	// Unwrap the raw value the same way the struct decode unwraps: only
	// when the top-level object has neither a dispatch_manifest key nor a
	// manifest key of its own, but does have a result key. An explicit JSON
	// null under either manifest key counts as ABSENT here, mirroring the
	// struct decode (which leaves a pointer field nil for null): wrappers
	// that serialize absent fields as null -- typical JS/TS hosts, and the
	// ts-host path is a live producer -- were accepted via the envelope
	// path before the submitted-bytes rework and must stay accepted
	// (WR-163.1-01). Any other non-null value under a manifest key blocks
	// the unwrap so the top-level shape is what structural validation
	// reports on, instead of being silently discarded.
	unwrappedRaw := raw
	if rawMap, ok := raw.(map[string]any); ok {
		dispatchManifestValue, hasDispatchManifest := rawMap["dispatch_manifest"]
		manifestValue, hasManifest := rawMap["manifest"]
		if (!hasDispatchManifest || dispatchManifestValue == nil) && (!hasManifest || manifestValue == nil) {
			if result, hasResult := rawMap["result"]; hasResult {
				unwrappedRaw = result
			}
		}
	}

	if envelope.Result.activeManifest() != nil && manifestSelectionMatchesRaw(envelope.Result, unwrappedRaw) {
		envelope.Result.submittedRaw = unwrappedRaw
		return envelope.Result, nil
	}

	if typeErrorTolerated {
		if violations := validateCompletionPacketStructure(unwrappedRaw); len(violations) > 0 {
			return codexExternalBuildCompletion{}, &completionContractError{Violations: violations}
		}
	}
	return codexExternalBuildCompletion{}, fmt.Errorf("completion file must include dispatch_manifest")
}

// manifestSelectionMatchesRaw reports whether the manifest key
// activeManifest() actually selected ("dispatch_manifest" when
// DispatchManifest is non-nil, else "manifest") is present in raw as a
// genuine JSON object. This is the ground truth loadExternalBuildCompletion
// cross-checks the decoded struct's activeManifest() pointer against, since
// a tolerated type error on that exact field leaves an
// allocated-but-never-populated zero-value struct behind the pointer rather
// than nil. Checking the SELECTED key -- not either key -- matters
// (WR-163.1-02): for {"dispatch_manifest": "bad", "manifest": {valid}} the
// tolerated type error allocates a zero DispatchManifest, activeManifest()
// prefers it over the valid Manifest, and an either-key check would accept
// the packet with a corrupt active manifest -- suppressing the structural
// violation for the wrong-typed key and later failing with the misleading
// "must come from `aether build --plan-only`" message instead.
func manifestSelectionMatchesRaw(completion codexExternalBuildCompletion, raw any) bool {
	rawMap, ok := raw.(map[string]any)
	if !ok {
		return false
	}
	var key string
	switch {
	case completion.DispatchManifest != nil:
		key = "dispatch_manifest"
	case completion.Manifest != nil:
		key = "manifest"
	default:
		return false
	}
	_, isObject := rawMap[key].(map[string]any)
	return isObject
}

func (c codexExternalBuildCompletion) activeManifest() *codexBuildManifest {
	if c.DispatchManifest != nil {
		return c.DispatchManifest
	}
	return c.Manifest
}

func (c codexExternalBuildCompletion) workerResults() []codexExternalBuildWorkerResult {
	results := make([]codexExternalBuildWorkerResult, 0, len(c.Dispatches)+len(c.Results)+len(c.Workers))
	results = append(results, c.Dispatches...)
	results = append(results, c.Results...)
	results = append(results, c.Workers...)
	return results
}

func runCodexBuildFinalize(root string, phaseNum int, completion codexExternalBuildCompletion, skipVerify bool) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	if store == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("no store initialized")
	}

	manifest := completion.activeManifest()
	if manifest == nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("completion file must include dispatch_manifest")
	}
	if !manifest.PlanOnly {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("dispatch_manifest must come from `aether build --plan-only`")
	}
	if manifest.Phase != phaseNum {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("completion phase %d does not match requested phase %d", manifest.Phase, phaseNum)
	}
	if len(manifest.Dispatches) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("dispatch_manifest contains no dispatches")
	}
	if err := validateFinalizerManifestRoot("dispatch_manifest", manifest.Root, root); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	now := time.Now().UTC()

	state, err := loadActiveColonyState()
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("%s", colonyStateLoadMessage(err))
	}
	if err := validateFinalizerManifestColonyMode("dispatch_manifest", manifest.ColonyMode, state); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if len(state.Plan.Phases) == 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("No project plan. Run `aether plan` first.")
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("phase %d not found (plan has %d phases)", phaseNum, len(state.Plan.Phases))
	}
	completionDigest, err := jsonSHA256(completion)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("hash completion packet: %w", err)
	}
	// WR-02: is this the identical packet that already produced committed
	// partial credit for this phase? Computed before the staleness guards
	// because a committed partial legitimately changed the very state they
	// compare against.
	partialReplay := isCommittedPartialAttemptReplay(*manifest, completionDigest)
	if err := validateBuildManifestPlanRevision(*manifest, state, partialReplay); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	binding, err := validateBuildAttemptManifestBinding(*manifest, state, partialReplay)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	// T-188-09 (D-10, D-11): a completion packet with no attempt binding at
	// all is still accepted -- this branch is deliberately kept, not
	// hardened into a refusal -- but it must never be silent. Warn on
	// stderr immediately, every time this is detected, regardless of
	// whether anything later in this function fails. The durable Events
	// record is appended near the end of this function (alongside this
	// same request's other Events, once `updatedState` is stable) rather
	// than here: `state` is still reassigned wholesale by
	// reconcilePriorCompletedPhaseTasksFromTrustedManifests below when a
	// prior completed phase needs task-status repair, and an event
	// appended to `state.Events` here would be silently discarded by that
	// reassignment in that case -- appending later is the only placement
	// that is correct on every occurrence, not just the common one.
	if binding.Legacy {
		visualFprintf(stderr, "warning: phase %d's build completion did not include the newer tracking details that link it back to one specific dispatched build, so it is being accepted using the older, less strictly checked method\n", phaseNum)
	}
	if binding.Bound && buildAttemptCompletionSealed(binding.Record) && binding.Record.CompletionSHA256 != "" && binding.Record.CompletionSHA256 != completionDigest {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("completion packet does not match the result already bound to attempt %s", binding.Record.ID)
	}
	// Route to committed-attempt reconciliation ONLY when THIS attempt is the
	// one whose lifecycle was committed -- proved by its own terminal
	// evidence, not by the colony happening to sit at BUILT.
	//
	// Without buildAttemptRecordedTerminalEvidence this condition also caught
	// a brand-new attempt that has never been finalized, whenever the phase
	// had been built before -- exactly what `--force` produces. That created a
	// deadlock with no in-band exit, reported from a live colony on
	// 2026-08-21 and reproduced by TestForcedRedispatchAfterBuiltIsNotADeadlock:
	//
	//   finalize:  "attempt X is already committed, so this different
	//               completion packet cannot replace it ... run `aether continue`"
	//   continue:  "no completed worker dispatches found -- build did not
	//               produce verifiable results"
	//
	// Each pointed at the other. The finalize half is a message I added on
	// 2026-08-19; it is correct for a genuinely committed attempt, and was
	// sending users nowhere for an attempt that had committed nothing.
	//
	// The two situations look alike and are not:
	//
	//   partial commit     colony state committed, journal write lost.
	//                      CompletionSHA256 set, Claims set, dispatches
	//                      completed. Reconciling is right; `aether continue`
	//                      genuinely works, because the dispatches are there.
	//
	//   forced redispatch  a fresh attempt on a phase whose PREVIOUS attempt
	//                      built. CompletionSHA256 empty, Claims nil,
	//                      dispatches still `planned`. Nothing of this attempt
	//                      has been committed, so there is nothing to
	//                      reconcile -- it is an ordinary finalize, and the
	//                      stale BUILT is the state --force exists to replace.
	if binding.Bound && binding.Record.Status != buildAttemptBuilt &&
		buildAttemptRecordedTerminalEvidence(binding.Record) &&
		state.State == colony.StateBUILT && state.CurrentPhase == phaseNum {
		return reconcileCommittedExternalBuildAttempt(state, phaseNum, binding, completionDigest)
	}
	if binding.Bound && binding.Record.Status == buildAttemptBuilt {
		return idempotentExternalBuildFinalizeResult(state, phaseNum, binding, completionDigest)
	}
	// WR-02: replaying the identical packet for a committed partial returns the
	// same answer -- including the same recovery command -- and commits nothing
	// new. Without this the runtime's own "rerun build-finalize with the same
	// completion packet" instruction had no reachable outcome for a partial.
	if binding.Bound && binding.Record.Status == buildAttemptPartial {
		return idempotentExternalPartialFinalizeResult(state, phaseNum, updatedPhaseForPartialReplay(state, phaseNum), binding, completionDigest)
	}
	if err := validateFinalizerManifestFreshness("dispatch_manifest", manifest.GeneratedAt, now); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	state, _, err = reconcilePriorCompletedPhaseTasksFromTrustedManifests(root, state, phaseNum)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	selectedTaskIDs := uniqueSortedStrings(manifest.SelectedTasks)
	phase := state.Plan.Phases[phaseNum-1]
	if err := validatePhaseCriterionEvidence(phase); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateBuildManifestTaskSetForPhase(codexContinueManifest{Present: true, Data: *manifest}, phase, true); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateSelectedBuildTasks(phase, selectedTaskIDs); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := runPreBuildGates(store.BasePath(), phaseNum); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := validateCodexBuildState(state, phaseNum, selectedTaskIDs, false); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	// D-06: the packet is atomic. Structural, claim-path, and dispatch-level
	// checks all accumulate into one violation list before any state is
	// touched -- no checkpoint save, attempt begin, digest bind, or
	// transition happens until this returns clean.
	if violations := validateCompletionPacketSemantics(root, completion); len(violations) > 0 {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, &completionContractError{Violations: violations}
	}
	dispatches, _, err := mergeExternalBuildResults(*manifest, completion.workerResults())
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	// Per SAFE-01, SAFE-02: validate build provenance before proceeding.
	// Rejects phantom builds where no worker produced successful results with file modifications.
	if err := validateBuildProvenanceForManifest(manifest, completion.workerResults()); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	// D-08/D-09 (195-06): the external/wrapper lane reuses the EXACT same
	// two-stage receipt trust boundary the native lane wired in 195-04
	// (admitCoherentJobTaskReceipts / finalizeCoherentJobTaskReceiptEvidence,
	// via resolveCoherentJobDispatchReceipts) -- never a second, external-only
	// validator. In-repo files already live in root at this point, so
	// admission and finalization run consecutively, exactly like the native
	// lane's executeCodexBuildDispatches. A worktree-backed completion's
	// files are NOT yet synced into root (mergePhaseWorktrees runs later,
	// below), so that lane instead routes through the SAME two stages with a
	// receipt-scoped sync step inserted between them
	// (resolveWorktreeExternalDispatchReceipts, plan 195-08) -- the identical
	// sequence the native worktree lane uses, never an external-only
	// validator, so both lanes credit the same tasks from the same evidence.
	if effectiveParallelMode(state) == colony.ModeWorktree {
		dispatches = resolveWorktreeExternalDispatchReceipts(root, phase, state, phaseNum, dispatches)
	} else {
		dispatches = resolveCoherentJobDispatchReceipts(root, phase, dispatches)
	}
	startedAt := parseManifestGeneratedAt(*manifest)
	completedAt := now
	checkpointRel := filepath.ToSlash(filepath.Join("checkpoints", fmt.Sprintf("pre-build-phase-%d.json", phaseNum)))
	buildDirRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum)))
	manifestRel := filepath.ToSlash(filepath.Join(buildDirRel, "manifest.json"))
	claimsRel := "last-build-claims.json"
	resultCollectionRel := filepath.ToSlash(filepath.Join(buildDirRel, "result-collection.json"))
	claims, err := completion.claimsOrAggregate(root, phaseNum, startedAt, dispatches)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	attachBuildArtifactEvidence(root, &claims)

	if err := store.SaveJSON(checkpointRel, state); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to checkpoint colony state: %w", err)
	}
	attemptRel := binding.Path
	if !binding.Bound {
		attemptRel, err = beginBuildAttempt(state, phaseNum, phase, startedAt, selectedTaskIDs, checkpointRel, manifestRel, claimsRel, "external-task", dispatches)
		if err != nil {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
	}
	if _, err := bindBuildAttemptCompletion(attemptRel, completion); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	attemptFinished := false
	finishAttempt := func(status, summary string, transitionErr error) {
		if attemptFinished {
			return
		}
		if err := transitionBuildAttempt(attemptRel, status, summary, nil, nil, "external-task", transitionErr); err == nil && buildAttemptStatusTerminal(status) {
			attemptFinished = true
		}
	}
	defer func() {
		if !attemptFinished {
			_ = transitionBuildAttempt(attemptRel, buildAttemptInterrupted, "build-finalize ended before durable finalization", nil, nil, "external-task", fmt.Errorf("build-finalize ended before durable finalization"))
		}
	}()
	if err := transitionBuildAttempt(attemptRel, buildAttemptDispatching, "external worker results received for validation", dispatches, nil, "external-task", nil); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist external result collection start", err)
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	// Prepare the updated state in memory first (needed for downstream writes).
	// D-08: BUILT means every selected task is actually proven done. A
	// grouped job that only credited part of its covered tasks (via receipt
	// finalization above) must NOT falsely report BUILT -- applyCodexBuildState
	// already leaves state.State at EXECUTING and the phase at
	// PhaseInProgress, which is the honest "still mid-build" signal; only a
	// fully-credited build advances it the rest of the way.
	buildFullyCredited := allSelectedBuildTasksCredited(selectedTaskIDs, dispatches)
	updatedState := state
	applyCodexBuildState(&updatedState, phaseNum, startedAt, selectedTaskIDs, colony.NormalizeVerificationDepth(manifest.ReviewDepth))
	if buildFullyCredited {
		updatedState.State = colony.StateBUILT
	}
	reconcileCompletedBuildTasks(&updatedState, phaseNum, dispatches)
	updatedPhase := updatedState.Plan.Phases[phaseNum-1]
	updatedState.Events = append(trimmedEvents(updatedState.Events),
		fmt.Sprintf("%s|build_completed|build-finalize|Phase %d external Task workers recorded", completedAt.Format(time.RFC3339), phaseNum),
	)
	if binding.Legacy {
		// Same fact as the stderr warning above, repeated here so it survives
		// past the terminal -- queryable later via `aether history`/`aether
		// status` -- rather than only a fleeting print (D-11). Appended here,
		// not immediately after validateBuildAttemptManifestBinding, because
		// this is the first point after `updatedState` is fully settled (see
		// the comment at the binding check above).
		updatedState.Events = append(updatedState.Events,
			fmt.Sprintf("%s|manifest_legacy_accepted|build-finalize|Phase %d build completion did not include the newer tracking details that link it back to one specific dispatched build, and was accepted using the older, less strictly checked method", completedAt.Format(time.RFC3339), phaseNum),
		)
	}

	if err := transitionBuildAttempt(attemptRel, buildAttemptTerminal, "external terminal worker results recorded before lifecycle projection", dispatches, &claims, "external-task", nil); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist external terminal worker results", err)
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := store.SaveJSON(claimsRel, claims); err != nil {
		finishAttempt(buildAttemptFailed, "failed to persist external current-build claims", err)
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write build claims: %w", err)
	}

	_, dispatches, err = writeCodexBuildOutcomeReports(root, updatedPhase, buildDirRel, dispatches, completedAt, "external-task")
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	finalManifest := buildCodexBuildManifest(root, updatedState, updatedPhase, checkpointRel, claimsRel, dispatches, startedAt, "external-task", selectedTaskIDs, manifest.WorkerBriefs, false, colony.NormalizeVerificationDepth(manifest.ReviewDepth))
	finalManifest.GeneratedAt = completedAt.Format(time.RFC3339)
	finalManifest.AttemptID = strings.TrimSpace(manifest.AttemptID)
	finalManifest.AttemptPath = filepath.ToSlash(strings.TrimSpace(manifest.AttemptPath))
	if err := store.SaveJSON(manifestRel, finalManifest); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write build manifest: %w", err)
	}

	// D-08 / ruling D11 rule 2: the program's own free checks (build, types,
	// lint, tests, claimed-files-exist, criterion evidence) run at build time
	// and are recorded as a report on the attempt journal -- never as an
	// advancement gate here. Advancing the phase is `continue`'s decision
	// alone (TestBuildFinalizeFreeChecksDoNotAdvanceThePhase). skipVerify
	// keeps its existing meaning: it is the switch for whether the free
	// checks run at all at build time; no new flag is added.
	if !skipVerify {
		contManifest := codexContinueManifest{Present: true, Path: manifestRel, Data: finalManifest}
		watcher := evaluateContinueWatcherVerification(contManifest)
		floor := runDeterministicFloor(context.Background(), root, updatedPhase, contManifest, watcher, 0)
		report := buildFreeCheckReportFromFloor(phaseNum, completedAt, floor)
		if err := attachBuildFreeCheckReport(attemptRel, report); err != nil {
			// A report the runtime failed to write is not a reason to fail a
			// build that otherwise completed -- warn and move on. It never
			// gates advancement either way.
			visualFprintf(stderr, "warning: could not record the build-time check report for phase %d: %v\n", phaseNum, err)
		} else if !report.Passed {
			visualFprintf(stderr, "warning: the program's own checks did not all pass for phase %d (%s) -- this does not block the build; the next step (`aether continue`) decides whether the phase can advance\n", phaseNum, report.Summary)
		}
	}

	if err := recordExternalBuildSpawnTree(dispatches); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	if err := persistExternalBuildHandoffs(root, phaseNum, dispatches, completion.workerResults()); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	resultCollection := buildExternalBuildResultCollectionReport(phaseNum, updatedPhase.Name, manifest.Dispatches, completion.workerResults(), dispatches, completedAt, selectedTaskIDs)
	if err := store.SaveJSON(resultCollectionRel, resultCollection); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to write result collection diagnostics: %w", err)
	}
	recoveryInstructions, err := buildExternalBuildRecoveryInstructions(phaseNum, dispatches)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}

	// Worktree mode: merge back completed branches before marking phase done.
	var worktreeMergeEvent string
	if effectiveParallelMode(updatedState) == colony.ModeWorktree {
		merged, failed, mergeErr := mergePhaseWorktrees(phaseNum)
		if mergeErr != nil {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("worktree merge-back failed: %w", mergeErr)
		}
		if len(failed) > 0 {
			return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("worktree merge-back blocked: %s", strings.Join(failed, "; "))
		}
		if len(merged) > 0 {
			worktreeMergeEvent = fmt.Sprintf("%s|worktree_merge|build-finalize|Merged %d worktree branch(es): %s",
				completedAt.Format(time.RFC3339), len(merged), strings.Join(merged, ", "))
			updatedState.Events = append(updatedState.Events, worktreeMergeEvent)
		}
	}

	// Atomically commit the colony state mutation (CR-02, 188-REVIEW.md).
	committedState, err := commitBuildFinalizeState(buildFinalizeCommitParams{
		PhaseNum:           phaseNum,
		StartedAt:          startedAt,
		SelectedTaskIDs:    selectedTaskIDs,
		ReviewDepth:        colony.NormalizeVerificationDepth(manifest.ReviewDepth),
		Dispatches:         dispatches,
		CompletedAt:        completedAt,
		LegacyManifest:     binding.Legacy,
		WorktreeMergeEvent: worktreeMergeEvent,
		UpdatedState:       updatedState,
		BuildFullyCredited: buildFullyCredited,
	})
	if err != nil {
		finishAttempt(buildAttemptFailed, "failed to commit external built lifecycle state", err)
		if errors.Is(err, errRuntimeStateSuperseded) {
			// Propagate the supersession error directly rather than wrapping
			// it in the generic "failed to save" message, which would read
			// as a storage/IO failure instead of the concurrency refusal it
			// actually is (CR-02).
			return nil, colony.ColonyState{}, colony.Phase{}, nil, err
		}
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("failed to save built colony state: %w", err)
	}
	// D-10: a fully-credited grouped job seals this attempt as built, exactly
	// as before. A partially-credited one (buildFullyCredited false) is
	// neither built nor a plain failure -- it is recorded as `partial`, and a
	// D-10 append-only recovery job is created below for the unfinished
	// tasks. The parent attempt's own dispatches/receipts/claims/completion
	// digest, already durably written above, are never rewritten by this.
	finalAttemptStatus := buildAttemptBuilt
	finalAttemptSummary := "external built lifecycle state committed"
	if !buildFullyCredited {
		finalAttemptStatus = buildAttemptPartial
		finalAttemptSummary = "external partial-terminal state committed; unfinished tasks recorded for a D-10 recovery job"
	}
	if err := transitionBuildAttempt(attemptRel, finalAttemptStatus, finalAttemptSummary, dispatches, &claims, "external-task", nil); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("built state committed but external build attempt journal is incomplete; rerun build-finalize with the same completion packet: %w", err)
	}
	attemptFinished = true
	updatedState = committedState
	updateSessionSummary("build-finalize", "aether continue", fmt.Sprintf("Phase %d external Task workers recorded (%d dispatches)", phaseNum, len(dispatches)))

	var partialRetryOutcome *partialBuildRetryOutcome
	if !buildFullyCredited {
		parentAttemptID := strings.TrimSuffix(filepath.Base(attemptRel), filepath.Ext(attemptRel))
		outcome, retryErr := reconcilePartialBuildRetry(updatedState, phaseNum, updatedPhase, parentAttemptID, time.Now().UTC(), dispatches)
		if retryErr != nil {
			visualFprintf(stderr, "warning: could not create a D-10 recovery job for phase %d's partial credit: %v\n", phaseNum, retryErr)
		} else {
			partialRetryOutcome = outcome
		}
	}

	// Collect pheromone suggestions once the build is durably committed.
	// Called exactly once per finalize (never inside the dispatch loop
	// above) so re-analysis cost scales with builds, not worker count.
	suggestAnalyzeRan, pendingSuggestionCount := collectPendingSuggestions(root)

	// File this run's per-worker token record. This call is what makes both
	// platform usage readers reachable from a real dispatch path; without it
	// they would be code nothing invokes, which is this repository's named
	// signature failure.
	//
	// Accounting is a record OF the build, never a gate ON it: a write that
	// fails is reported and the build still completes. A build must not be
	// lost because its bookkeeping could not be filed.
	spendNote := ""
	spendOutcome, spendErr := writeSpendRowsForRun(spendWriteRequest{
		Phase:      phaseNum,
		PhaseName:  updatedPhase.Name,
		Workflow:   spendWorkflowBuild,
		RepoRoot:   root,
		Platform:   manifest.HostPlatform,
		RunID:      spendRunIDFromAttempt(attemptRel),
		StartedAt:  spendRunStartFromManifest(manifest.GeneratedAt),
		EndedAt:    time.Now().UTC(),
		Dispatches: dispatches,
	})
	if spendErr != nil {
		spendNote = fmt.Sprintf("this build's per-worker token record could not be filed: %v", spendErr)
		visualFprintf(stderr, "warning: %s\n", spendNote)
	} else if len(spendOutcome.Notes) > 0 {
		spendNote = strings.Join(spendOutcome.Notes, "; ")
	}

	result := map[string]interface{}{
		"phase":                    phaseNum,
		"phase_name":               updatedPhase.Name,
		"state":                    updatedState.State,
		"plan_only":                false,
		"dispatch_mode":            "external-task",
		"spend_rows_written":       spendOutcome.RowsWritten,
		"spend_rows_measured":      spendOutcome.Reported,
		"dispatches":               codexBuildDispatchMaps(dispatches),
		"dispatch_count":           len(dispatches),
		"wave_count":               len(buildWaveExecutionPlans(dispatches, effectiveParallelMode(updatedState))),
		"parallel_mode":            string(effectiveParallelMode(updatedState)),
		"selected_tasks":           selectedTaskIDs,
		"checkpoint":               displayDataPath(checkpointRel),
		"manifest":                 displayDataPath(manifestRel),
		"claims_path":              displayDataPath(claimsRel),
		"attempt":                  displayDataPath(attemptRel),
		"result_collection":        displayDataPath(resultCollectionRel),
		"idempotent":               false,
		"next":                     "aether continue",
		"suggest_analyze_ran":      suggestAnalyzeRan,
		"pending_suggestion_count": pendingSuggestionCount,
	}
	if spendNote != "" {
		result["spend_ledger_note"] = spendNote
	}
	if pendingSuggestionCount > 0 {
		result["pending_suggestions_next"] = "aether suggest-approve"
	}
	if len(recoveryInstructions) > 0 {
		result["recovery_instructions"] = recoveryInstructions
	}
	if partialRetryOutcome != nil {
		result["recovery_job"] = true
		result["parent_attempt_id"] = partialRetryOutcome.ParentAttemptID
		result["retry_attempt_id"] = partialRetryOutcome.RetryAttemptID
		result["retry_attempt_path"] = partialRetryOutcome.RetryAttemptPath
		result["unfinished_task_ids"] = partialRetryOutcome.UnfinishedTaskIDs
		result["recovery_command"] = partialRetryOutcome.RedispatchCommand
		result["next"] = partialRetryOutcome.RedispatchCommand
	}
	addOrchestratorBoundaryGuidance(result, "build", updatedState, "aether continue", manifest.BoundaryQuestions)
	// One closing answer for the screen and the wrapper (Phase 197 plan 04).
	// This runs LAST because everything above it can still add the more
	// specific command this run knows about.
	closeLifecycleRun(result, updatedState, "build")
	return result, updatedState, updatedPhase, dispatches, nil
}

// buildFinalizeCommitParams carries what commitBuildFinalizeState needs to
// transition COLONY_STATE.json to StateBUILT for one phase.
type buildFinalizeCommitParams struct {
	PhaseNum           int
	StartedAt          time.Time
	SelectedTaskIDs    []string
	ReviewDepth        colony.VerificationDepth
	Dispatches         []codexBuildDispatch
	CompletedAt        time.Time
	LegacyManifest     bool
	WorktreeMergeEvent string
	UpdatedState       colony.ColonyState
	// BuildFullyCredited is precomputed once in runCodexBuildFinalize from
	// Dispatches/SelectedTaskIDs before this atomic commit re-reads a fresh
	// on-disk state -- it does not depend on which state happened to be on
	// disk, only on the terminal proof this build session actually produced
	// (D-08). false means a grouped job only proved part of its covered
	// tasks; the colony stays at EXECUTING (never a false BUILT).
	BuildFullyCredited bool
}

// commitBuildFinalizeState is runCodexBuildFinalize's one write against
// COLONY_STATE.json (CR-02, 188-REVIEW.md).
//
// This used to be `committedState = params.UpdatedState` inside the
// UpdateJSONAtomically closure -- discarding the primitive's own fresh
// on-disk read and replacing it wholesale with a value built earlier in
// runCodexBuildFinalize (from a `state` loaded once, long before the
// checkpoint save, build-attempt transitions, claims write, and, in
// worktree mode, a full worktree merge). Any concurrent write to
// COLONY_STATE.json during that window -- an operator pausing the colony --
// was silently discarded and replaced. It now mutates only the freshly-read
// value's own fields, re-validated by validateBuildFinalizeStateStillCurrent
// first.
//
// Unlike the direct-build path's second commit (runCodexBuildWithOptions,
// cmd/codex_build.go:692-707), which guards with
// validateRuntimeStateStillCurrent(..., colony.StateEXECUTING) because it
// commits a SEPARATE, EARLIER READY->EXECUTING checkpoint (SaveJSON, before
// worker dispatch) moments before this second call, the external/wrapper
// build-finalize flow never separately persists an EXECUTING checkpoint at
// all -- aether build --plan-only (runCodexBuildPlanOnlyWithOptions) returns
// a manifest without writing COLONY_STATE.json. This single write is the
// entire READY->BUILT transition for that flow, so its currency guard
// cannot require state.State == EXECUTING the way the direct path's second
// commit does; see validateBuildFinalizeStateStillCurrent's own doc comment.
func commitBuildFinalizeState(params buildFinalizeCommitParams) (colony.ColonyState, error) {
	var committedState colony.ColonyState
	err := store.UpdateJSONAtomically("COLONY_STATE.json", &committedState, func() error {
		if err := validateBuildFinalizeStateStillCurrent(committedState, params.PhaseNum); err != nil {
			return err
		}
		applyCodexBuildState(&committedState, params.PhaseNum, params.StartedAt, params.SelectedTaskIDs, params.ReviewDepth)
		if params.BuildFullyCredited {
			committedState.State = colony.StateBUILT
		}
		reconcileCompletedBuildTasks(&committedState, params.PhaseNum, params.Dispatches)
		committedState.Events = append(trimmedEvents(committedState.Events),
			fmt.Sprintf("%s|build_completed|build-finalize|Phase %d external Task workers recorded", params.CompletedAt.Format(time.RFC3339), params.PhaseNum),
		)
		if params.LegacyManifest {
			committedState.Events = append(committedState.Events,
				fmt.Sprintf("%s|manifest_legacy_accepted|build-finalize|Phase %d build completion did not include the newer tracking details that link it back to one specific dispatched build, and was accepted using the older, less strictly checked method", params.CompletedAt.Format(time.RFC3339), params.PhaseNum),
			)
		}
		if params.WorktreeMergeEvent != "" {
			committedState.Events = append(committedState.Events, params.WorktreeMergeEvent)
		}
		return nil
	})
	return committedState, err
}

// buildFullBuildTaskIDSet resolves the full set of task IDs this build
// session is actually responsible for: the explicit selection when the
// caller passed one, else every task ID any dispatch in this build covers
// (dispatchCoveredTaskIDs). An explicit selection is empty for the common
// "no --tasks filter" build -- validateSelectedBuildTasks already treats an
// empty selectedTaskIDs as "no filter, not zero tasks" -- so falling back to
// the dispatches' own covered-task union is what keeps that ordinary case
// from being misread as "nothing was selected, so everything trivially
// counts as credited."
func buildFullBuildTaskIDSet(selectedTaskIDs []string, dispatches []codexBuildDispatch) map[string]struct{} {
	set := make(map[string]struct{})
	for _, id := range selectedTaskIDs {
		if id = strings.TrimSpace(id); id != "" {
			set[id] = struct{}{}
		}
	}
	if len(set) > 0 {
		return set
	}
	for _, d := range dispatches {
		for _, id := range dispatchCoveredTaskIDs(d) {
			if id = strings.TrimSpace(id); id != "" {
				set[id] = struct{}{}
			}
		}
	}
	return set
}

// allSelectedBuildTasksCredited reports whether every task this build
// session is responsible for (buildFullBuildTaskIDSet) has actually been
// credited -- either by an ordinary whole-success dispatch
// (completed/completed_no_change, crediting every task it covers) or by
// root-evidenced per-task receipt credit
// (finalizeCoherentJobTaskReceiptEvidence, via dispatch.CompletedTaskIDs).
// It is what decides whether an external/wrapper build-finalize may
// transition the colony to BUILT (D-08): a grouped job that only proved
// part of its covered tasks must stay honestly mid-build rather than
// reporting a false BUILT. Uses completedBuildTaskIDs -- the SAME reader
// reconcileCompletedBuildTasks uses to mark individual tasks complete --
// so this check and the task statuses it gates can never disagree.
func allSelectedBuildTasksCredited(selectedTaskIDs []string, dispatches []codexBuildDispatch) bool {
	full := buildFullBuildTaskIDSet(selectedTaskIDs, dispatches)
	if len(full) == 0 {
		return true
	}
	credited := completedBuildTaskIDs(dispatches)
	for id := range full {
		if _, ok := credited[id]; !ok {
			return false
		}
	}
	return true
}

// unfinishedBuildTaskIDs returns the full-set task IDs allSelectedBuildTasksCredited
// found NOT credited, sorted, for diagnostics (result-collection.json) that
// must name both what finished and what did not (D-08/D-09).
func unfinishedBuildTaskIDs(selectedTaskIDs []string, dispatches []codexBuildDispatch) []string {
	full := buildFullBuildTaskIDSet(selectedTaskIDs, dispatches)
	credited := completedBuildTaskIDs(dispatches)
	var unfinished []string
	for id := range full {
		if _, ok := credited[id]; !ok {
			unfinished = append(unfinished, id)
		}
	}
	return uniqueSortedStrings(unfinished)
}

// creditedBuildTaskIDs returns the full-set task IDs allSelectedBuildTasksCredited
// found credited, sorted, for the same diagnostics.
func creditedBuildTaskIDs(selectedTaskIDs []string, dispatches []codexBuildDispatch) []string {
	full := buildFullBuildTaskIDSet(selectedTaskIDs, dispatches)
	credited := completedBuildTaskIDs(dispatches)
	var done []string
	for id := range full {
		if _, ok := credited[id]; ok {
			done = append(done, id)
		}
	}
	return uniqueSortedStrings(done)
}

// validateBuildFinalizeStateStillCurrent re-checks, against a freshly-read
// on-disk COLONY_STATE.json, that commitBuildFinalizeState's target phase is
// still the one the colony expects to commit -- the same fail-closed
// philosophy as validateRuntimeStateStillCurrent (cmd/codex_build.go), but
// shaped for THIS transition. The external/wrapper build-finalize flow never
// separately commits a READY->EXECUTING checkpoint before dispatch (see
// commitBuildFinalizeState's own doc comment); confirmed by this codebase's
// own fixtures (e.g. TestBuildFinalizeReconcilesJournalAfterBuiltStateCommit
// seeds State: colony.StateREADY, Status: colony.PhaseReady before calling
// runCodexBuildFinalize for the first time) -- so "still current" cannot
// mean "state is still EXECUTING." CurrentPhase == 0 is also accepted: a
// colony's very first phase may never have had CurrentPhase set by a prior
// advancePhase call (setupExternalBuildAttemptTest's own fixture uses
// CurrentPhase: 0 for phase 1). A concurrent pause, or a race that already
// advanced this phase past build, must still be caught -- that is exactly
// the clobber class CR-02 closes.
func validateBuildFinalizeStateStillCurrent(state colony.ColonyState, phaseNum int) error {
	if state.Paused {
		return runtimeStateSupersededError(phaseNum, "colony is paused")
	}
	if state.CurrentPhase != phaseNum && state.CurrentPhase != 0 {
		return runtimeStateSupersededError(phaseNum, fmt.Sprintf("current phase is %d", state.CurrentPhase))
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		return runtimeStateSupersededError(phaseNum, "phase is no longer present")
	}
	if state.Plan.Phases[phaseNum-1].Status == colony.PhaseCompleted {
		return runtimeStateSupersededError(phaseNum, fmt.Sprintf("phase status is %s", state.Plan.Phases[phaseNum-1].Status))
	}
	return nil
}

// collectPendingSuggestions runs suggest-analyze exactly once after a build
// has been durably committed, and reports its outcome without ever failing
// the caller. A suggestion-engine failure must never fail a build that
// otherwise succeeded (T-163-15) -- the error is logged to stderr, not
// propagated, and the distinction between "ran and found nothing" (ran=true,
// count=0) and "never ran" (ran=false) is kept visible on the return value
// rather than silently collapsed to the same zero.
func collectPendingSuggestions(root string) (ran bool, count int) {
	suggestResult, err := runSuggestAnalyze(root, false)
	if err != nil {
		visualFprintf(stderr, "warning: suggest-analyze did not run at build finalize: %v\n", err)
		return false, 0
	}
	if total, ok := suggestResult["total"].(int); ok {
		count = total
	}
	return true, count
}

func validateBuildManifestPlanRevision(manifest codexBuildManifest, state colony.ColonyState, partialReplay bool) error {
	if revisionID := strings.TrimSpace(manifest.PlanRevisionID); revisionID != "" && revisionID != activePlanRevisionID(state.Plan) {
		return fmt.Errorf("dispatch_manifest belongs to superseded plan revision %s; active revision is %s", revisionID, activePlanRevisionID(state.Plan))
	}
	if state.State == colony.StateBUILT && state.CurrentPhase == manifest.Phase {
		// Exact retry safety is proved by the durable attempt and completion
		// hashes. The lifecycle projection legitimately changed task statuses.
		return nil
	}
	if partialReplay {
		// WR-02: identical reasoning for a committed PARTIAL. Crediting some
		// of the phase's tasks legitimately changed the plan hash, so the
		// staleness check would refuse the very packet that produced it. The
		// packet's own digest already matched the durable attempt
		// (isCommittedPartialAttemptReplay), which is the same proof the
		// BUILT branch above relies on.
		return nil
	}
	if expectedHash := strings.TrimSpace(manifest.PlanStateHash); expectedHash != "" {
		currentHash, err := planStateHash(state.Plan)
		if err != nil {
			return fmt.Errorf("hash active plan while validating dispatch_manifest: %w", err)
		}
		if expectedHash != currentHash {
			return fmt.Errorf("dispatch_manifest plan state is stale; discard the packet and rerun `aether build %d --plan-only`", manifest.Phase)
		}
	}
	return nil
}

func idempotentExternalBuildFinalizeResult(state colony.ColonyState, phaseNum int, binding buildAttemptManifestBinding, completionDigest string) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	record := binding.Record
	if record.CompletionSHA256 == "" || record.CompletionSHA256 != completionDigest {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s has no matching durable completion packet", record.ID)
	}
	if state.State != colony.StateBUILT || state.CurrentPhase != phaseNum {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s is already finalized, but colony state has advanced; do not replay its completion packet", record.ID)
	}
	phase := state.Plan.Phases[phaseNum-1]
	if phase.Status != colony.PhaseInProgress {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s is already finalized, but phase %d is %s", record.ID, phaseNum, phase.Status)
	}
	dispatches := append([]codexBuildDispatch{}, record.Dispatches...)
	resultCollectionRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "result-collection.json"))
	result := map[string]interface{}{
		"phase":             phaseNum,
		"phase_name":        phase.Name,
		"state":             state.State,
		"plan_only":         false,
		"dispatch_mode":     "external-task",
		"dispatches":        codexBuildDispatchMaps(dispatches),
		"dispatch_count":    len(dispatches),
		"wave_count":        len(buildWaveExecutionPlans(dispatches, effectiveParallelMode(state))),
		"parallel_mode":     string(effectiveParallelMode(state)),
		"selected_tasks":    append([]string{}, record.SelectedTasks...),
		"checkpoint":        record.Checkpoint,
		"manifest":          record.Manifest,
		"claims_path":       record.ClaimsPath,
		"attempt":           displayDataPath(binding.Path),
		"result_collection": displayDataPath(resultCollectionRel),
		"idempotent":        true,
		"next":              "aether continue",
	}
	var boundaryQuestions []discussQuestion
	if record.PlanManifest != nil {
		boundaryQuestions = record.PlanManifest.BoundaryQuestions
	}
	addOrchestratorBoundaryGuidance(result, "build", state, "aether continue", boundaryQuestions)
	closeLifecycleRun(result, state, "build")
	return result, state, phase, dispatches, nil
}

// restatePartialCreditFromCommittedState rebuilds each dispatch's credited
// task set from the committed phase, which is the durable record of what the
// first finalize actually credited.
//
// A dispatch's CompletedTaskIDs are runtime-owned in-process state and are
// deliberately never serialized (CR-03, 195-REVIEW.md -- a wrapper-authored
// manifest must not be able to hand itself completion credit), so a record
// reloaded from the attempt journal carries none. The phase's own task
// statuses carry the same information and cannot be authored by a wrapper,
// which makes them the correct source for a replay.
func restatePartialCreditFromCommittedState(phase colony.Phase, dispatches []codexBuildDispatch) []codexBuildDispatch {
	completed := make(map[string]struct{}, len(phase.Tasks))
	for idx := range phase.Tasks {
		if phase.Tasks[idx].Status == colony.TaskCompleted {
			completed[buildTaskID(phase.Tasks[idx], idx)] = struct{}{}
		}
	}
	restated := make([]codexBuildDispatch, len(dispatches))
	copy(restated, dispatches)
	for i := range restated {
		var credited []string
		for _, id := range dispatchCoveredTaskIDs(restated[i]) {
			if _, ok := completed[id]; ok {
				credited = append(credited, id)
			}
		}
		restated[i].CompletedTaskIDs = credited
	}
	return restated
}

// updatedPhaseForPartialReplay returns the phase as it stands after the
// partial credit was committed. Kept tiny and separate so the replay path
// cannot accidentally reach for a stale copy.
func updatedPhaseForPartialReplay(state colony.ColonyState, phaseNum int) colony.Phase {
	if phaseNum >= 1 && phaseNum <= len(state.Plan.Phases) {
		return state.Plan.Phases[phaseNum-1]
	}
	return colony.Phase{}
}

// idempotentExternalPartialFinalizeResult is the partial-credit twin of
// idempotentExternalBuildFinalizeResult (WR-02, 195-REVIEW.md). It re-reports
// a committed partial from its own durable attempt record and mutates nothing:
// no colony state write, no attempt transition, no new credit.
//
// The recovery command is re-derived through planPartialBuildRetry, the pure
// half of the D-10 recovery planner, so a replay hands the owner the same
// command as the first call without writing anything. The recovery record
// itself is only ever LOOKED UP (findExistingBuildAttemptRetry); when the
// first call's record write failed, the replay reports the command and omits
// the record's id rather than creating one.
//
// NEW-05 (195-REVIEW.iter2.md): this used to call reconcilePartialBuildRetry,
// whose writing half creates the record when none is found -- so the "mutates
// nothing" promise above held only while the first call's record survived,
// and the one case that breaks it (that write having failed) is exactly the
// case a replay exists for.
func idempotentExternalPartialFinalizeResult(state colony.ColonyState, phaseNum int, phase colony.Phase, binding buildAttemptManifestBinding, completionDigest string) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	record := binding.Record
	if record.CompletionSHA256 == "" || record.CompletionSHA256 != completionDigest {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s has no matching durable completion packet", record.ID)
	}
	dispatches := restatePartialCreditFromCommittedState(phase, record.Dispatches)
	resultCollectionRel := filepath.ToSlash(filepath.Join("build", fmt.Sprintf("phase-%d", phaseNum), "result-collection.json"))
	result := map[string]interface{}{
		"phase":             phaseNum,
		"phase_name":        phase.Name,
		"state":             state.State,
		"plan_only":         false,
		"dispatch_mode":     "external-task",
		"dispatches":        codexBuildDispatchMaps(dispatches),
		"dispatch_count":    len(dispatches),
		"wave_count":        len(buildWaveExecutionPlans(dispatches, effectiveParallelMode(state))),
		"parallel_mode":     string(effectiveParallelMode(state)),
		"selected_tasks":    append([]string{}, record.SelectedTasks...),
		"checkpoint":        record.Checkpoint,
		"manifest":          record.Manifest,
		"claims_path":       record.ClaimsPath,
		"attempt":           displayDataPath(binding.Path),
		"result_collection": displayDataPath(resultCollectionRel),
		"idempotent":        true,
		"next":              "aether continue",
	}
	if plan, err := planPartialBuildRetry(phaseNum, phase, dispatches); err != nil {
		visualFprintf(stderr, "warning: could not restate phase %d's recovery job: %v\n", phaseNum, err)
	} else if plan != nil {
		result["recovery_job"] = true
		result["parent_attempt_id"] = record.ID
		result["unfinished_task_ids"] = append([]string{}, plan.UnfinishedTaskIDs...)
		result["recovery_command"] = plan.RedispatchCommand
		result["next"] = plan.RedispatchCommand
		if existingRel, existing, ok := findExistingBuildAttemptRetry(phaseNum, record.ID); ok {
			result["retry_attempt_id"] = existing.ID
			result["retry_attempt_path"] = displayDataPath(existingRel)
		}
	}
	var boundaryQuestions []discussQuestion
	if record.PlanManifest != nil {
		boundaryQuestions = record.PlanManifest.BoundaryQuestions
	}
	addOrchestratorBoundaryGuidance(result, "build", state, "aether continue", boundaryQuestions)
	closeLifecycleRun(result, state, "build")
	return result, state, phase, dispatches, nil
}

func reconcileCommittedExternalBuildAttempt(state colony.ColonyState, phaseNum int, binding buildAttemptManifestBinding, completionDigest string) (map[string]interface{}, colony.ColonyState, colony.Phase, []codexBuildDispatch, error) {
	record := binding.Record
	if record.CompletionSHA256 == "" || record.CompletionSHA256 != completionDigest || record.Claims == nil {
		// This is the idempotency path: re-submitting the SAME completion packet
		// for an already-committed build returns the same result safely. A
		// different packet is refused by design — a committed build is not
		// superseded by a later one.
		//
		// The old message ("without matching terminal evidence") read as an
		// invitation to go and produce matching evidence, which is impossible:
		// any redispatch yields a new digest. A real session spent six worker
		// dispatches discovering that, re-running the phase with four workers,
		// then six, then the full eleven, before concluding it could not be
		// done. Say what the situation is and name the path that works.
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf(
			"build attempt %s is already committed, so this different completion packet cannot replace it. "+
				"If files changed after the build signed off — a reviewer's findings fixed, for example — do not redispatch: "+
				"run `aether continue`, which re-runs verification and accepts amended artifacts when it passes green",
			record.ID)
	}
	manifestRel := strings.TrimPrefix(filepath.ToSlash(record.Manifest), ".aether/data/")
	var finalManifest codexBuildManifest
	if err := store.LoadJSON(manifestRel, &finalManifest); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s cannot reconcile without its final manifest: %w", record.ID, err)
	}
	if finalManifest.PlanOnly || finalManifest.Phase != phaseNum || finalManifest.AttemptID != record.ID || finalManifest.AttemptPath != displayDataPath(binding.Path) {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s final manifest does not prove the committed lifecycle state", record.ID)
	}
	claimsRel := strings.TrimPrefix(filepath.ToSlash(record.ClaimsPath), ".aether/data/")
	var persistedClaims codexBuildClaims
	if err := store.LoadJSON(claimsRel, &persistedClaims); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s cannot reconcile without persisted claims: %w", record.ID, err)
	}
	recordClaimsDigest, err := jsonSHA256(record.Claims)
	if err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	persistedClaimsDigest, err := jsonSHA256(persistedClaims)
	if err != nil || recordClaimsDigest != persistedClaimsDigest {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, fmt.Errorf("build attempt %s persisted claims do not match terminal evidence", record.ID)
	}
	if err := transitionBuildAttempt(binding.Path, buildAttemptBuilt, "reconciled attempt journal after built lifecycle commit", record.Dispatches, record.Claims, "external-task", nil); err != nil {
		return nil, colony.ColonyState{}, colony.Phase{}, nil, err
	}
	binding.Record.Status = buildAttemptBuilt
	return idempotentExternalBuildFinalizeResult(state, phaseNum, binding, completionDigest)
}

func buildExternalBuildRecoveryInstructions(phaseNum int, dispatches []codexBuildDispatch) ([]map[string]interface{}, error) {
	var failed []codexBuildDispatch
	for _, dispatch := range dispatches {
		// An honest completed_no_change is a success (ruling D6), so it must
		// never be routed into recovery/redispatch. Asking a worker to redo
		// work it correctly reported as already done is the fake-edit
		// pressure the no-change status exists to remove.
		if !externalBuildDispatchNeedsRecovery(dispatch.Status) {
			continue
		}
		failed = append(failed, dispatch)
	}
	if len(failed) == 0 {
		return nil, nil
	}

	wave := failed[0].Wave
	if wave <= 0 {
		wave = 1
	}
	budget := budgetFromRecoveryLog(phaseNum, wave)
	if budget == nil {
		budget = newRecoveryBudget(wave)
	}
	cb := globalCircuitBreaker
	if cb == nil {
		cb = NewCircuitBreaker(3)
	}
	workerDispatches := codexWorkerDispatchesForRecovery(dispatches, phaseNum)

	instructions := make([]map[string]interface{}, 0, len(failed))
	var logEntries []RecoveryLogEntry
	for _, dispatch := range failed {
		status := strings.ToLower(strings.TrimSpace(dispatch.Status))
		message := strings.TrimSpace(dispatch.Summary)
		if len(dispatch.Blockers) > 0 {
			message = strings.TrimSpace(message + " " + strings.Join(dispatch.Blockers, " "))
		}
		outcome := orchestrateRecovery(RecoveryContext{
			Phase:          phaseNum,
			Wave:           normalizedDispatchWave(dispatch),
			WorkerName:     dispatch.Name,
			TaskID:         dispatch.TaskID,
			Caste:          dispatch.Caste,
			Status:         status,
			ErrorMessage:   message,
			Dispatches:     workerDispatches,
			CircuitBreaker: cb,
			Budget:         budget,
		})
		logEntries = append(logEntries, outcome.LogEntries...)
		instructions = append(instructions, map[string]interface{}{
			"worker":             dispatch.Name,
			"task_id":            dispatch.TaskID,
			"caste":              dispatch.Caste,
			"status":             status,
			"action":             outcome.Action.Type,
			"peer":               outcome.Action.PeerName,
			"detail":             outcome.Action.Detail,
			"classification":     string(outcome.Classification),
			"failure_type":       string(outcome.FailureType),
			"rationale":          outcome.Rationale,
			"budget_remaining":   outcome.Action.BudgetRemaining,
			"recovery_exhausted": outcome.Exhausted,
		})
	}
	if err := appendRecoveryOutcomesToLog(phaseNum, budget, logEntries); err != nil {
		return nil, err
	}
	return instructions, nil
}

// codexWorkerDispatchesForRecovery rebuilds dispatches for a retry.
//
// It used to set nine fields and pass dispatch.Task — the one-line task string
// — as the whole brief. So a worker being retried *after failing* received
// strictly less than the attempt that had already failed: no capsule, no
// skills, no pheromones, no relay, no success criteria. Worst of all Root was
// empty, and cmd.Dir is only set when Root is non-empty, so the retry ran in
// the orchestrator's working directory rather than the repository it was
// supposed to be fixing.
//
// A retry is the moment context matters most. It now carries everything the
// original dispatch carried, and the handoff is re-resolved so the retry can
// see what the failed attempt reported.
func codexWorkerDispatchesForRecovery(dispatches []codexBuildDispatch, phaseNum int) []codex.WorkerDispatch {
	root := resolveAetherRoot()
	capsule := resolveCodexWorkerContext()
	// PheromoneSection is deliberately NOT resolved here (D-190-03-A / 190-05):
	// capsule already renders "## Pheromone Signals" unconditionally whenever a
	// signal is active (cmd/colony_prime_context.go:571). These WorkerDispatch
	// values are never passed through AssemblePrompt/AssembleHostedPrompt today
	// (buildExternalBuildRecoveryInstructions only reads them in-memory for
	// same-caste peer lookup), but populating a redundant PheromoneSection would
	// leave a duplication trap for the moment a future change wires this into
	// a live invocation, mirroring the pattern this plan just closed on the
	// live native/direct dispatch path.
	//
	// HandoffSection is deliberately NOT resolved here either (D-190-05-A /
	// 190-06), for the identical reason but a different field: capsule already
	// renders "## Previous Worker Handoffs" for "build"-workflow records
	// (cmd/colony_prime_context.go:695), and this function's per-dispatch
	// HandoffSection used the SAME "build" workflow tag -- an exact duplicate
	// of the capsule's own content, not the "materially different workflow"
	// case 190-06 fixed for continue/colonize/plan/seal/swarm. Removed for
	// consistency, same as PheromoneSection above.

	workers := make([]codex.WorkerDispatch, 0, len(dispatches))
	for _, dispatch := range dispatches {
		brief := strings.TrimSpace(dispatch.Brief)
		if brief == "" {
			brief = dispatch.Task
		}
		agentName := strings.TrimSpace(dispatch.AgentName)
		if agentName == "" {
			agentName = codexAgentNameForCaste(dispatch.Caste)
		}
		workers = append(workers, codex.WorkerDispatch{
			ID:                normalizedDispatchTaskID(dispatch),
			WorkerName:        dispatch.Name,
			AgentName:         agentName,
			Caste:             dispatch.Caste,
			TaskID:            dispatch.TaskID,
			TaskBrief:         brief,
			ContextCapsule:    capsule,
			SkillSection:      dispatch.SkillSection,
			PermissionProfile: dispatch.PermissionProfile,
			DeclaredPaths:     append([]string{}, dispatch.DeclaredPaths...),
			Root:              root,
			Wave:              normalizedDispatchWave(dispatch),
			Workflow:          "build",
			Phase:             phaseNum,
		})
	}
	return workers
}

func appendRecoveryOutcomesToLog(phaseNum int, budget *RecoveryBudget, entries []RecoveryLogEntry) error {
	file, err := recoveryLogReadPhase(phaseNum)
	if err != nil {
		file = RecoveryLogFile{Phase: phaseNum}
	}
	file.Entries = append(file.Entries, entries...)
	file.RecoveryBudget = budget
	rel := fmt.Sprintf("recovery-log-%d.json", phaseNum)
	return store.SaveJSON(rel, file)
}

// validateCompletionPacketSemantics is the single entrypoint for validating
// a submitted completion packet. It runs, in order, accumulating into one
// slice and never short-circuiting on the first problem:
//
//  1. validateCompletionPacketStructure (cmd/contract_schema.go, plan 01) --
//     structural/type/shape problems against the generated JSON Schema, run
//     against completion.structuralInput(): the wrapper's submitted,
//     envelope-unwrapped JSON when the packet came from
//     loadExternalBuildCompletion, or a struct round-trip via
//     completionPacketAsRaw only for packets constructed in-process
//     (stageBuildAttemptCompletionFromWorkerRuns, and every test that builds
//     a codexExternalBuildCompletion literal directly, where no submitted
//     bytes exist to validate).
//  2. validateExternalWorkerResultClaimPaths (task 1) -- claim-path
//     violations across every worker and field.
//  3. The dispatch-level checks inside mergeExternalBuildResults (task 2).
//
// A non-empty return means the whole packet is rejected (D-06): the caller
// must not mutate colony state, the attempt journal, or the completion
// digest binding until this returns. Plan 03 calls this same entrypoint
// from cmd/build_attempt.go's stage-time path so stage-time validation
// equals finalize-time validation (D-07).
func validateCompletionPacketSemantics(root string, completion codexExternalBuildCompletion) []contractViolation {
	var violations []contractViolation

	if raw, err := completion.structuralInput(); err != nil {
		violations = append(violations, contractViolation{
			Rule:    "schema.marshal",
			Message: fmt.Sprintf("failed to marshal completion packet for structural validation: %v", err),
		})
	} else {
		violations = append(violations, validateCompletionPacketStructure(raw)...)
	}

	violations = append(violations, validateExternalWorkerResultClaimPaths(root, completion.workerResults())...)

	if manifest := completion.activeManifest(); manifest != nil {
		_, mergeViolations, _ := mergeExternalBuildResults(*manifest, completion.workerResults())
		violations = append(violations, mergeViolations...)
	}

	return violations
}

// completionPacketAsRaw round-trips completion through encoding/json into a
// generic decoded value (map[string]any / []any / scalars), the shape
// validateCompletionPacketStructure expects.
func completionPacketAsRaw(completion codexExternalBuildCompletion) (any, error) {
	data, err := json.Marshal(completion)
	if err != nil {
		return nil, err
	}
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// Rule strings for mergeExternalBuildResults's dispatch-level violations.
// Ambiguous or already-matched name collisions (from
// selectExternalBuildResultForDispatch) are reported under
// worker.duplicate_result -- both are the same underlying problem, a name
// that does not resolve to exactly one worker result.
const (
	violationRuleNameRequired     = "worker.name_required"
	violationRuleDuplicateResult  = "worker.duplicate_result"
	violationRuleResultMissing    = "worker.result_missing"
	violationRuleIdentityMismatch = "worker.identity_mismatch"
	violationRuleStatusTerminal   = "worker.status_terminal"
	violationRuleHandoffValid     = "handoff.valid"
	// violationRuleNoChangeEvidence fires when a completed_no_change result
	// cannot say what it verified. An honest "nothing needed changing" is a
	// first-class success (owner ruling D6), but only WITH evidence: a
	// summary stating why, a passing handoff verification_status, and the
	// commands_run that prove somebody actually checked. Without those it is
	// indistinguishable from a worker that did nothing — the exact free-pass
	// loophole the phantom-build guards exist to close.
	violationRuleNoChangeEvidence = "worker.no_change_evidence"
	// violationRuleDispositionUnknown fires on a disposition value outside
	// the vocabulary ("" or "verified_existing"): one success status with a
	// disposition, never a second terminal state machine (ruling D6).
	violationRuleDispositionUnknown = "worker.disposition_unknown"
	// violationRuleCoveredTaskUnknown fires when a worker result's
	// covered_task_ids names a task ID that resolves to no dispatch anywhere
	// in the manifest -- an unrecognized claim, never silently accepted or
	// silently dropped (191.1-PATTERNS.md Pattern 6).
	violationRuleCoveredTaskUnknown = "worker.covered_task_unknown"
	// violationRuleCoveredTaskDuplicate fires when two different worker
	// results both claim covered_task_ids credit for the same task ID --
	// the second claim is rejected rather than silently overwriting the
	// first credit.
	violationRuleCoveredTaskDuplicate = "worker.covered_task_duplicate"
	// violationRuleCoveredTaskCreditUnevidenced fires when a covered_task_ids
	// claim's own CLAIMANT fails validation (CR-01, 191.1-REVIEW.md): its own
	// result is not a genuine success (completed/manually-reconciled), or
	// carries no evidence at all (no outputs/files_created/files_modified/
	// tests_written), or its name matches no dispatch anywhere in the
	// manifest. A worker whose own work is unproven, absent, or
	// unidentifiable can never durably credit ANOTHER dispatch as completed.
	violationRuleCoveredTaskCreditUnevidenced = "worker.covered_task_credit_unevidenced"
	// violationRuleBundledWorkSuspected is additive guidance (never a
	// replacement for the genuine violationRuleResultMissing violations it
	// rides alongside, D-04): it fires when exactly one dispatch has a real,
	// evidenced completed result and one or more other dispatches have no
	// result at all, naming the concrete covered_task_ids repair instead of
	// leaving a dead-end refusal.
	violationRuleBundledWorkSuspected = "worker.bundled_work_suspected"
)

// mergeExternalBuildResults merges a completion packet's worker results onto
// the plan-only manifest's dispatches, accumulating every dispatch-level
// problem it finds into a []contractViolation instead of returning on the
// first one (D-05/D-06). err is reserved for genuine internal failures the
// wrapper cannot fix by resubmitting a corrected packet; every case a
// wrapper CAN fix becomes a violation and the loop keeps going so later
// dispatches are still checked. When a dispatch's result cannot be resolved,
// that slot keeps its original, unmodified manifest dispatch.
func mergeExternalBuildResults(manifest codexBuildManifest, results []codexExternalBuildWorkerResult) ([]codexBuildDispatch, []contractViolation, error) {
	var violations []contractViolation
	resultByName := make(map[string]codexExternalBuildWorkerResult, len(results))
	for _, result := range results {
		name := result.effectiveName()
		if name == "" {
			violations = append(violations, contractViolation{
				Field:   "name",
				Rule:    violationRuleNameRequired,
				Message: "external worker result missing name",
			})
			continue
		}
		if existing, exists := resultByName[name]; exists {
			if useIncoming, ok := preferCompletedResultOverTimeout(existing.Status, result.Status); ok {
				if useIncoming {
					resultByName[name] = result
				}
				continue
			}
			violations = append(violations, contractViolation{
				Worker:  name,
				Field:   "name",
				Value:   name,
				Rule:    violationRuleDuplicateResult,
				Message: fmt.Sprintf("duplicate external worker result for %s", name),
			})
			continue
		}
		resultByName[name] = result
	}

	// covered_task_ids resolution (FIELD-02, 191.1-PATTERNS.md Pattern 6) runs
	// as its own pass, BEFORE the main per-dispatch loop below, and not
	// inline inside it: the main loop unconditionally resets
	// dispatches[i] = dispatch at the top of every iteration, so a credit
	// written into dispatches[j] while processing the covering dispatch's own
	// iteration would be silently overwritten once the loop reaches index j
	// on its own turn (and a covering worker can equally sit at a HIGHER
	// index than the dispatches it covers, so no loop ordering makes this
	// safe as an inline mutation). Resolving credits first, into a lookup the
	// main loop's own "missing result" branch consults, is what makes a
	// covered dispatch never see a violationRuleResultMissing in the first
	// place, honestly, regardless of index order.
	dispatchIndexByTaskID := make(map[string]int, len(manifest.Dispatches))
	for idx, d := range manifest.Dispatches {
		if taskID := strings.TrimSpace(d.TaskID); taskID != "" {
			if _, exists := dispatchIndexByTaskID[taskID]; !exists {
				dispatchIndexByTaskID[taskID] = idx
			}
		}
		// A manifest dispatch that is ITSELF a runtime-grouped coherent job
		// (planCoherentJobs) already covers more than its own
		// primary TaskID; a worker's covered_task_ids claim must resolve
		// against that full chain too, not just the chain's first step.
		for _, covered := range d.CoveredTaskIDs {
			if trimmed := strings.TrimSpace(covered); trimmed != "" {
				if _, exists := dispatchIndexByTaskID[trimmed]; !exists {
					dispatchIndexByTaskID[trimmed] = idx
				}
			}
		}
	}

	type coveredTaskCredit struct {
		coveringName   string
		coveringResult codexExternalBuildWorkerResult
	}
	// WR-02 (191.1-REVIEW.md): iterate the SAME name-deduplicated
	// resultByName map the main dispatch loop above already computes (post
	// preferCompletedResultOverTimeout resolution), not the raw results
	// slice. A worker that legitimately resubmits under its own identical
	// name -- first timeout, then completed, both carrying the same
	// covered_task_ids claim -- must be treated as ONE claim here too, the
	// same way the main loop already treats it as one legitimate
	// resubmission rather than a conflict. Iterating raw results made a
	// worker's own resubmission trip violationRuleCoveredTaskDuplicate
	// against itself.
	dispatchNameSet := make(map[string]struct{}, len(manifest.Dispatches)*2)
	for _, d := range manifest.Dispatches {
		if name := strings.TrimSpace(d.Name); name != "" {
			dispatchNameSet[name] = struct{}{}
			dispatchNameSet[stripWorkerRetrySuffix(name)] = struct{}{}
		}
	}
	coveredBy := make(map[int]coveredTaskCredit, len(resultByName))
	for _, result := range resultByName {
		if len(result.CoveredTaskIDs) == 0 {
			continue
		}
		coveringName := result.effectiveName()
		selfTaskID := strings.TrimSpace(result.TaskID)

		// CR-01 (191.1-REVIEW.md): validate the CLAIMANT itself before
		// validating any individual claimed task ID. Only a worker whose OWN
		// result is a genuine, evidenced success and corresponds to a real
		// dispatch in this manifest may grant covered_task_ids credit to
		// ANOTHER dispatch -- never a failed/blocked/timeout result, never a
		// claim carrying zero evidence, and never a fabricated name matching
		// no dispatch anywhere. Every disqualification below is a distinct,
		// named violation for EVERY task ID this claimant named (D-06: no
		// silent drop) so a buggy or malicious packet is refused with an
		// actionable reason instead of quietly losing the credit.
		ownStatus := normalizeExternalBuildStatus(result.Status)
		genuineSuccess := isSuccessfulExternalBuildStatus(ownStatus)
		hasEvidence := len(result.Outputs) > 0 || len(result.FilesCreated) > 0 || len(result.FilesModified) > 0 || len(result.TestsWritten) > 0
		// A completed_no_change claimant's evidence is the verification it
		// ran, not files it changed (it changed none, honestly). Its
		// commands_run satisfy the evidence requirement here; the merge
		// loop's no_change_evidence gate independently enforces the full
		// evidence rule on the claimant's own result.
		if !hasEvidence && isNoChangeExternalBuildStatus(ownStatus) {
			hasEvidence = len(result.Handoff.CommandsRun) > 0
		}
		_, recognizedClaimant := dispatchNameSet[coveringName]
		if !recognizedClaimant {
			_, recognizedClaimant = dispatchNameSet[stripWorkerRetrySuffix(coveringName)]
		}
		if !genuineSuccess || !hasEvidence || !recognizedClaimant {
			var reason string
			switch {
			case !recognizedClaimant:
				reason = fmt.Sprintf("claimant %q matches no dispatch in the manifest", coveringName)
			case !genuineSuccess:
				reason = fmt.Sprintf("claimant %q has its own status %q, not a genuine success", coveringName, result.Status)
			default:
				reason = fmt.Sprintf("claimant %q carries no outputs, files_created, files_modified, or tests_written to evidence the claim", coveringName)
			}
			for _, raw := range result.CoveredTaskIDs {
				coveredTaskID := strings.TrimSpace(raw)
				if coveredTaskID == "" || coveredTaskID == selfTaskID {
					continue
				}
				violations = append(violations, contractViolation{
					Worker:  coveringName,
					Field:   "covered_task_ids",
					Value:   coveredTaskID,
					Rule:    violationRuleCoveredTaskCreditUnevidenced,
					Message: fmt.Sprintf("%s claims covered_task_ids credit for task %s, but %s; refused", coveringName, coveredTaskID, reason),
				})
			}
			continue
		}

		for _, raw := range result.CoveredTaskIDs {
			coveredTaskID := strings.TrimSpace(raw)
			if coveredTaskID == "" || coveredTaskID == selfTaskID {
				continue // Blank, or a self-reference to the covering dispatch's own task -- not another dispatch.
			}
			idx, exists := dispatchIndexByTaskID[coveredTaskID]
			if !exists {
				violations = append(violations, contractViolation{
					Worker:  coveringName,
					Field:   "covered_task_ids",
					Value:   coveredTaskID,
					Rule:    violationRuleCoveredTaskUnknown,
					Message: fmt.Sprintf("%s claims covered_task_ids credit for task %s, but no dispatch in the manifest has that task ID", coveringName, coveredTaskID),
				})
				continue
			}
			if existing, already := coveredBy[idx]; already {
				violations = append(violations, contractViolation{
					Worker:  coveringName,
					Field:   "covered_task_ids",
					Value:   coveredTaskID,
					Rule:    violationRuleCoveredTaskDuplicate,
					Message: fmt.Sprintf("%s and %s both claim covered_task_ids credit for task %s; only one worker's result can be credited for it", existing.coveringName, coveringName, coveredTaskID),
				})
				continue
			}
			coveredBy[idx] = coveredTaskCredit{coveringName: coveringName, coveringResult: result}
		}
	}

	dispatches := make([]codexBuildDispatch, len(manifest.Dispatches))
	usedResults := make(map[string]bool, len(results))
	for i, dispatch := range manifest.Dispatches {
		// CR-03 (195-REVIEW.md): task credit and per-task claims are decided
		// by the runtime from evidence it checked itself, never accepted from
		// an inbound manifest. The struct tags already keep them off the wire;
		// clearing here as well means no in-memory path -- a test fixture, a
		// future decoder, a hand-built manifest -- can smuggle a verdict past
		// the receipt boundary either.
		dispatch.CompletedTaskIDs = nil
		dispatch.TaskClaims = nil
		dispatches[i] = dispatch
		resultName, result, ok, err := selectExternalBuildResultForDispatch(dispatch.Name, resultByName, usedResults)
		if err != nil {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "name",
				Rule:    violationRuleDuplicateResult,
				Message: err.Error(),
			})
			continue
		}
		if !ok {
			if credit, covered := coveredBy[i]; covered {
				// A different worker's result honestly named this dispatch's
				// task ID in covered_task_ids, validated above against the
				// manifest: the wrapper bundled this dispatch's real work
				// into that worker's single call. Credit it directly instead
				// of reporting a missing result -- the work was actually
				// done, just not filed under this dispatch's own name.
				// The credited dispatch inherits the claimant's outcome, not
				// a hardcoded "completed". A completed_no_change claimant
				// covered work that produced no files by definition; stamping
				// it "completed" made the covered dispatch fail continue
				// provenance for having no outputs -- the honest result
				// punished one hop downstream.
				dispatch.Status = "completed"
				creditStatus := normalizeExternalBuildStatus(credit.coveringResult.Status)
				if isNoChangeExternalBuildStatus(creditStatus) {
					dispatch.Status = creditStatus
					dispatch.Disposition = strings.ToLower(strings.TrimSpace(credit.coveringResult.Disposition))
					if dispatch.Disposition == "" && rawStatusCarriesVerifiedExisting(credit.coveringResult.Status) {
						dispatch.Disposition = "verified_existing"
					}
				}
				dispatch.Summary = fmt.Sprintf("covered by %s via covered_task_ids", credit.coveringName)
				if outputs := uniqueSortedStrings(append(append(append([]string{}, credit.coveringResult.Outputs...), credit.coveringResult.FilesCreated...), append(credit.coveringResult.FilesModified, credit.coveringResult.TestsWritten...)...)); len(outputs) > 0 {
					dispatch.Outputs = outputs
				}
				dispatches[i] = dispatch
				continue
			}
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "name",
				Rule:    violationRuleResultMissing,
				Message: fmt.Sprintf("missing external worker result for %s", dispatch.Name),
			})
			continue
		}
		usedResults[resultName] = true
		if err := validateExternalResultIdentity(dispatch, result); err != nil {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "identity",
				Rule:    violationRuleIdentityMismatch,
				Message: err.Error(),
			})
			continue
		}
		status := normalizeExternalBuildStatus(result.Status)
		if !isTerminalExternalBuildStatus(status) {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "status",
				Value:   result.Status,
				Rule:    violationRuleStatusTerminal,
				Message: fmt.Sprintf("external worker result for %s has non-terminal status %q", dispatch.Name, result.Status),
			})
			continue
		}
		if err := codex.ValidateWorkerHandoff(result.Handoff); err != nil {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "handoff",
				Rule:    violationRuleHandoffValid,
				Message: fmt.Sprintf("external worker result for %s has invalid handoff: %v", dispatch.Name, err),
			})
			continue
		}
		disposition := strings.ToLower(strings.TrimSpace(result.Disposition))
		if disposition == "" && rawStatusCarriesVerifiedExisting(result.Status) {
			disposition = "verified_existing"
		}
		if disposition != "" && disposition != "verified_existing" {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "disposition",
				Value:   result.Disposition,
				Rule:    violationRuleDispositionUnknown,
				Message: fmt.Sprintf("external worker result for %s has unknown disposition %q; the only recognized value is verified_existing", dispatch.Name, result.Disposition),
			})
			continue
		}
		if disposition != "" && !isNoChangeExternalBuildStatus(status) {
			violations = append(violations, contractViolation{
				Worker:  dispatch.Name,
				Field:   "disposition",
				Value:   result.Disposition,
				Rule:    violationRuleDispositionUnknown,
				Message: fmt.Sprintf("external worker result for %s carries disposition %q on status %q; a disposition only qualifies completed_no_change", dispatch.Name, disposition, status),
			})
			continue
		}
		if isNoChangeExternalBuildStatus(status) {
			if missing := noChangeEvidenceMissing(result); len(missing) > 0 {
				violations = append(violations, contractViolation{
					Worker:  dispatch.Name,
					Field:   "status",
					Value:   result.Status,
					Rule:    violationRuleNoChangeEvidence,
					Message: fmt.Sprintf("external worker result for %s claims completed_no_change without evidence — missing: %s", dispatch.Name, strings.Join(missing, "; ")),
				})
				continue
			}
		}
		dispatch.Status = status
		dispatch.Disposition = disposition
		dispatch.Summary = strings.TrimSpace(result.Summary)
		dispatch.Blockers = uniqueSortedStrings(result.Blockers)
		dispatch.Duration = result.Duration
		if outputs := uniqueSortedStrings(append(append(append([]string{}, result.Outputs...), result.FilesCreated...), append(result.FilesModified, result.TestsWritten...)...)); len(outputs) > 0 {
			dispatch.Outputs = outputs
		}
		// Threaded through unchanged, regardless of status (D-08/D-09): this
		// merge boundary is shared by native and external submissions alike,
		// and whether any receipt earns credit is a decision for
		// admitCoherentJobTaskReceipts/finalizeCoherentJobTaskReceiptEvidence
		// (cmd/coherent_job_receipts.go), never this function.
		dispatch.TaskReceipts = append([]codex.TaskReceipt{}, result.TaskReceipts...)
		dispatches[i] = dispatch
	}

	// D-04/criterion 2b: additive guidance, appended only after every genuine
	// violationRuleResultMissing violation above has already been recorded,
	// and never removing or replacing any of them (the packet really is
	// incomplete right now). When the packet's shape strongly resembles the
	// field failure this plan closes -- exactly one dispatch with a real,
	// evidenced completed result sitting next to one or more dispatches with
	// no result at all -- name the specific worker, the specific missing
	// dispatches, and the concrete covered_task_ids repair, instead of
	// leaving only a dead-end refusal with no path forward.
	var missingDispatchNames []string
	for _, v := range violations {
		if v.Rule == violationRuleResultMissing {
			missingDispatchNames = append(missingDispatchNames, v.Worker)
		}
	}
	if len(missingDispatchNames) > 0 {
		var completedWithEvidence []codexBuildDispatch
		for _, d := range dispatches {
			if d.Status == "completed" && len(d.Outputs) > 0 {
				completedWithEvidence = append(completedWithEvidence, d)
			}
		}
		if len(completedWithEvidence) == 1 {
			completed := completedWithEvidence[0]
			missingTaskIDs := make([]string, 0, len(missingDispatchNames))
			for _, name := range missingDispatchNames {
				for _, d := range dispatches {
					if d.Name == name && strings.TrimSpace(d.TaskID) != "" {
						missingTaskIDs = append(missingTaskIDs, strings.TrimSpace(d.TaskID))
						break
					}
				}
			}
			violations = append(violations, contractViolation{
				Worker: completed.Name,
				Field:  "covered_task_ids",
				Rule:   violationRuleBundledWorkSuspected,
				Message: fmt.Sprintf(
					"%s is the only dispatch with a real, evidenced completed result; %s have no result at all. If %s's work actually covered them too, resubmit %s's result with covered_task_ids naming their task IDs (%s) instead of leaving them unreported.",
					completed.Name, strings.Join(missingDispatchNames, ", "), completed.Name, completed.Name, strings.Join(missingTaskIDs, ", "),
				),
			})
		}
	}

	return dispatches, violations, nil
}

func selectExternalBuildResultForDispatch(expectedName string, resultByName map[string]codexExternalBuildWorkerResult, used map[string]bool) (string, codexExternalBuildWorkerResult, bool, error) {
	expectedName = strings.TrimSpace(expectedName)
	if expectedName == "" {
		return "", codexExternalBuildWorkerResult{}, false, nil
	}
	if result, ok := resultByName[expectedName]; ok {
		if used[expectedName] {
			return "", codexExternalBuildWorkerResult{}, false, fmt.Errorf("external worker result for %s matched more than one dispatch", expectedName)
		}
		return expectedName, result, true, nil
	}

	expectedBase := stripWorkerRetrySuffix(expectedName)
	matches := []string{}
	for name := range resultByName {
		if used[name] {
			continue
		}
		if stripWorkerRetrySuffix(name) == expectedBase {
			matches = append(matches, name)
		}
	}
	sort.Strings(matches)
	switch len(matches) {
	case 0:
		return "", codexExternalBuildWorkerResult{}, false, nil
	case 1:
		name := matches[0]
		return name, resultByName[name], true, nil
	default:
		return "", codexExternalBuildWorkerResult{}, false, fmt.Errorf("ambiguous external worker result for %s: %s", expectedName, strings.Join(matches, ", "))
	}
}

func stripWorkerRetrySuffix(name string) string {
	name = strings.TrimSpace(name)
	idx := strings.LastIndex(name, "-r")
	if idx <= 0 || idx+2 >= len(name) {
		return name
	}
	for _, r := range name[idx+2:] {
		if r < '0' || r > '9' {
			return name
		}
	}
	return name[:idx]
}

func persistExternalBuildHandoffs(root string, phaseNum int, dispatches []codexBuildDispatch, results []codexExternalBuildWorkerResult) error {
	resultByName := make(map[string]codexExternalBuildWorkerResult, len(results))
	for _, result := range results {
		if name := result.effectiveName(); name != "" {
			resultByName[name] = result
		}
	}
	usedResults := make(map[string]bool, len(results))
	for _, dispatch := range dispatches {
		resultName, result, ok, err := selectExternalBuildResultForDispatch(dispatch.Name, resultByName, usedResults)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		usedResults[resultName] = true
		status := normalizeExternalBuildStatus(result.Status)
		// A completed worker must relay something to the next phase. Empty
		// handoffs were previously accepted and persisted, which filled the
		// handoff store with content-free records — the chain was "written but
		// empty, read but not delivered." Failing here is what makes the
		// wrapper's mandatory-handoff instruction enforceable rather than prose.
		if status == buildWorkerCompleted && codex.IsEmptyWorkerHandoff(result.Handoff) {
			return fmt.Errorf("worker %s completed without a handoff; completed workers must relay changed_files, commands_run, verification_status, and next_worker_instructions so the next phase inherits their context", resultName)
		}
		filesCreated, err := validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("worker %s files_created", resultName), result.FilesCreated)
		if err != nil {
			return err
		}
		filesModified, err := validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("worker %s files_modified", resultName), result.FilesModified)
		if err != nil {
			return err
		}
		testsWritten, err := validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("worker %s tests_written", resultName), result.TestsWritten)
		if err != nil {
			return err
		}
		workerResult := &codex.WorkerResult{
			WorkerName:    dispatch.Name,
			Caste:         dispatch.Caste,
			TaskID:        dispatch.TaskID,
			Status:        status,
			Summary:       result.Summary,
			FilesCreated:  filesCreated,
			FilesModified: filesModified,
			TestsWritten:  testsWritten,
			Blockers:      result.Blockers,
			Handoff:       codex.NormalizeWorkerHandoff(root, result.Handoff),
		}
		if err := persistDispatchWorkerHandoff(codex.WorkerDispatch{
			WorkerName: dispatch.Name,
			Caste:      dispatch.Caste,
			TaskID:     dispatch.TaskID,
			Workflow:   "build",
			Phase:      phaseNum,
			Wave:       normalizedDispatchWave(dispatch),
			Root:       root,
		}, codex.DispatchResult{
			WorkerName:   dispatch.Name,
			Status:       status,
			WorkerResult: workerResult,
		}); err != nil {
			return err
		}
	}
	return nil
}

func validateExternalResultIdentity(dispatch codexBuildDispatch, result codexExternalBuildWorkerResult) error {
	dispatchSpec := workerIdentitySpec{
		Caste:         dispatch.Caste,
		Stage:         dispatch.Stage,
		TaskID:        dispatch.TaskID,
		Wave:          dispatch.Wave,
		ExecutionWave: normalizedDispatchWave(dispatch),
	}
	resultSpec := workerIdentitySpec{
		Caste:         result.Caste,
		Stage:         result.Stage,
		TaskID:        result.TaskID,
		Wave:          result.Wave,
		ExecutionWave: result.ExecutionWave,
	}
	return validateWorkerResultIdentity(dispatch.Name, dispatchSpec, resultSpec)
}

func normalizeExternalBuildStatus(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	switch status {
	case "complete", "done", "success", "succeeded", "passed", "code_written":
		return "completed"
	// An honest "nothing needed changing" is one success status (ruling D6);
	// verified_existing arrives as a raw status from workers but is a
	// DISPOSITION on completed_no_change, folded in by the merge path.
	case "no_change", "no-change", "nochange", "unchanged", "completed_no_change",
		"verified_existing", "already_complete", "already_correct":
		return "completed_no_change"
	// A quota/rate-limit stop is a resumable interruption, never an
	// ordinary code failure (ruling D7, spec §5 StopFailure semantics).
	case "interrupted", "suspended_quota", "rate_limit", "rate_limited":
		return "interrupted"
	case "fail", "error":
		return "failed"
	case "timed_out", "cancelled", "canceled":
		return "timeout"
	case "manual", "manually_reconciled":
		return "manually-reconciled"
	default:
		return status
	}
}

func isTerminalExternalBuildStatus(status string) bool {
	switch status {
	case "completed", "completed_no_change", "interrupted", "failed", "blocked", "timeout", "manually-reconciled":
		return true
	default:
		return false
	}
}

// isSuccessfulExternalBuildStatus reports whether a terminal status counts
// as the work SUCCEEDING. interrupted is deliberately absent: it is terminal
// (the worker stopped and the record is final) but the work is unfinished
// and resumable — counting it as success would complete tasks nobody did.
func isSuccessfulExternalBuildStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "completed", "completed_no_change", "manually-reconciled":
		return true
	default:
		return false
	}
}

// externalBuildDispatchNeedsRecovery reports whether a dispatch's terminal
// status means the work must be redispatched. An honest completed_no_change
// is a success (ruling D6) and must never be routed into recovery: asking a
// worker to redo work it correctly reported as already done is exactly the
// fake-edit pressure the no-change status exists to remove.
func externalBuildDispatchNeedsRecovery(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == "" {
		return false
	}
	return !isSuccessfulExternalBuildStatus(status)
}

func isNoChangeExternalBuildStatus(status string) bool {
	return status == "completed_no_change"
}

// rawStatusCarriesVerifiedExisting reports whether the worker's RAW status
// spelling asserted the stronger claim ("this already exists and I proved
// it") rather than merely "no change was needed" — folded into the
// disposition so the distinction survives normalization.
func rawStatusCarriesVerifiedExisting(rawStatus string) bool {
	switch strings.ToLower(strings.TrimSpace(rawStatus)) {
	case "verified_existing", "already_complete", "already_correct":
		return true
	default:
		return false
	}
}

// noChangeEvidenceMissing lists which of the three required evidence pieces
// a completed_no_change result lacks; empty means the evidence rule is met.
func noChangeEvidenceMissing(result codexExternalBuildWorkerResult) []string {
	return noChangeEvidenceMissingFrom(result.Summary, result.Handoff.VerificationStatus, result.Handoff.CommandsRun)
}

// noChangeEvidenceMissingFrom is the single definition of the no-change
// evidence rule. It takes the three pieces directly so BOTH lanes can apply
// it: the external/wrapper lane through mergeExternalBuildResults, and the
// in-process runtime lane through validateRuntimeNoChangeEvidence. When this
// rule lived only on the external result type, the runtime lane accepted an
// evidence-free completed_no_change and passed a build with no changes and
// nothing checked -- the phantom-build loophole, reopened on one lane.
func noChangeEvidenceMissingFrom(summary, verificationStatus string, commandsRun []string) []string {
	missing := []string{}
	if strings.TrimSpace(summary) == "" {
		missing = append(missing, "summary stating why no change was needed")
	}
	if !strings.EqualFold(strings.TrimSpace(verificationStatus), "pass") {
		missing = append(missing, "handoff verification_status: pass")
	}
	if len(commandsRun) == 0 {
		missing = append(missing, "handoff commands_run naming what was checked")
	}
	return missing
}

func parseManifestGeneratedAt(manifest codexBuildManifest) time.Time {
	if ts, err := time.Parse(time.RFC3339, strings.TrimSpace(manifest.GeneratedAt)); err == nil {
		return ts.UTC()
	}
	return time.Now().UTC()
}

func (c codexExternalBuildCompletion) claimsOrAggregate(root string, phaseNum int, startedAt time.Time, dispatches []codexBuildDispatch) (codexBuildClaims, error) {
	if c.Claims != nil {
		claims := *c.Claims
		claims.BuildPhase = phaseNum
		if strings.TrimSpace(claims.Timestamp) == "" {
			claims.Timestamp = startedAt.Format(time.RFC3339)
		}
		if err := validateAndNormalizeBuildClaims(root, "completion claims", &claims); err != nil {
			return codexBuildClaims{}, err
		}
		// codexBuildClaims.FilesCreated/FilesModified are non-omitempty --
		// the completion-packet schema requires them present as arrays.
		// validateAndNormalizeClaimPathsToRoot returns nil for an empty
		// input (e.g. a legitimate verification-only submission with no
		// file claims), which would otherwise marshal to JSON null and fail
		// structural validation. Normalize to an empty (not nil) array so a
		// genuinely empty claim set stays structurally valid.
		if claims.FilesCreated == nil {
			claims.FilesCreated = []string{}
		}
		if claims.FilesModified == nil {
			claims.FilesModified = []string{}
		}
		return claims, nil
	}

	byName := map[string]codexExternalBuildWorkerResult{}
	for _, result := range c.workerResults() {
		name := result.effectiveName()
		if name != "" {
			byName[name] = result
		}
	}
	usedResults := make(map[string]bool, len(byName))
	claims := codexBuildClaims{BuildPhase: phaseNum, Timestamp: startedAt.Format(time.RFC3339)}
	taskClaims := map[string]*codexBuildTaskClaim{}
	for _, dispatch := range dispatches {
		if dispatch.Status != "completed" {
			continue
		}
		resultName, result, ok, err := selectExternalBuildResultForDispatch(dispatch.Name, byName, usedResults)
		if err != nil {
			return codexBuildClaims{}, err
		}
		if !ok {
			continue
		}
		usedResults[resultName] = true
		claims.FilesCreated = append(claims.FilesCreated, result.FilesCreated...)
		claims.FilesModified = append(claims.FilesModified, result.FilesModified...)
		claims.TestsWritten = append(claims.TestsWritten, result.TestsWritten...)
		taskID := strings.TrimSpace(dispatch.TaskID)
		if taskID == "" {
			continue
		}
		entry, ok := taskClaims[taskID]
		if !ok {
			entry = &codexBuildTaskClaim{TaskID: taskID}
			taskClaims[taskID] = entry
		}
		entry.FilesCreated = append(entry.FilesCreated, result.FilesCreated...)
		entry.FilesModified = append(entry.FilesModified, result.FilesModified...)
		entry.TestsWritten = append(entry.TestsWritten, result.TestsWritten...)
	}
	// D-08/D-09: a grouped job's own Status may be non-whole-success
	// (failed/blocked/timeout) while still carrying root-evidenced,
	// receipt-credited task claims (dispatch.TaskClaims, populated only by
	// finalizeCoherentJobTaskReceiptEvidence). The loop above only walks
	// whole-success dispatches, so those receipt claims are folded in here,
	// separately, keyed by the CREDITED task's own ID -- never the covering
	// dispatch's primary TaskID -- so last-build-claims.json and the exact
	// per-task credit this build actually proved never disagree.
	for _, dispatch := range dispatches {
		for _, tc := range dispatch.TaskClaims {
			taskID := strings.TrimSpace(tc.TaskID)
			if taskID == "" {
				continue
			}
			entry, ok := taskClaims[taskID]
			if !ok {
				entry = &codexBuildTaskClaim{TaskID: taskID}
				taskClaims[taskID] = entry
			}
			entry.FilesCreated = append(entry.FilesCreated, tc.FilesCreated...)
			entry.FilesModified = append(entry.FilesModified, tc.FilesModified...)
			entry.TestsWritten = append(entry.TestsWritten, tc.TestsWritten...)
			claims.FilesCreated = append(claims.FilesCreated, tc.FilesCreated...)
			claims.FilesModified = append(claims.FilesModified, tc.FilesModified...)
			claims.TestsWritten = append(claims.TestsWritten, tc.TestsWritten...)
		}
	}
	claims.FilesCreated = uniqueSortedStrings(claims.FilesCreated)
	claims.FilesModified = uniqueSortedStrings(claims.FilesModified)
	claims.TestsWritten = uniqueSortedStrings(claims.TestsWritten)

	// Filesystem fallback: if claims are empty but builders completed, discover files via git.
	if len(claims.FilesCreated) == 0 && len(claims.FilesModified) == 0 && hasCompletedBuilders(dispatches) {
		created, modified := discoverChangedFilesFromGit()
		claims.FilesCreated = created
		claims.FilesModified = modified
	}
	if err := validateAndNormalizeBuildClaims(root, "aggregated worker claims", &claims); err != nil {
		return codexBuildClaims{}, err
	}

	if len(taskClaims) > 0 {
		taskIDs := make([]string, 0, len(taskClaims))
		for taskID := range taskClaims {
			taskIDs = append(taskIDs, taskID)
		}
		sort.Strings(taskIDs)
		for _, taskID := range taskIDs {
			entry := taskClaims[taskID]
			entry.FilesCreated = uniqueSortedStrings(entry.FilesCreated)
			entry.FilesModified = uniqueSortedStrings(entry.FilesModified)
			entry.TestsWritten = uniqueSortedStrings(entry.TestsWritten)
			if err := validateAndNormalizeBuildTaskClaim(root, "aggregated worker task claims", entry); err != nil {
				return codexBuildClaims{}, err
			}
			claims.TaskClaims = append(claims.TaskClaims, *entry)
		}
	}
	return claims, nil
}

// collectClaimPathViolations validates every path in paths against root,
// accumulating one contractViolation per bad path instead of returning on
// the first problem. Each violation names the worker and the concrete field
// it came from (files_created, files_modified, tests_written, outputs) so a
// wrapper author can act on the field, not a generic label. A claim under a
// sanctioned .aether/data scratch prefix
// (validateAndNormalizeClaimPathToRoot's ("", nil) branch) is accepted and
// silently dropped, exactly as before -- never a violation.
func collectClaimPathViolations(root, worker, field string, paths []string) []contractViolation {
	var violations []contractViolation
	for _, path := range uniqueSortedStrings(paths) {
		if _, err := validateAndNormalizeClaimPathToRoot(root, field, path); err != nil {
			violations = append(violations, contractViolation{
				Worker:  worker,
				Field:   field,
				Value:   path,
				Rule:    claimPathRuleFromError(err),
				Message: err.Error(),
			})
		}
	}
	return violations
}

// claimPathRuleFromError extracts the machine-readable rule string a
// validateAndNormalizeClaimPathToRoot error was wrapped with. Every return
// path inside that function wraps its error with *claimPathRuleError, so
// the fallback here is defensive only and should never trigger in practice.
func claimPathRuleFromError(err error) string {
	var ruleErr *claimPathRuleError
	if errors.As(err, &ruleErr) {
		return ruleErr.rule
	}
	return claimPathRuleEscapesRoot
}

// validateExternalWorkerResultClaimPaths validates every claimed path across
// every worker and every claim field (outputs, files_created,
// files_modified, tests_written), returning every violation found in one
// pass rather than the first. Worker label comes from
// codexExternalBuildWorkerResult.effectiveName(), falling back to "unnamed".
func validateExternalWorkerResultClaimPaths(root string, results []codexExternalBuildWorkerResult) []contractViolation {
	var violations []contractViolation
	for _, result := range results {
		name := result.effectiveName()
		if name == "" {
			name = "unnamed"
		}
		violations = append(violations, collectClaimPathViolations(root, name, "outputs", result.Outputs)...)
		violations = append(violations, collectClaimPathViolations(root, name, "files_created", result.FilesCreated)...)
		violations = append(violations, collectClaimPathViolations(root, name, "files_modified", result.FilesModified)...)
		violations = append(violations, collectClaimPathViolations(root, name, "tests_written", result.TestsWritten)...)
	}
	return violations
}

type codexResultCollectionReport struct {
	Workflow        string         `json:"workflow"`
	Phase           int            `json:"phase,omitempty"`
	PhaseName       string         `json:"phase_name,omitempty"`
	RecordedAt      string         `json:"recorded_at"`
	ExpectedWorkers int            `json:"expected_workers"`
	ReceivedResults int            `json:"received_results"`
	MatchedResults  int            `json:"matched_results"`
	StatusCounts    map[string]int `json:"status_counts,omitempty"`
	// CreditedTaskIDs and UnfinishedTaskIDs (D-08/D-09) name, separately,
	// exactly which selected tasks this build actually proved complete
	// (whole-success or root-evidenced receipt credit) and which did not --
	// never inferred from touched files, a worker's summary, or
	// covered_task_ids membership alone.
	CreditedTaskIDs         []string                     `json:"credited_task_ids,omitempty"`
	UnfinishedTaskIDs       []string                     `json:"unfinished_task_ids,omitempty"`
	Issues                  []codexResultCollectionIssue `json:"issues,omitempty"`
	Policy                  string                       `json:"policy"`
	ApprovedTempPath        string                       `json:"approved_temp_path"`
	SensitiveOutputRedacted bool                         `json:"sensitive_output_redacted"`
}

type codexResultCollectionIssue struct {
	Worker string `json:"worker,omitempty"`
	Caste  string `json:"caste,omitempty"`
	Status string `json:"status,omitempty"`
	Kind   string `json:"kind"`
	Detail string `json:"detail,omitempty"`
}

func buildExternalBuildResultCollectionReport(phaseNum int, phaseName string, expected []codexBuildDispatch, results []codexExternalBuildWorkerResult, dispatches []codexBuildDispatch, recordedAt time.Time, selectedTaskIDs []string) codexResultCollectionReport {
	report := codexResultCollectionReport{
		Workflow:                "build",
		Phase:                   phaseNum,
		PhaseName:               strings.TrimSpace(phaseName),
		RecordedAt:              recordedAt.UTC().Format(time.RFC3339),
		ExpectedWorkers:         len(expected),
		ReceivedResults:         len(results),
		MatchedResults:          len(dispatches),
		StatusCounts:            map[string]int{},
		CreditedTaskIDs:         creditedBuildTaskIDs(selectedTaskIDs, dispatches),
		UnfinishedTaskIDs:       unfinishedBuildTaskIDs(selectedTaskIDs, dispatches),
		Policy:                  "A structurally valid successful worker result (completed, completed_no_change, or manually-reconciled) wins over a timeout placeholder for the same worker; malformed JSON, duplicate terminal results, missing claims, stale manifests, and .aether/data completion files are rejected. Claims under sanctioned .aether/data subpaths (planning/, phase-research/, survey/, worker-debug/, reviews/) are tolerated and dropped from the claim set.",
		ApprovedTempPath:        finalizerCompletionTempPattern,
		SensitiveOutputRedacted: true,
	}
	for _, dispatch := range dispatches {
		status := normalizeExternalBuildStatus(dispatch.Status)
		if status == "" {
			status = "unknown"
		}
		report.StatusCounts[status]++
		switch {
		case isSuccessfulExternalBuildStatus(status):
			// completed, completed_no_change and manually-reconciled are all
			// successes; none is an unexpected status at finalization.
		case status == "interrupted":
			report.Issues = append(report.Issues, codexResultCollectionIssue{
				Worker: dispatch.Name,
				Caste:  dispatch.Caste,
				Status: status,
				Kind:   "worker_interrupted",
				Detail: "worker stopped before finishing its slice; the work is unfinished and the phase can be redispatched",
			})
		case status == "timeout":
			report.Issues = append(report.Issues, codexResultCollectionIssue{
				Worker: dispatch.Name,
				Caste:  dispatch.Caste,
				Status: status,
				Kind:   "collection_timeout",
				Detail: "worker reached terminal timeout status before a valid completed result was collected",
			})
		case status == "failed" || status == "blocked":
			report.Issues = append(report.Issues, codexResultCollectionIssue{
				Worker: dispatch.Name,
				Caste:  dispatch.Caste,
				Status: status,
				Kind:   "worker_failure",
				Detail: codex.SanitizeWorkerDiagnosticOutput(firstNonEmpty(dispatch.Summary, strings.Join(dispatch.Blockers, "; "))),
			})
		default:
			report.Issues = append(report.Issues, codexResultCollectionIssue{
				Worker: dispatch.Name,
				Caste:  dispatch.Caste,
				Status: status,
				Kind:   "unexpected_status",
				Detail: "worker result reached finalization with an unexpected status",
			})
		}
	}
	return report
}

func validateAndNormalizeBuildClaims(root, owner string, claims *codexBuildClaims) error {
	var err error
	if claims.FilesCreated, err = validateAndNormalizeClaimPathsToRoot(root, owner+" files_created", claims.FilesCreated); err != nil {
		return err
	}
	if claims.FilesModified, err = validateAndNormalizeClaimPathsToRoot(root, owner+" files_modified", claims.FilesModified); err != nil {
		return err
	}
	if claims.TestsWritten, err = validateAndNormalizeClaimPathsToRoot(root, owner+" tests_written", claims.TestsWritten); err != nil {
		return err
	}
	for i := range claims.TaskClaims {
		if err := validateAndNormalizeBuildTaskClaim(root, owner, &claims.TaskClaims[i]); err != nil {
			return err
		}
	}
	return nil
}

func validateAndNormalizeBuildTaskClaim(root, owner string, claim *codexBuildTaskClaim) error {
	taskLabel := strings.TrimSpace(claim.TaskID)
	if taskLabel == "" {
		taskLabel = "unassigned"
	}
	var err error
	if claim.FilesCreated, err = validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("%s task %s files_created", owner, taskLabel), claim.FilesCreated); err != nil {
		return err
	}
	if claim.FilesModified, err = validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("%s task %s files_modified", owner, taskLabel), claim.FilesModified); err != nil {
		return err
	}
	if claim.TestsWritten, err = validateAndNormalizeClaimPathsToRoot(root, fmt.Sprintf("%s task %s tests_written", owner, taskLabel), claim.TestsWritten); err != nil {
		return err
	}
	return nil
}

func recordExternalBuildSpawnTree(dispatches []codexBuildDispatch) error {
	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	entries, err := spawnTree.Parse()
	if err != nil {
		return fmt.Errorf("failed to read spawn tree: %w", err)
	}
	known := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		known[entry.AgentName] = struct{}{}
	}
	for _, dispatch := range dispatches {
		if _, ok := known[dispatch.Name]; !ok {
			if err := spawnTree.RecordSpawn("Queen", dispatch.Caste, dispatch.Name, dispatch.Task, 1); err != nil {
				return fmt.Errorf("failed to record external build dispatch %s: %w", dispatch.Name, err)
			}
			known[dispatch.Name] = struct{}{}
		}
		if err := spawnTree.UpdateStatus(dispatch.Name, dispatch.Status, dispatch.Summary); err != nil {
			return fmt.Errorf("failed to complete external build dispatch %s: %w", dispatch.Name, err)
		}
	}
	return nil
}

func hasCompletedBuilders(dispatches []codexBuildDispatch) bool {
	for _, d := range dispatches {
		if strings.EqualFold(d.Caste, "builder") && d.Status == "completed" {
			return true
		}
	}
	return false
}

func discoverChangedFilesFromGit() (created, modified []string) {
	if out, err := exec.Command("git", "diff", "--name-only", "--diff-filter=A", "HEAD").Output(); err == nil {
		created = parseGitNameOutput(out)
	}
	if out, err := exec.Command("git", "diff", "--name-only", "--diff-filter=M", "HEAD").Output(); err == nil {
		modified = parseGitNameOutput(out)
	}
	return created, modified
}

func parseGitNameOutput(out []byte) []string {
	var result []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return uniqueSortedStrings(result)
}

func validateAndNormalizeClaimPathsToRoot(root, field string, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return nil, nil
	}
	normalized := make([]string, 0, len(paths))
	for _, path := range uniqueSortedStrings(paths) {
		rel, err := validateAndNormalizeClaimPathToRoot(root, field, path)
		if err != nil {
			return nil, err
		}
		if rel != "" {
			normalized = append(normalized, rel)
		}
	}
	return uniqueSortedStrings(normalized), nil
}

// sanctionedDataClaimPrefixes lists the .aether/data/ subpaths a worker may
// honestly claim because a real runtime instruction orders the write there:
// the hook's scratch carve-outs (sanctionedDataWritePrefixes) plus the review
// ledgers written via `aether review-ledger-write` (cmd/review_ledger.go —
// e.g. .aether/data/reviews/history/ledger.json), which findingsInjectionForCaste
// tells watcher/chaos/measurer/archaeologist/gatekeeper/auditor briefs to run.
// A claim under one of these is accepted but dropped from the normalized claim
// set: it is colony state, not repo evidence, so it must not feed claim or
// criterion verification — and an honest declaration of a runtime-ordered
// write must never fail the whole completion packet.
func sanctionedDataClaimPrefixes() []string {
	prefixes := make([]string, 0, len(sanctionedDataWritePrefixes)+1)
	for _, prefix := range sanctionedDataWritePrefixes {
		prefixes = append(prefixes, strings.TrimPrefix(prefix, "/"))
	}
	return append(prefixes, ".aether/data/reviews/")
}

// claimPathRuleError wraps a claim-path validation error with a
// machine-readable rule classification. Callers (collectClaimPathViolations)
// extract the rule via errors.As instead of string-matching the message, so
// the rule stays correct even if the human-readable wording changes.
type claimPathRuleError struct {
	rule string
	err  error
}

func (e *claimPathRuleError) Error() string { return e.err.Error() }
func (e *claimPathRuleError) Unwrap() error { return e.err }

// Rule strings for claim-path validation failures. There are exactly four:
// a malicious null byte, a path that isn't repo-relative, a forbidden
// .aether/data claim, and every other way a path fails to resolve inside
// the repository boundary (escapes root, missing, symlink, directory,
// ambiguous match, unavailable root).
const (
	claimPathRuleNullByte     = "claim_path.null_byte"
	claimPathRuleRepoRelative = "claim_path.repo_relative"
	claimPathRuleAetherData   = "claim_path.aether_data"
	claimPathRuleEscapesRoot  = "claim_path.escapes_root"
)

// wrapClaimPathRule wraps a non-nil error with its rule classification.
// A nil err is passed through unchanged (the ok-with-no-error success case).
func wrapClaimPathRule(rule string, err error) error {
	if err == nil {
		return nil
	}
	return &claimPathRuleError{rule: rule, err: err}
}

func validateAndNormalizeClaimPathToRoot(root, field, claimed string) (string, error) {
	claimed = strings.TrimSpace(claimed)
	if claimed == "" {
		return "", nil
	}
	if strings.ContainsRune(claimed, 0) {
		return "", wrapClaimPathRule(claimPathRuleNullByte, fmt.Errorf("invalid %s claim %q: path contains a null byte", field, claimed))
	}
	policyClaim := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.ReplaceAll(claimed, "\\", "/"))))
	if filepath.IsAbs(claimed) || filepath.IsAbs(filepath.FromSlash(policyClaim)) || hasWindowsVolumePrefix(policyClaim) {
		return "", wrapClaimPathRule(claimPathRuleRepoRelative, fmt.Errorf("invalid %s claim %q: path must be repo-relative", field, claimed))
	}
	if policyClaim == ".aether/data" || strings.HasPrefix(policyClaim, ".aether/data/") {
		for _, prefix := range sanctionedDataClaimPrefixes() {
			if strings.HasPrefix(policyClaim, prefix) {
				return "", nil
			}
		}
		return "", wrapClaimPathRule(claimPathRuleAetherData, fmt.Errorf("invalid %s claim %q: path must not be under .aether/data", field, claimed))
	}
	if strings.TrimSpace(root) == "" {
		return "", wrapClaimPathRule(claimPathRuleEscapesRoot, fmt.Errorf("invalid %s claim %q: repository root is unavailable", field, claimed))
	}

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", wrapClaimPathRule(claimPathRuleEscapesRoot, fmt.Errorf("invalid %s claim %q: resolve repository root: %w", field, claimed, err))
	}
	rootEval, err := filepath.EvalSymlinks(rootAbs)
	if err != nil {
		rootEval = rootAbs
	}

	candidateAbs, directCandidate, err := candidateClaimAbsolutePath(rootAbs, claimed)
	if err != nil {
		return "", wrapClaimPathRule(claimPathRuleEscapesRoot, fmt.Errorf("invalid %s claim %q: %w", field, claimed, err))
	}
	if rel, ok, err := normalizeExistingClaimPath(rootEval, candidateAbs, field, claimed); ok || err != nil {
		return rel, wrapClaimPathRule(claimPathRuleEscapesRoot, err)
	}

	if directCandidate {
		if rel, ok, err := findUnambiguousRepoRelativeClaimPath(rootAbs, rootEval, field, claimed); ok || err != nil {
			return rel, wrapClaimPathRule(claimPathRuleEscapesRoot, err)
		}
	}
	return "", wrapClaimPathRule(claimPathRuleEscapesRoot, fmt.Errorf("invalid %s claim %q: path does not exist inside repository", field, claimed))
}

func hasWindowsVolumePrefix(path string) bool {
	if len(path) < 2 || path[1] != ':' {
		return false
	}
	ch := path[0]
	return (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func candidateClaimAbsolutePath(rootAbs, claimed string) (string, bool, error) {
	claimedSlash := filepath.ToSlash(claimed)
	if filepath.IsAbs(claimed) {
		return filepath.Clean(claimed), false, nil
	}
	cleanRel := filepath.Clean(filepath.FromSlash(claimedSlash))
	if cleanRel == "." {
		return "", false, fmt.Errorf("path is empty")
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return "", false, fmt.Errorf("path escapes repository")
	}
	return filepath.Join(rootAbs, cleanRel), true, nil
}

func normalizeExistingClaimPath(rootEval, candidateAbs, field, claimed string) (string, bool, error) {
	info, err := os.Lstat(candidateAbs)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("invalid %s claim %q: inspect path: %w", field, claimed, err)
	}
	if info.IsDir() {
		return "", true, fmt.Errorf("invalid %s claim %q: path is a directory", field, claimed)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return "", true, fmt.Errorf("invalid %s claim %q: path is a symlink", field, claimed)
	}
	resolved, err := filepath.EvalSymlinks(candidateAbs)
	if err != nil {
		return "", true, fmt.Errorf("invalid %s claim %q: resolve path: %w", field, claimed, err)
	}
	rel, err := repoRelativeClaimPath(rootEval, resolved)
	if err != nil {
		return "", true, fmt.Errorf("invalid %s claim %q: %w", field, claimed, err)
	}
	return rel, true, nil
}

func repoRelativeClaimPath(rootEval, resolved string) (string, error) {
	rel, err := filepath.Rel(rootEval, resolved)
	if err != nil {
		return "", fmt.Errorf("path is not relative to repository: %w", err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("path is outside repository")
	}
	return filepath.ToSlash(rel), nil
}

func findUnambiguousRepoRelativeClaimPath(rootAbs, rootEval, field, claimed string) (string, bool, error) {
	cleanClaim := filepath.ToSlash(filepath.Clean(filepath.FromSlash(claimed)))
	base := filepath.Base(cleanClaim)
	if base == "." || base == string(filepath.Separator) {
		return "", false, nil
	}
	out, err := exec.Command("git", "-C", rootAbs, "ls-files", "--cached", "--others", "--exclude-standard", "--", "*"+base).Output()
	if err != nil {
		return "", false, nil
	}
	candidates := parseGitNameOutput(out)
	matches := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.ToSlash(candidate)
		if candidate == cleanClaim || strings.HasSuffix(candidate, "/"+cleanClaim) || cleanClaim == base {
			matches = append(matches, candidate)
		}
	}
	matches = uniqueSortedStrings(matches)
	switch len(matches) {
	case 0:
		return "", false, nil
	case 1:
		rel, ok, err := normalizeExistingClaimPath(rootEval, filepath.Join(rootAbs, filepath.FromSlash(matches[0])), field, claimed)
		return rel, ok, err
	default:
		return "", true, fmt.Errorf("invalid %s claim %q: ambiguous repository path (%s)", field, claimed, strings.Join(matches, ", "))
	}
}

// normalizeClaimPathsToRoot resolves subdirectory-relative paths to repo-root-relative paths.
// If a path already resolves from root (file exists), it is kept as-is.
// If not found, it searches the repo for a matching file and replaces with the resolved path.
func normalizeClaimPathsToRoot(root string, paths []string) []string {
	if root == "" {
		return paths
	}
	result := make([]string, len(paths))
	for i, p := range paths {
		if p == "" {
			continue
		}
		if fileExists(filepath.Join(root, filepath.FromSlash(p))) {
			result[i] = p
			continue
		}
		if resolved := findRepoRelativePath(root, p); resolved != "" {
			result[i] = resolved
			continue
		}
		// Keep original — verification will flag it as missing
		result[i] = p
	}
	return result
}

// findRepoRelativePath searches for a file in the repo that matches the claimed path.
// Uses git ls-files for fast lookup and includes untracked files so newly-created
// files can satisfy basename-only worker claims before they are staged.
func findRepoRelativePath(root, claimed string) string {
	base := filepath.Base(claimed)
	if base == "." || base == string(filepath.Separator) {
		return ""
	}

	// Try git ls-files with basename pattern for fast lookup. Include untracked
	// files because worker-created files are often not staged when continue runs.
	out, err := exec.Command("git", "-C", root, "ls-files", "--cached", "--others", "--exclude-standard", "--", "*"+base).Output()
	if err == nil {
		candidates := parseGitNameOutput(out)
		if len(candidates) == 1 {
			return candidates[0]
		}
		if len(candidates) > 1 {
			if best := bestMatchForClaimedPath(claimed, candidates); best != "" {
				return best
			}
		}
	}

	// If git ls-files found nothing, the file likely doesn't exist in the repo.
	return ""
}

// bestMatchForClaimedPath scores candidates by counting matching trailing path segments.
// Tiebreaks by shortest total path length.
func bestMatchForClaimedPath(claimed string, candidates []string) string {
	claimedParts := strings.Split(filepath.ToSlash(claimed), "/")
	best := ""
	bestScore := 0
	bestLen := 0
	for _, c := range candidates {
		cParts := strings.Split(filepath.ToSlash(c), "/")
		score := 0
		minLen := len(claimedParts)
		if len(cParts) < minLen {
			minLen = len(cParts)
		}
		for i := 1; i <= minLen; i++ {
			if claimedParts[len(claimedParts)-i] == cParts[len(cParts)-i] {
				score++
			} else {
				break
			}
		}
		cLen := len(cParts)
		if best == "" || score > bestScore || (score == bestScore && cLen < bestLen) {
			best = c
			bestScore = score
			bestLen = cLen
		}
	}
	return best
}
