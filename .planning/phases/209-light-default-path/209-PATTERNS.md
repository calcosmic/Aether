# Phase 209: A Light Default Path - Pattern Map

**Mapped:** 2026-09-24
**Files analyzed:** 8 (derived from D-01/D-02/D-03; no explicit file list in CONTEXT.md)
**Analogs found:** 8 / 8

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `.aether/commands/go.yaml` | config (command source) | request-response | `.aether/commands/improve.yaml` | exact |
| `.claude/commands/ant-go.md` | route (wrapper) | request-response | `.claude/commands/ant-improve.md` | exact |
| `.claude/commands/ant/go.md` | route (wrapper) | request-response | `.claude/commands/ant/improve.md` | exact |
| `.opencode/commands/ant/go.md` | route (wrapper) | request-response | `.opencode/commands/ant/improve.md` | exact |
| `cmd/go_cmd.go` (new `aether go` cobra command + `resolveJobSizeRoute`) | controller / decision function | request-response | `cmd/review_depth.go` (`resolveVerificationDepth`) + `cmd/command_truth.go` (`quickCmd`, `runQuickJob`) | role-match |
| `cmd/root.go` (extend `frontDoorHelpGroups` / `frontDoorRenderedHelpGroups` membership + add advanced-commands gating) | controller (help renderer) | request-response | `cmd/root.go` itself (`configureFrontDoorHelp`, `renderFrontDoorHelp`) | exact |
| `cmd/advanced_commands.go` (new on/off setting: get/set) | config / model (state field + subcommand) | CRUD | `cmd/phase_commit.go` (`phaseCommitsCmd`, `ColonyState.PhaseCommits`) | exact |
| `cmd/go_cmd_test.go` / `cmd/front_door_209_test.go` (cheap in-process route + help tests) | test | request-response | `cmd/front_door_199_test.go`, `cmd/quick_do_test.go` | exact |
| `cmd/journey_live_test.go` extension or new `//go:build journey` case for `/ant-go` | test (paid real-flow) | request-response | `cmd/journey_live_test.go`, `cmd/journey.go`, `cmd/journey_seed.go` | role-match |

## Pattern Assignments

### 1. A new command end-to-end (`/ant-go`)

**Analog:** the most recently added command triplet, `improve` (added same wave as `spec`/`abandon`/`midden-review`, per `git log --diff-filter=A -- .aether/commands/*.yaml`).

**Exact file set, in order, for any new `/ant-*` command:**
1. `.aether/commands/go.yaml` — source of truth: `name`, `description`, `runtime.command`, `wrapper_additions.routing`, `guardrails`.
2. `.claude/commands/ant-go.md` — flat Claude Code command file.
3. `.claude/commands/ant/go.md` — namespaced Claude Code command file (byte-identical body to #2 apart from the managed-header comment).
4. `.opencode/commands/ant/go.md` — OpenCode mirror (byte-identical body).
5. (Codex) register in `codexPublicSkillCommands()` / the nine-skill list in `cmd/root.go` only if `/ant-go` should be one of the nine Codex public entrypoints — CLAUDE.md's "Codex Public Entrypoints" table currently names exactly nine; adding a tenth needs an explicit owner decision, so treat this as optional unless CONTEXT.md requires it (it does not).

**`.aether/commands/improve.yaml` excerpt (full file, 21 lines) to copy structure from:**
```yaml
name: ant-improve
description: "Look at how the colony's own suggestions are doing, or try one by hand"
source_of_truth: "Use the Go `aether` CLI as the source of truth."
runtime:
  command: "AETHER_OUTPUT_MODE=visual aether improve $ARGUMENTS"
wrapper_additions:
  routing: |
    Keep improvement inspection and trial runtime-first:
    1. ...
guardrails:
  - "Do not write colony state files, session files, or shadow/canary files by hand from this command spec."
  - "If docs and runtime disagree, runtime wins."
  - "Show this output to the owner in your own reply, unchanged ..."
  - "Show it in a fenced text block, from the first banner line (the line drawn with `━━`) to the end; leave out any running commentary above that line. After it, add at most two short sentences of your own, and never restate or replace the screen."
```
For `/ant-go`, `runtime.command` becomes `"AETHER_OUTPUT_MODE=visual aether go $ARGUMENTS"`, and `wrapper_additions.routing` should say plainly: pass the sentence straight through, let the runtime decide small vs. big, relay the plain-English "why" line verbatim.

**`.claude/commands/ant-improve.md` header + managed-file marker (lines 1-6) — copy verbatim pattern:**
```markdown
<!-- Aether-managed: runtime spec at .aether/commands/improve.yaml. Synced by aether update. -->
---
name: ant-improve
description: "Look at how the colony's own suggestions are doing, or try one by hand"
---

Use the Go `aether` CLI as the source of truth.
```
The body of `.claude/commands/ant-improve.md` and `.claude/commands/ant/improve.md` are byte-identical (verified: both files carry the identical prose block). `/ant-go`'s two Claude files must be identical to each other the same way.

**Parity test(s) enforcing this — name and use exactly these:**
- `cmd/subcommand_reachability_ratchet_test.go` — the orphan-reachability ratchet (WIRE-01). `callerWrapperCorpora` (lines ~46-50) lists the three wrapper trees a new command's caller evidence must appear in:
  ```go
  var callerWrapperCorpora = []string{
      filepath.Join(".claude", "commands", "ant"),
      filepath.Join(".opencode", "commands", "ant"),
      filepath.Join(".aether", "commands"),
  }
  ```
  A cobra command with no reference in any of these (and not in the shrink-only `testdata/orphan_allowlist_baseline.json`) fails this test by name.
- Search for the flat-vs-namespaced Claude file parity check with `grep -rn "ant-improve.md\|flat.*namespaced\|command_call_audit" cmd/*_test.go` before writing — `cmd/command_call_audit_test.go` (mentioned in `subcommand_reachability_ratchet_test.go`'s own comment as `auditedCorpora`) is the broader flag/call audit; read it before adding `go.yaml` to confirm no additional registration list needs updating (e.g. a YAML command inventory file under `.aether/commands/` that some test enumerates directory contents from).

### 2. The size decision itself (small vs. big routing)

**Analog for the *shape* of an automatic depth/route decision:** `cmd/review_depth.go`, `resolveVerificationDepth` (lines 185-201):
```go
// resolveVerificationDepth determines the 3-level verification depth for a phase.
// Priority: explicit heavy flag -> explicit light flag -> explicit --verification-depth string -> keyword match -> smart default.
func resolveVerificationDepth(phase colony.Phase, totalPhases int, lightFlag, heavyFlag bool, verificationDepthStr string) colony.VerificationDepth {
	if heavyFlag {
		return colony.VerificationDepthHeavy
	}
	if lightFlag {
		return colony.VerificationDepthLight
	}
	if verificationDepthStr != "" {
		return colony.NormalizeVerificationDepth(verificationDepthStr)
	}
	if phaseHasHeavyKeywords(phase.Name) {
		return colony.VerificationDepthHeavy
	}
	return resolveSmartVerificationDepth(phase, totalPhases)
}
```
This is the pattern to copy for a new `resolveJobSizeRoute(job string, ...) jobSizeRoute` function: a small, pure, priority-ordered decision function, in its own file, tested directly (not just through the command).

**Analog for "one worker, no pause" proportionality reasoning (directly reusable, do not duplicate):** `cmd/command_truth.go`, `runQuickJob` (around line 846-851) already implements the "small job" execution path with the exact CAP-029 proportionality comment:
```go
workerName := deterministicAntName("builder", job)
// CAP-029 proportionality: exactly one worker, recorded against this
// request's own attempt, before dispatch -- no check-in pause is ever
// added for a single worker with nothing pending.
attempt.recordDispatch(workerName, "builder", "dispatched")
```
`aether go` for the "small" branch should call `runQuickJob` (or a thin wrapper around it) directly rather than re-implementing single-worker dispatch — this is the one existing implementation of the quick-job route and must not be forked into a second copy.

**Analog for "the plan route" dispatch:** the planning/build entry points already reached by `aether plan` / `aether build` (see `cmd/codex_plan.go`, `cmd/codex_build.go`). `aether go`'s "big" branch should shell out to the same internal entry points those commands use, not reimplement planning.

**Where NOT to build a second decision:** `cmd/next_action.go`'s `resolveNextAction` (line 944) is the *one* place lifecycle "what to run next" text is generated — note its own doc comment: `"It lives HERE, in the resolver file, because this file is the single stated exemption of TestNextActionNeverHardcoded"`. A size router that also decides *and announces* a next command must not duplicate `resolveNextAction`'s job; if `/ant-go`'s plain-English "why" line needs to name a follow-up command, route the wording through `resolveNextAction`/the shared next-action machinery rather than hardcoding a new string literal — `cmd/next_action_hardcode_ratchet_test.go` is the structural ratchet that will catch a new hardcoded command string outside `cmd/next_action.go`.

**Structural "no second copy of one decision" tests to search before writing (pattern reference, not exhaustive — grep for the closest fit and follow its shape):**
- `cmd/status_line_test.go`, `TestStatusLineComesFromTheSharedDecision` (lines 68-73+): half of the test drives `resolveNextAction` over a fixture and checks the rendered surface carries exactly what that decision produced; half two is an AST guard refusing a command literal written directly in the rendering file. This is the exact template for a `TestGoRouteComesFromOneSizeDecision`-style test: one half runs `resolveJobSizeRoute` directly and diffs it against what the `go` command actually dispatches; the other half is an AST guard refusing route logic duplicated inside `cmd/go_cmd.go` itself.
- `cmd/recruitment_admission_test.go` / `TestOneAdmissionAuthority` pattern (named in CLAUDE.md's Biological Runtime section) — same "one gate, everyone must call it" test shape.

### 3. The help screen and its tiering (D-02)

**Analog:** `cmd/root.go`.

**Current group data + membership (copy this shape for the six-command default set):**
```go
var (
	frontDoorHelpOnce        sync.Once
	frontDoorDefaultHelpFunc func(*cobra.Command, []string)
	frontDoorHelpGroups      = []frontDoorHelpGroup{
		{
			id: frontDoorNormalGroupID, title: "Normal journey",
			entries: []frontDoorHelpEntry{
				{`/ant-init "goal"`, "Start a guided colony for one goal."},
				{"/ant-plan", "Turn the accepted goal and territory evidence into an executable phase plan."},
				...
			},
		},
		...
	}
)
```
`configureFrontDoorHelp` (line 441) assigns each registered cobra command to a `GroupID` via a `membership` map keyed by cobra command name — this is the mechanism to extend: add `"go": frontDoorNormalGroupID` and remove/relabel entries per the proposed six (`/ant-go`, `/ant-status`, `/ant-continue`, `/ant-flags`/`flag-list`, `/ant-resume`, `/ant-seal`) into a *new* smallest group, while every other existing entry moves to a group gated behind the new advanced-commands setting rather than being deleted from `frontDoorHelpGroups` (CLAUDE.md: "Hiding a command must never disable it").

`renderFrontDoorHelp` (line 524) is the actual render function — extend its group iteration to skip non-default groups unless the advanced-commands setting (see #3b) is on, and always render an "everything else" hint line pointing at the setting, mirroring the existing closing line:
```go
lines = append(lines, "", fmt.Sprintf("Use %s <command> for expert detail outside this journey map.", helpCommand))
```

**Locking test to extend, not duplicate:** `cmd/front_door_199_test.go`, `TestFrontDoorHelpGroups` (lines 17-56) — asserts group order (`"Normal journey", "Steer and inspect", "Expert maintenance"`), exact copy for specific commands via `wants`, and a forbidden-substring list for hidden plumbing:
```go
groups := []string{"Normal journey", "Steer and inspect", "Expert maintenance"}
...
wants := []string{
    `/ant-init "goal" Start a guided colony for one goal.`,
    ...
}
for _, forbidden := range []string{"pause-colony", "resume-colony", "/ant-recover", "/ant-abandon", "lay-eggs", "finalize", "protocol"} {
    if strings.Contains(strings.ToLower(got), strings.ToLower(forbidden)) {
        t.Errorf("ordinary help exposes hidden plumbing %q", forbidden)
    }
}
```
A new `cmd/front_door_209_test.go` should assert the *default* rendering shows exactly the owner-approved six (once the mandatory D-02 checkpoint screen is signed off) and that every non-default command is still reachable and still runs when typed directly (never disabled) — reuse `frontDoorHelpOutput199(t, root, width)` test harness helper from this same file rather than building a new help-capture harness.

### 3b. The advanced-commands on/off setting

**Analog:** `cmd/phase_commit.go` — `ColonyState.PhaseCommits` field + `phase-commits get|set` subcommand pair (lines 264-335):
```go
var phaseCommitsCmd = &cobra.Command{
	Use:   "phase-commits",
	Short: "Get or set the per-phase git save-point behaviour",
	Args:  cobra.NoArgs,
}

var phaseCommitsGetCmd = &cobra.Command{
	Use:   "get",
	...
	RunE: func(cmd *cobra.Command, args []string) error {
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputOK(map[string]interface{}{"mode": "on", "source": "default"})
			return nil
		}
		mode := string(state.PhaseCommits)
		source := "state"
		if mode == "" {
			mode = "on"
			source = "default"
		}
		outputOK(map[string]interface{}{"mode": mode, "source": source})
		return nil
	},
}

var phaseCommitsSetCmd = &cobra.Command{
	Use:   "set <on|off>",
	...
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := colony.PhaseCommitMode(raw)
		if !mode.Valid() {
			outputError(1, fmt.Sprintf("invalid phase-commits mode %q: must be on or off", string(mode)), nil)
			return nil
		}
		var state colony.ColonyState
		if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
			outputError(1, "COLONY_STATE.json not found", nil)
			return nil
		}
		state.PhaseCommits = mode
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			outputError(2, fmt.Sprintf("failed to save state: %v", err), nil)
			return nil
		}
		outputOK(map[string]interface{}{"mode": string(mode), "source": "cli"})
		return nil
	},
}

func init() {
	phaseCommitsSetCmd.Flags().String("mode", "", "on or off")
	phaseCommitsCmd.AddCommand(phaseCommitsGetCmd)
	phaseCommitsCmd.AddCommand(phaseCommitsSetCmd)
	rootCmd.AddCommand(phaseCommitsCmd)
}
```
Copy this exact shape for something like `aether advanced-commands get|set <on|off>` (or fold it into an existing preferences store if the planner decides a colony-scoped state field is wrong for something that should arguably be user/global-scoped — `/ant-preferences`'s hub `QUEEN.md` section is the alternate analog if this setting should be global rather than per-project; flag this choice for the planner, CONTEXT.md does not specify scope).

### 4. The escalation notice (D-03)

**Analog for where a one-line plain-English runtime note already reaches the owner without asking:** `cmd/review_depth.go`, the keyword-escalation note (lines 154-158) — this is the closest existing "the program silently escalated and said so" pattern:
```go
if phaseHasHeavyKeywords(phase.Name) {
    fmt.Fprintf(os.Stderr, "note: phase %q matched a security/release keyword; review depth escalated to heavy (pass --light to override)\n", phase.Name)
    return ReviewDepthHeavy
}
```
This precedent (escalate, print one plain note, continue — never ask, never block) is exactly D-03's shape and should be followed structurally, but the actual notice text must go through the visual output writer, not raw `fmt.Fprintf(os.Stderr, ...)`, because this command's output already goes through `outputWorkflow`/`outputOK` (see `quickCmd`'s `outputWorkflow(result, renderQuickVisual(result))`). Add the escalation sentence as a field on the `go` command's result map and render it as the first line of `renderGoVisual`, following the same convention `runQuickJob`'s existing "if it turns out to be bigger... stop and say so plainly" task-brief instruction (`cmd/command_truth.go` lines ~855-856) already gives the *worker*; the *runtime's own* notice (as opposed to the worker's self-report) belongs in the command result, not the task brief.

**Plain-English / voice-corpus constraints to satisfy (do not skip):** CLAUDE.md's "Classic Visual Voice (v1.28)" section — any new rendered screen line must carry a leading symbol from the one shared glyph table (`TestVoiceGlyphsHaveOneTable`), must not print raw internal state tokens (`TestVoicedScreensCarryNoRawStateToken`), and must explain any invented term the first time it appears on that screen (`TestVoicedScreensSpeakPlainEnglish`). If `/ant-go`'s screen is added to the eleven-screen "voice corpus," it must be registered there too (`TestEveryOrdinaryScreenIsMeasuredForVoice`) — check `cmd/classic_coverage_ratchet_test.go` (present in this repo) for the corpus registration list before finalizing the visual renderer.

**Translation layer for `/ant-…` command names in the notice:** `cmd/wrapper_command_names.go` and `frontDoorCommandForPlatform` (`cmd/root.go` line ~511) are the one place a bare `aether <cmd>` becomes `/ant-<cmd>` per platform — if the escalation notice needs to tell the owner "this became a plan," route the command name string through this existing translator, never hand-format `/ant-plan` directly inside the notice-building code (this is the same discipline the status line uses, see `TestStatusLineComesFromTheSharedDecision` above).

### 5. Real-flow / journey tests (UED-16, UED-17)

**Two distinct harnesses exist — use the cheap one for CI-run route proof, the expensive one only for the milestone-level real-chat walk:**

- **Cheap, fast, CI-safe, in-process:** `cmd/front_door_199_test.go` (`frontDoorHelpOutput199` helper: builds a temp root, writes `COLONY_STATE.json`/`spawn-tree.txt` fixtures directly, calls the render function in-process, asserts on returned string) and `cmd/quick_do_test.go` (`TestQuickSendsOneBuilderWithMemory`, `TestQuickVerdictFollowsTheProjectsOwnChecks`, etc. — uses a `quickJobCaptureInvoker` fake to intercept `codex.WorkerConfig` without spawning a real subprocess or real Claude session). **This is the right analog for `/ant-go`'s route-decision proof**: a `TestGoRouteChoosesQuickForATypoFix` / `TestGoRouteEscalatesWhenTheJobProvesBigger` pair, built the same way — real function calls, fake invoker, assert on the captured dispatch and on the returned result map's escalation-notice field. No `claude -p`, no cost, runs in the default `go test ./...` lane.
- **Expensive, paid, real-chat, gated:** `cmd/journey_live_test.go` (build-tagged `//go:build journey`), driven by `cmd/journey.go` (`journeyReportSchemaVersion`, `journeyDefaultReportRelativePath = ".aether/data/worker-debug/journey-report.json"`) and seeded via `cmd/journey_seed.go`/`cmd/journey_traps.go`. This is `make eval-gate-journey`'s real fourteen-step walk with a real `claude -p` invocation. **UED-17's "real run in a real chat" requirement, if it demands proof beyond the fast in-process tests, is satisfied by adding `/ant-go` as a step (or sub-step) inside this existing fourteen-step sequence** rather than building a fifteenth separate harness — check `cmd/journey_seed.go` for how a new step is declared into the existing const/name-list/completeness-predicate convention (the same "const block, names slice, names() helper, declared-membership predicate" pattern noted at the top of `cmd/journey.go` for `evalGateName`).

**Cost/format warning (from CLAUDE.md):** the paid journey walk costs real money and ~18 minutes and never runs in CI (`make eval-gate-journey`); do not wire `/ant-go` proof into that lane unless the phase's own success criteria genuinely require it. UED-16/17 as summarized in the phase context emphasize "the proof is a real run in a real chat" as a milestone-level standing constraint, not necessarily a new automated gate — flag this to the planner as a scope decision (build a small manual real-chat check the owner runs once, vs. wiring a new automated journey step).

## Shared Patterns

### Never refuse, only decide-and-continue
Source: `cmd/review_depth.go`'s keyword-escalation branch and `cmd/command_truth.go`'s quick-job task brief instruction ("stop and say so plainly ... instead of doing a large amount of work"). Apply to: `go_cmd.go`'s size router and its escalation path. Both D-01 and D-03 explicitly forbid adding a new refusal (CLAUDE.md "no new strict rules" + phase's own standing constraint) — every existing "decide, announce, continue" pattern in this repo already avoids blocking; do not introduce a confirmation prompt.

### One decision, never a second copy
Source: `cmd/next_action.go` (`resolveNextAction`, with `TestNextActionNeverHardcoded` / `next_action_hardcode_ratchet_test.go`) and `cmd/status_line_test.go` (`TestStatusLineComesFromTheSharedDecision`). Apply to: the size-route decision function and the help-screen next-step wording — both must reach through the one existing "what should the owner run next" authority rather than hand-typing a second `/ant-...` string.

### Command triplet parity
Source: `.aether/commands/improve.yaml` + `.claude/commands/ant-improve.md` + `.claude/commands/ant/improve.md` + `.opencode/commands/ant/improve.md`, enforced by `cmd/subcommand_reachability_ratchet_test.go`'s `callerWrapperCorpora`. Apply to: every new file in the `/ant-go` command triplet — write all four together, keep the two Claude bodies byte-identical, and never add the cobra command without adding wrapper caller evidence in the same change.

### Hiding never disables
Source: the existing "Expert maintenance" group in `frontDoorHelpGroups` (e.g. `/ant-maintenance`) already demonstrates a command that is deliberately kept out of the two more prominent help groups while remaining fully invokable — `TestFrontDoorHelpGroups`'s forbidden-substring list proves *hidden plumbing* is excluded, but registered, less-prominent commands like `/ant-maintenance` still render and still work. Apply to: every command demoted off the default six under D-02 — it moves group/visibility, its cobra registration and `RunE` are untouched.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| The owner checkpoint screen itself (D-02's mandatory rendered-menu sign-off) | n/a — a manual review artifact, not a source file | n/a | This is a plan-time gate (show the owner the real `aether help` output and get a ruling), not a code file; no codebase analog applies. The planner should treat it as a required manual step in one of the plans, using `frontDoorHelpOutput199`-style rendering to produce the actual screen bytes for the owner to review. |
| Timing harness for ROADMAP success criterion 4 (same jobs timed against plain Claude) | test / measurement script | batch | No existing "time this against a baseline" harness was found in `cmd/`; the closest precedent is the measured-cost reporting done manually for Phase 207's journey run (`.planning/phases/207-messy-practice-project-gate/207-JOURNEY-RUN.md`, plain prose with read timestamps), not an automated test. Treat as a manual measurement step, not a new Go test. |

## Metadata

**Analog search scope:** `cmd/*.go`, `cmd/*_test.go`, `.aether/commands/`, `.claude/commands/`, `.opencode/commands/`
**Files scanned:** ~40 (targeted greps + 6 full/partial reads: `cmd/root.go`, `cmd/phase_commit.go`, `cmd/review_depth.go`, `cmd/command_truth.go`, `cmd/next_action.go`, `cmd/journey.go`, `cmd/front_door_199_test.go`, `cmd/subcommand_reachability_ratchet_test.go`, `.aether/commands/improve.yaml`, `.claude/commands/ant-improve.md`, `.claude/commands/ant/improve.md`)
**Pattern extraction date:** 2026-09-24
