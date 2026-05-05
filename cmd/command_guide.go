package cmd

import (
	"fmt"
	"sort"
	"strings"

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
	Command        string   `json:"command"`
	Platform       string   `json:"platform"`
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
	switch platform {
	case "codex", "claude", "opencode":
	default:
		return commandGuideResult{}, fmt.Errorf("unsupported platform %q; expected codex, claude, or opencode", platform)
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
		Command:        command,
		Platform:       platform,
		Category:       def.Category,
		SkillReference: def.SkillReference,
		Intent:         def.Intent,
		Literal:        def.Literal,
		PreSteps:       append([]string(nil), def.PreSteps...),
		RunCommand:     def.RunCommand,
		PostSteps:      append([]string(nil), def.PostSteps...),
		DriftGuards:    append([]string(nil), def.DriftGuards...),
		RawBypass:      def.RawBypass,
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

	catalog["init"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillCreation,
		Intent:         "Refine a rough user goal into a deeper colony charter before runtime state is created.",
		Literal:        false,
		PreSteps: []string{
			"Load the aether-colony-creation Codex skill.",
			"Run `AETHER_OUTPUT_MODE=json aether init-research --goal \"<raw goal>\" --target .` for deterministic codebase context.",
			"Ask one compact batch of 4-7 questions when target users, success criteria, non-goals, constraints, risks, affected systems, or first milestone are unclear.",
			"Synthesize raw goal, user answers, and init-research output into a refined goal and charter JSON; do not echo the scan output as the final charter.",
			"Separate deterministic housekeeping warnings from at most 3 strategic AI-synthesized pheromone suggestions, and ask approval before writing any signal.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether init --charter-json '<synthesized charter JSON>' \"<refined goal>\"",
		PostSteps: []string{
			"Summarize the refined charter, approved strategic pheromones, and next runtime command.",
			"Route to `aether discuss` or `aether plan` based on unresolved clarification risk.",
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
			"For everything/all-of-the-above/full-system audits or large uncommitted diffs, split the topic into focused Oracle runs or start with `--depth quick`; do not collapse every area into one blocking balanced-depth prompt.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether oracle --depth <depth> --confidence-target <percent> --template <template> --background \"<synthesized prompt>\"",
		PostSteps: []string{
			"If the shell/tool call times out, run `aether oracle status` before declaring failure or switching to ad hoc agents.",
			"If OpenCode subprocess dispatch is unavailable, let Oracle use its automatic Codex/Claude fallback unless the user explicitly set `AETHER_WORKER_PLATFORM=opencode`.",
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
			"Select planning depth and decomposition depth with the user unless arguments already specify them.",
			"Run `AETHER_OUTPUT_MODE=visual aether status` for current colony context.",
			"Run `AETHER_OUTPUT_MODE=json aether plan --plan-only --depth <choice> --planning-depth <choice>` and parse `result.plan_manifest` or `result.planning_manifest`.",
			"If runtime returns `dispatch_mode: agent-delegate`, dispatch Scout and Route-Setter through the host platform instead of nested subprocess workers, then finalize with the returned manifest.",
			"If runtime reports unresolved clarifications, route to `aether discuss` before spawning planning workers unless the user explicitly approves assumptions.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether plan-finalize --completion-file <worker completion JSON>",
		PostSteps: []string{
			"Summarize depth, phase count, planning confidence, and actual planning workers.",
			"Route to `aether build 1` or the runtime-surfaced next build command.",
		},
		DriftGuards: intelligentCommandDriftGuards("plan", commandGuideSkillBuildCycle),
		RawBypass:   "If the user explicitly asks for raw/exact/no-orchestration plan, run their literal `aether plan ...` command.",
	}

	catalog["build"] = commandGuideDefinition{
		Category:       commandGuideCategoryFullOrchestration,
		SkillReference: commandGuideSkillBuildCycle,
		Intent:         "Use the runtime dispatch manifest to spawn platform workers and finalize the phase without hand-writing state.",
		Literal:        false,
		PreSteps: []string{
			"Load the aether-colony-build-cycle Codex skill.",
			"Run `AETHER_OUTPUT_MODE=visual aether status` and surface active REDIRECT, FOCUS, and FEEDBACK signals compactly.",
			"Run `AETHER_OUTPUT_MODE=json aether build <phase> --plan-only` and parse `result.dispatch_manifest`; do not parse visual output.",
			"Follow the installed build-wave playbook and use runtime-provided agent names, castes, task IDs, briefs, and skill_section values.",
			"Call `aether spawn-log` before each worker and `aether spawn-complete` after each terminal result.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether build-finalize <phase> --completion-file <worker completion JSON>",
		PostSteps: []string{
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
			"Use the default runtime path unless the user requested `--verification-depth heavy` or runtime asks for wrapper-spawned review workers.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=visual aether continue --skip-watchers --verification-depth standard $ARGUMENTS",
		PostSteps: []string{
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
		Intent:         "Seal from runtime output while preserving blocker, shelf, and Porter delivery guidance.",
		Literal:        false,
		PreSteps: []string{
			"Load the aether-colony-build-cycle Codex skill.",
			"Run `AETHER_OUTPUT_MODE=visual aether status` and confirm the colony is ready to seal.",
			"Surface blockers or shelf candidates from runtime output; do not inspect or edit state files directly.",
		},
		RunCommand: "AETHER_OUTPUT_MODE=json aether seal $ARGUMENTS",
		PostSteps: []string{
			"Summarize seal status, blockers if any, and the runtime recovery command.",
			"Run or recommend Porter delivery readiness only when the runtime seal path completes or asks for it.",
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
		"archaeology",
		"assumptions",
		"bump-version",
		"chaos",
		"colonize",
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
		"pause-colony",
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
		"resume-colony",
		"run",
		"shelf",
		"shelf-add",
		"shelf-dismiss",
		"shelf-list",
		"shelf-promote",
		"skill-create",
		"status",
		"swarm",
		"tunnels",
		"update",
		"verify-castes",
		"watch",
	}
}

func intelligentCommandDriftGuards(command, skill string) []string {
	return []string{
		fmt.Sprintf("When changing `%s` wrapper intelligence, update `.aether/commands/%s.yaml`, Claude/OpenCode wrappers, `%s` Codex skill, and `command-guide` together.", command, command, skill),
		"Runtime owns state mutation; wrappers and Codex skills may interview, synthesize, spawn, and summarize, but must not hand-edit state files.",
		"Keep YAML `codex_orchestration` metadata aligned with this guide; command-guide tests enforce that contract.",
	}
}
