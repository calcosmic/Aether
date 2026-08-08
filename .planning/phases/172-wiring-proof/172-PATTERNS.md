# Phase 172: Wiring Proof - Pattern Map

**Mapped:** 2026-08-08
**Files analyzed:** 6 (2 new, 4 modified)
**Analogs found:** 6 / 6

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `cmd/subcommand_reachability_ratchet_test.go` (NEW) | test (static analysis) | batch/transform (AST walk + corpus diff) | `cmd/visual_writer_discipline_test.go` (AST walker half) + `cmd/command_call_audit_test.go` (corpus-scan half) + `cmd/regression_test.go` (baseline-JSON-comparison half) | role-match (composite — three partial analogs, no single exact one) |
| `cmd/testdata/orphan_allowlist.json` (NEW) | config (checked-in test fixture / data) | batch (static baseline) | `cmd/testdata/command_catalog.json` (array-of-objects shape) + `cmd/testdata/regression_snapshot.json` (golden-file role) | role-match |
| `cmd/spawn.go` (MODIFY: `spawnCanSpawnCmd`) | controller (cobra command) | request-response | `cmd/spawn.go:spawnLogCmd` (sibling command, same file) | exact (same file, same command family) |
| `cmd/cli_flag_audit_test.go` (MODIFY: `skipSubcommands` guard) | test (static analysis) | batch/transform | `cmd/regression_test.go` (`loadRegressionSnapshot` + count comparison) | role-match |
| `cmd/command_call_audit_test.go` (MODIFY: `auditedCorpora`) | test (static analysis) | batch/transform | itself — `auditedCorpora` var declaration, same file, lines 40-46 | exact (in-place extension, not a new pattern) |
| `.github/workflows/ci.yml` (MODIFY: add named step(s)) | config (CI pipeline) | batch | itself — "Verify command catalog classification" step, lines 96-97 | exact (in-place extension) |

## Pattern Assignments

### `cmd/subcommand_reachability_ratchet_test.go` (NEW — test, batch/transform)

No single existing file does this end-to-end; it is assembled from three analogs, each covering one third of the job. Read all three before writing this file.

**Analog 1 — AST var-block enumeration:** `cmd/visual_writer_discipline_test.go`

**Imports** (lines 1-13):
```go
package cmd

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)
```

**Core pattern — the `declNameAndBody` walker** (lines 61-85, copy verbatim, it is already exactly what WIRE-01 needs):
```go
// declNameAndBody returns a stable name for a top-level declaration and a node
// to search under it. Function declarations give their own name; a var block
// gives its first declared name, which is how cobra commands are written
// (`var pheromoneDisplayCmd = &cobra.Command{...}`).
func declNameAndBody(decl ast.Decl) (string, ast.Node) {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Body == nil {
			return "", nil
		}
		return d.Name.Name, d.Body
	case *ast.GenDecl:
		if d.Tok != token.VAR {
			return "", nil
		}
		for _, spec := range d.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 {
				continue
			}
			return vs.Names[0].Name, vs
		}
	}
	return "", nil
}
```

**Enumeration driver shape** (lines 96-160, adapt — walk `cmd/*.go` non-test files, parse with `parser.ParseFile`, iterate `file.Decls`, call `declNameAndBody`, then `ast.Inspect` the body looking for `&cobra.Command{...}` composite literals instead of `fmt.Fprint*` calls):
```go
entries, err := os.ReadDir(".")
...
for _, entry := range entries {
	name := entry.Name()
	if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
		continue
	}
	file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
	...
	for _, decl := range file.Decls {
		enclosing, body := declNameAndBody(decl)
		if body == nil {
			continue
		}
		ast.Inspect(body, func(inner ast.Node) bool { ... })
	}
}
```
Per RESEARCH.md Pattern 1: extract the `Use:` string literal from the `*ast.CompositeLit` under `&cobra.Command{...}` to get the real command name (first whitespace-delimited token of `Use`, since some carry argument hints like `"skill-match [role] [task]"`). Cross-check against `rootCmd.Commands()`/`rootCmd.Find()` (see `cmd/cli_flag_audit_test.go:45-59` below) to also catch anonymous inline `AddCommand(&cobra.Command{...})` registrations that have no separate `var` declaration (e.g. `cmd/host_cmd.go:31-63`).

**Analog 2 — the three-corpus caller search:** `cmd/command_call_audit_test.go`

**Corpus list to reuse the shape of** (lines 40-46) — do NOT reuse this list itself (it is WIRE-03's flag-audit corpus, which includes doc-only trees like `command-playbooks/` that D-02 explicitly excludes as *caller* evidence). Copy the *shape* — a `[]string` of `filepath.Join(...)` directory trees — for the ratchet's own, narrower list of D-01(b) permitted caller corpora: `.claude/commands/ant`, `.opencode/commands/ant`, `.aether/commands`. `command-playbooks/` and `colony/playbooks/` must be excluded here per D-02/D-06 even though `command_call_audit_test.go` includes them for the different (flag-audit) purpose:
```go
var auditedCorpora = []string{
	filepath.Join(".claude", "commands", "ant"),
	filepath.Join(".opencode", "commands", "ant"),
	filepath.Join(".aether", "commands"),
	filepath.Join(".aether", "docs", "command-playbooks"),
	filepath.Join("colony", "playbooks"),
}
```

**Directory walk + name extraction, reuse near-verbatim for "does this corpus mention command X as an invocation or menu entry":** `collectDocumentedCalls` (lines 176-199) and `extractDocumentedCalls` (lines 76-141), including the fence-detection and backtick-detection logic (`backtickCallRe`, lines 63-73) — this is the established, working way to find `aether <name>` invocations in markdown/YAML without a hand-rolled regex-only scan (see RESEARCH.md "Don't Hand-Roll"). A menu entry naming the command (D-03) counts as caller evidence, so the ratchet's own extraction can be looser than the flag-audit's: a bare mention of the command name in a fenced block or as a wrapper's declared target is sufficient — it does not need full `validateCallAgainstCobra` arity validation, only "was this name referenced as something to execute."

**Token-boundary safety (Pitfall 2 — critical, do not skip):** never use `strings.Contains`/naive grep to check whether a corpus mentions a command name; `"skill-list"` is a literal substring of `"skill-list-lifecycle"` (`cmd/skill_lifecycle.go:125`). Reuse `tokenizeShellLike` (lines 243-280) and compare tokens exactly, the same way `validateCallAgainstCobra` does.

**Fixture self-test pattern (Pitfall 6 — load-bearing, must be copied exactly):** `TestAuditDetectsPositionalDrift` (lines 411-416):
```go
// Source: cmd/command_call_audit_test.go:411-416 (existing, verified pattern — reuse this
// shape for WIRE-01's own self-test proving the ratchet detects a synthetic orphan)
noArgsCmd := &cobra.Command{Use: "audit-selftest-noargs", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) {}}
rootCmd.AddCommand(noArgsCmd)
defer rootCmd.RemoveCommand(noArgsCmd)
```
The fixture command must be registered and removed inside the test function body — never at package scope — so it never becomes a real permanent orphan requiring its own baseline entry.

**Analog 3 — checked-in JSON baseline comparison:** `cmd/regression_test.go` (closest existing "read a `testdata/*.json` golden file and compare against live-computed state" pattern in this repo; note it is an *exact-match* comparator, not shrink-only — the shrink-only diff logic itself has no precedent and must be built fresh per RESEARCH.md).

**Baseline-read pattern to copy the shape of** (lines 45-57):
```go
// loadRegressionSnapshot reads the master regression golden file.
func loadRegressionSnapshot(t *testing.T) *RegressionSnapshot {
	t.Helper()
	data, err := os.ReadFile("testdata/regression_snapshot.json")
	if err != nil {
		t.Fatalf("read regression_snapshot.json: %v (run with -update-golden to create)", err)
	}
	var snap RegressionSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatalf("parse regression_snapshot.json: %v", err)
	}
	return &snap
}
```
Adapt for `orphan_allowlist.json`: read the committed baseline into a `[]OrphanEntry{Name, Reason, OwnerPhase string}`, then for `TestOrphanAllowlistOnlyShrinks`, assert every name in the *live* computed orphan set that needs baseline coverage is present in the baseline (fails on growth), and — per D-10 — do NOT compare counts (a one-out-one-in swap must fail) and do NOT compare against git history. The comparison must be a set-membership diff: any live name absent from the baseline is a failure naming that command.

**Failure-message shape to copy** (from `command_call_audit_test.go` lines 401-405, "name the command" convention):
```go
if len(violations) > 0 {
	t.Errorf("%d documented CLI call(s) violate the command's real argument contract:\n  %s",
		len(violations), strings.Join(violations, "\n  "))
}
```
WIRE-01's ratchet failure must name the orphaned command directly (per CONTEXT.md's "the orphan ratchet must name the command" requirement) — not "N orphans found," but each name listed.

---

### `cmd/testdata/orphan_allowlist.json` (NEW — config, batch)

**Analog:** `cmd/testdata/command_catalog.json` (array-of-objects shape) and the golden-file role of `cmd/testdata/regression_snapshot.json`.

**Shape to copy** (structured fields, not one free-text string, per RESEARCH.md Open Question 3 recommendation and D-08's requirement for a queryable owner phase):
```json
[
  {
    "name": "skill-index",
    "reason": "skill-lifecycle",
    "owner_phase": "178"
  }
]
```
Do NOT store this as a field on `cmd/testdata/command_catalog.json` (D-09) — that file is rewritten in place by `scripts/classify_commands.py:191` (`json.dump`), which would silently erase allowlist entries on a routine regeneration.

**Seeding note (Pitfall 1 — do not hardcode):** do not hand-write "8 skill-* entries." Build the scanner first, run it, and write whatever it actually reports (D-07). Research found strong evidence the honest count is 6, not 8 (`skill-parse-frontmatter` and `skill-cache-rebuild` have real callers via `.claude/commands/ant/skill-create.md:220-242`'s fenced "Run using the Bash tool" instructions) — but trust the scanner's own output over this research note if they disagree.

---

### `cmd/spawn.go` — `spawnCanSpawnCmd` (MODIFY — controller, request-response)

**Analog:** sibling command in the same file, `spawnLogCmd` (lines 13-79), for the `outputError`/exit-code convention; and the current broken command itself (lines 169-182).

**Current state** (lines 169-182, exact, to be replaced):
```go
var spawnCanSpawnCmd = &cobra.Command{
	Use:   "spawn-can-spawn",
	Short: "Check if spawning is allowed at given depth",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		depth := mustGetInt(cmd, "depth")

		outputOK(map[string]interface{}{
			"can_spawn": true,
			"depth":     depth,
		})
		return nil
	},
}
```

**Error-envelope / exit-code convention to reuse, from `spawnLogCmd`** (lines 33-36, the pattern `--enforce`'s deny path must follow):
```go
if parent == "" {
	outputError(1, "flag --parent is required", nil)
	return nil
}
```
`outputError` (defined `cmd/helpers.go:35-56`) calls `markRenderedCommandError(code)` (`cmd/root.go:219`), which is what drives the non-zero process exit at `cmd/root.go:245`/`250` — this is the established, repo-wide convention for a command that must signal failure via exit code while still returning `nil` from `RunE` (so cobra does not print its own usage text, which is not this repo's error-output convention). Do not call `os.Exit` directly and do not return a Go `error` from `RunE`.

**`init()` registration convention to extend** (lines 357-383, specifically line 370 and the flag block around it):
```go
spawnCanSpawnCmd.Flags().Int("depth", 0, "Spawn depth to check (required)")
```
Add `spawnCanSpawnCmd.Flags().Bool("enforce", false, "...")` alongside this line, in the same `init()` func, following the existing per-command flag registration block style used for every other command in this file.

**Target shape (from RESEARCH.md Code Examples, already adapted to this repo's conventions — implement this):**
```go
var spawnCanSpawnCmd = &cobra.Command{
	Use:   "spawn-can-spawn",
	Short: "Check if spawning is allowed at given depth",
	Args:  cobra.MaximumNArgs(1), // D-14: accept the positional depth workers.md:292 sends
	RunE: func(cmd *cobra.Command, args []string) error {
		depth := mustGetInt(cmd, "depth")
		if len(args) == 1 {
			if parsed, err := strconv.Atoi(args[0]); err == nil {
				depth = parsed // positional wins when both forms are present
			}
		}
		enforce, _ := cmd.Flags().GetBool("enforce")
		canSpawn := true // Phase 173 (SPAWN-01) makes this a real decision

		if enforce && !canSpawn {
			outputError(1, fmt.Sprintf("spawn denied at depth %d", depth), nil)
			return nil
		}
		outputOK(map[string]interface{}{
			"can_spawn": canSpawn,
			"depth":     depth,
		})
		return nil
	},
}
```
Note both existing calling conventions must keep working: `.aether/workers.md:292` sends `aether spawn-can-spawn {your_depth} --enforce` (positional), while `.aether/docs/command-playbooks/build-full.md:854` and `build-wave.md:675` send `aether spawn-can-spawn --depth {depth}` (flag, no positional, no `--enforce`) — `cobra.MaximumNArgs(1)` accepts both (0 or 1 positionals).

---

### `cmd/cli_flag_audit_test.go` — `skipSubcommands` shrink-only guard (MODIFY — test, batch/transform)

**Analog:** the existing map itself (lines 37-40) plus `cmd/regression_test.go`'s baseline-comparison shape (same pattern reused at smaller scale — no new file needed, just a companion test function in the same file or the new ratchet file).

**Current unguarded map** (lines 21-40, exact):
```go
func TestCLIFlagAudit(t *testing.T) {
	markdownDirs := []string{
		"../.claude/commands/ant/",
		"../.opencode/commands/ant/",
		"../.aether/docs/command-playbooks/",
	}

	re := regexp.MustCompile(`aether\s+([\w][\w-]*)\s+((?:--[\w][\w-]*(?:=\S*|\s+\S*)?\s*)*)`)

	// Subcommands known to be called from markdown but intentionally not
	// registered as direct subcommands (they are shell-only, aliases, or
	// handled by other mechanisms).
	skipSubcommands := map[string]bool{
		"verify-castes":     true, // markdown-only command, no Go subcommand
		"pending-decisions": true, // playbook shorthand; actual Go subcommands are pending-decision-{add,list,resolve}
	}
```
Per D-12, this 2-entry map needs the same shrink-only treatment as `orphan_allowlist.json` (D-10/D-11): move it to (or mirror it in) a committed baseline comparable at test time, so a silent third entry cannot be added without the test catching it. Reuse the same comparator built for `TestOrphanAllowlistOnlyShrinks` rather than inventing a second mechanism — this is explicitly the same discipline applied to a second, previously-unguarded list.

---

### `cmd/command_call_audit_test.go` — `auditedCorpora` extension (MODIFY — test, batch/transform)

**Analog:** itself. This is an in-place data extension — but **not** a standalone one.

> **⚠ CORRECTION (added during plan revision).** This section previously described the corpus
> addition as sufficient "with zero other code changes", inheriting that claim from RESEARCH.md
> Pattern 2. Both were wrong. `tokenizeShellLike` collapses `result=$(aether` (the shape at
> `.aether/workers.md:292`) into a single token matching neither `aether` nor `*/aether`, so
> the line is discarded before validation ever runs. The corpus extension must be preceded by
> the tokenizer fix in **`172-00-PLAN.md`** (wave 1), which is why `172-03` declares
> `depends_on: ["172-00", "172-01"]`. Every "no extractor change needed" phrase below is
> superseded by that plan.

**Current corpus list** (lines 40-46, exact):
```go
var auditedCorpora = []string{
	filepath.Join(".claude", "commands", "ant"),
	filepath.Join(".opencode", "commands", "ant"),
	filepath.Join(".aether", "commands"),
	filepath.Join(".aether", "docs", "command-playbooks"),
	filepath.Join("colony", "playbooks"),
}
```
`collectDocumentedCalls` (lines 176-199) walks each entry with `filepath.Walk`, which is *recursive*. Per RESEARCH.md Pitfall 3 / Open Question 1 (resolved: use the literal top-level glob), the 4 top-level tracked `.aether/*.md` files (`CONTEXT.md`, `CROWNED-ANTHILL.md`, `QUEEN.md`, `workers.md` — `HANDOFF.md` is gitignored, exclude it) must be added as an **explicit file list**, not a directory-walk entry, since `auditedCorpora` today assumes every entry is a directory to `filepath.Walk`. Either extend `collectDocumentedCalls` with a second, non-recursive file-list corpus type, or add a small parallel loop that calls `extractDocumentedCalls` directly on each of the 4 named files (reusing the function at lines 76-141, ~~unchanged — no extractor change needed, confirmed by manual trace in RESEARCH.md Pattern 2: the fence/backtick/placeholder handling already correctly parses `workers.md:292`'s fenced `aether spawn-can-spawn {your_depth} --enforce` line~~ **as repaired by `172-00-PLAN.md` — the pre-fix function silently drops that line, see the correction above**), which will then report `unknown flag --enforce` before any `cmd/spawn.go` fix, via `validateCallAgainstCobra`, lines 284-375.

**Failure-message shape already correct, no changes needed** (lines 396-398):
```go
rel, _ := filepath.Rel(root, c.File)
violations = append(violations, fmt.Sprintf("%s:%d: `%s` — %s", rel, c.Line, c.Raw, v))
```
This already produces `file:line:` plus the flag error (`unknown flag --%s`, line 338) — satisfies WIRE-03's "name file, line, and flag" requirement with zero new code once the corpus exists.

---

### `.github/workflows/ci.yml` — new named CI step(s) (MODIFY — config, batch)

**Analog:** the existing "Verify command catalog classification" step (lines 96-97) and "Run Go tests" step (lines 41-42).

**Exact shape to copy** (lines 93-97):
```yaml
      - name: Parity tests
        run: go test ./cmd/... -run "TestParity|TestGoOnly" -count=1 -timeout 900s -v

      - name: Verify command catalog classification
        run: python3 scripts/verify_catalog_classified.py --strict
```
Add a new step in this same style, placed near this block (per RESEARCH.md/CONTEXT.md's "near the existing Verify command catalog classification step"), running only the wiring/flag tests by name — mirroring the "Parity tests" step's `-run` pattern:
```yaml
      - name: Verify subcommand wiring and CLI flag contracts
        run: go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced|TestOrphanAllowlistOnlyShrinks|TestCommandCallsMatchCobraContracts|TestDocumentedCommandNamesResolve' -count=1 -timeout 900s -v
```
Per D-15, this must be its own **named** step distinct from the blanket "Run Go tests" step (lines 41-42) — those tests already run as part of `go test ./...` there too; the named step exists for legibility of failure, not additional coverage (confirmed no gap: `go test ./... -count=1` already picks up any new `cmd/*_test.go` automatically).

## Shared Patterns

### JSON success/error envelope (`outputOK`/`outputError`)
**Source:** `cmd/helpers.go:21-56`
**Apply to:** `cmd/spawn.go`'s `spawnCanSpawnCmd` modification
```go
func outputOK(result interface{}) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		outputError(2, fmt.Sprintf("failed to marshal command result: %v", err), nil)
		return
	}
	fmt.Fprintf(stdout, "{\"ok\":true,\"result\":%s}\n", string(resultJSON))
}

func outputError(code int, message string, details interface{}) {
	markRenderedCommandError(code)
	...
}
```
`markRenderedCommandError` (`cmd/root.go:219`) is the sole mechanism in this repo for making a command's process exit non-zero while still going through the structured JSON/visual error envelope. Every new "deny → non-zero exit" path in this phase must go through it, not `os.Exit`.

### AST var-block enumeration (`declNameAndBody`)
**Source:** `cmd/visual_writer_discipline_test.go:65-85`
**Apply to:** `cmd/subcommand_reachability_ratchet_test.go`
Already a project-established idiom used twice (also in `cmd/go_source_hint_audit_test.go`) specifically because a `FuncDecl`-only AST walk misses every cobra command (they are `var xCmd = &cobra.Command{RunE: func(){...}}`). Any new AST walk over `cmd/*.go` in this phase must use this two-case switch, not a `FuncDecl`-only walk.

### Corpus-based markdown/YAML call extraction (`extractDocumentedCalls` family)
**Source:** `cmd/command_call_audit_test.go:75-280`
**Apply to:** both `cmd/subcommand_reachability_ratchet_test.go` (loosely, for "is this name mentioned as something to run") and `cmd/command_call_audit_test.go`'s own `auditedCorpora` extension (unchanged, just a new corpus entry)
This is the sole fence-aware, placeholder-aware, prose-negation-aware markdown/YAML command-call parser in the repo. Do not hand-roll a second regex-only scanner (see RESEARCH.md "Don't Hand-Roll" and Anti-Patterns — this is exactly the blind spot that made the older `cli_flag_audit_test.go` miss positional-argument violations).

### Committed JSON baseline + comparator + named CI gate
**Source:** `cmd/testdata/command_catalog.json` + `scripts/verify_catalog_classified.py` + `.github/workflows/ci.yml:96-97` (and, for the read/compare half specifically, `cmd/regression_test.go:45-57`)
**Apply to:** `cmd/testdata/orphan_allowlist.json` + `TestOrphanAllowlistOnlyShrinks` + the new CI step
Three-part template already proven in this repo: policy doc (optional, e.g. `.aether/docs/command-catalog-policy.md` precedent) + checked-in `testdata/*.json` + a test/script that reads and validates it + a named CI step. WIRE-01's new piece is the *shrink-only* comparison logic itself (set-membership diff against a committed copy, D-10) — that specific comparator has no precedent and must be built fresh, but the surrounding scaffold should follow this template exactly.

### Token-boundary-safe name matching
**Source:** `cmd/command_call_audit_test.go`'s `tokenizeShellLike` (lines 243-280) and `validateCallAgainstCobra`'s exact-token resolution via `rootCmd.Find` (lines 284-305)
**Apply to:** `cmd/subcommand_reachability_ratchet_test.go`'s caller-corpus search
Never use `strings.Contains`/substring grep to decide whether a corpus references a command name — `"skill-list"` is a substring of `"skill-list-lifecycle"` (Pitfall 2). Tokenize first, compare exact tokens.

## No Analog Found

None — every file in scope has at least a role-match or exact analog. The one genuinely novel piece of logic (the shrink-only baseline comparator itself) has no prior implementation in this repo to copy, but it does have a structural template to follow (see "Committed JSON baseline + comparator + named CI gate" above) and a very close instinct-preservation precedent worth noting for shape only: `cmd/command_call_severity.go:1-45`'s `gateClassifiedCommands` map — "a map that must be reviewed deliberately, not grown reflexively," though that map has no runtime enforcement of the discipline (a plain Go map literal), which is exactly the gap D-10/D-11 close by comparing against a *committed, diffable* baseline file instead of trusting comment discipline alone.

## Metadata

**Analog search scope:** `cmd/*.go`, `cmd/*_test.go`, `cmd/testdata/*.json`, `.github/workflows/ci.yml`, `scripts/*.py`, `.aether/workers.md`, `.aether/docs/command-catalog-policy.md`
**Files scanned:** `cmd/visual_writer_discipline_test.go`, `cmd/command_call_audit_test.go` (full, 751 lines), `cmd/cli_flag_audit_test.go` (full, 188 lines), `cmd/spawn.go` (full, 383 lines), `cmd/helpers.go` (partial, lines 15-70), `cmd/root.go` (partial, exit-code lines), `cmd/regression_test.go` (partial, lines 1-120), `cmd/command_call_severity.go` (partial, lines 1-40), `cmd/command_source_hygiene_test.go` (partial, `repoRootForCommandSourceTest`), `cmd/testdata/command_catalog.json` (head), `.github/workflows/ci.yml` (full), `scripts/verify_catalog_classified.py` (full), `.aether/workers.md` (lines 280-300)
**Pattern extraction date:** 2026-08-08
