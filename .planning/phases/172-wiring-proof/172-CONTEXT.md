# Phase 172: Wiring Proof - Context

**Gathered:** 2026-08-08
**Status:** Ready for planning

<domain>
## Phase Boundary

This phase delivers **the standard**, not a capability. Two automated checks and one
broken command fixed:

1. A test that fails when a registered cobra subcommand has no caller outside its own
   definition file, seeded with a shrink-only baseline of today's orphans (WIRE-01).
2. `aether spawn-can-spawn 5 --enforce` — the exact string `.aether/workers.md:292`
   instructs every worker to run — succeeds instead of erroring (WIRE-02).
3. A test that fails, naming file/line/flag, when any `.aether/*.md` instruction invokes
   a CLI flag the binary does not register (WIRE-03).

It lands first in v1.26 so the ratchet is shaped by the standard rather than by whatever
the later phases happen to ship.

**Explicitly NOT in this phase:** making `spawn-can-spawn` return a real deny decision
(Phase 173), reclaiming or removing the 8 `skill-*` orphans (Phase 178), any new
delegation, spend, roster or skill capability.

*Plain English: this builds the smoke alarm before the next eight phases build the things
that could catch fire. Nothing new gains the ability to do anything here.*

</domain>

<decisions>
## Implementation Decisions

### What counts as a caller (the reachability rule)

- **D-01:** A caller is something that **executes** the command. Three kinds count:
  (a) Go code outside the command's own definition file, (b) a platform wrapper the
  runtime actually loads — `.claude/commands/ant/*.md`, `.opencode/commands/ant/*.md`,
  `.aether/commands/*.yaml` — and (c) a shipped hook or script. Nothing else.
- **D-02:** Documentation does **not** count. `CLAUDE.md`, `AGENTS.md`, `docs/specs/*`,
  `.aether/docs/*`, audit reports, `cmd/testdata/command_catalog.json` and
  `cmd/testdata/parity_snapshot.json` are all excluded as caller evidence. This is
  load-bearing: the 8 known `skill-*` orphans appear in ~20 files today and reach nobody.
  Under any looser rule the ratchet passes on day one and never fires again.
- **D-03:** A **menu entry counts as a caller.** If a `/ant-…` wrapper or a
  `.aether/commands/*.yaml` spec names the command, it is user-reachable and therefore
  called. This is what prevents the hundreds of legitimately human-invoked commands
  (`status`, `spend`, `focus`) from flooding the baseline.
- **D-04:** **Rejected:** exempting commands by their `classification` tier in
  `cmd/testdata/command_catalog.json`. Those tiers are auto-assigned by
  `scripts/classify_commands.py` from name heuristics, so a single mis-guessed tier would
  silently grant a permanent exemption with no one reviewing it.
- **D-05:** **Test files are not callers.** A command whose only reference outside its own
  file is `*_test.go` is an orphan. A passing test proves the command works; it does not
  prove anything uses it — and "works, and nothing calls it" is precisely the failure mode
  this phase exists to catch.
- **D-06:** The retired playbooks (`.aether/docs/command-playbooks/*.md`, not loaded by the
  runtime since v1.25) stay **in scope for the flag audit** — a stale doc naming a
  non-existent flag is still misleading to a human reader — but a mention there is **never**
  caller evidence. This is exactly how `suggest-analyze` stayed marked complete for a
  milestone with its only call site in an unloaded playbook behind `2>/dev/null`.

### The baseline allowlist

- **D-07:** Every orphan the scan finds goes into the baseline **honestly**, even if the
  real number is far above the 8 expected. This phase does not expand into repair work.
- **D-08:** Each entry carries a **reason tag** identifying why it is tolerated and which
  phase owns it (e.g. `skill-lifecycle / phase-178`). Required because Phase 178's success
  criterion is *"the allowlist drops from 8 skill entries to 0"* — with an untagged flat
  list of possibly 60 entries, that criterion becomes unmeasurable.
- **D-09:** The allowlist lives in **its own dedicated file**, not as a field on
  `cmd/testdata/command_catalog.json`. Reason: `scripts/classify_commands.py` rewrites that
  catalog in place (`json.dump` at line 191), so allowlist entries stored there could be
  silently erased by a routine regeneration. A debt record that can vanish is not a record.
- **D-10:** Shrink-only is enforced by **comparing against a committed baseline copy**. Any
  name present in the live list and absent from the baseline fails the test.
  **Rejected:** counting entries (a one-out-one-in swap passes silently) and comparing
  against git history (CI shallow clones and rewritable history make it fail or pass for
  reasons unrelated to the list).
- **D-11:** **No escape hatch.** Additions to the allowlist are hard-blocked. There is no
  reason-field override, no reviewer sign-off path. Build it and connect it, or delete it.
  Deliberately changing the committed baseline is possible but is a visible, on-the-record
  edit to a checked-in file — not a runtime bypass.
- **D-12:** The flag audit's existing `skipSubcommands` map in `cmd/cli_flag_audit_test.go`
  (currently 2 entries, unguarded) gets the **same shrink-only treatment**. Otherwise
  exemption pressure simply relocates to the unguarded list.

### The documented safety command (WIRE-02)

- **D-13:** `--enforce` is implemented with **real semantics now**: a deny answer causes a
  non-zero exit. Today `spawn-can-spawn` unconditionally returns `can_spawn: true`, so the
  observable behaviour is unchanged and the criterion *"exits 0"* holds — but when Phase 173
  makes the answer honest, enforcement already works with no further wiring.
  **Rejected:** registering `--enforce` as an inert accepted-but-ignored flag. Shipping a
  flag whose name promises enforcement and delivers none is the exact anti-pattern this
  milestone is named after.
- **D-14:** `spawnCanSpawnCmd` currently declares `Args: cobra.NoArgs`, so the documented
  invocation fails on the positional `5` as well as on the unknown flag. **Fix the command,
  not the manual** — accept the positional depth exactly as `.aether/workers.md:292` writes
  it. **Rejected:** rewriting `workers.md` to use the existing `--depth` flag, because
  instruction docs in this repo have a demonstrated history of drifting out of sync with
  the runtime, and the criterion is pinned to the string the manual actually contains.

### CI surfacing

- **D-15:** Both checks get **their own named CI step** running just the wiring and flag
  tests, in addition to the existing `go test ./...` coverage. A failure must read as a
  wiring problem, not as one anonymous failure among hundreds. Criterion 4's proof is
  behavioural — delete a caller, watch the gate go red — not a reading of the workflow file.

### Claude's Discretion

- Exact file name, format (JSON vs. plain text) and location of the allowlist and its
  baseline copy.
- Test names beyond the roadmap-pinned `TestNoRegisteredSubcommandIsUnreferenced`.
- How the caller scan is implemented (AST walk vs. text scan) and how it identifies a
  command's "own definition file".
- Whether the flag audit is an extension of the existing `TestCLIFlagAudit` or a new test —
  provided `.aether/*.md` enters scope and failures name file, line and flag.
- The precise shape of the reason tag.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase definition and standard
- `.planning/ROADMAP.md` §"Phase 172: Wiring Proof" (lines 619-628) — goal, the four
  success criteria, and the pinned test name
- `.planning/ROADMAP.md` §"v1.26 Intelligent Orchestration" (lines 35-64) — the organising
  finding and why the ordering is not negotiable
- `.planning/REQUIREMENTS.md` §"Wiring Proof (WIRE)" (lines 30-34) — WIRE-01, WIRE-02, WIRE-03
- `CLAUDE.md` §"Definition of Done" — the governing standard: a requirement is satisfied only
  when a command exists that someone can run, and that command fails when the requirement is
  unmet. Also the corollary preferring proportion/invariant assertions over section-presence

### The thing being fixed (WIRE-02)
- `.aether/workers.md` §"Step-by-Step Spawn Protocol" line 292 — the documented invocation
  `aether spawn-can-spawn {your_depth} --enforce` that errors today
- `cmd/spawn.go:169-182` — `spawnCanSpawnCmd`; `Args: cobra.NoArgs`, no `--enforce`,
  returns `can_spawn: true` unconditionally
- `cmd/spawn.go:358-370` — the `init()` flag registrations for the spawn command family

### Existing machinery to extend or avoid
- `cmd/cli_flag_audit_test.go` — `TestCLIFlagAudit` already scans `.claude/commands/ant/`,
  `.opencode/commands/ant/` and `.aether/docs/command-playbooks/` for `aether <cmd> --flag`
  and checks registration. Does **not** cover `.aether/*.md`. Holds the unguarded
  `skipSubcommands` map from D-12
- `cmd/testdata/command_catalog.json` — 400 command entries with `name`, `flags`,
  `classification`, `since_version`. Read-only reference for this phase; **not** the
  allowlist home (D-09)
- `scripts/classify_commands.py:191` — rewrites the catalog in place; the reason D-09 rules
  the catalog out as storage
- `.aether/docs/command-catalog-policy.md` — the written policy for the catalog and its
  classification tiers; the precedent for how a shrink-only policy doc should read
- `scripts/verify_catalog_classified.py` — the existing catalog CI gate; precedent for a
  named CI step (D-15)
- `.github/workflows/ci.yml` — the release gate. `go test ./... -count=1` already runs
  everything under `cmd/`; the named step from D-15 is added here

### Downstream dependents (do not implement, but do not block)
- `.planning/ROADMAP.md` §"Phase 173: Delegation Guard" — owns the real deny decision that
  D-13's `--enforce` will carry
- `.planning/ROADMAP.md` §"Phase 178: Skill Authoring Hardening" criterion 1 — measures
  success as the allowlist's 8 skill entries dropping to 0, which is why D-08 requires tags

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `cmd/cli_flag_audit_test.go`: a working markdown→CLI scanner with alias resolution,
  per-flag checking and deduplicated failure output. WIRE-03 is closer to an extension of
  this than a new build — the missing pieces are `.aether/*.md` coverage and a guarded
  skip-list.
- `rootCmd.Commands()` + `Flags().VisitAll` (same file, lines 43-58): the established way
  to enumerate every registered subcommand and its flags at test time. The orphan ratchet
  needs the same enumeration.
- `cmd/testdata/command_catalog.json`: a maintained inventory of all 400 commands, usable
  as cross-check data even though it is not the allowlist home.
- `scripts/verify_catalog_classified.py` + its CI step: the working template for
  "policy doc + checked-in data file + named CI gate".

### Established Patterns
- Tests in `cmd/` run against the live `rootCmd` in-process and read repo files by relative
  path (`../.claude/...`). No fixture harness needed.
- Data that gates CI lives in `cmd/testdata/` as checked-in JSON with a companion policy
  doc in `.aether/docs/`.
- `go test ./... -count=1` in CI already picks up any new `cmd/*_test.go` automatically —
  D-15's named step is for legibility, not coverage.

### Integration Points
- `cmd/spawn.go` `init()` — where `--enforce` is registered and `Args` is relaxed.
- `.aether/workers.md:292` — the documented string that must execute successfully; leave
  the wording alone per D-14.
- `.github/workflows/ci.yml` — one new named step alongside "Verify command catalog
  classification".

### Known Trap
- 8 `skill-*` commands (`skill-index`, `skill-detect`, `skill-match`, `skill-inject`,
  `skill-list`, `skill-diff`, `skill-cache-rebuild`, `skill-parse-frontmatter`) resolve to
  ~20 referencing files under a naive grep, including `CLAUDE.md`, `AGENTS.md`,
  `docs/specs/2026-03-22-aether-skills-layer-design.md`, both catalog JSON files, and
  `.aether/docs/command-playbooks/build-context.md`. Any implementation that treats those
  as callers produces a ratchet that reports zero orphans on a codebase with at least 8.
  **This is the acceptance trap for WIRE-01.**

</code_context>

<specifics>
## Specific Ideas

- The user's framing for D-13, kept because it captures the intent precisely: the manual
  tells workers to "phone the office and check you're allowed, and do what they say." The
  phone line is broken (this phase fixes it) and the office currently says yes to everyone
  (Phase 173 fixes that). "And do what they say" must be real machinery now, so that when
  the office learns to say no, the obedience already exists.
- Failure output is for a non-technical operator: the orphan ratchet must name the command,
  and the flag test must name file, line and flag. "Tests failed" is not an acceptable
  surface for either.

</specifics>

<deferred>
## Deferred Ideas

- **Making `spawn-can-spawn` return a real deny decision** — Phase 173 (SPAWN-01).
  Explicitly out of scope here; this phase only makes the documented invocation executable.
- **Reclaiming or removing the 8 `skill-*` lifecycle commands** — Phase 178 (SKILL-01),
  measured by those allowlist entries reaching 0.
- **The wider unreachable-command sweep** beyond the ratchet itself — already parked as
  RECLAIM in `.planning/REQUIREMENTS.md` §Future Requirements.
- **Retiring the playbook directory entirely** — came up while ruling on D-06. Not this
  phase; the files remain human reference material.

### Reviewed Todos (not folded)
- `2026-08-01-finalize-reconcile-task-evidence-gate.md` — "continue-finalize does not count
  `--reconcile-task` as recorded reconciliation for the implementation_evidence gate."
  Matched on generic keywords only. It is a runtime behaviour bug, not a wiring question.
  Stays in the backlog.
- `2026-08-01-ts-host-preflight-hardcoded-timeout.md` — "ts-host preflight timeout hardcoded,
  not configurable, runs in repo cwd." Same reason; also already carried as inserted
  Phase 163.2 in the v1.25 roadmap. Stays in the backlog.

</deferred>

---

*Phase: 172-Wiring Proof*
*Context gathered: 2026-08-08*
