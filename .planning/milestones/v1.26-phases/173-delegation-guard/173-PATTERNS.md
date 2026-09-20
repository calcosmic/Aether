# Phase 173: Delegation Guard - Pattern Map

**Mapped:** 2026-08-12
**Files analyzed:** 11 (7 modified, 4 new)
**Analogs found:** 11 / 11 — this phase is a "switch on and prove" phase; every file below
already has a same-repo sibling doing the same shape of thing correctly. There is no file
in this phase that needs an invented pattern.

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `cmd/spawn.go` (`spawnCanSpawnDecision`, `spawnLogCmd`, `spawnTreeActiveCmd`, `spawnTreeDepthCmd`) | controller/CLI command | request-response (guard decision) | itself — the seam already exists at `cmd/spawn.go:176`, widen in place | exact |
| `cmd/internal_cmds.go` (`spawnCanSpawnSwarmCmd`, D-19 fix) | controller/CLI command | request-response (guard decision) | `cmd/spawn.go`'s `spawnCanSpawnCmd` (the sibling guard, already correct in shape once SPAWN-01 lands) | role-match |
| `cmd/spawn_budget.go` (NEW — whole-run tree budget) | service | CRUD (count + compare) | `cmd/queen_spawn_budget.go` (vocabulary/shape to borrow, NOT to reuse as the same budget — see Anti-Pattern below) + `pkg/agent/spawn_tree.go`'s `CurrentRun`/`EntriesForRun` (the actual data source) | role-match (vocabulary) / exact (data source) |
| `cmd/spawn_ancestor.go` (NEW — ancestor-chain cycle walk) | utility | transform (pure function over parsed tree) | `pkg/agent/spawn_tree.go:Parse()` + `cmd/internal_cmds.go:502-513`'s lookup shape (rewritten against `Parse()`, not hand-rolled) | role-match |
| `cmd/spawn_reap.go` (NEW — reaper + `spawn-reap*` commands) | service + controller | batch (scan) + CRUD (mutate status) | `pkg/agent/spawn_tree.go`'s `UpdateStatus`/`UpdateStatusPreserveActivity` (the mutation primitive) + `cmd/spawn.go`'s command-registration shape | role-match |
| `cmd/hook_cmds.go` (`hookPreToolUseCmd`, new `Task` matcher + deny path) | middleware | event-driven (pre-tool-use hook) | itself — extend the existing `Write|Edit` branch pattern with a new `Task` branch; `protectedHookWriteReason` is the fail-closed decision-function shape to copy | exact |
| `.claude/settings.json` (`PreToolUse` matcher list) | config | event-driven (hook registration) | itself — add a second matcher entry alongside the existing `Write\|Edit` entry | exact |
| `pkg/agent/spawn_tree.go` (`RecordSpawn` depth derivation; possible `abandoned` status) | model | CRUD | itself — `normalizeSpawnStatus`/`IsTerminalSpawnStatus`/`IsLiveSpawnStatus` already the place new statuses are added | exact |
| `.aether/workers.md` (depth table + two `--depth 0` lines) | config/doc | — | itself — corpus-content correction, no code analog needed | n/a |
| `cmd/spawn_enforce_test.go` (SPAWN-01/02/03/05 guard tests — NEW test functions) | test | request-response (guard assertions) | itself — `TestSpawnCanSpawnEnforceDeniesWithNonZeroExit`'s decision-substitution shape | exact |
| `cmd/hook_cmds_test.go` (SPAWN-04 hook test cases — NEW test functions) | test | event-driven (hook stdin/stdout) | itself — `TestHookPreToolUseBlocksMainBranchWhenRedirectActive`'s stdin-JSON + state-fixture shape | exact |

## Pattern Assignments

### `cmd/spawn.go` — widen `spawnCanSpawnDecision`, derive depth in `spawn-log`, extend `spawn-tree-active`/`spawn-tree-depth` (controller, request-response)

**Analog:** itself. This is the one true chokepoint the research names — do not create a
second seam elsewhere.

**The decision seam today** (`cmd/spawn.go:170-178`, confirmed by direct read):
```go
// spawnCanSpawnDecision is the allow/deny seam Phase 173 (SPAWN-01) replaces
// with a real decision. Today it unconditionally allows every depth. It is a
// package-level function variable (not a plain func) specifically so a test
// can substitute a deny answer for the duration of a single test case,
// driving --enforce's deny-to-non-zero-exit path without waiting for
// Phase 173 to implement the real cap logic.
var spawnCanSpawnDecision = func(depth int) (bool, string) {
	return true, ""
}
```
Widen the func-var signature in place (per RESEARCH.md's Pattern 1 recommendation) — do not
add a parallel `var` elsewhere. Keep it a package-level `var func`, not a plain `func`, so
`cmd/spawn_enforce_test.go` can keep substituting it exactly as
`TestSpawnCanSpawnEnforceDeniesWithNonZeroExit` already does (see below).

**The refusal-to-exit-code machinery that already works** (`cmd/spawn.go:180-218`, confirmed):
```go
var spawnCanSpawnCmd = &cobra.Command{
	Use:   "spawn-can-spawn",
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		depth := mustGetInt(cmd, "depth")
		if len(args) == 1 {
			parsed, err := strconv.Atoi(args[0])
			if err != nil {
				outputError(1, fmt.Sprintf("invalid depth %q: must be an integer", args[0]), nil)
				return nil
			}
			depth = parsed // positional wins over --depth
		}
		enforce, _ := cmd.Flags().GetBool("enforce")
		canSpawn, reason := spawnCanSpawnDecision(depth)
		if enforce && !canSpawn {
			msg := fmt.Sprintf("spawn denied at depth %d", depth)
			if reason != "" {
				msg = fmt.Sprintf("%s: %s", msg, reason)
			}
			outputError(1, msg, nil)
			return nil
		}
		outputOK(map[string]interface{}{"can_spawn": canSpawn, "depth": depth})
		return nil
	},
}
```
Do not touch this shape — only the decision function's inputs/outputs change. Preserve the
positional-wins-over-flag dual-input contract (`.aether/workers.md:292` sends the positional
form; build playbooks send `--depth`); this is what `TestSpawnCanSpawnAcceptsDocumentedInvocation`
locks.

**The chokepoint that must change** — `spawnLogCmd` (`cmd/spawn.go:14-79`, confirmed):
```go
var spawnLogCmd = &cobra.Command{
	Use:   "spawn-log",
	RunE: func(cmd *cobra.Command, args []string) error {
		...
		depth, _ := cmd.Flags().GetInt("depth") // <-- caller-supplied, currently trusted
		st := agent.NewSpawnTree(store, "spawn-tree.txt")
		if err := st.RecordSpawn(parent, caste, name, task, depth); err != nil { // <-- passed through unchanged
```
Replace the `cmd.Flags().GetInt("depth")` value with a value derived by looking up `parent`
in `st.Parse()` and taking `parentEntry.Depth + 1` (SPAWN-02). D-19 fail-closed applies here:
decide and document the coordinator's own sentinel parent name as the one legitimate
"not found → depth 0" case; every other unresolved parent must deny, not silently default.

**`spawn-tree-active` to extend for SPAWN-07** (`cmd/spawn.go:248-294`, confirmed) already
emits `Name/Parent/Caste/Task/Depth/Status/SpawnedAt` per entry — the raw data D-14's readable
render needs is already there. What's missing is depth-indentation and English rendering (see
`casteLabel`/`casteEmoji` under Shared Patterns below); do not rebuild the data-gathering half.

**`spawn-tree-depth` for D-07's fixed number** (`cmd/spawn.go:296-322`, confirmed) already
computes `max(entries[i].Depth)` — once `spawn-log` derives depth correctly, this command
needs no changes; it already reports whatever is recorded.

---

### `cmd/internal_cmds.go` — invert D-19's fail-open defect in `spawn-can-spawn-swarm` (controller, request-response)

**Analog:** `cmd/spawn.go`'s sibling guard, post-SPAWN-01. Fix `spawn-can-spawn-swarm` in
place; RESEARCH.md's Pitfall 5 explicitly warns against assuming a merge is required or that
fixing one command fixes the other — they are separate, currently-divergent code paths.

**The exact defect, confirmed by direct read** (`cmd/internal_cmds.go:549-558`):
```go
var state colony.ColonyState
if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
	// No state = no spawns, budget available   <-- WRONG per D-19
	outputOK(map[string]interface{}{
		"can_spawn":        true,
		"remaining_budget": 5,
		"current_spawns":   0,
	})
	return nil
}
```
**Fix shape:** on load error, return `can_spawn: false` with a named reason (e.g.
`"colony state unreadable: cannot verify spawn budget"`). Note this command takes no
`--enforce` flag today — confirm in planning whether SPAWN-04/D-19's "exits non-zero"
expectation is scoped to this command; ROADMAP criterion 4 names the JSON payload's
`can_spawn` field specifically, so that must flip at minimum.

**Also note the hand-rolled lookup at `cmd/internal_cmds.go:493-535`** (`spawnGetDepthCmd`) —
this is the ORPHANED command RESEARCH.md flags as containing reusable shape. It hand-parses
`spawn-tree.txt` with `splitLines`/`splitPipe` instead of `agent.SpawnTree.Parse()`:
```go
data, err := store.ReadFile("spawn-tree.txt")
...
lines := splitLines(string(data))
for _, line := range lines {
	fields := splitPipe(line)
	// Format: timestamp|parent|caste|agentName|task|depth|status
	if len(fields) >= 7 && fields[3] == name {
		found = true
		depth = parseIntSafe(fields[5])
		break
	}
}
```
**Correction to the phase brief's framing:** this command's logic is real and correct in
outcome, but per RESEARCH.md's own "Don't Hand-Roll" table and Anti-Pattern list, do NOT copy
this hand-rolled parsing into `spawn-log`'s new depth-derivation code. Reuse the *shape*
("find the most recent entry with matching `AgentName`, read its `Depth`") but implement it
against `agent.SpawnTree.Parse()`, matching `cmd/internal_cmds.go:153-168`'s
`latestSpawnEntryByName` helper (already in `cmd/spawn.go`'s own file, confirmed above), which
already does exactly this the correct way:
```go
func latestSpawnEntryByName(st *agent.SpawnTree, name string) *agent.SpawnEntry {
	entries, err := st.Parse()
	if err != nil {
		return nil
	}
	for i := len(entries) - 1; i >= 0; i-- {
		if entries[i].AgentName == name {
			entry := entries[i]
			return &entry
		}
	}
	return nil
}
```
This is the actual best analog for SPAWN-02's parent lookup — closer than `spawn-get-depth`,
already in the same file, and already Parse()-based. Point the planner here first.

---

### `cmd/spawn_budget.go` (NEW) — whole-run tree budget (service, CRUD/count)

**Analog for vocabulary/shape (borrow the shape, NOT the budget):** `cmd/queen_spawn_budget.go`.

**Analog for the actual data source:** `pkg/agent/spawn_tree.go`'s run-tracking API — already
wired into every lifecycle command via `cmd/spawn_runs.go`.

**Confirmed, already-working run/entry API** (`pkg/agent/spawn_tree.go:162-197`):
```go
// CurrentRun returns the current or most recent tracked run.
func (st *SpawnTree) CurrentRun() (SpawnRun, bool, error) { ... }

// EntriesForRun returns spawn entries that belong to the given run's time window.
func (st *SpawnTree) EntriesForRun(runID string) ([]SpawnEntry, error) { ... }
```
**Composition to write (new, per RESEARCH.md Code Examples):**
```go
func spawnsConsumedThisRun(st *agent.SpawnTree) (int, error) {
	run, ok, err := st.CurrentRun()
	if err != nil || !ok {
		return 0, err // D-19: caller must decide fail-closed when run identity is unknown
	}
	entries, err := st.EntriesForRun(run.ID)
	if err != nil {
		return 0, err
	}
	return len(entries), nil // D-03: every helper counted, live or completed
}
```
**Existing lifecycle wiring to reuse, not reinvent** (`cmd/spawn_runs.go`, full file, confirmed):
```go
func beginRuntimeSpawnRun(command string, startedAt time.Time) (*runtimeSpawnRun, error) {
	tree := agent.NewSpawnTree(store, "spawn-tree.txt")
	run, err := tree.BeginRun(command, startedAt)
	...
	return &runtimeSpawnRun{Tree: tree, Run: run}, nil
}
func finishRuntimeSpawnRun(handle *runtimeSpawnRun, status string, endedAt time.Time) {
	...
	_ = handle.Tree.EndRun(handle.Run.ID, status, endedAt)
}
```
This is already called from build/continue/colonize/plan/swarm — SPAWN-03 needs zero new
run-tracking infrastructure, only the count composed above, read inside the widened
`spawnCanSpawnDecision`.

**Anti-pattern warning (load-bearing, repeat to the planner):** `cmd/queen_spawn_budget.go`'s
`queenSpawnBudget`/`queenSpawnBudgetForPhase` is a **pre-dispatch, per-wave WORKER SELECTION**
budget — it decides which castes get dispatched in a wave, BEFORE any spawn happens. It is not
an enforcement gate at spawn-record time. D-04 explicitly requires the wave cap and the tree
budget to stay two visibly different quantities; reusing `queenSpawnBudget`'s struct or
constants for SPAWN-03 would silently merge them. Borrow only the naming convention
(`...Budget` struct with a `Reason string` field, e.g. `queenSpawnBudget.Reason` at
`cmd/queen_spawn_budget.go:16`) — not the type itself.

**Caveat to carry into the plan (D-21 bounded-claim requirement):** `filterEntriesForRun`
(`pkg/agent/spawn_tree.go:490-522`, confirmed) windows by `ActivityTimestamp`/`Timestamp` —
wall-clock, not a transactional link to the run ID. State this as a named, bounded
approximation in the plan, not as an absolute guarantee.

---

### `cmd/spawn_ancestor.go` (NEW) — ancestor-chain cycle detection (utility, transform)

**Analog:** `agent.SpawnTree.Parse()` (confirmed, `pkg/agent/spawn_tree.go:561-569`) is the
correct read primitive; no existing ancestor-walk helper exists anywhere in the repo — this is
genuinely new logic, but it must be built ON TOP of `Parse()`, not a fourth hand-rolled
pipe-parser (RESEARCH.md's Anti-Pattern list explicitly calls out three existing divergent
hand-rolled parsers as a pattern to stop repeating: `spawn-get-depth`, `spawn-tree-depth`'s own
`Parse()` use is already correct, and `spawn-can-spawn-swarm`'s budget count in
`cmd/internal_cmds.go:570-582` is a third hand-rolled one worth noting but out of this file's
direct scope).

```go
// Source: pkg/agent/spawn_tree.go:563-569 (confirmed, HIGH confidence)
func (st *SpawnTree) Parse() ([]SpawnEntry, error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	entries, _, err := st.parseFile()
	return entries, err
}
```
Walk `[]SpawnEntry` from the would-be child backward via `ParentName` links, comparing
`(Caste, normalized Task)` at each ancestor against the proposed spawn — SPAWN-05. There is no
existing task-normalization helper in the repo (confirmed by absence, not by omission — grep
of `spawn_tree.go` and `internal_cmds.go` shows no such function); this is genuinely new, small,
pure text processing and should get its own bounded unit test per RESEARCH.md's V5 flag.

---

### `cmd/spawn_reap.go` (NEW) — reaper + manual orphan commands (service + controller, batch + CRUD)

**Analog for the mutation primitive:** `pkg/agent/spawn_tree.go`'s `UpdateStatus` /
`UpdateStatusPreserveActivity` (confirmed, `pkg/agent/spawn_tree.go:228-296`):
```go
func (st *SpawnTree) UpdateStatus(name string, status string, summary string) error {
	return st.updateStatus(name, status, summary, false)
}
func (st *SpawnTree) UpdateStatusPreserveActivity(name string, status string, summary string) error {
	return st.updateStatus(name, status, summary, true)
}
```
`UpdateStatusPreserveActivity` is the one to reach for: reaping an entry should not make it
look like fresh activity (which would poison SPAWN-03's run-window count). This already exists
and is already exercised by other callers — do not write a third status-update path.

**Analog for command registration and JSON envelope:** `cmd/spawn.go`'s existing command
list-and-init shape (`spawnTreeActiveCmd` for "list", `spawnCompleteCmd` for "mutate one
entry") — model `spawn-orphans`/`spawn-reap` on that pairing (a read command + a mutate
command), not a single combined command.

**Terminal-status registration point** (`pkg/agent/spawn_tree.go:678-686`, confirmed) — if the
reaper needs a distinct `"abandoned"` status (per RESEARCH.md's recommended project structure
note), this is the one place it must be added on both sides:
```go
func IsLiveSpawnStatus(status string) bool {
	switch normalizeSpawnStatus(status) {
	case "spawned", "starting", "active", "running":
		return true
	default:
		return false
	}
}
func IsTerminalSpawnStatus(status string) bool {
	switch normalizeSpawnStatus(status) {
	case "completed", "failed", "blocked", "timeout", "superseded", "manually-reconciled", "skipped":
		return true
	default:
		return false
	}
}
```
Adding `"abandoned"` to `IsTerminalSpawnStatus` (and NOT to `IsLiveSpawnStatus`) is what makes
D-18 ("reaping releases budget") work for free through SPAWN-03's counting — confirm the
budget-counting code counts `EntriesForRun` regardless of live/terminal status per D-03 ("every
helper counted, live or completed"), so reaping changes `spawn-tree-active`'s *view* but not
the run's total consumed count; D-18 must be satisfied by the budget composition explicitly
excluding abandoned entries, or by a separate "remaining" calculation — this is a real design
decision the plan must state, not assume.

**No heartbeat / clock-injection gap (RESEARCH.md Pitfall 4, confirmed real):**
`SpawnEntry.ActivityTimestamp` is set at spawn and only touched again at completion — there is
no periodic liveness signal. `RecordSpawn` calls `time.Now().UTC()` directly
(`pkg/agent/spawn_tree.go:210`) with no injectable clock. A reaper test needs either (a) a
small clock-injection seam added to `SpawnTree`, or (b) a helper that writes an
already-stale-timestamped entry directly to the store bypassing `RecordSpawn`. Flag this as a
Wave 0 test-infrastructure decision, not an afterthought.

---

### `cmd/hook_cmds.go` — new `Task` matcher + deny path in `hookPreToolUseCmd` (middleware, event-driven)

**Analog:** itself, extend the existing `Write|Edit` branch with a new branch for `Task`. The
`protectedHookWriteReason` function is the fail-closed decision-function SHAPE to copy exactly
— it returns `""` for "allow" and a human-readable string for "deny", and the caller wraps
that in `emitHookBlock` only when non-empty.

**The existing dispatch shape** (`cmd/hook_cmds.go:34-90`, confirmed):
```go
RunE: func(cmd *cobra.Command, args []string) error {
	input := readClaudeHookInput()
	toolName := input.ToolName
	...
	if strings.EqualFold(toolName, "Write") || strings.EqualFold(toolName, "Edit") {
		if reason := protectedHookWriteReason(target, cwd); reason != "" {
			... tracer.LogIntervention(...) ...
			return emitHookBlock(reason)
		}
		if reason := redirectWriteReason(target, cwd); reason != "" {
			...
			return emitHookBlock(reason)
		}
	}
	return nil
},
```
Add a sibling `if strings.EqualFold(toolName, "Task") { ... }` branch. Follow the same
tracer.LogIntervention-then-emitHookBlock pattern for consistency with the existing audit
trail.

**The fail-closed decision-function shape to copy** (`cmd/hook_cmds.go:239-266`, confirmed):
```go
func protectedHookWriteReason(target, cwd string) string {
	normalized := normalizeHookPath(target, cwd)
	if normalized == "" {
		return "" // NOTE: empty target = allow here, but see caveat below
	}
	...
	switch {
	case strings.Contains(slash, "/.aether/data/"):
		return "Protected colony state path. ..."
	...
	default:
		return ""
	}
}
```
**MISLEADING-ANALOG WARNING (flagged per the task brief's explicit ask):**
`protectedHookWriteReason`'s early return — `if normalized == "" { return "" }` — is a
fail-**open** branch: an unresolvable target is currently treated as "allow." That is
acceptable for THIS function (an empty/unresolvable write target has nothing to protect), but
it is the wrong branch to copy verbatim for SPAWN-04's requester-depth resolution. D-20 requires
the OPPOSITE: when the requester's depth cannot be resolved, deny. Do not pattern-match this
function's control flow shape without inverting that one branch — copy the
"decision-function-returns-a-reason-string" shape, not the "unresolved input defaults to empty
string" behavior.

**The deny-emission primitive** (`cmd/hook_cmds.go:366-377`, confirmed, reuse unchanged):
```go
func emitHookBlock(reason string) error {
	payload := map[string]string{"decision": "block", "reason": reason}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, string(encoded))
	return nil
}
```
**Stdin parsing already handles the JSON envelope shape**
(`cmd/hook_cmds.go:16-25, 186-203`, confirmed) — `claudeHookInput` struct and
`readClaudeHookInput()`. Extend `claudeHookInput` with the `agent_id`/`agent_type` fields
SPAWN-04 needs (RESEARCH.md's Assumption A1/A2 — verify the real field names against a live
captured hook payload in Wave 0 before locking field names into this struct).

---

### `.claude/settings.json` — register the new `Task` matcher (config, event-driven)

**Analog:** itself, confirmed full file (49 lines):
```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Write|Edit",
        "hooks": [{"type": "command", "command": "aether hook-pre-tool-use", "timeout": 10}]
      }
    ],
    ...
  }
}
```
Add a second object to the `PreToolUse` array with `"matcher": "Task"` (or `"Agent|Task"` per
the architecture diagram in RESEARCH.md — confirm the exact tool name Claude Code uses for
subagent dispatch in Wave 0's empirical check) pointing at the same
`"command": "aether hook-pre-tool-use"` — the Go-side `RunE` already receives `tool_name` in
its stdin JSON and can dispatch on it, so no new binary/command is needed, only a new matcher
entry routing more events into the existing one.

---

### `.aether/workers.md` — correct the depth table and the two `--depth 0` lines (doc)

**No code analog — corpus content fix.** Two separate defects, confirmed by direct read:

1. Lines 46, 328 (confirmed): `aether spawn-log --parent "Prime-1" --caste "builder" --name
   "Hammer-42" --task "implementing auth module" --depth 0` — sends `--depth 0` for what is
   actually a depth-1 child. Once SPAWN-02 lands this specific value becomes irrelevant (the
   flag is ignored server-side), but the doc should still stop instructing something wrong,
   since `spawn-can-spawn`'s courtesy pre-check still reads a caller-declared depth.

2. Lines 263-272 (confirmed, full table read) — the entire "Depth-Based Behavior" table
   contradicts D-01/D-05 beyond the two named lines: it has depth-2 "Specialist" workers ABLE
   to spawn ("Yes (if surprised)", max 2 sub-spawns — i.e. depth 3), and a separate "Global
   Cap: Maximum 10 workers per phase" line that matches neither D-02's 20 nor
   `queenSpawnBudget`'s per-wave 4-8. RESEARCH.md's Pitfall 1 is explicit: a test that only
   checks the two `--depth 0` strings are gone will pass while this table's residue survives.
   The plan needs a content-level test asserting the table (or its replacement) states
   depth-2 entries cannot spawn, not merely a literal-string absence check.

---

### `cmd/spawn_enforce_test.go` — SPAWN-01/02/03/05 guard tests (test, request-response)

**Analog:** itself. Two existing tests are the exact shape to replicate:

**Decision-substitution shape** (`cmd/spawn_enforce_test.go:153-205`, confirmed, excerpt):
```go
func TestSpawnCanSpawnEnforceDeniesWithNonZeroExit(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	origDecision := spawnCanSpawnDecision
	defer func() { spawnCanSpawnDecision = origDecision }()
	spawnCanSpawnDecision = func(depth int) (bool, string) {
		return false, "depth exceeds cap"
	}
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	renderedCommandExitCode.Store(0)
	rootCmd.SetArgs([]string{"spawn-can-spawn", "7", "--enforce"})
	_ = rootCmd.Execute()
	if code := int(renderedCommandExitCode.Load()); code == 0 {
		t.Fatalf("deny + --enforce did not set a non-zero exit code...")
	}
	...
}
```
Once `spawnCanSpawnDecision`'s signature is widened, this test (and every new one) substitutes
a fake decision the same way — `saveGlobals`/`resetRootCmd`/`newTestStore` are the established
per-test fixture trio; reuse them, do not build parallel fixture helpers.

**Corpus-tracking shape for `.aether/workers.md`** (`cmd/spawn_enforce_test.go:24-58`, confirmed)
— `extractSpawnCanSpawnInvocation` reads the live file and regex-extracts the documented
invocation rather than hardcoding a copy, so it starts validating new wording automatically.
Use the same "read the real file, don't duplicate its string" approach for the SPAWN-06/D-07
corpus assertion (`.aether/workers.md` no longer instructs `--depth 0` for children).

**CI wiring is automatic if tests land here** — confirmed
(`cmd/ci_wiring_gate_test.go:87-93`, full var read):
```go
var wiringGateGuardFiles = []string{
	"subcommand_reachability_ratchet_test.go",
	"command_call_audit_test.go",
	"cli_flag_audit_test.go",
	"spawn_enforce_test.go",
	"ci_wiring_gate_test.go",
}
```
`spawn_enforce_test.go` is already in this list. Any new top-level `func Test...` added to this
specific file is automatically required (and automatically verified, by
`TestWiringGateStepRunsEveryWiringTest`) to appear in `.github/workflows/ci.yml:100`'s `-run`
filter. Putting new SPAWN-01/02/03/05 tests in a NEW file (e.g. `spawn_budget_test.go`) forfeits
this — the ratchet only walks the five files in `wiringGateGuardFiles` above. If a new file is
genuinely needed (RESEARCH.md's Wave 0 gaps suggest `spawn_budget_test.go`/`spawn_reap_test.go`
might be cleaner), the plan MUST add that filename to `wiringGateGuardFiles` in the same wave,
or the CI-wiring proof silently does not cover it.

---

### `cmd/hook_cmds_test.go` — SPAWN-04 hook test cases (test, event-driven)

**Analog:** `TestHookPreToolUseBlocksMainBranchWhenRedirectActive`
(`cmd/hook_cmds_test.go:260-333`, confirmed, excerpted above) — the established shape for a
hook test: `saveGlobalsCmd`/`resetRootCmd`/`newTestStoreCmd` fixture trio, a real `git init` in
a temp dir when branch detection matters, `s.SaveJSON("COLONY_STATE.json", state)` for state
setup, `setHookStdin(t, ...)` to inject the hook's stdin JSON, `rootCmd.SetArgs([]string{"hook-pre-tool-use"})`
+ `rootCmd.Execute()`, then unmarshal `buf` and assert `result["decision"] == "block"` and
`result["reason"]` contains the expected substring. Copy this shape for the new `Task`-matcher
test cases; the new tests only need a different `tool_name`/`tool_input` in the injected stdin
and different state/fixture setup for depth resolution.

---

## Shared Patterns

### Fail-closed decision function (D-19/D-20)
**Source of the CORRECT shape:** `cmd/hook_cmds.go`'s `protectedHookWriteReason` — returns a
named reason string on deny, empty string on allow, called from a single dispatch point.
**Source of the DEFECT to invert:** `cmd/internal_cmds.go:549-558` (`spawn-can-spawn-swarm`'s
`can_spawn: true` on state-load error).
**Apply to:** every new guard function in `cmd/spawn_budget.go`, `cmd/spawn_ancestor.go`, the
widened `spawnCanSpawnDecision`, and the hook's `Task` branch. Every one of these must return
deny (not allow) when its data source (`COLONY_STATE.json`, `spawn-tree.txt`, run state) is
unreadable or the identity it needs cannot be resolved.

### Locked, cross-process-safe read/write of `spawn-tree.txt`
**Source:** `pkg/storage/storage.go:48-77` (confirmed) — `Store.AtomicWrite` (exclusive
`Lock`), `Store.UpdateFile` (read-modify-write under one exclusive `Lock`), `Store.ReadFile`
(confirmed at `pkg/storage/storage.go:295-296`, uses `RLock`). `pkg/agent/spawn_tree.go`'s
`RecordSpawn`/`UpdateStatus` already go through `st.store.UpdateFile`; `spawn-tree-active`
already goes through the RLock'd read path via `Parse()`/`parseFile()` → `st.store.ReadFile`.
**Apply to:** SPAWN-07's live-safe render, SPAWN-08's reaper scan and mutation, SPAWN-03's
budget count. Do not add a new locking scheme — every one of these already gets safe
concurrent access for free by going through `SpawnTree`'s existing methods.

### Non-technical readable caste identity (D-14)
**Source:** `cmd/codex_visuals.go:38-157, 3808-3846` (confirmed) — `casteEmojiMap`,
`casteLabelMap`, `casteColorMap`, and the accessor functions `casteEmoji(caste)`,
`casteLabel(caste)`, `casteIdentity(caste)` (which composes `emoji + " " + colorized label`,
e.g. `"🔨 Builder"`).
```go
func casteIdentity(caste string) string {
	return casteEmoji(caste) + " " + colorizeCaste(caste, casteLabel(caste))
}
```
**Apply to:** SPAWN-07's `spawn-tree-active` render. Do not add a second caste-name/emoji map
in `spawn.go` — call these existing accessors. This is the single source of truth per
CLAUDE.md's UX Architecture section (`cmd/codex_visuals.go` owns caste identity, not per-command
maps).

### Midden write for whole-run-ceiling refusals only (D-11)
**Source:** `cmd/midden_cmds.go:400-454` (confirmed, excerpt) — `middenWriteCmd` reads/creates
`colony.MiddenFile`, appends a `colony.MiddenEntry{ID, Timestamp, Category, Source, Message}`,
generates `ID` as `midden_{unix}_{pid}`.
**Apply to:** exactly the D-02 whole-run-ceiling refusal path in the widened
`spawnCanSpawnDecision` (or its caller) — call `midden-write`-equivalent logic (either shell out
via the existing command, or call the same underlying write path directly) ONLY when the reason
is "budget", never for a routine depth-cap refusal. D-11 is explicit that logging every refusal
is the wrong behavior — this must be conditional on reason, not blanket.

### Hermetic test fixture registration for guard tests
**Source:** `cmd/subcommand_reachability_ratchet_test.go:1739-1741` (confirmed) —
```go
selftest := &cobra.Command{Use: orphanLeaf, Run: func(*cobra.Command, []string) {}}
rootCmd.AddCommand(selftest)
defer rootCmd.RemoveCommand(selftest)
```
**Apply to:** any new test in this phase that needs to register a synthetic command or
otherwise mutate `rootCmd` for the duration of one test case — register inside the test body,
remove via `defer`, never leave global cobra state mutated across tests.

## No Analog Found

None. Every file this phase touches has a same-repo sibling doing the same shape of work
correctly (or, for `.aether/workers.md`, is itself the thing being corrected against the
locked D-05/D-01/D-02 decisions). The only genuinely novel logic is the ancestor-chain walk
(`cmd/spawn_ancestor.go`) and the reaper's staleness check (`cmd/spawn_reap.go`) — both are
new pure/mutation logic layered on `agent.SpawnTree.Parse()`/`UpdateStatus`, not new
infrastructure.

## Corrections to the Task Brief's Suggested Analogs

- **`cmd/internal_cmds.go:493-535` (`spawn-get-depth`) is real and orphaned as stated**, but is
  NOT the best lookup-logic analog to copy: it hand-parses with `splitLines`/`splitPipe`
  instead of `agent.SpawnTree.Parse()`. The better analog, in the same target file
  (`cmd/spawn.go`), is `latestSpawnEntryByName` (`cmd/spawn.go:153-168`) — already
  `Parse()`-based, already correct, already in scope. Cite `spawn-get-depth` to the planner as
  confirmation the LOOKUP SHAPE is right, but point implementation at
  `latestSpawnEntryByName`.
- **`protectedHookWriteReason` is a fail-open example for unresolvable input**, not fail-closed
  — flagged explicitly above. It is still the right function to copy for its
  "return a reason string, empty means allow, single dispatch point" shape; the one branch
  (`if normalized == "" { return "" }`) must be inverted for D-20's specific requirement, not
  reused verbatim.
- **`cmd/queen_spawn_budget.go` must be cited as a NEGATIVE analog for the actual budget
  value** — reuse only its naming convention (`...Budget` struct with a `Reason` field), never
  its type or its numbers, per D-04.

## Metadata

**Analog search scope:** `cmd/` (spawn*.go, hook_cmds.go, internal_cmds.go,
queen_spawn_budget.go, codex_visuals.go, midden_cmds.go, ci_wiring_gate_test.go,
subcommand_reachability_ratchet_test.go, spawn_enforce_test.go, hook_cmds_test.go),
`pkg/agent/spawn_tree.go` (full file), `pkg/storage/storage.go`, `.aether/workers.md`,
`.claude/settings.json`, `cmd/testdata/orphan_allowlist.json`
**Files scanned (full or targeted read):** 15
**Pattern extraction date:** 2026-08-12
