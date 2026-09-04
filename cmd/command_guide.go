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
	if platform == "codex" || def.Literal {
		return def
	}
	adapted := def
	adapted.SkillReference = ""
	if len(adapted.PreSteps) > 0 && strings.HasPrefix(adapted.PreSteps[0], "Load the ") {
		adapted.PreSteps = append([]string(nil), adapted.PreSteps...)
		adapted.PreSteps[0] = fmt.Sprintf("Use the generated %s slash-command wrapper for `%s`; do not load Codex lifecycle skills.", platform, command)
	}
	return adapted
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

	catalog["init"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillCreation,
		Intent:         "Start a guided colony for one goal.",
		Literal:        false,
		PreSteps: []string{
			"Load the aether-colony-creation Codex skill.",
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
			"Render the exact Codex-native closeout `Next Up: aether plan`; do not translate it into Claude/OpenCode or a deferred Codex-native lifecycle surface.",
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
			"Load the aether-colony-research Codex skill.",
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
		Intent:         "Select planning depth, enforce clarification gates, and use runtime manifests for Scout and Route-Setter orchestration.",
		Literal:        false,
		PreSteps: []string{
			"Load the aether-colony-build-cycle Codex skill.",
			"Decision Moment 1 (depth proposal): request a first manifest with `aether host plan`, omitting `--depth`, `--planning-depth`, and `--verification-depth` unless arguments already specify them, then print `result.depth_proposal_card` verbatim — do not restate, summarize, or re-reason the recommendations. Accept on a single confirmation, or request a fresh manifest with the named knob's flag set when the user picks a different option.",
			"Run `AETHER_OUTPUT_MODE=visual aether status` for current colony context.",
			"Inspect every planning dispatch `permission_profile` before spawning it and pass it through verbatim from the manifest; never substitute or broaden it. Scout's canonical profile is `workspace_write`, scoped behaviorally to writing only under `.aether/data/phase-research`.",
			"When revising future work after a completed phase, pass `--refresh --revision-type <type> --revision-reason <why>` to every host-plan iteration. Research and verification revisions also require repository-relative `--revision-evidence <path>` files.",
			"Run `aether host plan --depth <choice> --planning-depth <choice> --verification-depth <choice>` to fetch one planning-iteration manifest via the TS host. Parse `result.plan_manifest` or `result.planning_manifest`.",
			"Save the full JSON envelope to a temporary manifest file for later ceremony rendering.",
			"Treat the manifest's `planning_run_id`, `iteration`, `target_confidence`, `max_iterations`, `previous_confidence`, `selected_gaps`, `previous_plan_draft`, and `expected_workers` as authoritative loop state.",
			"If the manifest includes `revision`, preserve it and the worker briefs verbatim: completed phases are immutable and Route-Setter outputs replacement unfinished phases only; Go assigns final IDs atomically.",
			"When the manifest includes `queen_execution_policy.spawn_budget`, surface selected/pruned caste reasons so users can see why workers were or were not spawned.",
			"Decision Moment 2 (research batch, the second and final decision moment): print `result.research_proposal_card` verbatim when non-empty, approve with `aether plan-research-approve --approve-all` or flip specific phases with `--flip <ids>`, then request a fresh manifest so gated `phase_research` dispatches appear. Surface `result.research_warning` whenever `result.research_awaiting_approval` is true.",
			"Before rendering spawn ceremonies or spawning workers, inspect `result.orchestrator_boundary_guidance` and the matching manifest `orchestrator_boundary_guidance`: if active or `next` is `aether discuss`, stop, show the summary, route to `aether discuss`, tell the user to rerun `after_discuss_next`, and request a fresh plan-only manifest after the answer is resolved.",
			"Render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony spawn-plan --workflow plan --manifest-file <manifest file>`.",
			"If runtime returns `dispatch_mode: agent-delegate`, dispatch Scout and Route-Setter through the host platform instead of nested subprocess workers, then finalize with the returned manifest.",
			"If runtime reports unresolved clarifications, route to `aether discuss` before spawning planning workers unless the user explicitly approves assumptions.",
			"Before each manifest wave, render `AETHER_FORCE_COLOR=1 AETHER_OUTPUT_MODE=visual aether ceremony wave-start --workflow plan --manifest-file <manifest file> --execution-wave <execution_wave>`.",
			"Spawn every manifest dispatch as visible live Task/subagent panels with caste-labelled descriptions: one Scout plus any phase_research Scouts in wave 1, then exactly one Route-Setter in wave 2; do not use background-only dispatch as the ceremony.",
			"Pass each dispatch `brief` verbatim, honor its read budget and no-repeat loop guard, and mark workers blocked rather than manually reconciling read loops as completed.",
			"Include the Scout terminal result in the Route-Setter prompt. If `selected_gaps` or `previous_plan_draft` are present, require fresh evidence or resolved gaps before allowing confidence to rise.",
			"Call `aether spawn-log` before each planning worker and `aether spawn-complete` after each terminal result.",
			finalizerCompletionContractStep("plan"),
			"Build the completion packet with `planning_run_id`, `iteration`, Scout `scout_report`, Route-Setter `phase_plan`, and a compact `source_summary`; never reuse a completion packet for another iteration.",
			"After each terminal result, render `AETHER_OUTPUT_MODE=visual aether ceremony worker-complete --workflow plan --worker-file <approved temp worker result JSON>`.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <approved temp completion JSON>",
		PostSteps: []string{
			"If the JSON finalizer returns `requires_next_iteration: true`, do not render final closeout or claim a completed plan; request a fresh `aether host plan` manifest with the same loop controls and repeat Scout -> Route-Setter -> finalizer.",
			"After the JSON finalizer succeeds with a completed plan, run `AETHER_OUTPUT_MODE=visual aether ceremony closeout --workflow plan --completion-file <approved temp completion JSON>`.",
			"Summarize depth, phase count, planning confidence, `planning_loop.stop_reason`, and actual planning workers.",
			"For a revision, surface the accepted `plan_revision` reason and preserved/superseded/replacement phase IDs, and discard all packets from the parent revision.",
			"Route to `aether build 1` or the runtime-surfaced next build command.",
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
			"Load the aether-colony-build-cycle Codex skill.",
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
			"Route first to `aether plan`.",
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
			"Load the aether-colony-build-cycle Codex skill.",
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
			"Load the aether-colony-build-cycle Codex skill.",
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
			"If phase advanced, summarize verification and route to the next `aether build <phase>`.",
			"If blocked, follow the runtime recovery command first.",
			"If complete, route to `aether seal`.",
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
			"Load the aether-colony-build-cycle Codex skill.",
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
			"Load the aether-colony-research Codex skill.",
			"Run `AETHER_OUTPUT_MODE=json aether discuss-analyze --target .` for suggested codebase-aware questions.",
			"Present a compact set of questions covering architecture, dependencies, testing, deployment, performance, and user intent where relevant.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether discuss $ARGUMENTS",
		PostSteps: []string{
			"Persist answers with `aether discuss --resolve <id> --answer \"<answer>\"` when runtime supplies IDs.",
			"If discussion_status is settled, route back to `aether plan`.",
		},
		DriftGuards: intelligentCommandDriftGuards("discuss", commandGuideSkillResearch),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration discuss, run their literal `aether discuss ...` command.",
	}

	return catalog
}

func commandGuideLiteralCommands() []string {
	return []string{
		"abandon",
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
		"insert-phase",
		"interpret",
		"lay-eggs",
		"maturity",
		"medic",
		"memory-details",
		"migrate-state",
		"organize",
		"patrol",
		"pause",
		"pause-colony",
		"phase",
		"pheromones",
		"porter",
		"preferences",
		"profile",
		"queen-compose",
		"quick",
		"recover",
		"redirect",
		"reference-index",
		"reference-list",
		"reference-match",
		"resume",
		"resume-colony",
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

func intelligentCommandDriftGuards(command, skill string) []string {
	return []string{
		fmt.Sprintf("When changing `%s` wrapper intelligence, update `.aether/commands/%s.yaml`, Claude/OpenCode wrappers, `%s` Codex skill, and `command-guide` together.", command, command, skill),
		"Runtime owns state mutation; wrappers and Codex skills may interview, synthesize, spawn, and summarize, but must not hand-edit state files.",
		"Choose exactly one worker launch owner per run: platform-native Task/subagent panels after a dry-run manifest, or Go-adapter subprocess execution through the TS host/direct runtime. Never dispatch both paths for the same manifest.",
		"Treat AETHER_WORKER_PLATFORM as a hard provider pin. If that provider is unavailable, stop with the Go-owned diagnostic; never fall back to another provider.",
		"Keep YAML `codex_orchestration` metadata aligned with this guide; command-guide tests enforce that contract.",
	}
}
