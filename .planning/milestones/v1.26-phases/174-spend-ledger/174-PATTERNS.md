# Phase 174: Spend Ledger - Pattern Map

**Mapped:** 2026-08-13
**Files analyzed:** 13 (7 new source + 4 new test + 6 modified)
**Analogs found:** 12 / 13

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `pkg/codex/usage.go` (extend `Source`) | model/value-type | transform | itself — extend in place | exact (existing file) |
| `pkg/codex/usage_test.go` (extend) | test | transform | `TestClaudeUsageTotalIncludesCacheReadAndCreation` in same file | exact |
| `cmd/spend_ledger.go` | service (persistence + roll-up) | CRUD (append-only, read-aggregate) | `pkg/agent/spawn_tree.go` (`SpawnTree`, `RecordSpawn`, `ToJSON`) | role-match (Go struct → wraps `pkg/storage.Store`, not a fresh design) |
| `cmd/spend_ledger_test.go` | test | CRUD / invariant | `cmd/consolidation_dryrun_test.go` (fixture-seed + assert pattern) + `cmd/caste_relevance_test.go`'s `TestQueenOrchestratePreservesSafetyCastes` (invariant-style test) | role-match |
| `cmd/spend_cmd.go` (`aether spend`) | controller (Cobra subcommand, read-only) | request-response | `cmd/proof_cmd.go` (`proofCmd`, `buildProofOutput`, `outputWorkflow`) | exact — same shape: load state, build a report struct, render JSON + visual, mutate nothing |
| `cmd/spend_cmd_test.go` | test (idempotency) | request-response | `cmd/consolidation_dryrun_test.go`'s `assertConsolidationDryRunIsPure` | exact — this is the named precedent CLAUDE.md and RESEARCH.md both cite for D-04 |
| `cmd/wrapper_usage_claude.go` | parser/utility | file-I/O, transform | `pkg/codex/usage.go`'s `ParseUsage`/`usageFromEvent` (line-scan + tolerant decode) + `cmd/hook_cmds.go`'s `claudeHookInput` (payload struct shape) | role-match — no line-oriented JSONL parser exists yet; nearest shape is `ParseUsage`'s per-line `json.Unmarshal` + skip-on-error loop |
| `cmd/wrapper_usage_claude_test.go` | test (fixture-based) | transform | `pkg/codex/usage_test.go` (literal externally-sourced expected values, never re-derived from code under test) | exact pattern, new fixture format |
| `cmd/wrapper_usage_opencode.go` | parser/utility (discovery + parse) | file-I/O, transform | same as above; discovery-by-time-window has no direct analog — nearest is `pkg/agent/spawn_tree.go`'s `filterEntriesForRun` (bounding records to a `[StartedAt, EndedAt]` window) | partial match |
| `cmd/wrapper_usage_opencode_test.go` | test (fixture-based) | transform | same as `wrapper_usage_claude_test.go` | exact pattern, new fixture format |
| `cmd/hook_cmds.go` (extend `claudeHookInput` + persist `TranscriptPath`) | middleware (hook receiver) | event-driven | itself — extend in place; persistence side is genuinely new (see No Analog Found) | role-match |
| `cmd/internal_worker_adapter.go` (`mapInternalWorkerResult` copies `Usage`) | transform (adapter) | request-response | itself — one-line addition to an existing struct literal | exact (existing file) |
| `cmd/codex_build_finalize.go` (`codexExternalBuildWorkerResult` gains `Usage`) | model + validation | request-response | itself — extend struct; path-boundary validation reuses `validateAndNormalizeClaimPathToRoot` if the ledger ever accepts a claimed path | exact (existing file) |
| `cmd/ceremony_cmd.go` (`renderCeremonyCloseoutVisual` gains token line) | controller (shared render function) | request-response | itself — extend in place; nearest sibling writer is `writeCeremonyWorkerSummary` | exact (existing file) |
| `CLAUDE.md` "## Token Budget" rename | docs | — | n/a (doc-only edit) | n/a |

## Pattern Assignments

### `pkg/codex/usage.go` (extend — add a third `Source` tier)

**Analog:** itself. This file already solves the exact problem (Source tagging,
`Measured()`, `billedTotal()`); Pattern 3 in RESEARCH.md says add a constant, not
a new type.

**Current source tagging** (`pkg/codex/usage.go:37-49`):
```go
// Measured reports whether the usage came from the provider rather than a
// local estimate.
func (u WorkerUsage) Measured() bool { return u.Source == UsageSourceProvider }

// Empty reports whether nothing at all was recorded.
func (u WorkerUsage) Empty() bool {
	return u.InputTokens == 0 && u.OutputTokens == 0 && u.TotalTokens == 0 && u.Source == ""
}

const (
	UsageSourceProvider = "provider"
	UsageSourceEstimate = "estimate"
)
```

**What to copy:** Add `UsageSourceSessionTranscript = "session-transcript"` (or
equivalent name) alongside the existing two constants — do not introduce a
parallel enum type. Decide in the plan whether `Measured()` widens to include
it (RESEARCH.md recommends yes, tracked separately from `provider`-grade) —
this is a named open design decision, not settled by research.

**billedTotal arithmetic to reuse verbatim, never re-derive** (`pkg/codex/usage.go:227-243`):
```go
// billedTotal sums the token counts a provider actually bills for.
//
// Anthropic reports input, cache_read and cache_creation as DISJOINT counts —
// total = input + cache_read + cache_creation. ... a 186x undercount.
func (u WorkerUsage) billedTotal() int64 {
	return u.InputTokens + u.CachedInputTokens + u.CacheCreationTokens + u.OutputTokens
}
```
Any new total this phase computes (ledger grand total, per-worker subtotal)
must route through this method or call it — never restate `input+output`.

**Error handling / tolerant parsing pattern to mirror in the new parsers**
(`pkg/codex/usage.go:80-113`):
```go
func ParseUsage(rawOutput string) (WorkerUsage, bool) {
	if strings.TrimSpace(rawOutput) == "" {
		return WorkerUsage{}, false
	}
	var found bool
	var usage WorkerUsage
	for _, line := range strings.Split(rawOutput, "\n") {
		line = strings.TrimSpace(stripANSIEscapeCodes(line))
		if line == "" || !strings.HasPrefix(line, "{") {
			continue
		}
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue // malformed line skipped, never fatal
		}
		if parsed, ok := usageFromEvent(event); ok {
			usage = parsed // later events supersede earlier — last statement wins
			found = true
		}
	}
	if !found {
		return WorkerUsage{}, false
	}
	usage.Source = UsageSourceProvider
	...
}
```
This "skip malformed line, never crash, last-wins" shape is exactly the
V5-input-validation discipline RESEARCH.md's Security Domain requires for the
new Claude-transcript and OpenCode-storage parsers.

---

### `pkg/codex/usage_test.go` (extend)

**Analog:** itself — `TestClaudeUsageTotalIncludesCacheReadAndCreation`
(`pkg/codex/usage_test.go:54-69`):
```go
func TestClaudeUsageTotalIncludesCacheReadAndCreation(t *testing.T) {
	raw := `{"type":"result","usage":{"input_tokens":50,"cache_read_input_tokens":100000,"cache_creation_input_tokens":2000,"output_tokens":500}}`
	usage, ok := ParseUsage(raw)
	if !ok {
		t.Fatal("expected usage from a cache-heavy result event")
	}
	if usage.CacheCreationTokens != 2000 {
		t.Errorf("cache_creation = %d, want 2000 — the field was not parsed at all", usage.CacheCreationTokens)
	}
	const want = 50 + 100000 + 2000 + 500
	if usage.TotalTokens != want {
		t.Errorf("total = %d, want %d; Anthropic's input/cache_read/cache_creation counts are disjoint",
			usage.TotalTokens, want)
	}
}
```

**What to copy:** The `want` constant is a **literal arithmetic expression of
externally-sourced numbers**, never a call back into `billedTotal()` or the
parser itself. Every new test this phase adds (ledger subtotal test, wrapper
parser test) must follow this exact shape — CLAUDE.md's Pitfall 1 names this
file's own git history as the reason the rule exists.

---

### `cmd/spend_ledger.go` (new — persistence, measured/estimated split, roll-up invariant)

**Analog:** `pkg/agent/spawn_tree.go` (`SpawnTree`) for the record-append shape
and parent-linkage field naming, backed by `pkg/storage.Store` for the actual
durable write (do not hand-roll file locking).

**Struct/field naming to mirror** (`pkg/agent/spawn_tree.go:28-47`):
```go
type SpawnEntry struct {
	Timestamp         string // Field 1: ISO 8601 UTC (2006-01-02T15:04:05Z)
	ActivityTimestamp string
	ParentName        string // Field 2: parent agent name
	Caste             string
	AgentName         string
	Task              string
	Depth             int
	Status            string
	Summary           string
}

type SpawnRun struct {
	ID        string `json:"id"`
	Command   string `json:"command"`
	StartedAt string `json:"started_at"`
	EndedAt   string `json:"ended_at,omitempty"`
	Status    string `json:"status"`
}
```
A ledger row should carry the same `ParentName`/`Caste`/`AgentName` vocabulary
so a row can join to a `SpawnEntry` by name without a translation layer — D-07
requires exactly this join, not a re-derivation of parent linkage.

**Record-append pattern to mirror, using `pkg/storage.Store.UpdateFile` (read-modify-write under one lock) instead of hand-rolled locking** (`pkg/agent/spawn_tree.go:234-265`):
```go
func (st *SpawnTree) RecordSpawn(parent, caste, name, task string, depth int) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if st.store == nil {
		return fmt.Errorf("spawn tree: store is nil")
	}
	if err := st.store.UpdateFile(st.filePath, func(existing []byte) ([]byte, error) {
		entries, completions, err := parseSpawnTreeBytes(existing)
		if err != nil {
			return nil, err // corrupt bytes stay on disk as evidence
		}
		entry := SpawnEntry{ /* ... */ }
		entries = append(entries, entry)
		return formatSpawnTreeLines(entries, completions), nil
	}); err != nil {
		return err
	}
	return st.reloadLocked()
}
```

**`pkg/storage.Store` methods available** (`pkg/storage/storage.go:41-266` — do not hand-roll any of these):
```go
func (s *Store) BasePath() string
func (s *Store) AtomicWrite(path string, data []byte) error
func (s *Store) UpdateFile(path string, mutate func(existing []byte) ([]byte, error)) error
func (s *Store) SaveJSON(path string, data interface{}) error
func (s *Store) LoadJSON(path string, dest interface{}) error
func (s *Store) LoadRawJSON(path string) ([]byte, error)
func (s *Store) SaveRawJSON(path string, data []byte) error
func (s *Store) AppendJSONL(path string, entry interface{}) error
func (s *Store) ReadJSONL(path string) ([]json.RawMessage, error)
func (s *Store) ReadFile(path string) ([]byte, error)
```
`AppendJSONL`/`ReadJSONL` are the most direct fit for "per-run ledger files,
kept modestly" (Claude's Discretion) — one JSONL row per worker dispatch,
append-only, no read-modify-write race.

**Run-boundary linkage to reuse, not re-derive** (`pkg/agent/spawn_tree.go:126-213`):
```go
func (st *SpawnTree) BeginRun(command string, startedAt time.Time) (SpawnRun, error)
func (st *SpawnTree) EndRun(runID, status string, endedAt time.Time) error
func (st *SpawnTree) CurrentRun() (SpawnRun, bool, error)
func (st *SpawnTree) EntriesForRun(runID string) ([]SpawnEntry, error)
```
SPEND-06's "for the current run" and SPEND-08's "this phase used" both map
directly onto `CurrentRun()`/`EntriesForRun()` — the run boundary already
exists; the ledger should key rows to the same `RunID`, not invent its own
run concept.

**JSON shape convention to mirror for any read-side rendering** (`pkg/agent/spawn_tree.go:724-732`):
```go
type spawnEntryJSON struct {
	Name        string `json:"name"`
	Parent      string `json:"parent"`
	Caste       string `json:"caste"`
	Task        string `json:"task"`
	Status      string `json:"status"`
	SpawnedAt   string `json:"spawned_at"`
	CompletedAt string `json:"completed_at"`
}
```

---

### `cmd/spend_ledger_test.go` (new)

**Analogs:** `cmd/consolidation_dryrun_test.go` (fixture-seeding + hash-based
purity assertion) for SPEND-01 durability, and `cmd/caste_relevance_test.go`'s
`TestQueenOrchestratePreservesSafetyCastes` for the invariant-style shape
SPEND-05's "grand total equals sum of rows" needs — assert the property, not
that a field is present (CLAUDE.md Definition of Done corollary).

**Fixture-seed pattern to copy** (`cmd/consolidation_dryrun_test.go:34-79`):
```go
func seedConsolidationFixture(t *testing.T) (dataDir string) {
	t.Helper()
	s, _ := newTestStore(t)
	store = s
	dataDir = s.BasePath()
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{ /* ... */ }); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}
	return dataDir
}
```
Use the same `newTestStore(t)` + package-level `store = s` swap for
`cmd/spend_ledger_test.go`'s own fixture setup — this is the standing test
harness convention in `cmd/`, not something to reinvent.

**Hash-based mutation-detection helper to reuse for SPEND-01's "readable after
the process exits" test** (`cmd/consolidation_dryrun_test.go:22-32`):
```go
func hashFileForTest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "absent"
	}
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
```

---

### `cmd/spend_cmd.go` (new — `aether spend` Cobra command)

**Analog:** `cmd/proof_cmd.go` (`proofCmd`) — closest existing read-only
inspection command: loads state, builds a report struct, renders JSON +
plain-English visual, touches nothing.

**Command + RunE pattern to copy** (`cmd/proof_cmd.go:83-96`):
```go
var proofCmd = &cobra.Command{
	Use:   "proof",
	Short: "Inspect runtime context proof and skill proof for the active colony",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		state, err := loadActiveColonyState()
		if err != nil {
			outputError(1, colonyStateLoadMessage(err), nil)
			return nil
		}
		result := buildProofOutput(skillWorkspaceRoot(), state)
		outputWorkflow(result, renderProofVisual(state, result))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(proofCmd)
}
```
`spendCmd` should follow this exact shape: `Use: "spend"`, `Args:
cobra.NoArgs` (or a bounded flag set for filters like `--include-estimates`),
a `buildSpendOutput(...)` report builder, and `outputWorkflow(result,
renderSpendVisual(result))`. Register with `rootCmd.AddCommand(spendCmd)` in
its own `init()`, same as every other top-level subcommand in `cmd/` (see
`internal_worker_adapter.go:101-106` and `ceremony_cmd.go:127-148` for two
more instances of the same registration idiom).

---

### `cmd/spend_cmd_test.go` (new — idempotency/purity test)

**Analog:** `cmd/consolidation_dryrun_test.go`'s `assertConsolidationDryRunIsPure`
— the exact precedent CLAUDE.md and RESEARCH.md both name for D-04.

**Pattern to copy directly** (`cmd/consolidation_dryrun_test.go:81-132`):
```go
func assertConsolidationDryRunIsPure(t *testing.T, command string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	dataDir := seedConsolidationFixture(t)
	watched := []string{
		filepath.Join(dataDir, "instincts.json"),
		// ... every file the command could plausibly touch
	}
	before := make(map[string]string, len(watched))
	for _, p := range watched {
		before[p] = hashFileForTest(t, p)
	}

	rootCmd.SetArgs([]string{command, "--dry-run"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("%s --dry-run failed: %v", command, err)
	}

	for _, p := range watched {
		if after := hashFileForTest(t, p); after != before[p] {
			t.Errorf("%s --dry-run mutated %s (before=%s after=%s)", command, filepath.Base(p), before[p][:12], after[:12])
		}
	}
	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("%s --dry-run not ok: %s", command, buf.String())
	}
}
```
Write an `assertSpendCommandIsIdempotent(t, ...)` in the same shape: run
`aether spend` twice against the same seeded ledger, hash the ledger file (and
any other file it could touch) before/after each run, and assert byte-identity
across both runs — not just "no mutation," but literally
`TestSpendCommandIsIdempotent` per the Validation Architecture's named test.
Note the companion negative test this file also proves is worth writing:
`TestConsolidationRealRunStillMutates` shows the harness is not vacuously
green by confirming a real (mutating) counterpart still changes the hash —
consider the same "prove the harness can detect a mutation" companion test if
`aether spend` ever gains a write-side counterpart.

---

### `cmd/wrapper_usage_claude.go` (new — parse Claude Code session JSONL)

**Analog:** `pkg/codex/usage.go`'s `ParseUsage`/`usageFromEvent` for the
tolerant line-by-line JSON decode shape; `cmd/hook_cmds.go`'s `claudeHookInput`
for the payload-struct-with-comments convention.

**Payload struct convention to mirror when adding `TranscriptPath`**
(`cmd/hook_cmds.go:16-35`):
```go
type claudeHookInput struct {
	HookEventName string `json:"hook_event_name"`
	ToolName      string `json:"tool_name"`
	// ...
	// AgentID, AgentType, and SessionID are Phase 173 (SPAWN-04) additions.
	// These field names come from current official Claude Code hooks
	// documentation and are a hypothesis, not a confirmed contract — Task 3
	// of the 173-01 plan runs a real nested dispatch and records the actual
	// observed field names in 173-HOOK-FINDINGS.md. ...
	AgentID   string `json:"agent_id"`
	AgentType string `json:"agent_type"`
	SessionID string `json:"session_id"`
}
```
Add `TranscriptPath string \`json:"transcript_path"\`` the same way — a plain
string field, empty-string-safe decode, with a comment documenting that its
exact contract was empirically confirmed (not assumed) per
`173-HOOK-FINDINGS.md`.

**Line-scan tolerant decode shape to reuse for the JSONL transcript** (same
`ParseUsage` excerpt as above under `pkg/codex/usage.go`) — use `bufio.Scanner`
per RESEARCH.md's Standard Stack (transcripts observed up to 600+ KB; do not
`os.ReadFile` the whole thing), skip malformed lines, never fail the whole
parse on one bad line.

**Path-boundary validation to reuse before opening `transcript_path`**
(`cmd/codex_build_finalize.go:1629-1676`, `validateAndNormalizeClaimPathToRoot`):
```go
func validateAndNormalizeClaimPathToRoot(root, field, claimed string) (string, error) {
	claimed = strings.TrimSpace(claimed)
	if claimed == "" {
		return "", nil
	}
	if strings.ContainsRune(claimed, 0) {
		return "", wrapClaimPathRule(claimPathRuleNullByte, fmt.Errorf("invalid %s claim %q: path contains a null byte", field, claimed))
	}
	// ... IsAbs / .. traversal checks, symlink-eval, existence check
}
```
RESEARCH.md's Security Domain explicitly names this function as the pattern to
mirror for validating `transcript_path` stays under the expected
`~/.claude/projects/` root before it is opened. The new validator will differ
in its expected root (a `$HOME`-relative directory, not the repo root), but
the null-byte check, `IsAbs`/traversal rejection, and symlink-eval-then-check
shape should be copied structurally.

**Correlation key discipline (Pitfall 3, structural not textual):** match on
`tool_use_id` first (exact), fall back to the `{emoji} {Caste} {name}: {task}`
description text only as a secondary signal — see RESEARCH.md's Pattern 2 for
the two real observed transcript shapes (foreground `tool_result` vs async
`<task-notification>`) that the parser must handle.

---

### `cmd/wrapper_usage_claude_test.go` / `cmd/wrapper_usage_opencode_test.go` (new)

**Analog:** `pkg/codex/usage_test.go` — literal, externally-sourced expected
values (see excerpt under `usage_test.go` above). RESEARCH.md's Validation
Architecture requires a **committed fixture file** (trimmed/redacted real
transcript shape) rather than depending on this machine's live home
directory — CI will not have `~/.claude/projects/` or
`~/.local/share/opencode/storage/`.

---

### `cmd/wrapper_usage_opencode.go` (new — discover + parse OpenCode session storage)

**Analog (parsing):** same `ParseUsage`-style tolerant decode as Claude's
parser. **Analog (discovery-by-time-window):** `pkg/agent/spawn_tree.go`'s run
boundary — `filterEntriesForRun` bounds spawn entries to
`[run.StartedAt, run.EndedAt]`; the OpenCode session discovery needs the same
"bound to the current run's time window, fall back to `estimate` if
ambiguous" shape (RESEARCH.md Pitfall 4). No exact discovery-by-heuristic
analog exists elsewhere in the codebase — flagged below under No Analog
Found.

---

### `cmd/hook_cmds.go` (extend — persist `TranscriptPath` once per run)

**Analog:** itself, `claudeHookInput` (see excerpt above) for the field
addition. The *persistence* side (writing the captured path to a durable
per-run file so `build-finalize` can read it later) has no direct precedent —
`captureRawHookPayload` is the nearest existing mechanism but is an **opt-in
debug capture gated by an environment variable**, not the real delivery path:

```go
// captureRawHookPayload appends the raw stdin bytes received by a hook to
// the file named by AETHER_HOOK_CAPTURE_FILE, when that environment variable
// is set. This is Phase 173 (SPAWN-04) Wave 0's opt-in evidence recorder...
// It is off by default (empty env var short-circuits immediately) and every
// error path returns silently -- a capture failure must never affect the
// hook's allow/deny answer (T-173-02).
func captureRawHookPayload(raw []byte) {
	path := strings.TrimSpace(os.Getenv("AETHER_HOOK_CAPTURE_FILE"))
	if path == "" {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(raw)
	_, _ = f.Write([]byte("\n"))
}
```
Do not reuse this function itself (it is explicitly the opt-in debug path,
and its own comment documents why a home-directory-writable switch was
removed as a security finding in 173-REVIEW.md CR-01). The new persistence
must write through `pkg/storage.Store` under the ledger's own
Go-owned run directory, matching the "never a plain env-var/home-dir file a
worker can aim" discipline that removal established, and must fail soft (a
missing/unwritable transcript path must never fail the hook's allow/deny
decision — same non-blocking discipline `captureRawHookPayload` already
demonstrates).

---

### `cmd/internal_worker_adapter.go` (extend — `mapInternalWorkerResult` copies `Usage`)

**Analog:** itself — one-field addition to an existing mapping function.

**Current gap, exact fix site** (`cmd/internal_worker_adapter.go:474-499`):
```go
func mapInternalWorkerResult(result codex.WorkerResult, invokeErr error) *internalWorkerResult {
	errorText := ""
	if result.Error != nil {
		errorText = sanitizeInternalWorkerAdapterError(result.Error.Error())
	} else if invokeErr != nil {
		errorText = sanitizeInternalWorkerAdapterError(invokeErr.Error())
	}
	return &internalWorkerResult{
		Name:          result.WorkerName,
		Caste:         result.Caste,
		TaskID:        result.TaskID,
		Status:        result.Status,
		Summary:       strings.TrimSpace(result.Summary),
		FilesCreated:  append([]string(nil), result.FilesCreated...),
		FilesModified: append([]string(nil), result.FilesModified...),
		TestsWritten:  append([]string(nil), result.TestsWritten...),
		Artifacts:     result.Artifacts,
		ScoutReport:   result.ScoutReport,
		ToolCount:     result.ToolCount,
		Blockers:      append([]string(nil), result.Blockers...),
		Spawns:        append([]string(nil), result.Spawns...),
		Duration:      result.Duration.Seconds(),
		Error:         errorText,
		Handoff:       result.Handoff,
		// Usage: result.Usage,  <-- ADD THIS: the field exists on codex.WorkerResult
		//                           and is silently dropped today (SPEND-01 gap)
	}
}
```
Also add `Usage codex.WorkerUsage \`json:"usage,omitempty"\`` to the
`internalWorkerResult` struct itself (`cmd/internal_worker_adapter.go:48-65`,
alongside the existing `ToolCount int \`json:"tool_count,omitempty"\`` field —
copy that field's `omitempty` convention).

**Bounded-input precedent to reuse if the new parsers need a size cap**
(`cmd/internal_worker_adapter.go:20-24`):
```go
const (
	internalWorkerAdapterSchemaVersion = 1
	internalWorkerRequestMaxBytes      = 2 << 20
	internalWorkerTimeoutMax           = 60 * time.Minute
)
```
RESEARCH.md's Security Domain names this exact constant as the precedent for
bounding the transcript-file read against resource exhaustion (Denial of
Service row in Known Threat Patterns).

---

### `cmd/codex_build_finalize.go` (extend — `codexExternalBuildWorkerResult` gains `Usage`)

**Analog:** itself — struct field addition, same as `internal_worker_adapter.go`.

**Current gap, exact struct** (`cmd/codex_build_finalize.go:61-90`):
```go
type codexExternalBuildWorkerResult struct {
	Stage         string              `json:"stage,omitempty"`
	Wave          int                 `json:"wave,omitempty"`
	ExecutionWave int                 `json:"execution_wave,omitempty"`
	Caste         string              `json:"caste,omitempty"`
	Name          string              `json:"name"`
	AntName       string              `json:"ant_name,omitempty"`
	Task          string              `json:"task,omitempty"`
	Status        string              `json:"status"`
	Summary       string              `json:"summary,omitempty"`
	TaskID        string              `json:"task_id,omitempty"`
	TaskIndex     int                 `json:"task_index,omitempty"`
	DependsOn     []string            `json:"depends_on,omitempty"`
	Outputs       []string            `json:"outputs,omitempty"`
	Blockers      []string            `json:"blockers,omitempty"`
	Duration      float64             `json:"duration,omitempty"`
	ToolCount     int                 `json:"tool_count,omitempty"`
	FilesCreated  []string            `json:"files_created,omitempty"`
	FilesModified []string            `json:"files_modified,omitempty"`
	TestsWritten  []string            `json:"tests_written,omitempty"`
	Handoff       codex.WorkerHandoff `json:"handoff,omitempty"`
	// Usage codex.WorkerUsage `json:"usage,omitempty"`  <-- ADD THIS, per D-06
	//   this must be populated from the independently-read transcript/session
	//   artifact, NEVER from a value the wrapper JSON itself asserts — an LLM
	//   typing a number into this field is exactly the D-06 violation.
}
```
Per D-06 and Pattern 1 (Anti-Patterns), this field must **never** be trusted
if a wrapper-path completion packet sets it directly — the Go runtime must
independently resolve it from the transcript/session artifact keyed by
`tool_use_id`/worker name, and any value present in the raw external JSON for
this field should be ignored or explicitly rejected, mirroring how
`validateExternalResultIdentity` (line 1191) already refuses to trust
externally-asserted identity fields without cross-checking them against the
issued dispatch.

**Existing structural-input validation entry point to extend, if the ledger
needs its own contract violation category** (`cmd/codex_build_finalize.go:918`,
`validateCompletionPacketSemantics`) — follow the existing
`completionContractError`/`contractViolation` accumulation pattern rather than
returning a bare `error` for any new "usage claim was ignored" reporting.

---

### `cmd/ceremony_cmd.go` (extend — `renderCeremonyCloseoutVisual` gains the token line)

**Analog:** itself — insertion point is between `writeCeremonyCloseoutNotice`
and `writeCeremonyWorkerSummary`.

**Exact insertion point** (`cmd/ceremony_cmd.go:582-584`):
```go
writeCeremonyCloseoutNotice(&b, result)
writeCeremonyWorkerSummary(&b, result)   // <-- add a writeCeremonySpendLine(&b, result) call near here
writeCeremonyPlanSummary(&b, result)
```

**Sibling writer function to mirror in shape and tone** (`cmd/ceremony_cmd.go:734-751`):
```go
func writeCeremonyWorkerSummary(b *strings.Builder, result map[string]interface{}) {
	workers := mapSliceValue(result["completion_workers"])
	completed := intValue(result["completion_completed"])
	blocked := intValue(result["completion_blocked"])
	failed := intValue(result["completion_failed"])
	total := intValue(result["completion_worker_count"])
	if total == 0 && len(workers) > 0 {
		total = len(workers)
	}
	if total > 0 {
		fmt.Fprintf(b, "\nWorkers: %d completed  %d blocked  %d failed  (%d total)\n", completed, blocked, failed, total)
	}
	toolCount := 0
	for _, worker := range workers {
		toolCount += intValue(worker["tool_count"])
	}
	if toolCount > 0 {
		fmt.Fprintf(b, "Tools: %d calls across workers\n", toolCount)
	}
	// ...
}
```
Write `writeCeremonySpendLine(b, result)` in the same shape: pull a
pre-computed field off `result` (e.g. `result["spend_total_tokens"]`,
`result["spend_measured"]` bool) — **do not compute the figure inside the
render function**; `renderCeremonyCloseout` (the caller, line 214-261) is
where `result[...]` gets populated from loaded state today, so the ledger
read/roll-up belongs there, keeping the render function a pure formatter per
the UX Architecture ownership model ("wrapper markdown just renders what Go
returns" — here the equivalent split is compute-in-`renderCeremonyCloseout`,
format-in-`renderCeremonyCloseoutVisual`). One plain-English line, tokens only
(D-01), e.g. `"This phase used ~12,400 tokens (measured)"` or `"...(partly
estimated)"` — Claude's Discretion on exact mixed-run wording.

---

## Shared Patterns

### Cobra subcommand registration
**Source:** `cmd/internal_worker_adapter.go:101-106`, `cmd/ceremony_cmd.go:127-148`, `cmd/proof_cmd.go:100-102`
**Apply to:** `cmd/spend_cmd.go`
```go
func init() {
	rootCmd.AddCommand(spendCmd)
}
```
Every top-level subcommand in `cmd/` registers itself this way in its own
file's `init()`. There is no central command table in `cmd/aether/main.go` to
edit — `cmd/main.go` does not exist; the binary entrypoint is
`cmd/aether/main.go`, and registration is fully decentralized per-file.

### Durable JSON/JSONL persistence with locking
**Source:** `pkg/storage/storage.go:141-266` (`SaveJSON`, `LoadJSON`, `AppendJSONL`, `ReadJSONL`)
**Apply to:** `cmd/spend_ledger.go`
```go
func (s *Store) AppendJSONL(path string, entry interface{}) error
func (s *Store) ReadJSONL(path string) ([]json.RawMessage, error)
```
Never hand-roll file locking or atomic writes — `pkg/storage.Store` already
owns this discipline and every other durable colony file goes through it.

### Idempotency / dry-run purity testing
**Source:** `cmd/consolidation_dryrun_test.go:81-132` (`assertConsolidationDryRunIsPure`)
**Apply to:** `cmd/spend_cmd_test.go` (SPEND-06, D-04)
See full excerpt above under `cmd/spend_cmd_test.go`. This is a named
CLAUDE.md precedent — do not write a new idempotency-test helper from
scratch.

### Literal, externally-sourced test expectations (never restate the parser under test)
**Source:** `pkg/codex/usage_test.go:54-69`
**Apply to:** every new test file in this phase (`spend_ledger_test.go`,
`wrapper_usage_claude_test.go`, `wrapper_usage_opencode_test.go`,
`spend_cmd_test.go`'s figures if any)
```go
const want = 50 + 100000 + 2000 + 500 // literal, not usage.billedTotal()
```

### Path-boundary validation before reading an untrusted path
**Source:** `cmd/codex_build_finalize.go:1629-1676` (`validateAndNormalizeClaimPathToRoot`)
**Apply to:** `cmd/wrapper_usage_claude.go` (validating `transcript_path` stays
under `~/.claude/projects/`), `cmd/wrapper_usage_opencode.go` (validating
discovered session paths stay under
`~/.local/share/opencode/storage/`)

### Bounded-size untrusted input read
**Source:** `cmd/internal_worker_adapter.go:22` (`internalWorkerRequestMaxBytes = 2 << 20`)
**Apply to:** any transcript/session file read — combine with
`bufio.Scanner` line-scanning per RESEARCH.md's Standard Stack, never
`os.ReadFile` a transcript wholesale.

### Caste identity rendering (emoji + colored label)
**Source:** `cmd/codex_visuals.go:3808-3846` (`casteEmoji`, `casteLabel`, `casteIdentity`)
```go
func casteIdentity(caste string) string {
	return casteEmoji(caste) + " " + colorizeCaste(caste, casteLabel(caste))
}
```
**Apply to:** `aether spend`'s detail view (D-05: "workers appear under their
colony identity") and any per-worker line the closeout token summary might
list — call `casteIdentity(caste)` rather than re-deriving emoji/label maps.

### Grand-total-equals-sum-of-rows invariant test shape
**Source:** `cmd/caste_relevance_test.go:395` (`TestQueenOrchestratePreservesSafetyCastes`)
**Apply to:** `cmd/spend_ledger_test.go`'s SPEND-05 test
Assert the property algebraically across generated/varied inputs, not that a
specific field exists — CLAUDE.md's "Proportion/invariant tests preferred
over 'section exists' tests" corollary, restated directly in RESEARCH.md's
Phase Requirements for SPEND-05.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `cmd/wrapper_usage_opencode.go`'s session-discovery-by-time-window logic | utility | file-I/O, transform | No existing code discovers an artifact by matching a directory tree plus a time window with no first-class identifier to anchor on. `pkg/agent/spawn_tree.go`'s `filterEntriesForRun` bounds *already-tagged* entries to a time window — it does not *discover* untagged files by scanning a directory and matching timestamps. This is genuinely new machinery; build it narrowly (bound to `SpawnRun.StartedAt`/`EndedAt`, fall back to `estimate` on ambiguity per Pitfall 4) rather than searching further for a non-existent precedent |
| `cmd/hook_cmds.go`'s durable (non-debug) persistence of `TranscriptPath` once per run | middleware (hook → durable store) | event-driven → file-I/O | `captureRawHookPayload` is the nearest neighbor but is explicitly an opt-in debug-only capture path (gated by an env var, silent-fail on every error) — not a real delivery mechanism. The real path (hook fires → Go runtime persists `TranscriptPath` to a ledger-adjacent, Go-owned file → `build-finalize` reads it back) has no existing precedent to copy structurally beyond "use `pkg/storage.Store`, fail soft, never let a missing path fail the hook's allow/deny decision" |

## Metadata

**Analog search scope:** `pkg/codex/`, `pkg/agent/`, `pkg/storage/`, `cmd/` (targeted: `*_cmd.go`, `hook_cmds.go`, `codex_build_finalize.go`, `internal_worker_adapter.go`, `ceremony_cmd.go`, `codex_visuals.go`, `consolidation_dryrun_test.go`, `caste_relevance_test.go`, `proof_cmd.go`)
**Files scanned:** 13 read directly (full or targeted ranges); ~15 more located via `grep`/`wc -l` to confirm structure without reading in full
**Pattern extraction date:** 2026-08-13
