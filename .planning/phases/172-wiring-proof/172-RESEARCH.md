# Phase 172: Wiring Proof - Research

**Researched:** 2026-08-08
**Domain:** Go/cobra CLI static-analysis testing (repo-internal; no external library research needed)
**Confidence:** HIGH — every claim below was verified by reading or grepping this repo's actual source during this session, not from training-data recall. Where I could not verify something (mostly implementation-detail choices left to discretion), it is marked `[ASSUMED]` and logged.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**What counts as a caller (the reachability rule)**
- **D-01:** A caller is something that **executes** the command. Three kinds count:
  (a) Go code outside the command's own definition file, (b) a platform wrapper the
  runtime actually loads — `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`,
  `.aether/commands/*.yaml` — and (c) a shipped hook or script. Nothing else.
- **D-02:** Documentation does **not** count. `CLAUDE.md`, `AGENTS.md`, `docs/specs/*`,
  `.aether/docs/*`, audit reports, `cmd/testdata/command_catalog.json` and
  `cmd/testdata/parity_snapshot.json` are all excluded as caller evidence.
- **D-03:** A **menu entry counts as a caller.** If a `/ant-…` wrapper or a
  `.aether/commands/*.yaml` spec names the command, it is user-reachable and therefore
  called.
- **D-04:** **Rejected:** exempting commands by their `classification` tier in
  `cmd/testdata/command_catalog.json`.
- **D-05:** **Test files are not callers.** A command whose only reference outside its own
  file is `*_test.go` is an orphan.
- **D-06:** The retired playbooks (`.aether/docs/command-playbooks/*.md`, not loaded by the
  runtime since v1.25) stay **in scope for the flag audit** but a mention there is **never**
  caller evidence.

**The baseline allowlist**
- **D-07:** Every orphan the scan finds goes into the baseline **honestly**, even if the
  real number is far above or below the 8 expected.
- **D-08:** Each entry carries a **reason tag** identifying why it is tolerated and which
  phase owns it (e.g. `skill-lifecycle / phase-178`).
- **D-09:** The allowlist lives in **its own dedicated file**, not as a field on
  `cmd/testdata/command_catalog.json` (that file is rewritten in place by
  `scripts/classify_commands.py:191`).
- **D-10:** Shrink-only is enforced by **comparing against a committed baseline copy**.
  Rejected: counting entries, comparing against git history.
- **D-11:** **No escape hatch.** Additions to the allowlist are hard-blocked.
- **D-12:** The flag audit's existing `skipSubcommands` map in `cmd/cli_flag_audit_test.go`
  (currently 2 entries, unguarded) gets the **same shrink-only treatment**.

**The documented safety command (WIRE-02)**
- **D-13:** `--enforce` is implemented with **real semantics now**: a deny answer causes a
  non-zero exit. Today `spawn-can-spawn` unconditionally returns `can_spawn: true`, so
  observable behaviour is unchanged and "exits 0" holds today. Rejected: an inert
  accepted-but-ignored flag.
- **D-14:** `spawnCanSpawnCmd` currently declares `Args: cobra.NoArgs`. **Fix the command,
  not the manual** — accept the positional depth exactly as `.aether/workers.md:292`
  writes it. Rejected: rewriting `workers.md` to use `--depth`.

**CI surfacing**
- **D-15:** Both checks get **their own named CI step**, in addition to the existing
  `go test ./...` coverage.

### Claude's Discretion
- Exact file name, format (JSON vs. plain text) and location of the allowlist and its
  baseline copy.
- Test names beyond the roadmap-pinned `TestNoRegisteredSubcommandIsUnreferenced`.
- How the caller scan is implemented (AST walk vs. text scan) and how it identifies a
  command's "own definition file".
- Whether the flag audit is an extension of the existing `TestCLIFlagAudit` or a new test —
  provided `.aether/*.md` enters scope and failures name file, line and flag.
- The precise shape of the reason tag.

### Deferred Ideas (OUT OF SCOPE)
- Making `spawn-can-spawn` return a real deny decision — Phase 173 (SPAWN-01).
- Reclaiming or removing the 8 `skill-*` lifecycle commands — Phase 178 (SKILL-01).
- The wider unreachable-command sweep beyond the ratchet itself — parked as RECLAIM.
- Retiring the playbook directory entirely.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| WIRE-01 | A test fails when a registered subcommand has no caller outside its own definition, seeded with today's known orphans in a shrink-only allowlist | See "Architecture Patterns → Pattern 1" and "Common Pitfalls → Pitfall 1/2/6" for the exact enumeration mechanism, the true (verified) orphan set, and the substring/self-reference traps |
| WIRE-02 | The documented invocation in `.aether/workers.md` matches a flag that exists | See "Code Examples → spawn-can-spawn fix" for the exact current code (`cmd/spawn.go:169-182`), the exit-code convention to reuse, and the dual-calling-convention constraint from `command-playbooks` |
| WIRE-03 | A test fails when a `.aether/*.md` instruction names a CLI flag the binary does not register | See "Architecture Patterns → Pattern 2" for the verified scope gap in the existing `TestCommandCallsMatchCobraContracts` and the exact minimal extension that seeds the failure |
</phase_requirements>

## Summary

This phase is almost entirely repo archaeology, and the archaeology paid off: **the machinery
for both WIRE-01 and WIRE-03 already exists in this repo, one phase-160 ("Fail Loudly")
generation removed.** `cmd/command_call_audit_test.go` is a working, cobra-native audit
(`TestCommandCallsMatchCobraContracts`) that already resolves subcommand names and validates
flags/positionals against the real `rootCmd` tree, with a scoped corpus, fence/backtick-aware
extraction, and file:line:flag failure messages. It does **not** currently scan
`.aether/workers.md` — that file sits directly outside its five audited corpora — which is
exactly why `--enforce` on `.aether/workers.md:292` has never been caught. Extending its
`auditedCorpora` list with the top-level `.aether/*.md` files is, on direct trial-by-reasoning
against the code, sufficient to make the test fail today naming `workers.md:292` and
`unknown flag --enforce` — which is precisely the WIRE-03 seed criterion. No new parser is
required; a new corpus entry is.

WIRE-01 has no existing analog test, but has an existing analog **pattern**:
`cmd/visual_writer_discipline_test.go` already walks `var xCmd = &cobra.Command{...}` GenDecl
blocks with `go/ast` (not just `FuncDecl`) to find cobra command definitions and their
enclosing file — the repo-memory note that "an AST guard must walk var blocks" is still
correct today and directly reusable (`declNameAndBody`, same file). What WIRE-01 needs beyond
that: comparing each registered command name against three permitted caller corpora (Go
non-test code outside its own file, loaded platform wrappers, shipped hooks/scripts), and
a shrink-only baseline comparison, for which this repo currently has **no existing pattern to
reuse** — the closest analog (`cmd/testdata/command_catalog.json` +
`scripts/verify_catalog_classified.py --strict`) is a policy-doc-plus-gate template worth
copying, but its `--strict` check enforces *count ranges*, not a shrink-only diff against a
committed copy. That comparison logic must be built fresh.

**A significant correction to the phase's own premise:** direct inspection of
`.claude/commands/ant/skill-create.md` (lines 220-242) shows `skill-parse-frontmatter` and
`skill-cache-rebuild` — two of the eight commands CONTEXT.md names as the seeded orphan set —
are genuinely invoked ("Run using the Bash tool") from a live, loaded wrapper
(`/ant-skill-create`, mirrored in `.opencode/commands/ant/skill-create.md`). Under D-01(b)
these are real callers. The other six (`skill-index`, `skill-detect`, `skill-match`,
`skill-inject`, `skill-list`, `skill-diff`) have zero references anywhere outside
`cmd/skills.go` and its `_test.go`, confirmed by targeted grep — those are genuine orphans.
This does not conflict with D-07 ("every orphan the scan finds goes into the baseline
honestly, even if the real number is far above the 8 expected") — it means the honest number
today is very likely **6**, not 8, and the planner should build the scanner to report
whatever it actually finds rather than hand-seed a list of 8 names. See Pitfall 1 for detail
and Assumption A1 for the residual uncertainty.

**Primary recommendation:** Extend the existing `cmd/command_call_audit_test.go` machinery
(new corpus entry) for WIRE-03, and add one new test file reusing
`visual_writer_discipline_test.go`'s AST var-block walker for WIRE-01's command enumeration,
paired with a hand-built shrink-only baseline comparison (new — nothing to reuse) stored as
`cmd/testdata/orphan_allowlist.json`. Fix `cmd/spawn.go`'s `spawnCanSpawnCmd` in place for
WIRE-02, reusing the existing `outputError`/exit-code convention already used by its sibling
`spawn-log` command in the same file.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Orphan subcommand detection (WIRE-01) | Go test (`cmd/`, CI) | — | Static analysis over `rootCmd` and repo source; no runtime component |
| `spawn-can-spawn --enforce` (WIRE-02) | API/Backend (Go CLI, `cmd/spawn.go`) | — | A cobra subcommand is this repo's "backend" unit; it is invoked as a subprocess by wrapper markdown, never by a browser/frontend tier |
| Flag/instruction drift detection (WIRE-03) | Go test (`cmd/`, CI) | — | Parses markdown/YAML corpora and validates against the live cobra command tree |
| CI surfacing (all three) | CI / Build pipeline (`.github/workflows/ci.yml`) | — | Named steps alongside the existing `go test ./...` and `verify-catalog` steps |

There is no browser, SSR, or database tier in this phase — it is CLI-and-CI only. This map is
included for completeness per the research template; there is nothing to sanity-check a
planner against here beyond "don't put any of this behind a network boundary that doesn't
exist."

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go/ast`, `go/parser`, `go/token` | Go 1.26 stdlib (`go.mod` toolchain, verified via `.github/workflows/ci.yml` `go-version: "1.26"`) | Parse `cmd/*.go` to enumerate `var xCmd = &cobra.Command{...}` declarations and their enclosing file | Already imported and used for exactly this purpose in `cmd/go_source_hint_audit_test.go` and `cmd/visual_writer_discipline_test.go` — no new dependency [VERIFIED: repo grep] |
| `github.com/spf13/cobra` | already a direct dependency (`rootCmd.Commands()`, `rootCmd.Find()`, `cmd.Args()`) | Enumerate registered commands and validate real argument/flag contracts | Already the CLI framework; `cmd/command_call_audit_test.go` already validates against it live | 
| `github.com/spf13/pflag` | already a direct dependency | Walk `Flags()`/`PersistentFlags()` to check flag registration | Used today in `cmd/cli_flag_audit_test.go:12` |
| `encoding/json` | stdlib | Read/write the committed baseline allowlist | Consistent with every other `cmd/testdata/*.json` fixture in this repo |

### Supporting

None. This phase needs zero new third-party packages — confirmed by grepping `go.mod` for
anything AST/lint-related and finding none beyond stdlib usage already present.

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Hand-rolled `go/ast` walk | `golang.org/x/tools/go/packages` (type-checked SSA-level call graph) | Would give true call-graph reachability, including transitive Go-internal calls — but is a new dependency, much slower in CI, and this repo has already chosen the lighter `go/ast`-only approach twice (two existing tests). Overkill for a repo where the Go-internal caller category has zero real instances today (see Pitfall 5) |
| Regex-based markdown scan (old `cli_flag_audit_test.go` style) | Cobra-native resolution (`rootCmd.Find`) | The regex approach is a known, documented blind spot — it cannot see positional-argument violations, which is the entire reason `command_call_audit_test.go` was built in the first phase after it. Do not extend the older test; extend the newer one |

**Installation:** none required.

**Version verification:** N/A — no new packages.

## Architecture Patterns

### System Architecture Diagram

```
                    ┌──────────────────────────────┐
                    │   Go source (cmd/*.go)        │
                    │   var xCmd = &cobra.Command{}  │
                    │   + init() { rootCmd.AddCommand }│
                    └───────────────┬────────────────┘
                                    │ go/ast parse (declNameAndBody-style)
                                    ▼
                    ┌──────────────────────────────┐
                    │  Enumerate every registered    │
                    │  subcommand name + its own      │
                    │  definition file (WIRE-01)      │
                    └───────────────┬────────────────┘
                                    │ for each name, search 3 permitted
                                    │ caller corpora (D-01/D-02/D-03/D-05):
              ┌─────────────────────┼─────────────────────────┐
              ▼                     ▼                          ▼
   ┌─────────────────┐   ┌─────────────────────┐   ┌───────────────────────┐
   │ Go code outside   │   │ Loaded platform       │   │ Shipped hooks/scripts │
   │ own file, non-test │   │ wrapper: .claude/…/ant,│  │ .aether/utils/hooks/*.js│
   │ (near-zero hits —  │   │ .opencode/…/ant,       │   │ scripts/*.sh checked   │
   │ see Pitfall 5)      │   │ .aether/commands/*.yaml│   │ into the repo          │
   └─────────────────┘   └─────────────────────┘   └───────────────────────┘
              │                     │                          │
              └─────────────────────┴─────────────┬────────────┘
                                                    ▼
                                     found in ≥1 corpus? ── yes → not an orphan
                                                    │ no
                                                    ▼
                                  in committed baseline (cmd/testdata/orphan_allowlist.json)?
                                    │ yes, unchanged        │ no, or grew
                                    ▼                        ▼
                            PASS (tolerated debt)     FAIL, naming the command
                                                       (TestNoRegisteredSubcommandIsUnreferenced)

   ─────────────────────────── separately (WIRE-03) ───────────────────────────

   ┌───────────────────────────┐
   │ .aether/*.md, .claude/…,   │   extractDocumentedCalls() — already exists,
   │ .opencode/…, .aether/      │   fence+backtick aware, prose/negation aware
   │ commands/, .aether/docs/    │──────────────┐
   │ command-playbooks/,         │              ▼
   │ colony/playbooks/            │   validateCallAgainstCobra() — already exists,
   └───────────────────────────┘   resolves against rootCmd.Find, checks every
                                    flag against the real FlagSet, checks
                                    positionals against the real Args validator
                                                    │
                                                    ▼
                                    FAIL naming file:line:flag (or "positional
                                    args … rejected by the command's own Args
                                    validator")
```

### Recommended Project Structure

No new package or directory. Everything lands in the existing `cmd/` package, following the
existing file-per-audit convention:

```
cmd/
├── spawn.go                              # WIRE-02: fix in place (Args, new --enforce flag, deny→exit)
├── command_call_audit_test.go            # WIRE-03: add ".aether" (top-level *.md) to auditedCorpora
├── cli_flag_audit_test.go                # WIRE-03 (D-12): give skipSubcommands shrink-only treatment
├── subcommand_reachability_ratchet_test.go   # WIRE-01: NEW — TestNoRegisteredSubcommandIsUnreferenced
│                                              #   + TestOrphanAllowlistOnlyShrinks + fixture test
├── testdata/
│   └── orphan_allowlist.json             # WIRE-01: NEW — committed baseline, own dedicated file (D-09)
```

### Pattern 1: Enumerating registered commands via `go/ast` var-block walk

**What:** Parse every non-test `.go` file in `cmd/` for top-level `var xCmd = &cobra.Command{Use: "name", ...}`
declarations, recording the command name and the file it was declared in. This is the
"own definition file" WIRE-01 needs, and it is the same walk `visual_writer_discipline_test.go`
already performs for a different purpose.

**When to use:** For WIRE-01's command enumeration. Do NOT rely on `rootCmd.Commands()` alone
for the "own file" part — cobra's runtime tree tells you the command exists and its `Use`
string, but not which `.go` file declared it. You need the AST walk for that half; you need
`rootCmd.Commands()`/`rootCmd.Find()` for confirming registration is real (catches anonymous
inline commands like `hostCmd.AddCommand(&cobra.Command{...})` in `cmd/host_cmd.go:31-63`,
which have no separate `var` declaration to find via AST alone — cross-check both).

**Example:**
```go
// Source: cmd/visual_writer_discipline_test.go:61-85 (existing, verified pattern)
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
Extract the `Use:` string literal from the `*ast.CompositeLit` under each `&cobra.Command{...}`
to get the actual command name (not the Go var name) — `Use` sometimes carries argument
hints, e.g. `Use: "skill-match [role] [task]"` (`cmd/skills.go:264`); take the first
whitespace-delimited token.

### Pattern 2: Extending the cobra-native flag/call audit (WIRE-03)

> **⚠ CORRECTION (added during plan revision — this section's central claim was wrong).**
> This section concludes that adding `.aether/workers.md` to the corpus is sufficient with
> **"zero other code changes"**. That conclusion is **disproven**. It rests on a manual trace
> that started at `validateCallAgainstCobra` and never checked whether the call reaches it.
> It does not: `workers.md:292` reads `result=$(aether spawn-can-spawn {your_depth} --enforce)`,
> and `tokenizeShellLike` collapses `result=$(aether` into a single token matching neither
> `aether` nor `*/aether`, so the binary-detection test never fires and the line is skipped
> before any validation happens. Adding the corpus alone would ship a test that passes while
> the bug it exists to catch sits inside its declared scope.
>
> **Authoritative source: `172-00-PLAN.md`**, which fixes the tokenizer in wave 1 as a
> blocking prerequisite. Read the paragraphs below as a record of what was believed at
> research time, not as instructions. Everything else in this section — the corpus list,
> the non-recursive-glob requirement, the failure-message shape — remains accurate.
>
> *Kept rather than deleted deliberately: this phase exists to catch claims that drift from
> reality, so its own research doc records the correction instead of quietly erasing it.*

**What:** `cmd/command_call_audit_test.go`'s `auditedCorpora` (lines 40-46) currently lists
five directory trees. It does not include `.aether/workers.md` or any other top-level
`.aether/*.md` file — confirmed by direct inspection: `.aether/workers.md`,
`.aether/QUEEN.md`, `.aether/CONTEXT.md`, `.aether/CROWNED-ANTHILL.md` are git-tracked
(`git ls-files .aether/*.md`) but none of their parent directory is in `auditedCorpora`.
`workers.md:292` (`result=$(aether spawn-can-spawn {your_depth} --enforce)`) sits inside a
fenced ` ```bash ` block. ~~so `extractDocumentedCalls` would parse it correctly as a fenced
invocation the moment the file is in scope — no extractor change needed, only a corpus
addition.~~ **Corrected:** the fence handling is fine, but the *tokenizer* is not — the
`result=$(` prefix makes the first token `result=$(aether`, which fails binary detection, so
the line is discarded before parsing. A corpus addition alone is **not** sufficient; the
tokenizer fix in `172-00-PLAN.md` must land first.

**When to use:** Add exactly one new corpus entry. Because `auditedCorpora` is walked with
`filepath.Walk` over a directory (`cmd/command_call_audit_test.go:184-196`), and
`.aether/*.md` per the roadmap wording is a **top-level, non-recursive** glob (not
`.aether/**/*.md`), the correct new entry is not a directory but an explicit list of the
4-5 top-level `.aether/*.md` files, OR the corpus-walk helper needs a "non-recursive
directory scan" mode distinct from the existing recursive `filepath.Walk`. See Open
Question 1 for the exact scope tradeoff.

**Verified today-state (before the fix):**
```
grep -n "spawn-can-spawn\|--enforce\|--depth" .aether/workers.md
# 292:result=$(aether spawn-can-spawn {your_depth} --enforce)
```
Running `validateCallAgainstCobra` by hand against this call today (traced manually against
the live `rootCmd`): `rootCmd.Find(["spawn-can-spawn", "{your_depth}", "--enforce"])` resolves
`spawnCanSpawnCmd`; the loop hits `{your_depth}` first (accepted as a positional/placeholder
token), then `--enforce`, which is not in `target.Flags()`, `target.InheritedFlags()`, or
`rootCmd.PersistentFlags()` — the function returns `"unknown flag --enforce"` **before ever
reaching the positional-arity check**. ~~This means: once `.aether/workers.md` is in scope,
the test fails today with exactly the message the roadmap wants ("naming the file, the
line, and the offending flag") with **zero other code changes** — the existing extractor and
validator are sufficient. This is strong evidence WIRE-03 is a one-line-corpus extension,
not a new parser.~~

**↑ STRUCK — the trace above is correct but starts one step too late.** It assumes the call
reaches `validateCallAgainstCobra`. It never does: `tokenizeShellLike` yields
`result=$(aether` as one token, which fails the binary-detection test, so
`extractDocumentedCalls` discards the line before validation. The validator behaviour
described above is real and will produce exactly that message — but only after
`172-00-PLAN.md` repairs tokenization. WIRE-03 is a parser fix **plus** a corpus extension,
which is why 172-00 blocks 172-02 and 172-03.

**Example — the corpus list to extend:**
```go
// Source: cmd/command_call_audit_test.go:40-46 (current)
var auditedCorpora = []string{
	filepath.Join(".claude", "commands", "ant"),
	filepath.Join(".opencode", "commands", "ant"),
	filepath.Join(".aether", "commands"),
	filepath.Join(".aether", "docs", "command-playbooks"),
	filepath.Join("colony", "playbooks"),
}
```

### Pattern 3: `outputError`'s deferred-exit-code convention (for WIRE-02's `--enforce`)

**What:** Commands in this repo signal a non-zero process exit while still returning `nil`
from `RunE` and printing a structured JSON/visual error envelope, via
`markRenderedCommandError(code)` inside `outputError` (`cmd/helpers.go:35-36`,
`cmd/root.go:219`). `spawnCanSpawnCmd`'s sibling `spawnLogCmd`, four cases up in the same
file, already uses this convention for its own required-flag validation
(`cmd/spawn.go:35-45`).

**When to use:** For `--enforce`'s deny path. Do not `os.Exit()` directly and do not return
a Go `error` from `RunE` (that path exists too via cobra, but it prints cobra's own usage
text, which is not this repo's convention — every other command in `cmd/spawn.go` uses
`outputError`).

**Example:**
```go
// Adapted from the established pattern at cmd/spawn.go:17-45 and cmd/gate.go:87
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
Note the `.aether/docs/command-playbooks/build-full.md:854` and `build-wave.md:675` calling
convention is `aether spawn-can-spawn --depth {depth}` (flag form, no positional, no
`--enforce`) — `cobra.MaximumNArgs(1)` accepts this unchanged (0 positionals is within the
max), so both calling conventions keep working. Neither playbook file was updated to add
`--enforce`; that is out of scope (they are unloaded playbooks, D-06).

### Anti-Patterns to Avoid

- **Building a new markdown/flag parser for WIRE-03:** the existing one in
  `command_call_audit_test.go` already does fence detection, backtick detection, placeholder
  handling (`$ARGUMENTS`, `<phase>`, `{name}`), shell-operator truncation, and prose/negation
  filtering (`"Do not attempt to run"`). A hand-rolled regex (like the *older*
  `cli_flag_audit_test.go`) would reintroduce the exact positional-argument blind spot that
  motivated building the newer test in the first place.
- **Treating "referenced anywhere" as caller evidence for WIRE-01:** the naive grep for the
  8 `skill-*` names surfaces ~20 files (CLAUDE.md, AGENTS.md, `docs/specs/*`, both catalog
  JSONs, retired playbooks) that CONTEXT.md's D-02 explicitly excludes. A scan that doesn't
  restrict itself to the three D-01 corpora will report zero orphans and never fire again.
- **Treating "the underlying Go function is called" as evidence the CLI subcommand is
  reachable:** `cmd/codex_build.go:2869` calls `matchSkillsForWorkflow` directly — the same
  business logic `skill-match`'s `RunE` uses — but nothing ever shells out
  `aether skill-match`. If the scan matches on shared function/identifier names instead of
  on the literal invoked command string, it will incorrectly clear `skill-match` as "used."
  See Pitfall 5.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Parsing `aether <cmd> [flags] [args]` out of markdown/YAML, fence-aware, placeholder-aware | A new regex/tokenizer | `extractDocumentedCalls` / `tokenizeShellLike` / `isPlaceholder` / `isShellOperator` in `cmd/command_call_audit_test.go:76-280` | Already handles the exact edge cases that broke the older `cli_flag_audit_test.go` (positionals, placeholders, negated instructions, quoted args with spaces) |
| Validating a call's flags and positional arity against the real command | Manual flag-name string comparison | `validateCallAgainstCobra` (same file, lines 284-375) — resolves via `rootCmd.Find`, checks `target.Flags()`/`InheritedFlags()`/`rootCmd.PersistentFlags()`, and validates positionals through `target.Args(target, positionals)` (cobra's own validator, e.g. `cobra.NoArgs`, `cobra.MaximumNArgs`) | Uses cobra's own arity contract instead of guessing; this is precisely the fix for the positional blind spot the phase-160 postmortem documents |
| Enumerating cobra command declarations across `.go` files including `RunE` closures inside `var` blocks | A `FuncDecl`-only AST walk (misses cobra commands entirely, since they are `var xCmd = &cobra.Command{RunE: func(){...}}`) | `declNameAndBody` in `cmd/visual_writer_discipline_test.go:65-85` | Documented in that file's own comment: a `FuncDecl`-only walk is "how `pheromone-display` writing a raw table... hid" — the exact blind spot this phase must not reintroduce for WIRE-01 |
| A "policy doc + checked-in data + CI gate" scaffold | Inventing a new doc structure | `.aether/docs/command-catalog-policy.md` + `cmd/testdata/command_catalog.json` + `scripts/verify_catalog_classified.py --strict` + its named `Verify command catalog classification` CI step | Established, working three-part template in this exact repo; WIRE-01's allowlist should follow the same shape (policy doc explaining the shrink-only rule + `cmd/testdata/orphan_allowlist.json` + named CI step), even though the comparison LOGIC (shrink-only diff) has no existing implementation to copy |

**Key insight:** every piece of WIRE-03 and half of WIRE-01 (the enumeration half) already
exists in this codebase from a prior phase in the same problem family (phase 160, "Fail
Loudly," commented `LOUD-03/04/05` in `command_call_audit_test.go`). The remaining net-new
work is: (1) a corpus addition, (2) a shrink-only baseline comparator (genuinely new, no
existing pattern), and (3) the `cmd/spawn.go` fix. Do not re-derive what phase 160 already
built correctly.

## Common Pitfalls

### Pitfall 1: The seeded "8 skill-* orphans" is very likely 6, not 8 — verify at implementation time, don't hardcode
**What goes wrong:** Treating CONTEXT.md's "8 `skill-*` lifecycle commands" as a literal list
to seed the baseline with, rather than letting the built scanner report what it actually
finds.
**Why it happens:** `.claude/commands/ant/skill-create.md:220-242` (mirrored in
`.opencode/commands/ant/skill-create.md`) contains explicit "Run using the Bash tool"
instructions for `aether skill-parse-frontmatter --file ...` and `aether skill-cache-rebuild`
inside fenced bash blocks. `/ant-skill-create` is a real, loaded wrapper (listed in
`.claude/rules/aether-colony.md`'s Advanced commands table). Under D-01(b)/D-03 this is
unambiguous caller evidence for those two commands. The other six
(`skill-index`, `skill-detect`, `skill-match`, `skill-inject`, `skill-list`, `skill-diff`)
have zero references anywhere outside `cmd/skills.go` and `cmd/skills_test.go`, confirmed by
targeted `grep -rl` across `cmd/`, `pkg/`, `.claude/commands/ant/`, `.opencode/commands/ant/`,
`.aether/commands/`.
**How to avoid:** Build the scan first, run it, and seed the baseline from its real output
— per D-07, this is already the locked behaviour. Do not special-case or manually correct
the list to match "8." If the scan disagrees with the roadmap's stated count, that is the
scan doing its job; note the discrepancy in the phase's SUMMARY for Phase 178's benefit
(its success criterion measures "the allowlist's 8 skill entries dropping to 0" — it will
need to read whatever the actual seeded count is, tagged `skill-lifecycle / phase-178`,
rather than assume 8).
**Warning signs:** If the implemented scan reports exactly 8 skill-lifecycle orphans, that is
a sign the scan is (incorrectly) not crediting `skill-create.md`'s fenced invocations as
caller evidence — recheck the extractor's corpus coverage and fence handling before trusting
the count.

### Pitfall 2: Substring matching on command names produces false "has a caller" results
**What goes wrong:** `grep -l "skill-list"` matches `cmd/skill_lifecycle.go` because
`"skill-list"` is a literal substring of the unrelated command name `"skill-list-lifecycle"`
(`cmd/skill_lifecycle.go:125`). A text-scan-based caller check that isn't word/token-boundary
aware will silently mark `skill-list` as "referenced" and clear it from the orphan list
incorrectly.
**Why it happens:** Cobra command names in this repo share long common prefixes deliberately
(`spawn-log`/`spawn-complete`/`spawn-can-spawn`/`spawn-tree-*`, `skill-*` family,
`pending-decision-*`). Any substring-based grep will cross-contaminate.
**How to avoid:** Match command names as whole tokens — reuse `validateCallAgainstCobra`'s
approach of tokenizing first and comparing exact strings (or `rootCmd.Find` on the tokenized
first argument), not `strings.Contains`.
**Warning signs:** A caller-evidence hit inside another command's own `Use:` string or var
name, rather than inside an actual invocation.

### Pitfall 3: `.aether/*.md` scope is genuinely ambiguous between "top-level glob" and "recursive tree" — pick top-level, and say why
**What goes wrong:** Interpreting the roadmap's `.aether/*.md` as `.aether/**/*.md` (recursive)
massively over-scopes the corpus: `git ls-files '.aether/**/*.md'` returns 250 tracked files,
769 `aether `-shaped tokens live under `.aether/docs/` alone, 158 under `.aether/skills/`, 92
under `.aether/references/`, 34 under `.aether/templates/` — none of that surface has been
vetted for the extractor's prose/fence heuristics, and it pulls in `.aether/docs/*`
generally, which D-02 explicitly excludes as **caller** evidence (a separate question from
flag-audit scope, but the same directory) risking scope creep unrelated to this phase's
stated boundary ("no new delegation, spend, roster or skill capability").
**Why it happens:** `.aether/*.md` is genuine shell-glob syntax for "top-level files only" —
but it is easy to read informally as "anything under `.aether` that's markdown."
**How to avoid:** Interpret literally: the top-level tracked files are
`.aether/CONTEXT.md`, `.aether/CROWNED-ANTHILL.md`, `.aether/QUEEN.md`, `.aether/workers.md`
(4 git-tracked; a 5th, `.aether/HANDOFF.md`, is gitignored — `.gitignore:87` — and should be
excluded, since an audit over an untracked, session-local file would behave differently in
CI than locally). This is also the **minimal** scope that satisfies the seeded failure
(`workers.md:292`) with zero noise: confirmed by grep that `CONTEXT.md`, `QUEEN.md`, and
`CROWNED-ANTHILL.md` contain zero fenced-code-block `aether` invocations (they are runtime
session logs full of unfenced prose mentioning "aether continue", "aether discuss" etc. as
narrative bug reports, which the existing extractor's fence/backtick requirement already
filters out).
**Warning signs:** A newly-added corpus producing dozens of unrelated violations from
`.aether/docs/` or `.aether/skills/` prose — a sign the scope was read recursively rather
than as a literal top-level glob. Flagged as Open Question 1 for explicit confirmation.

### Pitfall 4: The two existing flag-audit tests are not redundant — know which one you're extending
**What goes wrong:** Assuming `cmd/cli_flag_audit_test.go` (`TestCLIFlagAudit`) is the
current/authoritative flag audit and extending it, when `cmd/command_call_audit_test.go`
(`TestCommandCallsMatchCobraContracts`, from phase 160) superseded it in capability (cobra-
native arity checking) but not in code (the older test still exists and still runs — both
are live in CI today).
**Why it happens:** Both files audit overlapping but non-identical corpora
(`cli_flag_audit_test.go` also scans `.aether/docs/command-playbooks/`, which
`command_call_audit_test.go` also scans — but the older one uses a pure regex that cannot see
positional-argument violations, per that file's own docstring: "This resolves RESEARCH.md
open questions 1 and 2").
**How to avoid:** Extend `command_call_audit_test.go`'s `auditedCorpora` for WIRE-03's new
scope. Leave `cli_flag_audit_test.go` as-is except for D-12's shrink-only treatment of its
`skipSubcommands` map (a narrower, separate, explicitly-locked task).
**Warning signs:** Two near-duplicate new corpus entries added in two different files for the
same `.aether/*.md` scope.

### Pitfall 5: The Go-code caller category (D-01a) has zero real instances in this codebase today — do not over-build for it
**What goes wrong:** Building elaborate call-graph or SSA analysis to catch "Go code outside
the command's own file calls this command," when a targeted search
(`grep -rn "exec.Command(\"aether\"" cmd/ pkg/`) finds **zero** self-invocations of the
`aether` binary anywhere in the source tree. This repo's Go code never shells out to itself;
every "Go code" reference to a command name found during this research turned out to be
either (a) a string literal *telling a human/worker* to run the command
(`cmd/platform_sync.go:119`, already covered by the separate `TestGoSourceHintsMatchCobraContracts`
audit — not execution) or (b) a shared underlying function called directly, bypassing the CLI
entirely (`matchSkillsForWorkflow` in `cmd/codex_build.go:2869` — see the Anti-Patterns
section above).
**Why it happens:** D-01(a) is written as a real category because it is architecturally
possible (a parent cobra command calling `rootCmd.Execute()` recursively, or `exec.Command`
self-invocation for a hook), just not present today.
**How to avoid:** Implement the Go-caller check narrowly and honestly — e.g. search non-test
`.go` files for `exec.Command("aether"` / `exec.Command(os.Args[0]` string-literal patterns
naming the subcommand — and expect it to contribute zero hits today. Do not treat "the
underlying function is referenced elsewhere" as equivalent; that produces false negatives
(clears real orphans). This is flagged as Open Question 2 — the planner/discuss-phase should
confirm this narrow interpretation explicitly, since it is the one D-01 sub-clause this
research could not fully pin down against a real example.
**Warning signs:** The orphan count comes back suspiciously low (near zero) — a sign the
Go-caller check is matching on shared identifiers rather than actual execution evidence.

### Pitfall 6: The ratchet's own fixture command must not permanently trip the ratchet it tests
**What goes wrong:** Success criterion 1 requires "a fixture that registers exactly such a
command" (one with no caller) to prove the ratchet fails correctly. If that fixture is a
real, permanently-registered `rootCmd.AddCommand` in production code, it becomes a real
orphan forever, either requiring a permanent baseline entry (undermining "shrink-only, honest
debt" — a fixture isn't debt) or breaking the ratchet on every run.
**Why it happens:** The most obvious way to write "register a command with no caller" is to
just add one in a `_test.go` file at package scope.
**How to avoid:** Follow the existing precedent exactly:
`TestAuditDetectsPositionalDrift` in `cmd/command_call_audit_test.go:411-449` registers its
throwaway fixture **inside the test function**, via `rootCmd.AddCommand(noArgsCmd)` followed
immediately by `defer rootCmd.RemoveCommand(noArgsCmd)`. The command exists only for the
duration of that one test, is never present when the orphan-scan test itself runs, and never
touches the committed baseline. Use the identical pattern for WIRE-01's self-test.
**Warning signs:** A new permanent entry in `cmd/testdata/orphan_allowlist.json` with a
reason tag like "test fixture" — that is the self-referential trap manifesting; it should
never need a baseline entry at all if scoped as a function-local fixture.

## Code Examples

### Fixture pattern for the orphan-detection self-test (reused verbatim from an existing test)
```go
// Source: cmd/command_call_audit_test.go:411-416 (existing, verified pattern — reuse this shape for WIRE-01's own self-test)
noArgsCmd := &cobra.Command{Use: "audit-selftest-noargs", Args: cobra.NoArgs, Run: func(*cobra.Command, []string) {}}
rootCmd.AddCommand(noArgsCmd)
defer rootCmd.RemoveCommand(noArgsCmd)
```

### Existing severity-classification precedent for a shrink-only-adjacent map
```go
// Source: cmd/command_call_severity.go:1-45 (existing) — not shrink-only itself, but the
// established shape for "a map that must be reviewed deliberately, not grown reflexively."
// gateClassifiedCommands lists commands whose failure-to-execute halts a run; everything
// else defaults to "enrichment." WIRE-01's allowlist needs the inverse discipline: growth
// must be IMPOSSIBLE, not just discouraged by comment, which this existing map does not
// enforce — it can grow silently today. Do not copy its enforcement (none); copy its
// "one deliberate map with recorded rationale per entry" shape instead.
```

### The `spawn-can-spawn` current, broken state (exact reproduction)
```go
// Source: cmd/spawn.go:169-181 (current, verified by direct read)
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
// cmd/spawn.go:370 (init):
// spawnCanSpawnCmd.Flags().Int("depth", 0, "Spawn depth to check (required)")
// No --enforce flag registered anywhere in this file.
```
Running `aether spawn-can-spawn 5 --enforce` today: cobra rejects the positional `5` first
(`Args: cobra.NoArgs` — `command accepts 0 arg(s), received 1`) before it would even get to
the unknown `--enforce` flag. Both defects (D-13, D-14) must be fixed together for the
documented string to succeed.

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Regex-based markdown CLI-call scanning (`cli_flag_audit_test.go`, flags only, no positional check) | Cobra-native resolution against the live `rootCmd` tree (`command_call_audit_test.go`, flags AND positionals) | Phase 160 ("Fail Loudly," commented `LOUD-03/04/05`) — a prior phase in this same repo, this milestone cycle | The old approach is a documented, acknowledged blind spot ("cli_flag_audit_test.go matches only `--flag` tokens with a regex and never consults cobra's Args validator" — comment at `command_call_audit_test.go:17-21`). WIRE-03 should extend the newer approach, not the older one |
| `FuncDecl`-only AST walks for finding Go command definitions | `GenDecl`(VAR)-aware AST walks (`declNameAndBody`) | Also phase 160 (`visual_writer_discipline_test.go`) | A `FuncDecl`-only walk misses every cobra command, since they are declared as `var xCmd = &cobra.Command{RunE: func(){...}}` — the repo's own comment documents this having hidden a real bug (`pheromone-display` writing unrouted output) |

**Deprecated/outdated:** none of this phase's dependencies are deprecated; `go/ast`/`go/parser`
are stable Go stdlib with no relevant version churn.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The true orphan count for the `skill-*` family is 6, not 8, because `skill-parse-frontmatter` and `skill-cache-rebuild` are genuinely invoked from `.claude/commands/ant/skill-create.md` and its OpenCode mirror. Verified by direct file read of the fenced bash blocks (lines 220-242) and cross-checked against the absence of any `.claude/commands/ant/skill-list.md`, `skill-index.md`, etc. wrapper. HIGH confidence on the mechanics; marked ASSUMED only because I have not executed the actual scanner code (it does not exist yet) to confirm it reaches the same conclusion — a subtle extraction bug could still produce a different number. | Summary, Pitfall 1 | Low — D-07 already requires the scan's real output to be trusted over any pre-stated count, so this assumption cannot cause a wrong build; it only means the planner should not be surprised if the seeded baseline differs from "8" |
| A2 | `.aether/*.md` in WIRE-03's success criterion means the top-level, non-recursive glob (4 tracked files: `CONTEXT.md`, `CROWNED-ANTHILL.md`, `QUEEN.md`, `workers.md`), not `.aether/**/*.md`. Reasoned from literal glob syntax plus the fact that this is the minimal scope that satisfies the seeded `workers.md:292` failure with zero unrelated noise (verified no fenced `aether` invocations exist in the other 3 top-level files). Not directly confirmed by the user. | Architecture Patterns → Pattern 2, Pitfall 3, Open Question 1 | Medium — if the intended scope is actually broader (e.g., also `.aether/docs/*.md` beyond the already-covered `command-playbooks/` subfolder), the test would need a second, larger corpus pass with unvalidated false-positive risk from 769+ untriaged doc invocations |
| A3 | D-01(a)'s "Go code outside the command's own definition file" caller category should be implemented narrowly as a search for `exec.Command("aether"...)`/self-invocation string-literal patterns, and is expected to contribute zero matches in this codebase today. Based on an exhaustive grep finding no such self-invocation pattern anywhere in `cmd/` or `pkg/`. | Pitfall 5, Open Question 2 | Medium — if the intended meaning is broader (e.g., "any Go code that transitively reaches the same business logic," which DOES have instances via `matchSkillsForWorkflow`), the seeded orphan set could shrink to include `skill-match`/`skill-inject` incorrectly, undermining the entire ratchet's premise |
| A4 | `spawn-can-spawn`'s bare (non-`--enforce`) invocation should keep returning exit 0 with `can_spawn` in the JSON payload even when the (future, Phase 173) decision is `false` — i.e., `--enforce` is what upgrades a report into a gate, matching the existing `gate-check` precedent (`cmd/gate.go` — always exits via `outputOK`, pass/fail lives in the JSON) for the unflagged case. Not explicitly stated in CONTEXT.md, inferred from D-13's exact wording ("`--enforce` is implemented with real semantics... a deny answer causes a non-zero exit" — implying the non-`--enforce` path does not) and from the one behavioral precedent (`gate-check`) found in this codebase. | Code Examples → spawn-can-spawn fix | Low — this only affects a flag (`--enforce`'s absence) that Phase 173 will exercise for real; getting it wrong changes when the deny becomes a hard failure, not whether the mechanism exists |

**If this table is empty:** N/A — see above.

## Open Questions (RESOLVED)

All three were answered before or during planning. Each keeps its original wording so a
future reader can see what was uncertain at research time, followed by a **Resolution**
line naming where the answer was made.

1. **Does `.aether/*.md` in WIRE-03 mean the literal top-level glob, or "anything markdown
   under `.aether/`"?**
   - What we know: the seeded failure (`workers.md:292`) is satisfied by the narrowest
     possible reading (4 top-level tracked files). The 3 other top-level files
     (`CONTEXT.md`, `QUEEN.md`, `CROWNED-ANTHILL.md`) are runtime session logs, not
     instruction documents, and contribute zero fenced `aether` invocations, so including
     them is safe but not load-bearing.
   - What's unclear: whether the roadmap author intended the glob literally, or as shorthand
     for "the parts of `.aether/` that carry worker instructions" (which would also pull in
     `.aether/docs/`, `.aether/skills/`, `.aether/references/`, `.aether/templates/` —
     250 tracked `.md` files, largely unvalidated against this extractor's heuristics).
   - Recommendation: implement the literal top-level glob (satisfies every stated success
     criterion with the least new false-positive surface) and record the interpretation
     explicitly in the plan/commit message so a future reader can see the choice was made on
     purpose, not by omission. If the planner or a downstream reviewer wants the broader
     scope, that is a one-line corpus change later, not a redesign.
   - **Resolution:** the literal top-level glob. Decided by the user before planning:
     the four git-tracked `.aether/*.md` files plus `.aether/docs/command-playbooks/*.md`,
     explicitly **not** recursive over `.aether/**/*.md`. Recorded in `172-VALIDATION.md`
     section "Resolved Scope Call" and in `172-03-PLAN.md`'s `<corpus_scope>` block.

2. **What exactly counts as "Go code outside the command's own definition file" (D-01a)
   caller evidence, given the pattern has zero real examples today?**
   - What we know: no `exec.Command("aether"...)` self-invocation exists anywhere in the
     codebase. Shared business-logic functions (e.g., `matchSkillsForWorkflow`) ARE called
     directly by other Go files, bypassing the CLI subcommand entirely.
   - What's unclear: whether D-01(a) should ever fire for a case like `skill-match`/
     `skill-inject`, where the *logic* is demonstrably used but the *subcommand entry point*
     is not.
   - Recommendation: implement narrowly (self-invocation string-literal search only, per
     Assumption A3) since that is the only reading consistent with "a caller is something
     that executes the command" (D-01's own definition — using shared logic is not executing
     the command, it's reusing code). Flag this choice for the discuss-phase or planner to
     confirm before implementation, since getting it wrong in the loose direction would
     silently clear real orphans (`skill-match`, `skill-inject`) that the phase explicitly
     wants caught.
   - **Resolution:** implement narrowly, as recommended. Fixed in `172-02-PLAN.md`'s
     `<caller_corpora>` block, clause (a): only a string-literal `exec.Command("aether", ...)`
     / `exec.Command(os.Args[0], ...)` self-invocation counts, and a shared business-logic
     function being called elsewhere explicitly does **not**. Expected to contribute zero
     hits today, and implemented anyway so the category is honest rather than absent.

3. **Exact reason-tag shape and allowlist file format (Claude's Discretion, but worth a
   concrete recommendation).**
   - What we know: D-08 requires each entry to name a reason and an owning phase (example
     given: `skill-lifecycle / phase-178`). D-09 requires its own dedicated file, not
     `command_catalog.json`.
   - What's unclear: JSON vs. plain text; single string vs. structured fields.
   - Recommendation: `cmd/testdata/orphan_allowlist.json`, an array of
     `{"name": "skill-index", "reason": "skill-lifecycle", "owner_phase": "178"}` objects —
     structured fields (not one free-text string) so `TestOrphanAllowlistOnlyShrinks` and any
     future Phase 178 test can query by `owner_phase` programmatically, which a flat reason
     string would make brittle to parse.
   - **Resolution:** adopted as recommended. The shape is fixed in `172-PATTERNS.md`
     section `cmd/testdata/orphan_allowlist.json` and in `172-02-PLAN.md` task 1 step 5:
     a JSON array of objects with exactly `name`, `reason`, `owner_phase`, sorted by
     `name`, with `skill-*` orphans tagged `owner_phase: "178"` and everything else
     `"RECLAIM"`. `172-04-PLAN.md` reuses the same shape for the flag audit's skip list.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All three requirements (test authoring, `go build`) | ✓ | go1.26 (per `.github/workflows/ci.yml` `go-version: "1.26"`; local toolchain confirmed working via successful `go build ./cmd/aether` during this research session) | — |
| `go/ast`, `go/parser`, `go/token` | WIRE-01 enumeration | ✓ | stdlib, already imported in 2 existing test files | — |
| `github.com/spf13/cobra`, `github.com/spf13/pflag` | All three requirements | ✓ | already vendored/direct deps | — |
| GitHub Actions runner (`ubuntu-latest`) | D-15's named CI step | ✓ | `.github/workflows/ci.yml` confirmed present and running `go test ./...` today | — |

**Missing dependencies with no fallback:** none.

**Missing dependencies with fallback:** none.

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go's built-in `testing` package (no third-party test framework anywhere in `cmd/`) |
| Config file | none — Go tests need no config; CI invocation is `.github/workflows/ci.yml`'s "Run Go tests" step |
| Quick run command | `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced|TestOrphanAllowlist|TestCommandCallsMatchCobraContracts|TestSpawnCanSpawn' -v` |
| Full suite command | `go test ./... -count=1 -timeout 900s` (matches CI's "Run Go tests" step exactly) |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| WIRE-01 | A registered subcommand with no caller outside its own file fails the ratchet, seeded honestly, shrink-only | unit (static analysis + fixture) | `go test ./cmd -run TestNoRegisteredSubcommandIsUnreferenced -v` | ❌ Wave 0 — new file `cmd/subcommand_reachability_ratchet_test.go` |
| WIRE-01 | The allowlist may only shrink vs. the committed baseline | unit | `go test ./cmd -run TestOrphanAllowlistOnlyShrinks -v` | ❌ Wave 0 — same new file |
| WIRE-01 (D-12) | The flag audit's `skipSubcommands` map gets the same shrink-only treatment | unit | `go test ./cmd -run TestFlagAuditSkipListOnlyShrinks -v` (name TBD, Claude's discretion) | ❌ Wave 0 |
| WIRE-02 | `aether spawn-can-spawn 5 --enforce` exits 0 today (no visible behaviour change yet) | integration (subprocess or in-package `RunE` invocation) | `go test ./cmd -run TestSpawnCanSpawnAcceptsDocumentedInvocation -v` (name TBD) | ❌ Wave 0 — add to `cmd/spawn_test.go` or a new `cmd/spawn_enforce_test.go` |
| WIRE-02 | `--enforce` causes non-zero exit on a real deny (forward-looking; unreachable today since `can_spawn` is hardcoded true) | unit | test constructs the command with a stubbed deny path, or asserts the exit-code wiring exists via `markRenderedCommandError` — exact mechanism is an implementation detail | ❌ Wave 0 |
| WIRE-03 | `.aether/workers.md:292`'s `--enforce` flag is caught as unregistered before the fix, and passes after | unit (this is the existing `TestCommandCallsMatchCobraContracts`, corpus-extended) | `go test ./cmd -run TestCommandCallsMatchCobraContracts -v` | ✅ exists (`cmd/command_call_audit_test.go`) — extend `auditedCorpora`, no new test file needed |
| WIRE-03 | A `.aether/*.md` flag violation names file, line, and flag | unit | same as above — verified today that `validateCallAgainstCobra`'s error strings already include `unknown flag --%s` | ✅ exists |

### Sampling Rate
- **Per task commit:** `go test ./cmd -run 'TestNoRegisteredSubcommandIsUnreferenced|TestOrphanAllowlist|TestCommandCallsMatchCobraContracts|TestSpawnCanSpawn|TestDocumentedCommandNamesResolve' -v`
- **Per wave merge:** `go test ./cmd/... -count=1 -timeout 900s -v` (the full `cmd` package, since these tests interact with the shared `rootCmd` global and could have ordering side effects worth catching early)
- **Phase gate:** `go test ./... -count=1 -timeout 900s` (matches the exact CI "Run Go tests" step) green, AND the new named CI step(s) added per D-15 green, before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `cmd/subcommand_reachability_ratchet_test.go` — new file: `TestNoRegisteredSubcommandIsUnreferenced`, `TestOrphanAllowlistOnlyShrinks`, and a fixture self-test proving the ratchet actually detects a synthetic orphan (mirroring `TestAuditDetectsPositionalDrift`'s function-local `AddCommand`/`defer RemoveCommand` pattern — see Pitfall 6)
- [ ] `cmd/testdata/orphan_allowlist.json` — new committed baseline file, seeded from the scanner's real (not pre-assumed) output
- [ ] `.aether/docs/` policy doc for the allowlist (optional but recommended, following the `command-catalog-policy.md` precedent named in Don't Hand-Roll)
- [ ] Extend `cmd/cli_flag_audit_test.go`'s `skipSubcommands` handling for D-12's shrink-only rule (currently an unguarded 2-entry map)
- [ ] `cmd/spawn.go` — add `--enforce` bool flag, relax `Args` to `cobra.MaximumNArgs(1)`, parse the positional depth, wire deny→non-zero-exit (unreachable today, but the wiring must exist per D-13)
- [ ] `.github/workflows/ci.yml` — two new named steps per D-15 (or one combined step covering both new test groups; Claude's discretion), inserted near the existing "Verify command catalog classification" step
- [ ] `cmd/command_call_audit_test.go` — extend `auditedCorpora` with the 4 top-level `.aether/*.md` files (see Open Question 1 for the scope call)
- [ ] Framework install: none — `go test` is already the toolchain

*(No existing test infrastructure covers these three requirements; all listed gaps are real.)*

## Security Domain

This phase's `security_enforcement` status was not found set to `false` in
`.planning/config.json` (the file has no such key), so this section is included per the
template's default-enabled rule. In substance, this phase has an unusually thin security
surface: it adds no network endpoint, no user-input parsing of untrusted external data (the
markdown/YAML corpora scanned are files already checked into this repo, authored by the
project's own maintainers), and no authentication/session/crypto code.

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | No auth surface in this phase |
| V3 Session Management | no | No session surface in this phase |
| V4 Access Control | no | No access-control surface; the "no escape hatch" (D-11) is a process/governance control, not an ASVS access-control mechanism |
| V5 Input Validation | marginal | The markdown/YAML corpus files are trusted repo-internal content, not attacker-controlled input — but the extractor should still not panic/crash on malformed markdown (existing `extractDocumentedCalls` already tolerates missing files via `continue`, `cmd/command_call_audit_test.go:181-183`); reuse that defensive pattern for any new corpus reads |
| V6 Cryptography | no | No cryptographic operations in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| A malicious PR adds a new orphaned subcommand alongside a fabricated allowlist entry to smuggle it past review | Tampering | D-11's "no escape hatch" + D-10's "compare against a committed baseline copy" already mitigate this: any new allowlist entry not present in the last-reviewed baseline fails CI, and the baseline itself is a visible diff in code review — there is no runtime bypass, only a reviewable file edit |
| A CI-only bypass (e.g., an env var that disables the ratchet) reintroduces silent orphan accumulation | Tampering / Repudiation | Do not implement any `SKIP_RATCHET`-style environment override — none of the existing audit tests in this repo have one (confirmed by grep for `os.Getenv` inside `command_call_audit_test.go`, `cli_flag_audit_test.go` — none found), and D-11 explicitly forbids an escape hatch for WIRE-01 specifically |

## Sources

### Primary (HIGH confidence — direct file reads/greps performed in this session)
- `cmd/spawn.go` (full `spawnCanSpawnCmd`, `init()`, sibling `spawnLogCmd`) — current broken state of WIRE-02
- `.aether/workers.md` lines 1-400 (grepped for all `aether` invocations, read lines 270-310 in full) — the documented WIRE-02 string and its surrounding protocol
- `cmd/command_call_audit_test.go` (full file, 752 lines) — the existing WIRE-03-adjacent audit family (`TestCommandCallsMatchCobraContracts`, `TestAuditDetectsPositionalDrift`, `TestDocumentedCommandNamesResolve`, `TestDocumentedSubcommandsAreSeverityClassified`, `TestGateClassifiedCallsHaveGateWiring`), `auditedCorpora`, `extractDocumentedCalls`, `validateCallAgainstCobra`
- `cmd/cli_flag_audit_test.go` (full file, 188 lines) — the older regex-based audit, its `skipSubcommands` map (D-12's target)
- `cmd/visual_writer_discipline_test.go` (full file) — the `declNameAndBody` AST var-block-walk precedent for WIRE-01
- `cmd/go_source_hint_audit_test.go` (partial) — the third member of the audit family, confirms Go-source string-literal hints are already separately checked
- `cmd/command_call_severity.go` (full file) — the gate/enrichment classification precedent
- `.aether/docs/command-catalog-policy.md` (partial) — the policy-doc-plus-gate template precedent
- `cmd/skills.go`, `cmd/skill_lifecycle.go` — all 8 `skill-*` `Use:` declarations and their `init()` registrations
- `.claude/commands/ant/skill-create.md` lines 200-287, `.opencode/commands/ant/skill-create.md` (same lines) — the caller evidence for `skill-parse-frontmatter`/`skill-cache-rebuild` (Assumption A1)
- `cmd/codex_build.go` line 2869, `cmd/platform_sync.go` line 119 — the direct-function-call vs. printed-hint distinction (Pitfall 5)
- `.github/workflows/ci.yml` (full file) — the exact release-gate command (D-15's target) and the "Verify command catalog classification" named-step precedent
- `Makefile` line 37-38 — the secondary local `make test` entry point
- `scripts/classify_commands.py` line 191, `scripts/verify_catalog_classified.py` — confirms `command_catalog.json` is unsafe for allowlist storage (D-09) and that no other `cmd/testdata/*.json` file is touched by that script
- `.aether/commands/skill-create.yaml` — an example `.aether/commands/*.yaml` menu-entry spec (D-03 mechanism)
- `.gitignore` lines 81-99 — confirms which `.aether/*.md` files are tracked vs. session-local
- `git ls-files '.aether/*.md' '.aether/**/*.md'` — the exact 250-file recursive count vs. 4-file top-level count underlying Pitfall 3/Open Question 1
- `.planning/phases/172-wiring-proof/172-CONTEXT.md`, `172-DISCUSSION-LOG.md` — locked decisions and their rationale
- `.planning/REQUIREMENTS.md`, `.planning/ROADMAP.md`, `.planning/STATE.md` — requirement text, success criteria, milestone framing
- Direct `go build ./cmd/aether` executed in this session — confirmed the toolchain builds clean and `--help` lists ~369 top-level commands

### Secondary (MEDIUM confidence)
None used — this phase required no external/web research; every claim traces to a primary
repo-internal source verified in this session.

### Tertiary (LOW confidence)
None.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — zero new dependencies, all verified present via direct grep of existing imports
- Architecture: HIGH for WIRE-02/WIRE-03 (directly traced through existing, working code); MEDIUM for WIRE-01's shrink-only comparator (the pattern to reuse for enumeration is proven, but the comparator itself has no precedent in this repo, so its exact shape is a design decision, not a verified fact)
- Pitfalls: HIGH — all six are grounded in direct file reads/greps performed this session, not inference

**Research date:** 2026-08-08
**Valid until:** This is repo-internal, near-zero external-dependency research; it is stable until the underlying `cmd/*.go` files it cites are next modified by another phase. Recommend re-verifying line numbers (not conclusions) if planning is delayed past other in-flight `cmd/` work — 30 days is a reasonable outside bound given this repo's commit velocity.
