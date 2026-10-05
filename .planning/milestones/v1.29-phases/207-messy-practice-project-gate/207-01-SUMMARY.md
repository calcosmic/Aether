---
phase: 207-messy-practice-project-gate
plan: 01
subsystem: testing
tags: [claude-code-headless, eval-gates, go-test, release-gate, shell-harness]

# Dependency graph
requires:
  - phase: 206-screens-reach-the-owner
    provides: "scripts/proof-screens-reach-the-owner.sh -- the isolation pattern (AETHER_HUB_DIR only, HOME untouched), the claude -p invocation shape, the transient-failure retry-once policy, and the --max-budget-usd availability probe, all extended (not reinvented) by this plan"
provides:
  - "scripts/build-messy-practice-project.sh -- a committed, idempotent script that builds a real Aether-initialised practice project and seeds it with the leftover-junk-data trap (UED-07's first trap; 208 more traps expand this in 207-02)"
  - "cmd/journey.go -- the transcript/report library (journeyStepVocabulary, journeyMenuCommandNames, journeyBashToolCallCount, classifyJourneyFailure, journeyGateVerdict) every later plan in this phase extends, never replaces"
  - "cmd/journey_traps.go -- the shared trap manifest reader both the builder script (via jq) and Go read, so they can never disagree about which traps exist"
  - "cmd/journey_live_test.go -- TestJourney, the //go:build journey live harness, proven end-to-end against a real claude -p chat for the status step"
  - "the journey eval gate (cmd/eval_gates.go, cmd/testdata/eval-gates/gates.json, make eval-gate-journey) -- the eighth named release gate"
affects: [207-02-messy-practice-project-gate, 207-03-messy-practice-project-gate, 207-04-messy-practice-project-gate, 207-05-messy-practice-project-gate, 207-06-messy-practice-project-gate]

# Actuals (#2632)
actuals:
  tokens: 15400
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "on-disk session transcript for slash-command proof: Claude Code's --output-format stream-json stdout never echoes the outgoing user turn, so the <command-message>/<command-name> tags a typed menu command expands into only ever exist in the real, persisted session transcript at ~/.claude/projects/<encoded-cwd>/<session-id>.jsonl -- discovered empirically this plan, not documented anywhere in the official headless docs"
    - "closed-vocabulary convention reused a third time: journeyStep/journeyFailureKind follow the exact const-block/names-slice/names()/declared-membership-predicate shape evalGateName already established"
    - "one gate verdict function forever: journeyGateVerdict is the single place a journey report is judged pass/fail; 207-04 adds rules to this same function rather than creating a second one"

key-files:
  created:
    - scripts/build-messy-practice-project.sh
    - cmd/journey.go
    - cmd/journey_traps.go
    - cmd/journey_live_test.go
    - cmd/journey_test.go
    - cmd/testdata/journey/traps.json
  modified:
    - cmd/eval_gates.go
    - cmd/testdata/eval-gates/gates.json
    - Makefile

key-decisions:
  - "Assertions run against Claude Code's own on-disk session transcript (~/.claude/projects/<encoded-cwd>/<session-id>.jsonl), not the captured --output-format stream-json stdout -- the stdout stream never carries the <command-name> tag; only the persisted transcript does. Located by filename (<session-id>.jsonl, a UUID) via a bounded filepath.WalkDir under ~/.claude/projects, deliberately avoiding a reimplementation of Claude Code's own undocumented cwd-to-directory-name encoding."
  - "--allowedTools Bash is added alongside --permission-mode acceptEdits on both claude -p invocations after acceptEdits alone was observed auto-rejecting the Bash tool call with 'This command requires approval' on a fresh session/cwd; the plan text named acceptEdits only, this is a Rule 1 auto-fix (the step could not otherwise run at all)."
  - "The session-establishing (warm-up) call's own --max-budget-usd is set to $2.00, not the driven step's $1.00: the FIRST call in a session pays the one-time cost of building this repository's own large CLAUDE.md/hooks/skills prompt cache from cold, observed empirically at ~$0.49 for a single-word reply; every later --resume'd call reads from that cache instead. Provisional per 207-RESEARCH.md Assumption A2."
  - "evalGateRequiresClaudeCLICredentials is a distinct requirement value from evalGateRequiresProviderCredentials (207-RESEARCH.md Assumption A3): provider_credentials means an LLM worker under test has API keys, claude_cli_credentials means the developer's own machine has an interactively signed-in claude CLI -- different failure modes that must not be conflated in a CI/local decision."

requirements-completed: [UED-07, UED-08]

coverage:
  - id: D1
    description: "A committed script builds a real, Aether-initialised practice project from nothing and is safely re-runnable against its own output, leaving the seeded trap in place"
    requirement: "UED-07"
    verification:
      - kind: integration
        ref: "scripts/build-messy-practice-project.sh run twice against the same destination, then jq -e '.entries | length >= 2' on the resulting midden.json"
        status: pass
      - kind: integration
        ref: "scripts/build-messy-practice-project.sh pointed at a non-empty, unmarked destination -- refuses by name, exit 1"
        status: pass
      - kind: integration
        ref: "scripts/build-messy-practice-project.sh with an unbuildable declared trap id -- refuses by name, exit 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "One lifecycle menu command (/ant-status) runs through a real claude -p chat with hooks and menu commands loaded, and the harness proves it from the transcript and disk state, never from wording"
    requirement: "UED-08"
    verification:
      - kind: integration
        ref: "go test -tags journey -run TestJourney/status ./cmd -- real claude -p chat, asserts <command-name>/ant-status</command-name> and >=1 Bash tool call from the on-disk session transcript, plus COLONY_STATE.json parses"
        status: pass
    human_judgment: false
  - id: D3
    description: "The journey report is versioned JSON with declared/executed step counts, and journeyGateVerdict refuses a partial (one-step) run rather than accepting it"
    requirement: "UED-08"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyGateRefusesAPartialRun"
        status: pass
      - kind: integration
        ref: "TestJourney/status itself writes a one-step report and asserts journeyGateVerdict refuses it, naming the scope"
        status: pass
    human_judgment: false
  - id: D4
    description: "make eval-gate-journey exists, is declared in the committed gate manifest alongside the other seven gates, and sets no skip-enabling environment variable"
    requirement: "UED-08"
    verification:
      - kind: unit
        ref: "cmd/eval_gates_test.go#TestEvalGateVocabularyMatchesTheManifest, #TestEvalGateSelectionMatchesRealTests/journey, #TestEvalGateBudgetsArePositive, #TestEvalGateManifestLoadIsStable"
        status: pass
      - kind: unit
        ref: "cmd/eval_gates_test.go#TestDefaultSuiteDiscoveredCountIsUnchangedByTags"
        status: pass
    human_judgment: false
  - id: D5
    description: "The transcript parser is proven against a genuinely captured Claude Code session, not a hand-typed fixture, and the test can fail"
    requirement: "UED-08"
    verification:
      - kind: unit
        ref: "cmd/journey_test.go#TestJourneyTranscriptReadsRealCaptures (against cmd/testdata/stop-hook/menu-command-transcript.jsonl)"
        status: pass
    human_judgment: false

duration: 65min
completed: 2026-09-22
status: complete
---

# Phase 207 Plan 01: One Lifecycle Step, End to End Summary

**A committed script builds a real, trap-seeded Aether practice project; a real `claude -p` chat drives `/ant-status` inside it; the harness proves it from Claude Code's own on-disk session transcript and `COLONY_STATE.json`, never from prose; and `make eval-gate-journey` is now the eighth named release gate.**

## Performance

- **Duration:** 65 min
- **Tasks:** 3
- **Files created:** 6
- **Files modified:** 3

## Accomplishments

- `scripts/build-messy-practice-project.sh`: a committed, idempotent builder that constructs a real practice project (`git init`, an initial commit, real `aether init` + `aether update --force`) and applies every trap declared in `cmd/testdata/journey/traps.json` by id, seeding `.aether/data/midden.json` with two unacknowledged entries for the `leftover-junk-data` trap. Refuses by name a non-empty, unmarked destination or an undeclared trap id it doesn't know how to build.
- `cmd/journey.go`: the transcript/report library -- a fourteen-step closed vocabulary (`journeyStepVocabulary`) mapped to menu commands, `journeyMenuCommandNames`/`journeyBashToolCallCount` (JSONL transcript readers), `classifyJourneyFailure` (the small transient-failure regex), the versioned `journeyReport` struct, and `journeyGateVerdict` -- the one function that will ever decide whether a journey report passes the gate.
- `cmd/journey_live_test.go` (`//go:build journey`): `TestJourney/status` drives `/ant-status` through a real, budget-capped `claude -p` chat inside a script-built practice project and asserts from Claude Code's own on-disk session transcript (`~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`) that the menu command and a Bash tool call really happened, then from `COLONY_STATE.json` on disk. Writes a one-step report and asserts `journeyGateVerdict` refuses it by name.
- `journey` registered as the eighth named eval gate: `evalGateJourney` + the distinct `claude_cli_credentials` requirement in `cmd/eval_gates.go`, a manifest entry in `cmd/testdata/eval-gates/gates.json` with an explicitly provisional budget, and `make eval-gate-journey` in the `Makefile`.
- `cmd/journey_test.go`: five offline tests proving the transcript parser (against the real captured fixture), the failure classifier, the report round trip, and the gate verdict's refusal logic -- no chat, no money.

## Task Commits

1. **Task 1: One lifecycle step, end to end** - `4666da47` (feat)
2. **Task 2: Register journey as the eighth named gate** - `d697de00` (feat)
3. **Task 3: Offline tests for the transcript, classifier and report library** - `36c36301` (test)

_No separate plan-metadata commit yet -- STATE.md/ROADMAP.md/REQUIREMENTS.md are updated and committed after this file is written, per the executor's atomic close-out order._

## Files Created/Modified

- `scripts/build-messy-practice-project.sh` - builds and seeds the practice project; idempotent, refuses unknown destinations/traps by name
- `cmd/journey.go` - transcript reading, failure classification, report writer/reader, gate verdict
- `cmd/journey_traps.go` - shared trap manifest reader (script + Go)
- `cmd/journey_live_test.go` - the live `TestJourney` harness (`//go:build journey`)
- `cmd/journey_test.go` - offline unit tests for the library in `cmd/journey.go`
- `cmd/testdata/journey/traps.json` - the one declared trap list (`leftover-junk-data` for this plan)
- `cmd/eval_gates.go` - `evalGateJourney`, `evalGateRequiresClaudeCLICredentials`, header comment count corrected to eight
- `cmd/testdata/eval-gates/gates.json` - the `journey` gate entry
- `Makefile` - `eval-gate-journey` target and `.PHONY` entry

## Decisions Made

- Assertions read Claude Code's own **on-disk session transcript**, not the `--output-format stream-json` stdout -- discovered by direct experiment this session (see key-decisions above for the full rationale and the polling/lookup mechanism used).
- Added `--allowedTools Bash` alongside `--permission-mode acceptEdits` (Rule 1 auto-fix: `acceptEdits` alone auto-rejected the Bash call with "This command requires approval" on a fresh session).
- Warm-up call's own budget cap raised to $2.00 (session-establishing calls pay the cold prompt-cache cost; the driven step's cap stays at the plan's specified $1.00).
- `claude_cli_credentials` kept distinct from `provider_credentials` in the eval-gate requirement vocabulary, per 207-RESEARCH.md Assumption A3.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `--permission-mode acceptEdits` alone did not approve the Bash tool call**
- **Found during:** Task 1, first live run of `TestJourney/status`
- **Issue:** The status step's Bash tool call was auto-rejected ("This command requires approval", `non_execution_kind: user-rejected"`) with only `--permission-mode acceptEdits` set, so the chat never ran `aether status` at all.
- **Fix:** Added `--allowedTools Bash` to both `claude -p` invocations (the session-establishing call and the step-driving call).
- **Files modified:** cmd/journey_live_test.go
- **Verification:** `TestJourney/status` then passed, transcript shows the Bash tool call succeeded.
- **Committed in:** 4666da47 (Task 1 commit)

**2. [Rule 1 - Bug] Session-establishing call's $0.25 budget cap was insufficient**
- **Found during:** Task 1, first live run
- **Issue:** The warm-up call failed with `terminal_reason: budget_exhausted` / `"Reached maximum budget ($0.25)"` -- a single-word reply on this machine cost $0.49, almost entirely cache-creation tokens for this repository's own large CLAUDE.md/hooks/skills system prompt.
- **Fix:** Raised the warm-up call's `--max-budget-usd` to $2.00, documented as deliberately more generous than the driven step's own $1.00 cap.
- **Files modified:** cmd/journey_live_test.go
- **Verification:** Warm-up call then completed with `is_error: false` and a real `session_id`.
- **Committed in:** 4666da47 (Task 1 commit)

**3. [Rule 1 - Bug] `<command-name>` tag never appears in the streamed stdout**
- **Found during:** Task 1, second live run (after fixes 1-2)
- **Issue:** `TestJourney/status` failed its own `<command-name>` assertion even though the chat visibly ran `/ant-status` correctly (confirmed by reading the assistant's own status-card reply in the stream). Direct inspection showed `--output-format stream-json` stdout contains `system`/`assistant`/`user`(tool_result) events but never echoes the outgoing user turn itself -- so the `<command-message>`/`<command-name>` tags a typed slash command expands into (the exact shape in `cmd/testdata/stop-hook/menu-command-transcript.jsonl`) simply do not exist in that stream. They exist only in Claude Code's own persisted, on-disk session transcript at `~/.claude/projects/<encoded-cwd>/<session-id>.jsonl`.
- **Fix:** Added `journeyFindSessionTranscript` (locates `<session-id>.jsonl` by filename under `~/.claude/projects/`, polling up to 10s for the write to land) and pointed both the `<command-name>` and Bash-tool-call assertions at that file instead of the captured stdout. The stdout is still captured (to a separate `streamOutPath`) for debugging, but is no longer the assertion source.
- **Files modified:** cmd/journey_live_test.go
- **Verification:** `TestJourney/status` passed end-to-end: transcript carries 1 Bash tool call and command names `[/ant-status]`.
- **Committed in:** 4666da47 (Task 1 commit)

**4. [Rule 2 - Missing Critical] `journeyContainsString` was only defined behind the `journey` build tag**
- **Found during:** Task 3, writing the default-build offline tests
- **Issue:** `cmd/journey_test.go` (default build, no build tag) needed the same membership check `cmd/journey_live_test.go` (`//go:build journey`) already defined -- the default build would not compile.
- **Fix:** Moved `journeyContainsString` into `cmd/journey.go` (default build, shared by both files).
- **Files modified:** cmd/journey.go, cmd/journey_live_test.go
- **Verification:** `go build ./cmd/aether` and `go vet ./cmd/...` clean on both the default and `-tags journey` builds.
- **Committed in:** 4666da47 (Task 1 commit, since journey.go/journey_live_test.go were part of that task's file set)

---

**Total deviations:** 4 auto-fixed (3 Rule 1 bugs discovered only by actually running the live chat, 1 Rule 2 missing-critical build fix)
**Impact on plan:** All four were necessary for the tracer to actually pass against a real chat, exactly the kind of "architectural dead end costs one commit instead of ten" discovery this plan's own objective names as its purpose. No scope creep -- deviation 3 in particular changes which artifact later plans in this phase must read (the on-disk transcript, not the captured stdout), which is now documented in `cmd/journey_live_test.go`'s own comments for 207-04 to build on.

## Issues Encountered

None beyond the deviations above, all resolved during Task 1's live-run iteration before moving to Task 2/3.

## User Setup Required

None - no external service configuration required. Requires a locally signed-in `claude` CLI, already present on this machine (verified via `claude --version` -> `2.1.278`).

## Next Phase Readiness

- The tracer slice (`journey.go`, `journey_traps.go`, `journey_live_test.go`, `journey_test.go`, the builder script, the `journey` eval gate) is proven end-to-end against a real chat and is ready for 207-02 to expand: more traps in `cmd/testdata/journey/traps.json` (8 more, per UED-07's full roadmap list), more steps in `journeyStepVocabulary` driven through the chat, and 207-04's flip to `scope: "whole-chain"` plus the trial-count/flaky-vs-real rules in the same `journeyGateVerdict` function.
- **Load-bearing discovery for 207-02 onward:** any new step or trap assertion must read the on-disk session transcript (`journeyFindSessionTranscript`), never the captured `--output-format stream-json` stdout, for any `<command-name>` or transcript-shape assertion.
- No blockers.

---
*Phase: 207-messy-practice-project-gate*
*Completed: 2026-09-22*

## Self-Check: PASSED

All 9 created/modified files verified present with `[ -f ]`; all 3 task commit hashes (`4666da47`, `d697de00`, `36c36301`) verified present in `git log --oneline --all`.
