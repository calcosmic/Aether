# Research: reliability techniques for the "Daily driver" milestone (2026-09-21)

All repos verified via `gh api repos/<r>` on 2026-09-21 (stars / last push). Every claim cites a file read today.

Stars: OpenHands/OpenHands 88.7k; Aider-AI/aider 49.1k (last push 2026-05); SWE-agent/SWE-agent 20.4k; cline/cline 69.0k;
anthropics/claude-code 147k; anthropics/claude-code-action 8.9k; anthropics/claude-plugins-official 36.6k; aaif-goose/goose (ex block/goose) 54.5k;
openai/codex 125.7k; gptme/gptme 4.4k; promptfoo/promptfoo 25.3k; obra/superpowers 289.7k; jj-vcs/jj 31.7k; cli/cli 46.4k;
hashicorp/terraform 49.7k; Homebrew/brew 49.7k; pre-commit/pre-commit 15.6k; restic/restic 36.2k; syncthing/syncthing 88.8k; git/git (mirror).
Not used (failed the bar or stale): plandex (last push 2025-10), envinfo (repo not found under the guessed owner).

## 1. End-to-end / agent evals in CI

### 1a. Real `claude -p` sessions as tests — obra/superpowers
- `tests/claude-code/test-helpers.sh`: `run_claude()` wraps `timeout N claude -p "$prompt" [--allowed-tools=...]`; assertions are
  plain grep helpers: `assert_contains`, `assert_not_contains`, `assert_count`, `assert_order` (A appears before B). Matching is
  deliberately case-insensitive "because models freely capitalize".
- `tests/claude-code/test-subagent-driven-development-integration.sh`: builds a throwaway project, runs
  `cd "$TEST_PROJECT" && timeout 1800 claude -p "$PROMPT" --plugin-dir ... --permission-mode bypassPermissions | tee out`,
  then finds the SESSION JSONL (`ls -t "$SESSION_DIR"/*.jsonl | head -1` — they cd into the temp project precisely so the
  transcript lands in its own folder and is not confused with concurrent sessions) and asserts on TOOL CALLS in the transcript
  (`grep '"name":"Skill".*"skill":"superpowers:..."'`, counts of `"name":"(Agent|Task)"`), plus on FILES ON DISK
  (`grep "export function add" src/math.js`), plus a negative check that nothing extra was built. Ends with a token-usage report
  (`analyze-token-usage.py`).
- `tests/claude-code/README.md`: two tiers — fast tests by default (~2 min), `--integration` for the 10–30 min run.
- Takeaway: assert on (1) state on disk, (2) tool calls in the JSONL transcript, (3) ordering — never on model prose.

### 1b. Real model, tiny deterministic prompts — anthropics/claude-code-action
- `.github/workflows/test-settings.yml`: runs the real action with prompt `Use Bash to echo "Hello from settings test"`, then a
  shell step greps the execution file for the marker and for the absence of `Permission to use Bash ... has been denied`.
  A paired job asserts the DENY case. Fork PRs skipped (no credentials).
- `.github/workflows/test-structured-output.yml`: uses `--json-schema` so the result is machine-checkable with `jq`.
- Takeaway: make the prompt so literal that the model has no latitude; assert with grep/jq; test allow AND deny.

### 1c. Official headless knobs (code.claude.com/docs/en/headless.md, cli-reference.md)
- `--output-format stream-json --verbose` gives NDJSON events; `--include-hook-events` adds hook lifecycle events
  (SessionStart/Setup hook events always included). Denials appear as `permission_denied` system messages and in
  `permission_denials` on the final result.
- `--max-budget-usd` (print mode; subagent spend counts), `--max-turns` (exits with error at limit), `--json-schema`,
  `--permission-mode dontAsk|acceptEdits|auto`, `--allowedTools "Bash(aether *)"`, `--no-session-persistence`, `--setting-sources`.
- CAVEAT: `--bare` skips hooks, skills, custom commands, subagents, CLAUDE.md — and docs say it "will become the default for -p in a
  future release". Aether's journey MUST NOT use bare (it is testing hooks + commands) and should pin this explicitly
  (e.g. `--setting-sources user,project,local`) so a future default flip does not silently hollow out the gate.
- SessionStart `initialUserMessage` exists for -p mode (hooks.md SessionStart table).

### 1d. Record/replay so runs are cheap and deterministic — goose, SWE-agent, codex, OpenHands
- goose `crates/goose/src/providers/testprovider.rs`: `TestProvider::new_recording(inner, file)` wraps a real provider and saves
  `{input, output}` records keyed by SHA-256 of the message list; `new_replaying(file)` serves them with no network. Hash input is
  normalised first (strips internal metadata) so replays survive irrelevant changes. Recordings are committed:
  `crates/goose-cli/src/scenario_tests/recordings/<provider>/<scenario>.json`; runner in `scenario_runner.rs` takes a validator
  closure per scenario and can target one provider via `GOOSE_TEST_PROVIDER`. MCP servers get the same treatment
  (`crates/goose/tests/mcp_replays/`).
- SWE-agent `sweagent/agent/models.py`: `ReplayModel` replays the actions of a `.traj` file; `InstantEmptySubmitModel`; every model
  config has `per_instance_cost_limit`, `total_cost_limit`, `per_instance_call_limit`. `sweagent run replay --traj_path` re-executes
  recorded actions against a real environment ("Debugging and testing of tools and environment behavior"). Fixture repo is a
  purpose-built public repo, `SWE-agent/test-repo` ("Repo with very simple issues"); trajectories in `tests/test_data/trajectories/`.
- codex `codex-rs/core/tests/common/responses.rs`: wiremock `MockServer` fakes the model API with scripted SSE; `ResponseMock`
  captures every request so tests assert what the agent SENT back (`saw_function_call`, `function_call_output_text`).
- OpenHands: two tiers — `.github/workflows/mock-llm-e2e.yml` starts `tests/e2e/mock-llm/scripts/mock-llm-server.py` and runs
  Playwright e2e against it on CI with a global timeout; `tests/e2e/live*/` hits real providers, is excluded from the default test
  run, skips providers with no creds, and its README keeps a dated "Last validated result" table.
- Relevance: the REPLAY tier suits Aether's Go-side journey (drive `aether` commands with scripted worker outputs — the six bugs
  were all in the Go program, not in the model). The LIVE tier stays `claude -p`.

### 1e. Flake policy — cline, gptme
- cline `evals/smoke-tests/README.md`: 3 trials per scenario by default; reports pass@k (any trial) AND pass^k (all trials =
  reliability). Scenario = `config.json` {prompt, expectedFiles, expectedContent[{file, contains}], timeout} + optional `template/`
  dir of starting files.
- cline `evals/analysis/patterns/cline-failures.yaml`: regex table classifying each failure as `transient` (429, timeout, 503 →
  retriable), `harness`, `environment`, `auth`, `policy`, or `provider_bug` (with issue link). Only non-transient classes count
  against the product.
- gptme `.github/workflows/eval-ci.yml`: 5 tiny evals on Haiku ("~$0.03-0.08/run"), `--timeout 60 --parallel 5`, path-filtered
  triggers, skips drafts and forks, cancels stale runs, and is `continue-on-error: true` ("Non-blocking in Phase 1 — informational
  only") — i.e. they did NOT make it a hard gate at first.
- aider `benchmark/README.md`: run inside Docker because model output is executed; `--tries`, `--threads`; `--stats` on a results dir.

## 2. "Never a dead end" in strict CLIs

- jj structured hints: `cli/src/command_error.rs` — `CommandError { hints: Vec<ErrorHint> }`, `.hinted("Run \`jj git remote rename\`
  to give a different name.")`. The hint is a FIELD of the error type, not prose glued into the message. `cli/src/cli_util.rs`:
  stale state → `.hinted("Run \`jj workspace update-stale\` to update it.")`, and with config `snapshot.auto-update-stale` the
  command recovers by itself instead of refusing (lines ~498-512). If the needed operation was lost it creates a
  "RECOVERY COMMIT" rather than failing.
- jj operation log: `docs/operation-log.md` — every mutating command records an operation with a full view snapshot; `jj undo`,
  `jj op revert`, `jj op restore`, `--at-op` to inspect the past. Also gives lock-free concurrency (no stale locks to clear).
  `docs/conflicts.md`: conflicts are recorded in the commit and "the rebase operation will succeed" — the tool records the
  problem as state and keeps going instead of refusing.
- terraform `internal/command/clistate/state.go`: `LockErrorMessage` names the escape hatch (`-lock=false`, "not recommended");
  `UnlockErrorMessage` names `force-unlock` and states exactly when it is safe. Escape + danger statement in the same message.
- gh `pkg/cmd/root/help.go`: "To get started with GitHub CLI, please run:  gh auth login" + the env-var alternative.
- git `unpack-trees.c` `report_collided_checkout()`: on a case-insensitive FS a clone with colliding paths SUCCEEDS and WARNS,
  listing each collided path — does not refuse.
- syncthing `lib/fs/casefs.go`: a filesystem wrapper that resolves the REAL on-disk case of every path and returns a typed
  `CaseConflictError{Given, Real}` whose message tells the user what to do. Case handled once, at the FS boundary.
- restic `doc/040_backup.rst`: "Symlinks are archived as symlinks, restic does not follow them." Change detection only for
  regular files. (The rule Aether's pause fingerprint broke: use lstat, hash the link target string, never descend.)
- brew `Library/Homebrew/cmd/doctor.rb`: doctor warnings are advisory ("If everything you use Homebrew for is working fine:
  please don't worry"), checks individually runnable (`--list-checks`, named args).

## 3. Claude Code hooks and output delivery (code.claude.com/docs/en/hooks.md, fetched today)

ANSWER: YES. `systemMessage` (top-level field of hook JSON output) is "Warning message shown to the user" — delivered by
Claude Code itself, not via the model. "To surface a message to the user on any platform, return systemMessage in JSON output."
- Works on Stop, PostToolUse, PreToolUse, UserPromptSubmit, SessionStart etc. DISCARDED on: MessageDisplay, Notification,
  PreCompact, PostCompact, SessionEnd, Setup, InstructionsLoaded, ConfigChange, Elicitation, WorktreeRemove. On
  FileChanged/CwdChanged it is only a brief terminal notification. From an `async: true` hook it goes to Claude, NOT the user.
- Cap: 10,000 characters per field; over that it is replaced by a file path + 2,000-char preview. No setting raises it.
- Labelled as a warning in the UI — presentation is not Aether's to style. In stream-json it may arrive as an
  `SDKInformationalMessage` (so the journey test can assert on it).
- `suppressOutput` "has no effect"; a successful hook's plain stdout is never shown in the transcript.
- `additionalContext` (inside `hookSpecificOutput`) goes to CLAUDE only, never shown as a chat message.
- Stop hook: input carries `last_assistant_message`, `stop_hook_active`, `transcript_path`. Block with
  `{"decision":"block","reason":"..."}`; gentler variant `hookSpecificOutput.additionalContext` shows as "Stop hook feedback" not an
  error. Hard cap: Claude Code overrides after 8 consecutive blocks. `{"continue":false,"stopReason":"..."}` shows stopReason to user.
- Proven usage: anthropics/claude-code `plugins/hookify/core/rule_engine.py` returns `{"decision":"block","reason":m,
  "systemMessage":m}` on Stop and a bare `{"systemMessage":...}` for warn-only rules (a ready-made warn-vs-block split);
  anthropics/claude-plugins-official `plugins/ralph-loop/hooks/stop-hook.sh` emits `decision/reason/systemMessage` via `jq -n`.
- PostToolUse gets `tool_response`; handlers can be narrowed with `"if": "Bash(aether *)"` (best-effort filter, docs say so).
- MessageDisplay hook + `displayContent`: rewrites what is RENDERED of an assistant message (transcript unchanged). Assistant text
  only; 10 s default timeout; could in principle splice a screen in, but it is fragile — see Traps.
- Skills/commands: `` !`cmd` `` is "dynamic context injection" — output is inlined into the PROMPT Claude receives; it is not
  shown to the user. It removes a tool round-trip but does not solve visibility.
- Status line: multi-line supported, `refreshInterval` ≥1 s, 300 ms debounce — good for a permanent one-line "phase 3/7, next:
  /ant-continue", not for full screens.
- SessionStart: superpowers `hooks/session-start` shows the exact JSON shape and warns Claude Code reads both
  `additional_context` and `hookSpecificOutput` without de-duplication — emit only one.

## 4. Real-use feedback loops
- pre-commit `pre_commit/error_handler.py`: on any unexpected error prints one line + "Check the log at <path>", and the log is
  pre-formatted markdown (version info block, error, traceback) ready to paste into an issue. Distinguishes FatalError (exit 1,
  expected) from unexpected (exit 3).
- brew `cmd/gist-logs.rb`: one command bundles logs + system config into a gist, `--new-issue` files it; truncates the middle
  of large logs keeping front and back.
- goose `documentation/docs/troubleshooting/diagnostics-and-reporting.md`: `goose session diagnostics --session-id X` → one JSON
  with system info, config, session data, recent logs; plus a viewer script.
- cline failure-pattern YAML (above) doubles as a triage table for field reports.

## What Aether should steal (ranked)
1. Hook `systemMessage` to put screens in front of the owner (C). Source: hooks.md JSON-output table; hookify rule_engine.py;
   ralph-loop stop-hook.sh. PostToolUse handler with `if: "Bash(aether *)"` re-emits the command's rendered screen as
   systemMessage (≤10k chars); the Stop hook becomes a backstop only. Small.
2. Refusal = typed error with a mandatory "next command" field (B). Source: jj command_error.rs hints; terraform state.go;
   gh help.go. One Go refusal type {what, why, next_command, protects_data bool}; one test walks every construction site and
   fails on an empty next_command, and the journey asserts each printed next_command actually runs. Medium.
3. Journey assertions on disk state + JSONL tool calls + ordering, never prose (A). Source: superpowers
   test-subagent-driven-development-integration.sh + test-helpers.sh; claude-code-action test-settings.yml. Medium.
4. Budget/turn/time caps and explicit non-bare pin (A). Source: cli-reference (`--max-budget-usd`, `--max-turns`), headless.md
   bare-mode note; SWE-agent cost limits. Small.
5. Failure classifier + 3 trials with pass^k (A). Source: cline failures.yaml + smoke-tests README. Transient (429/overloaded/
   timeout) → retry once and do not count; anything else fails the gate. Small.
6. Two tiers: replayed/scripted-worker journey on every change, live `claude -p` journey before release (A). Source: goose
   testprovider.rs, SWE-agent ReplayModel, OpenHands mock-llm vs live. Medium–large; do the live tier first.
7. Messy fixture repo as a committed builder script, not a tarball (A). Source: cline `template/` dirs; SWE-agent/test-repo.
   Script must create: symlink to a directory, symlink loop, case-only filename pair (skip/detect on case-insensitive FS),
   pre-existing archive name differing only by case, stale lock, superseded spec, leftover junk data, dirty git tree. Small–medium.
8. Filesystem rules at one boundary (B). Source: restic (never follow symlinks), syncthing casefs.go (resolve real case, typed
   error), git report_collided_checkout (warn + list, don't refuse). Small each.
9. Refusal/crash log + one-command report bundle (D). Source: pre-commit error_handler.py, brew gist-logs, goose diagnostics.
   Every refusal appends to a local log; `aether report` bundles version, last N refusals, state summary as paste-ready markdown.
   Small–medium.
10. Self-heal stale state instead of refusing (B). Source: jj `snapshot.auto-update-stale`, recovery commit. Apply to the
    "planning run pinned to a superseded spec" class: detect, re-pin, say so in one line. Medium.
11. Operation log + undo (B). Source: jj operation-log.md. Strongest dead-end cure but large; defer past the freeze.

## Traps
- MessageDisplay/`displayContent` to splice screens into assistant text: display-only, runs per streamed batch, 10 s timeout,
  falls back silently — a second, flakier channel than systemMessage.
- Relying on the Stop hook to force the model to paste the screen: 8-consecutive-block override, costs a model turn each time,
  and the model may paraphrase. Use as backstop only.
- `` !`cmd` `` in commands for visibility: feeds the prompt, not the user's screen.
- `--bare` for speed/determinism: removes exactly the hooks and commands under test.
- Asserting on model prose or golden full transcripts: superpowers needed case-insensitive keyword greps even for simple checks;
  golden transcripts churn with every model update.
- Making the live gate hard from day one: gptme shipped theirs as `continue-on-error` first. With an unclassified flake rate a
  hard gate trains you to re-run until green. Classify first (item 5), then harden.
- VCR replay of `claude -p` itself via a fake Anthropic endpoint: I found no >1k-star repo doing this for Claude Code; request
  hashing breaks whenever Claude Code changes its system prompt. Replay at Aether's own worker boundary instead. (Unverified
  that it cannot work — just no proven example.)
- LLM-as-judge grading (promptfoo-style rubrics) for the release gate: adds a second nondeterministic component; every Aether
  outcome that matters is checkable on disk.
- A `doctor` that fails on advisory findings: brew explicitly tells users to ignore doctor warnings if things work.
- Docker sandboxing (aider/SWE-agent style): needed when executing untrusted generated code at scale; for one owned fixture
  repo in a temp dir with a temp HOME it is ceremony.
