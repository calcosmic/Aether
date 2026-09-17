package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/spf13/cobra"
)

const (
	commandGuideCategoryLiteral           = "literal"
	commandGuideCategoryFullOrchestration = "full-orchestration"
	commandGuideCategorySemiIntelligent   = "semi-intelligent"
	commandGuideCategoryRuntimeNative     = "runtime-native"

	commandGuideSkillCreation   = "aether-colony-creation"
	commandGuideSkillResearch   = "aether-colony-research"
	commandGuideSkillBuildCycle = "aether-colony-build-cycle"
)

type commandGuideDefinition struct {
	Category       string   `json:"category"`
	SkillReference string   `json:"skill_reference,omitempty"`
	Intent         string   `json:"intent"`
	Literal        bool     `json:"literal"`
	PreSteps       []string `json:"pre_steps,omitempty"`
	RunCommand     string   `json:"run_command"`
	PostSteps      []string `json:"post_steps,omitempty"`
	DriftGuards    []string `json:"drift_guards,omitempty"`
	RawBypass      string   `json:"raw_bypass,omitempty"`
}

type commandGuideResult struct {
	Command          string                 `json:"command"`
	Platform         string                 `json:"platform"`
	Category         string                 `json:"category"`
	SkillReference   string                 `json:"skill_reference,omitempty"`
	Intent           string                 `json:"intent"`
	Literal          bool                   `json:"literal"`
	PreSteps         []string               `json:"pre_steps,omitempty"`
	RunCommand       string                 `json:"run_command"`
	PostSteps        []string               `json:"post_steps,omitempty"`
	DriftGuards      []string               `json:"drift_guards,omitempty"`
	RawBypass        string                 `json:"raw_bypass,omitempty"`
	PlatformContract codex.PlatformContract `json:"platform_contract"`
}

var commandGuidePlatform string

var commandGuideCmd = &cobra.Command{
	Use:   "command-guide <command>",
	Short: "Return platform orchestration guidance for an Aether command",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		result, err := buildCommandGuide(args[0], commandGuidePlatform)
		if err != nil {
			outputError(1, err.Error(), nil)
			return nil
		}
		outputOK(result)
		return nil
	},
}

func init() {
	commandGuideCmd.Flags().StringVar(&commandGuidePlatform, "platform", "codex", "Target platform: codex, claude, or opencode")
	rootCmd.AddCommand(commandGuideCmd)
}

func buildCommandGuide(command, platform string) (commandGuideResult, error) {
	command = strings.TrimSpace(strings.TrimPrefix(command, "ant-"))
	platform = strings.ToLower(strings.TrimSpace(platform))
	if platform == "" {
		platform = "codex"
	}
	var runtimePlatform codex.Platform
	switch platform {
	case "codex":
		runtimePlatform = codex.PlatformCodex
	case "claude":
		runtimePlatform = codex.PlatformClaude
	case "opencode":
		runtimePlatform = codex.PlatformOpenCode
	default:
		return commandGuideResult{}, fmt.Errorf("unsupported platform %q; expected codex, claude, or opencode", platform)
	}
	platformContract, ok := codex.PlatformContractFor(runtimePlatform)
	if !ok {
		return commandGuideResult{}, fmt.Errorf("platform %q has no runtime support contract", platform)
	}

	definitions := commandGuideCatalog()
	def, ok := definitions[command]
	if !ok {
		names := make([]string, 0, len(definitions))
		for name := range definitions {
			names = append(names, name)
		}
		sort.Strings(names)
		return commandGuideResult{}, fmt.Errorf("unknown command guide %q; known commands: %s", command, strings.Join(names, ", "))
	}
	def = adaptCommandGuideDefinitionForPlatform(command, platform, def)

	return commandGuideResult{
		Command:          command,
		Platform:         platform,
		Category:         def.Category,
		SkillReference:   def.SkillReference,
		Intent:           def.Intent,
		Literal:          def.Literal,
		PreSteps:         append([]string(nil), def.PreSteps...),
		RunCommand:       def.RunCommand,
		PostSteps:        append([]string(nil), def.PostSteps...),
		DriftGuards:      append([]string(nil), def.DriftGuards...),
		RawBypass:        def.RawBypass,
		PlatformContract: platformContract,
	}, nil
}

func adaptCommandGuideDefinitionForPlatform(command, platform string, def commandGuideDefinition) commandGuideDefinition {
	if command == "build" && platform == "codex" {
		return codexNativeBuildCommandGuide(def)
	}
	if platform == "codex" || def.Literal {
		return def
	}
	adapted := def
	adapted.SkillReference = ""
	if len(adapted.PreSteps) > 0 && strings.HasPrefix(adapted.PreSteps[0], "Use `$ant-") {
		adapted.PreSteps = append([]string(nil), adapted.PreSteps...)
		adapted.PreSteps[0] = fmt.Sprintf("Use the generated %s slash-command wrapper for `%s`; do not load Codex lifecycle skills.", platform, command)
	}
	// Public suggestions retain each host's spelling. Executable commands and
	// raw bypass instructions are not display text and are never rewritten.
	for _, steps := range []*[]string{&adapted.PreSteps, &adapted.PostSteps, &adapted.DriftGuards} {
		rendered := append([]string(nil), (*steps)...)
		for i := range rendered {
			rendered[i] = strings.ReplaceAll(rendered[i], "$ant-", "/ant-")
		}
		*steps = rendered
	}
	return adapted
}

// Keep native transport instructions out of the shared primary-platform catalog.
func codexNativeBuildCommandGuide(def commandGuideDefinition) commandGuideDefinition {
	def.PreSteps = append([]string(nil), def.PreSteps...)
	def.PostSteps = append([]string(nil), def.PostSteps...)
	def.DriftGuards = append([]string(nil), def.DriftGuards...)
	def.Intent = "Use the accepted Go manifest with a native Codex child; persist its bound terminal result before aggregate staging and runtime finalization."
	var steps []string
	for _, step := range def.PreSteps {
		if strings.Contains(step, "build-wave playbook") || strings.Contains(step, "Spawn parallel waves") || strings.Contains(step, "Pass worker briefs verbatim") || strings.Contains(step, "After all terminal results") {
			continue
		}
		steps = append(steps, step)
	}
	def.PreSteps = append(steps,
		"The host native spawn_agent operation is the sole launcher. Do not execute internal-worker-adapter, a provider subprocess, or a second aether build dispatch for native work.",
		"Write strict JSON requests into an absolute regular file in a new aether-worker-request-* directory under the system temporary directory. Every request requires schema_version:1, phase and the exact manifest execution_binding. For each accepted dispatch run `aether codex-native-worker reserve --request <file>` with worker_name, task_id, actual host_session_id, canonical workspace and host_permission:workspace_write. Unsupported workspace or permissions refuse before spawning.",
		"Only a fresh launch_allowed:true response permits one spawn_agent call. Use the response dispatch agent_name/caste and worker name, and pass worker.native.prompt bytes verbatim. The child must wait without editing or running checks until released. Save the actual returned child ID. Replayed or unresolved reservations never authorize another spawn.",
		"Run `aether codex-native-worker bind --request <file>` with the same execution_binding, worker_name, task_id, host_session_id, worker.provider_run_id as launch_id, actual child_id, dispatch_sha256 and prompt_sha256 from worker.native. Only after bind succeeds send worker.native.release verbatim to that child using the native messaging tool. Wait for the actual child's terminal response.",
		"Immediately run `aether codex-native-worker record --request <file>` with all bound identity fields, result (the child's actual nonempty JSON result), source_event_id and source_event_sha256 of the retained raw terminal event. Retain raw parent/child tool events. Unknown outcomes remain unresolved. Never substitute parent edits or manufactured results for the child.",
		"Run `aether codex-native-worker stage --request <file>` with schema_version, phase and execution_binding only after required terminal results are saved; use result.completion_path. On a fresh session first run `aether codex-native-worker inspect --request <file>` for the exact saved binding; retain finished helpers without respawning them. inspect is read-only. No helper is mandatory beyond the runtime-selected team.")
	def.PreSteps = append(def.PreSteps,
		"When a bound native child asks a material question, run `aether codex-native-worker question --request <file>` with its exact saved assignment/child fields and question:{question_id:<stable actual host question/event key>,question:<the child's exact question>}. Run the same question operation without a question field after recording a terminal result to admit its saved handoff open_decisions. Do this before asking the owner; never pass native questions through generic handoff text answers.",
		"For pending questions, show the runtime-returned decision.description verbatim. Its answer_request_path contains the exact native_binding and question with an empty answer. After the actual owner supplies the answer, copy only those answer bytes into that file and execute the exact returned `aether decision-answer --native-request <file>` answer_command. Do not invent approval, substitute an answer, change bindings or impersonate checkpoint/reviewer-waiver authorization. Harness/predeclared answers are test authorization, never owner testimony. Identical answer replay retains the original receipt; conflicts refuse.",
		"For an answered current live child, run `aether codex-native-worker context --request <file>` with the same exact binding. awaiting_delivery returns context_delivery.payload, payload_sha256, child_id and decision_ids. Send that exact payload unchanged to that exact child using the host's native messaging operation. Only after observing completed send, run observe with observation_status:context_delivered, the exact context_delivery, context_send:{status:completed,child_id,message_sha256:<payload_sha256>}, actual observed_at, source_event_id and source_event_sha256. Pending/failed sends never acknowledge delivery. A context read is not consultation or evidence that the answer influenced work.",
		"On resume, inspect the saved attempt first and re-run question for that same bound child. An answered question is not re-asked; no_updates needs no send and a requested delivered context receipt is historical. Stale goal/session/attempt/child or paused-state refusal leaves the choice pending; follow the runtime recovery command rather than re-stamping old text. A terminal child's question/answer stays attached to its saved handoff; never reopen it, respawn it or give its answer to a sibling. Surface its returned next_command for the current runtime decision.",
	)
	def.RunCommand = "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by codex-native-worker stage>"
	def.DriftGuards = append(def.DriftGuards, "Native records do not grant completion credit; only the existing Go finalizer does. Missing host capability or child evidence is incomplete, never simulated success.")
	return def
}

func commandGuideCatalog() map[string]commandGuideDefinition {
	catalog := make(map[string]commandGuideDefinition, len(commandGuideLiteralCommands())+7)
	for _, command := range commandGuideLiteralCommands() {
		catalog[command] = commandGuideDefinition{
			Category:   commandGuideCategoryLiteral,
			Intent:     "Run the runtime command directly. No AI interview or wrapper orchestration is required.",
			Literal:    true,
			RunCommand: fmt.Sprintf("AETHER_OUTPUT_MODE=visual aether %s $ARGUMENTS", command),
			RawBypass:  "Literal passthrough is the default for this command.",
			DriftGuards: []string{
				"Keep this command literal unless its Claude/OpenCode wrappers grow AI reasoning or worker orchestration.",
				"If this command becomes intelligent, update command-guide, YAML codex_orchestration metadata, Codex skills, and wrapper docs in the same change.",
			},
		}
	}
	catalog["insert-phase"] = commandGuideDefinition{
		Category: commandGuideCategoryLiteral,
		Intent: "Insert a corrective phase through the Go-owned resolver. Pass one issue sentence with " +
			"`aether insert-phase \"problem to stabilise\"`. For non-interactive automation, use " +
			"`aether insert-phase --after 2 --name \"Stabilize login retries\" --description \"login retries lose state\" --constraints \"do not change the provider\"`.",
		Literal:    true,
		RunCommand: "AETHER_OUTPUT_MODE=visual aether insert-phase $ARGUMENTS",
		RawBypass:  "Literal passthrough is the default; Go resolves shorthand, prompts, and explicit flags.",
		DriftGuards: []string{
			"Keep phase position, name, and description resolution in the Go command; wrappers only pass through $ARGUMENTS.",
			"Keep the shorthand and non-interactive automation examples aligned across command-guide, canonical YAML, and both managed wrappers.",
		},
	}
	catalog["pause"] = commandGuideDefinition{
		Category:   commandGuideCategoryLiteral,
		Intent:     "Use `aether pause` and `aether resume` as the only public pause and return commands. Pause stops at a runtime-confirmed safe boundary and commits one resumable handoff.",
		Literal:    true,
		RunCommand: "AETHER_OUTPUT_MODE=visual aether pause $ARGUMENTS",
		RawBypass:  "Literal passthrough is the default; the Go runtime owns safe-boundary detection and every handoff write.",
		DriftGuards: []string{
			"Relay the runtime handoff, receipt, provenance, state effect, and next action without inventing another return route.",
			"Codex must not inspect, select, or edit recovery evidence or lifecycle state.",
		},
	}
	catalog["resume"] = commandGuideDefinition{
		Category: commandGuideCategoryLiteral,
		Intent: "Validate and restore the safest honest recovery point through the Go runtime. Keep its provenance groups distinct: " +
			"Confirmed, Reconstructed, Conflicting, and Unknown.",
		Literal:    true,
		RunCommand: "AETHER_OUTPUT_MODE=visual aether resume $ARGUMENTS",
		RawBypass:  "Literal passthrough is the only public recovery path; the Go runtime owns evidence classification and restoration.",
		DriftGuards: []string{
			"Conflicting or Unknown evidence stops with state effect none; relay the named conflict or missing fact and the runtime's exact next action.",
			"Codex must not inspect, select, or edit recovery evidence or lifecycle state.",
		},
	}
	catalog["entomb"] = commandGuideDefinition{
		Category:   commandGuideCategoryLiteral,
		Intent:     "Archive and clear a sealed colony only when the owner separately chooses this optional post-seal action.",
		Literal:    true,
		RunCommand: "AETHER_OUTPUT_MODE=visual aether entomb $ARGUMENTS",
		RawBypass:  "Literal passthrough is the default; the Go runtime owns confirmation, archive verification, publication, and active-state clearing.",
		DriftGuards: []string{
			"Keep sealed state available for review until the owner explicitly invokes this command.",
			"Never invoke entomb automatically or fold it into the seal workflow.",
		},
	}
	catalog["spec"] = commandGuideDefinition{
		Category: commandGuideCategoryRuntimeNative,
		Intent: "Inspect, revise, approve, or repair the canonical owner-readable Specification through the Go runtime. " +
			"Specification approval is separate from planning stop and exact plan-candidate acceptance.",
		Literal:    true,
		RunCommand: "AETHER_OUTPUT_MODE=visual aether spec $ARGUMENTS",
		RawBypass:  "Runtime-native passthrough is the default; Codex uses `aether spec`, while Claude/OpenCode use their generated `/ant-spec` wrapper over the same Go operations.",
		DriftGuards: []string{
			"Keep all nine typed body categories, immutable revision/hash bindings, exact approval token, projection repair, replay result, receipt, and next action owned by Go.",
			"Do not parse `.aether/SPEC.md` as authority or treat Specification approval as a planning stop, candidate acceptance, plan activation, or build eligibility.",
			"When this surface changes, update `.aether/commands/spec.yaml`, Claude/OpenCode wrappers, `cmd/contracts/spec.md`, and `command-guide` together.",
		},
	}

	catalog["init"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillCreation,
		Intent:         "Start a guided colony for one goal.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("init", commandGuideSkillCreation),
			"Stage 1 — Queen opening: name the requested goal and repository, then run `AETHER_OUTPUT_MODE=json aether init-research --goal \"<raw goal>\" --target .` for deterministic codebase context.",
			"An existing active colony is refused before storage opens; the refusal changes no files.",
			"Stage 2 — Setup: do not require a separate setup command; aether init performs safe automatic bootstrap and reports Ready, Bootstrapped, or an actionable failure.",
			"Go owns setup, registry updates, accepted-charter persistence, colony state creation, territory evidence, and init result truth.",
			"If prior_colonies.count > 0, show a Prior Context section (up to 3 recent colonies' goals and outcomes from prior_colonies.recent) before asking for the new goal.",
			"Stage 3 — Accepted intent: ask one compact batch of 4-7 questions when target users, success criteria, non-goals, constraints, risks, affected systems, or first milestone are unclear.",
			"Synthesize raw goal, user answers, and init-research output into a refined goal and charter JSON; do not echo the scan output as the final charter.",
			"Persist the owner-approved goal and material constraints as accepted-charter/v1 through aether init.",
			"Before creating colony state, ask the user to choose Colony Mode or Orchestrator Mode. Explain that Colony Mode is the existing default, while Orchestrator Mode asks guided boundary questions at phase points for tighter user control. If the user skips the choice or the host is non-interactive, default to Colony Mode.",
			"Separate deterministic housekeeping warnings from at most 3 strategic AI-synthesized pheromone suggestions, and ask approval before writing any signal.",
			"Run `aether shelf-list --json --status shelved`, let the user choose per entry, and carry the chosen IDs into the init call with `--promote-shelf` / `--dismiss-shelf` -- never promote or dismiss before the user approves, since a cancel or a failed init must leave the backlog untouched.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether init --colony-mode <selected colony|orchestrator> --charter-json '<synthesized charter JSON>' \"<refined goal>\"",
		PostSteps: []string{
			"Stage 4 — Territory: read the typed territory outcome from the runtime result rather than inspecting survey files or inferring freshness. Territory result is exactly one of Fresh, Refreshed, Stale—refresh required, or Unavailable.",
			"Stage 5 — Closeout: summarize the colony name, accepted goal, runtime-created artifacts, approved strategic pheromones, and typed territory result.",
			"Render the exact Codex skill closeout `Next Up: $ant-plan`; the executable route remains `aether plan`.",
		},
		DriftGuards: intelligentCommandDriftGuards("init", commandGuideSkillCreation),
		RawBypass:   "If the user explicitly asks for raw/exact/no-interview init, run their literal `aether init ...` command and say the synthesis layer was bypassed.",
	}

	catalog["oracle"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillResearch,
		Intent:         "Turn a loose research request into a scoped Oracle prompt, template, and confidence target before starting the Oracle loop.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("oracle", commandGuideSkillResearch),
			"Ask one compact batch of 3-6 questions when topic, audience, decision criteria, output type, constraints, or persistence expectations are unclear.",
			"Infer the Oracle template: PRD -> prd, tech comparison -> tech-eval, architecture -> architecture-review, bug/root cause -> bug-investigation, best practices -> research-brief.",
			"Synthesize the answers into a precise research prompt; do not pass a vague raw prompt through unchanged.",
			"Present research depth as selectable options unless the user already gave one: quick (5 iterations), balanced/standard (15), deep (30), or exhaustive/marathon (50). Do not hide this behind a raw flag-only flow.",
			"If the user gives an exact iteration cap, pass `--max-iterations <1-50>`.",
			"Ask the user to choose target confidence unless they already gave one: 80%, 90%, 95% recommended, or 99%; pass the selected number as `--confidence-target <percent>`.",
			"For long OpenCode-hosted runs, prefer `--background`; Oracle will detach a controller, preserve `.aether/oracle` state, and report progress through `aether oracle status`.",
			"If runtime auto-detaches because it detected a hosted Claude/OpenCode agent session, treat that as a normal background run and inspect progress through `aether oracle status`.",
			"For everything/all-of-the-above/full-system audits or large uncommitted diffs, split the topic into focused Oracle runs or start with `--depth quick`; do not collapse every area into one blocking balanced-depth prompt.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> --background \"<synthesized prompt>\"",
		PostSteps: []string{
			"If the shell/tool call times out, run `aether oracle status` before declaring failure or switching to ad hoc agents.",
			"If OpenCode subprocess dispatch is unavailable, let Oracle use its automatic Codex/Claude fallback unless the user explicitly set `AETHER_WORKER_PLATFORM=opencode`.",
			"Do not fake Oracle worker completion; if no dispatcher is available, surface the blocker and keep the saved Oracle workspace.",
			"Summarize confidence, blockers, and concrete recommendations from runtime output.",
			"Suggest persisting high-value findings as pheromones or hive wisdom only with user approval.",
		},
		DriftGuards: intelligentCommandDriftGuards("oracle", commandGuideSkillResearch),
		RawBypass:   "If the user explicitly asks for raw/exact/no-interview oracle, run their literal `aether oracle ...` command.",
	}

	catalog["plan"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Plan from one approved Specification through one Go-authorized Scout or Route-Setter stage at a time, then review and explicitly accept the exact stopped candidate.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("plan", commandGuideSkillBuildCycle),
			"Run `AETHER_OUTPUT_MODE=visual aether status` and use its lifecycle facts as context; do not infer planning authority by reading state files.",
			"Run `AETHER_OUTPUT_MODE=json aether spec --inspect` before selecting a preset. Continue only when Go reports the exact current Specification revision/hash APPROVED, its projection synchronized, and affected scope reconciled. Follow the runtime's exact repair or approval action otherwise; Specification approval is not plan acceptance.",
			"When no explicit valid policy was supplied, request `aether host plan` with no preset and render `preset_required` as four equal choices only: Fast 80/up to 4 passes, Balanced 90/up to 6, Deep 95/up to 8, and Exhaustive 99/up to 12. Do not preselect or recommend one; cancelled or invalid input dispatches no worker.",
			"After one exact choice, request `aether host plan --preset <fast|balanced|deep|exhaustive>`. Exact target/max flags may bypass only the card when Go maps them to one preset. Routine read-only phase research is automatic inside the preset and has no owner approval step.",
			"When revising future work after a completed phase, pass `--refresh --revision-type <type> --revision-reason <why>` to every host-plan iteration. Research and verification revisions also require repository-relative `--revision-evidence <path>` files.",
			"Save the structured response outside `.aether/data` and read only `result.plan_manifest.stage_manifest` as worker authority. It binds the authorization ID, run/pass/preset, approved Specification, base plan, prior card, input frontier, expected caste/result type, and caste-specific predecessor fields; the only worker result types are `planning-scout-result/v1` and `planning-route-setter-result/v1`.",
			"Dispatch exactly the single current stage. Scout is first; Route-Setter exists only in a returned `route_stage_manifest` bound to the exact Scout receipt; a later Scout exists only in a returned `scout_stage_manifest` after a completed card. Never predict, combine, or reuse a stage.",
			"Inspect the current dispatch `permission_profile` and pass it through verbatim; never substitute or broaden it. Use the runtime-provided name, caste, brief, result contract, evidence frontier, weakest gap, Scout receipt, and candidate snapshot exactly as present.",
			"If the manifest includes revision data, preserve completed phases and unaffected stable IDs. Route-Setter proposes only the current plan content and five readiness assessments; Go validates evidence, derives overall readiness/semantic delta/weakest gap/stop policy, and assigns authority-bearing IDs.",
			"When the manifest includes `queen_execution_policy.spawn_budget`, surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"Before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest file>`.",
			"Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest file> --execution-wave <execution_wave>` for the one current stage.",
			"Spawn that one manifest dispatch in the visible live Task/subagent panels with a caste-labelled description; do not dispatch both planning castes or background-only work as the ceremony.",
			"Pass the exact brief and stage result contract, honor the read budget and no-repeat guard, and mark a blocked/failed worker honestly. Never synthesize evidence, a successful result, receipt, score, stop, recommendation, candidate, acceptance, state transition, or next action.",
			"Call `aether spawn-log` before each planning worker and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("plan"),
			"Build the completion packet from the unchanged current `plan_manifest` plus exactly one strict `scout_result` or `route_result`. Never combine stages, submit legacy whole-chain worker arrays, or reuse a completion packet.",
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <approved temp worker result JSON>`.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <approved temp completion JSON>",
		PostSteps: []string{
			"Phase 199 contract: require automatic typed territory freshness from the runtime before planning; do not inspect or infer it from files.",
			"After Scout finalization, render `stage_receipt`, admitted evidence, gaps, and the exact next boundary. Dispatch Route-Setter only from `route_stage_manifest`; if `decision_cards` are present, render the complete evidence-first batch and collect all exact owner answers before resubmitting the issued resume token.",
			"Let Go resolve an owner batch as `direct_resume` or `successor_spec_required`. Direct resume returns only the exact previously authorized stage. A successor remains DRAFT: run the exact `aether spec` approval action and reconcile affected scope before requesting another Scout. Codex never edits SPEC or chooses this branch.",
			"After every Route-Setter finalization, render the complete immutable `iteration_card` before acting. Show fresh evidence, all five before/after assessments, Go-derived overall against target, weakest gap, semantic delta, stop/pause reason, and `evidence_that_would_change`.",
			"Continue only from a returned `scout_stage_manifest` targeting the weakest evidenced gap. A later material decision pauses only after the completed card. Routine phase research remains automatic and never creates a separate approval command.",
			"When Go returns a stopped `plan_candidate`, label it NOT ACTIVE and run `AETHER_OUTPUT_MODE=json aether plan --candidate`. Render the full proposal, approved Specification/base/timeline bindings, every card, five scores, residual gaps and evidence that would change them, semantic delta, and Queen recommendation with producer/rationale/evidence.",
			"Only after explicit owner confirmation execute the review result's full `acceptance_command` verbatim. The generic legacy acceptance flag is never a shortcut. Stale or divergent acceptance leaves the active plan unchanged and routes back to `aether plan --candidate`.",
			"Only a successful `acceptance_receipt` makes the new PlanRevision READY. Then render plan closeout and offer an equal guided-build/Autopilot choice; neither route is preselected or recommended. Route guided work to `$ant-build 1` or the runtime-surfaced exact command, and Autopilot to `aether run`.",
			"For a revision, surface preserved, affected, superseded, and replacement semantic IDs from the accepted revision and discard all stale stage packets, decisions, candidates, and acceptance commands.",
			"Treat exact replay as idempotent receipt retention. A stale or divergent replay stops with state unchanged and the runtime's exact recovery command; never substitute current IDs into an old packet.",
		},
		DriftGuards: intelligentCommandDriftGuards("plan", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration plan, run their literal `aether plan ...` command.",
	}

	catalog["colonize"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Use the runtime survey manifest to spawn visible platform surveyors and finalize survey state without hand-writing data files.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("colonize", commandGuideSkillBuildCycle),
			"Do not copy repo-local legacy commands back into target repos; the published platform wrappers and TS host are the orchestration surface.",
			"Run `aether host colonize $ARGUMENTS` to fetch the survey manifest via the TS host. Parse `result.colonize_manifest`; do not parse visual output.",
			"Save the full JSON envelope to a temporary manifest file, then render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow colonize --manifest-file <manifest file>`.",
			"When the colonize manifest includes `queen_execution_policy.spawn_budget`, surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"Before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"If runtime returns `dispatch_mode: agent-delegate`, dispatch the four Surveyor workers through the host platform instead of nested subprocess workers.",
			"Use runtime-provided agent names, castes, task IDs, briefs, output_paths, and skill_section values.",
			"Before each manifest wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow colonize --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"Spawn surveyors as visible live Task/subagent panels with caste-labelled descriptions; do not use background-only dispatch as the ceremony.",
			"Call `aether spawn-log` before each surveyor and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("colonize"),
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow colonize --worker-file <approved temp worker result JSON>`.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether colonize-finalize --completion-file <approved temp completion JSON>",
		PostSteps: []string{
			"After the JSON finalizer succeeds, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow colonize --completion-file <approved temp completion JSON>`.",
			"Summarize actual surveyors, survey files, and any runtime-surfaced warning.",
			"Route first to `$ant-plan`.",
		},
		DriftGuards: intelligentCommandDriftGuards("colonize", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration colonize, run their literal `aether colonize ...` command.",
	}

	catalog["swarm"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Use the runtime swarm manifest to spawn visible bug-destroyer workers and finalize swarm artifacts without hand-writing data files.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("swarm", commandGuideSkillBuildCycle),
			"If the user provides no problem description, use the generated wrapper's direct watch path: `AETHER_OUTPUT_MODE=visual aether swarm --watch`.",
			"For bug-destroyer targets, run `AETHER_OUTPUT_MODE=json aether swarm --plan-only $ARGUMENTS` and parse `result.swarm_manifest`.",
			"Save the full JSON envelope to a temporary manifest file, then render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow swarm --manifest-file <manifest file>`.",
			"Dispatch wave 1 investigation workers through the host platform, then wave 2 builder, then wave 3 watcher.",
			"Use runtime-provided agent names, castes, roles, waves, task IDs, briefs, and response contracts.",
			"Before each manifest wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow swarm --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"Spawn swarm workers as visible live Task/subagent panels with caste-labelled descriptions; do not use background-only dispatch as the ceremony.",
			"Call `aether spawn-log` before each worker and `aether spawn-complete` after each terminal result.",
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow swarm --worker-file <worker result JSON>`.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether swarm-finalize --completion-file <worker completion JSON>",
		PostSteps: []string{
			"After the JSON finalizer succeeds, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow swarm --completion-file <worker completion JSON>`.",
			"Summarize actual workers, root cause, fix or blocker, and verification evidence.",
			"Route first to the runtime-surfaced `next` command.",
		},
		DriftGuards: intelligentCommandDriftGuards("swarm", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration swarm, run their literal `aether swarm ...` command.",
	}

	catalog["build"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Use the runtime dispatch manifest to spawn platform workers and finalize the phase without hand-writing state.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("build", commandGuideSkillBuildCycle),
			"Run `AETHER_OUTPUT_MODE=visual aether status` and surface active REDIRECT, FOCUS, and FEEDBACK signals compactly.",
			"Run `aether build <phase> --plan-only` to fetch the dispatch manifest directly from the Go runtime without dispatching workers. Parse `result.dispatch_manifest`; do not parse visual output.",
			"Save the full JSON envelope to a temporary manifest file for later ceremony rendering.",
			"Coherent jobs: several related tasks become one job for one worker, and grouping is a proposal rather than a decision. Go owns accepted groups, completion credit, retry, worktree reconciliation, and check-in policy; the wrapper proposes, renders, spawns, and submits.",
			"Propose a grouping only with both a relationship and a benefit -- \"these are related\" is not a reason. `--job-proposal` is repeatable: one JSON object per group with `name`, `task_ids`, `owner_caste`, `relationship`, `benefit`, and optional `owner_reason`, e.g. `aether build --job-proposal '{\"name\":\"templates\",\"task_ids\":[\"2\",\"3\",\"4\"],\"owner_caste\":\"builder\",\"relationship\":\"these tasks edit the same templates\",\"benefit\":\"one worker avoids repeated setup and write conflicts\"}' <phase> --plan-only`. With no proposal the runtime still groups tasks joined by a dependency chain or by meaningful shared implementation files; incidental overlap through a README, changelog, or dependency manifest joins nothing.",
			"Read `job_decisions` and relay it in plain English. Each entry's `status` is `accepted` or `refused`; a refusal names `offending_task_id`, `dependency_id`, and the `replacement_job_names` the runtime substituted, and only that group is repaired. Each dispatch then carries `job_name`, `job_reason`, `job_source` (`queen`, `automatic`, `single`, or `retry`) and `covered_task_ids` in order -- render them, never edit them, and never re-propose a refused grouping unchanged. A real dependency cycle blocks dispatch for the whole phase, names the cycle, and names the plan repair: surface it and stop.",
			"Team check-in: `checkin_requested` is the runtime's decision and `checkin_reason` says why -- never infer either from the flags passed. `one_worker_fast_path` means one worker with nothing left for the owner to decide: render `checkin_summary` as a short non-blocking note and continue without asking. `non_interactive` (autopilot or `--no-checkin`) skips the stage. `explicit_checkin`, `pending_owner_decision`, and `default_pause` all keep the full blocking check-in including the forced-reviewer waiver flow. `--checkin` is the owner override forcing the pause on a decision-free one-worker build, and combining it with `--no-checkin` is refused by name before any side effect.",
			"Read `queen_execution_policy.spawn_budget` from the dispatch manifest and surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"If provider dispatch is unavailable, surface only the Go-owned structured availability message: provider, sanitized cause, and next action. Do not include raw provider stdout, stderr, tokens, or auth probe output.",
			"Before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow build --manifest-file <manifest file>` for the old-style caste-colored spawn ceremony.",
			"Follow the installed build-wave playbook and use runtime-provided agent names, castes, task IDs, briefs, and skill_section values.",
			"Before dispatching each worker, inspect its typed `permission_profile`. Do not broaden it: repository_read_only must run through a host-enforced no-write boundary; scoped_write or test_write must be rejected until the selected host reports enforcement. Treat `behavioral_restrictions` under workspace_write as required behavior, not as a sandbox claim.",
			"Before each manifest wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow build --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"Spawn parallel waves as visible live Task/subagent panels with caste-labelled descriptions; do not use background-only dispatch as the ceremony.",
			"Pass worker briefs verbatim: `dispatch.brief_path` (a repo-display path to the file holding the composed brief, byte for byte) is now the routine channel every dispatch carries -- the runtime writes the composed brief to disk and reports the path, so inline JSON briefs of 6-22KB never hit Read-tool long-line truncation. Inline `dispatch.brief` appears only in the rare case where the runtime could not write the file for that dispatch; honor it verbatim when it is the only one present. Whichever one a dispatch carries, use it verbatim, never merge, summarize, or reconstruct. Enforce read cache discipline: if a worker keeps re-reading the same unchanged file, mark it blocked with the missing context instead of waiting for another loop.",
			"Call `aether spawn-log` before each worker and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("build"),
			"After all terminal results are accepted, run `AETHER_OUTPUT_MODE=json aether build-completion-stage <phase> --completion-file <approved temp completion JSON>` exactly once. Parse `result.completion_path`; this Go-owned packet is the recovery source if the wrapper stops before finalization.",
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow build --worker-file <approved temp worker result JSON>`.",
			"Task receipts: a dispatch's `covered_task_ids` is that worker's assignment scope, not credit for it. A worker that finishes only part of its job submits a `task_receipts` array -- one entry per proved task with `task_id`, `status`, `summary`, `files_created`, `files_modified`, `tests_written`, and its own `handoff`. A task with no receipt is unfinished; never infer completion because a related file changed.",
			"An accepted task receipt is admission, not completion credit: only the runtime's root-backed finalization can grant `completed_task_ids`. Never author `covered_task_ids` or `completed_task_ids` by hand in a manifest or in colony state; the runtime owns both.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <Go-owned completion_path returned by build-completion-stage>",
		PostSteps: []string{
			"For Autopilot, show displayed Autopilot bounds before runtime work begins, preserve concrete repair/debt receipts from runtime results, and allow independent safe-path continuation while only unsafe or blocked work pauses.",
			"After the JSON finalizer succeeds, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow build --completion-file <Go-owned completion_path>`.",
			"Read the finalizer's own answer instead of assuming a job finished whole: each dispatch's `completed_task_ids` is what the runtime actually credited, and `recovery_job` set to true means part of the job was proven and part was not. `unfinished_task_ids` lists what remains, `parent_attempt_id` and `retry_attempt_id` link the appended recovery attempt to the original one, and `recovery_command` is the exact command that redispatches only the unfinished tasks. Relay `recovery_command`; never ask a new worker to redo credited work.",
			"In worktree mode one job takes one worktree, one branch, and one merge-back. The runtime admits receipts, syncs only what it admitted back to the project root, then credits. Anything the worker touched but never proved is neither synced nor destroyed -- it stays on a preserved branch the runtime names, and must be reported that way rather than as lost or as done.",
			"Summarize actual workers, completed tasks, and the most relevant signal or risk.",
			"Route first to `aether continue`.",
		},
		DriftGuards: intelligentCommandDriftGuards("build", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration build, run their literal `aether build ...` command.",
	}

	catalog["continue"] = commandGuideDefinition{
		Category:       commandGuideCategorySemiIntelligent,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Run runtime-owned verification by default, with Codex orchestration only for heavy external review manifests.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("continue", commandGuideSkillBuildCycle),
			"Run `AETHER_OUTPUT_MODE=visual aether status` and frame continue as verification, not another build pass.",
			"Use the default runtime path unless the user requested `--classic-ceremony`, `--verification-depth heavy`, or runtime asks for wrapper-spawned review workers.",
			"For classic/heavy external review, run `aether host continue --dry-run --classic-ceremony $ARGUMENTS` (or `aether host continue --dry-run --verification-depth heavy $ARGUMENTS`) to fetch the manifest via the TS host without dispatching reviewers. Parse `result.manifest.continue_manifest`; do not parse visual output.",
			"For heavy external review, save the full JSON envelope to a temporary manifest file for later ceremony rendering.",
			"For heavy external review, when the manifest includes `queen_execution_policy.spawn_budget`, surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"For heavy external review, before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"For heavy external review, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow continue --manifest-file <manifest file>`.",
			"Before each heavy review wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow continue --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"For heavy external review, spawn reviewers as visible live Task/subagent panels with caste-labelled descriptions; do not use background-only dispatch as the ceremony.",
			"Pass reviewer briefs verbatim and enforce read cache discipline: if a reviewer keeps re-reading the same unchanged file or artifact, mark it blocked with the missing context.",
			"For heavy external review, call `aether spawn-log` before each reviewer and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("continue"),
			"For heavy external review, after each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow continue --worker-file <approved temp worker result JSON>`.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether continue --verification-depth standard $ARGUMENTS",
		PostSteps: []string{
			"For heavy external review, after `continue-finalize` succeeds, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow continue --completion-file <approved temp completion JSON>`.",
			"If phase advanced, summarize verification and route to the next `$ant-build <phase>`.",
			"If blocked, follow the runtime recovery command first.",
			"If complete, route to `$ant-seal`.",
		},
		DriftGuards: intelligentCommandDriftGuards("continue", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration continue, run their literal `aether continue ...` command.",
	}

	catalog["seal"] = commandGuideDefinition{
		Category:       commandGuideCategorySemiIntelligent,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Use the runtime seal manifest to spawn visible final-review workers, then finalize sealing through the runtime.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("seal", commandGuideSkillBuildCycle),
			"Keep explicit seal: Force flags pass only when directly supplied by the owner. The Go runtime owns final review, preflight, confirmation, transaction, and rendering. A forced-incomplete closure is not verified success. Full workflow coverage and native-worker parity remain pending.",
			"Run `AETHER_OUTPUT_MODE=visual aether status` and confirm the colony is ready to seal.",
			"Run `aether host seal $ARGUMENTS` to fetch the seal manifest via the TS host. Parse `result.seal_manifest`; do not parse visual output.",
			"If runtime reports blockers or recovery guidance, surface that output and stop.",
			"Save the full JSON envelope to a temporary manifest file for later ceremony rendering.",
			"When the seal manifest includes `queen_execution_policy.spawn_budget`, surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"Before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow seal --manifest-file <manifest file>`.",
			"Dispatch Gatekeeper, Auditor, and Probe through the host platform using runtime-provided names, castes, task IDs, briefs, and skill sections.",
			"Before each final-review wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow seal --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"Spawn final-review workers as visible live Task/subagent panels with caste-labelled descriptions; do not use background-only dispatch as the ceremony.",
			"Call `aether spawn-log` before each final-review worker and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("seal"),
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow seal --worker-file <approved temp worker result JSON>`.",
			"Preserve optional worker `findings`, `issues`, `recommendations`, `weak_spots`, `edge_cases_discovered`, and `reusable_lessons` fields in the completion JSON; the finalizer persists them.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether seal-finalize --completion-file <approved temp completion JSON>",
		PostSteps: []string{
			"After the JSON finalizer succeeds, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow seal --completion-file <approved temp completion JSON>`.",
			"Summarize actual final-review workers, blockers if any, and the runtime seal result.",
			"Use `.aether/data/seal/final-review.json` and review ledgers for durable final-review findings; do not rely on chat-only summaries.",
			"Follow the runtime's Porter readiness output, but do not run delivery commands unless the user selects them.",
			"After sealing, run `AETHER_OUTPUT_MODE=visual aether status` first to review the retained sealed state; `aether entomb` is a separate optional owner-confirmed archive-and-clear action.",
			"Never invoke entomb automatically; sealing retains active state for owner review.",
		},
		DriftGuards: intelligentCommandDriftGuards("seal", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration seal, run their literal `aether seal ...` command.",
	}

	catalog["discuss"] = commandGuideDefinition{
		Category:       commandGuideCategorySemiIntelligent,
		SkillReference: commandGuideSkillResearch,
		Intent:         "Use codebase-aware analysis to ask better clarification questions before planning.",
		Literal:        false,
		PreSteps: []string{
			codexGuideSupportStep("discuss", commandGuideSkillResearch),
			"Run `AETHER_OUTPUT_MODE=json aether discuss-analyze --target .` for suggested codebase-aware questions.",
			"Present a compact set of questions covering architecture, dependencies, testing, deployment, performance, and user intent where relevant.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether discuss $ARGUMENTS",
		PostSteps: []string{
			"Persist answers with `aether discuss --resolve <id> --answer \"<answer>\"` when runtime supplies IDs.",
			"If discussion_status is settled, route back to `$ant-plan`.",
		},
		DriftGuards: intelligentCommandDriftGuards("discuss", commandGuideSkillResearch),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration discuss, run their literal `aether discuss ...` command.",
	}

	return catalog
}

func commandGuideLiteralCommands() []string {
	return []string{
		"archaeology",
		"ask",
		"assumptions",
		"bump-version",
		"chaos",
		"council",
		"data-clean",
		"dream",
		"entomb",
		"export-signals",
		"feedback",
		"flag",
		"flags",
		"focus",
		"help",
		"history",
		"import-signals",
		"improve",
		"insert-phase",
		"interpret",
		"lay-eggs",
		"maturity",
		"maintenance",
		"medic",
		"memory-details",
		"migrate-state",
		"organize",
		"patrol",
		"pause",
		"phase",
		"pheromones",
		"porter",
		"preferences",
		"profile",
		"queen-compose",
		"quick",
		"redirect",
		"reference-index",
		"reference-list",
		"reference-match",
		"resume",
		"run",
		"shelf",
		"shelf-add",
		"shelf-dismiss",
		"shelf-list",
		"shelf-promote",
		"skill-create",
		"status",
		"tunnels",
		"unblock",
		"update",
		"verify-castes",
		"watch",
	}
}

// The support path is relative to the selected public skill, never the repo.
// SkillReference remains the stable internal identity used by YAML metadata.
func codexGuideSupportStep(command, support string) string {
	return fmt.Sprintf("Use `$ant-%s` in Codex. Read `../support/%s.md`, resolved relative to the installed ant-%s/SKILL.md, not the working directory; this is private support, not a separately discoverable helper skill.", command, support, command)
}

func intelligentCommandDriftGuards(command, skill string) []string {
	return []string{
		fmt.Sprintf("When changing `%s` wrapper intelligence, update `.aether/commands/%s.yaml`, Claude/OpenCode wrappers, `.aether/skills/colony/%s/SKILL.md` (source for installed `../support/%s.md`), and `command-guide` together.", command, command, skill, skill),
		"Runtime owns state mutation; wrappers and Codex skills may interview, synthesize, spawn, and summarize, but must not hand-edit state files.",
		"Choose exactly one worker launch owner per run: platform-native Task/subagent panels after a dry-run manifest, or Go-adapter subprocess execution through the TS host/direct runtime. Never dispatch both paths for the same manifest.",
		"Treat AETHER_WORKER_PLATFORM as a hard provider pin. If that provider is unavailable, stop with the Go-owned diagnostic; never fall back to another provider.",
		"Keep YAML `codex_orchestration` metadata aligned with this guide; command-guide tests enforce that contract.",
	}
}
