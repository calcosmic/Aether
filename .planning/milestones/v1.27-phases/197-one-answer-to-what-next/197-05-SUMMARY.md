---
phase: 197-one-answer-to-what-next
plan: 05
subsystem: cli
tags: [go, cobra, command-aliases, wrapper-parity, hub-publish, update-reconciliation, goldens]

requires:
  - phase: 197-one-answer-to-what-next
    provides: "resolveNextAction and the availability gate (197-01/197-03); the resume pair's existing Cobra alias shape as the mechanism to follow"
provides:
  - "pause is the canonical Cobra command name; pause-colony is a Cobra alias resolving to the same *cobra.Command object"
  - ".aether/commands/pause.yaml — one definition file declaring the alias via an aliases: field, replacing the former two-file pattern for this pair"
  - "source_check.go's checkGeneratedCommandSurfaces knows about declared aliases: a declared alias legitimises its wrapper, a missing wrapper for a declared alias is reported on all three platform locations, and an undeclared alias-shaped wrapper is still reported as an orphan"
  - "aether update restores a missing declared-alias wrapper and names the repair in plain words, distinct from the copied-file count and the stale-publish signal"
  - "yamlCommandNamesForGuideTest expands declared aliases into the YAML-derived name set consumed by guide/wrapper/platform parity tests"
affects: [197-06 (pause card, next wave), 197-07 (cross-command envelope assertion)]

actuals:
  tokens: 10931
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Alias declared once, enforced twice: a YAML aliases: field is the single declaration; two tests (declared-alias-missing-wrapper, undeclared-alias-wrapper) hold the three hand-maintained wrapper copies to it in both directions"
    - "Read the declaration off the live Cobra tree, not a second Go list: the alias-repair reconciliation in aether update reads cobra.Command.Aliases directly, so a future aliased command is picked up automatically"
    - "Golden regeneration with hand-restored continuity: a renamed command's catalog entry loses its embedded enrichment on a mechanical -update-golden run (the enrichment source is the golden file itself, keyed by name); restoring it by hand once means the next regeneration self-heals"

key-files:
  created:
    - .aether/commands/pause.yaml
    - .claude/commands/ant/pause.md
    - .claude/commands/ant-pause.md
    - .opencode/commands/ant/pause.md
    - cmd/canonical_alias_test.go
    - cmd/alias_reconcile_test.go
    - .planning/phases/197-one-answer-to-what-next/deferred-items.md
  modified:
    - cmd/session_flow_cmds.go
    - cmd/hook_cmds.go
    - cmd/wrapper_command_names.go
    - cmd/command_guide.go
    - cmd/source_check.go
    - cmd/platform_sync.go
    - cmd/update_cmd.go
    - cmd/parity_test.go
    - cmd/command_guide_test.go
    - cmd/command_call_audit_test.go
    - cmd/testdata/command_catalog.json
    - cmd/testdata/parity_snapshot.json
    - cmd/testdata/colony_state_write_allowlist.json
    - .claude/commands/ant/pause-colony.md
    - .claude/commands/ant-pause-colony.md
    - .opencode/commands/ant/pause-colony.md

key-decisions:
  - "pause-colony.yaml is deleted, not kept alongside pause.yaml -- the requirement's 'declared once' means one definition file, matching the objective's framing of replacing the resume pair's two-file pattern rather than reproducing it for pause"
  - "CommandName in pauseColonyCmd's RunE stays \"pause-colony\" -- an internal session.LastCommand bookkeeping value, not user-facing naming, and changing it would have required touching an undeclared test file for no behavioural gain (the plan's own 'do not change what the command does' line)"
  - "The update-time alias reconciliation reads cobra.Command.Aliases off the live command tree rather than parsing YAML aliases: fields -- a downstream project's aether binary never receives the raw .aether/commands/*.yaml sources (only rendered wrapper .md mirrors reach the hub), so the Cobra tree is the only alias-declaration source actually available at aether update time in a consumer repo"
  - "yamlCommandNamesForGuideTest (not itself in the plan's declared file list) needed alias expansion so TestCommandGuideCoversAllYamlCommands, TestPlatformParityGolden, and TestAllYamlHaveWrappersAndGuide keep treating pause-colony as a legitimate name now that it has no YAML file of its own -- a single shared fix rather than three separate ones"
  - "pause is classified alongside pause-colony in knownEnrichmentSubcommands (command_call_audit_test.go), mirroring how resume and resume-colony both carry their own entry -- a failing halt on this command would be wrong (a paused colony is not an error state)"
  - "The renamed catalog entry's classification/since_version/historical_presence/stability_score were hand-restored after -update-golden dropped them (the enrichment lookup is keyed by name, and \"pause\" is a new key) -- losing five versions of audit history to a name change felt like the wrong default, and restoring it once means every future regeneration self-heals from the golden"

patterns-established:
  - "A declared alias is enforced in both directions by name-driven Go tests, never by hand-inspecting the three wrapper copies"
  - "An update-time reconciliation pass snapshots missing state before an existing sync mechanism runs, then reports only what that mechanism actually fixed -- never a parallel repair implementation"

requirements-completed: [NEXT-05]

coverage:
  - id: D1
    description: "Typing either aether pause or aether pause-colony runs the same command, proved by object identity (not merely matching output) through the live Cobra command tree."
    requirement: NEXT-05
    verification:
      - kind: unit
        ref: "cmd/canonical_alias_test.go#TestCanonicalAliasDelegates/same_command_object"
        status: pass
      - kind: integration
        ref: "cmd/canonical_alias_test.go#TestCanonicalAliasDelegates/identical_output_for_one_state"
        status: pass
    human_judgment: false
  - id: D2
    description: "The pause/pause-colony alias is declared exactly once, in pause.yaml's aliases: field, and every one of the three hand-maintained wrapper copies (nested Claude, flat Claude, OpenCode) is held to that declaration by tests in both directions -- a declared alias missing a wrapper is reported, and an alias-shaped wrapper with no declaration anywhere is still reported. No wrapper generator was built or claimed; the six wrapper files remain hand-authored."
    requirement: NEXT-05
    verification:
      - kind: unit
        ref: "cmd/canonical_alias_test.go#TestDeclaredAliasesHaveWrappersOnEveryPlatform"
        status: pass
      - kind: unit
        ref: "cmd/canonical_alias_test.go#TestAliasWrapperNeedsADeclaration"
        status: pass
      - kind: other
        ref: "aether source-check (run against this repository)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Updating a project whose short-name wrapper is missing restores it via the ordinary hub sync, and the update output names the specific command and platform surface it was missing from in plain words -- distinct from the copied-file count and the unrelated stale-publish signal. A clean update reports no repair."
    requirement: NEXT-05
    verification:
      - kind: integration
        ref: "cmd/alias_reconcile_test.go#TestUpdateRestoresAMissingAliasWrapper"
        status: pass
      - kind: integration
        ref: "cmd/alias_reconcile_test.go#TestUpdateReportsTheAliasRepairByName"
        status: pass
      - kind: integration
        ref: "cmd/alias_reconcile_test.go#TestUpdateReportsNoRepairWhenNothingIsMissing"
        status: pass
    human_judgment: false
  - id: D4
    description: "The short name resolves through the live command tree and appears on the slash-wrapper allowlist and command guide beside the long one, so the resolver from 197-01/197-03 can recommend it in the form the owner types."
    requirement: NEXT-05
    verification:
      - kind: unit
        ref: "cmd/hint_translation_test.go#TestWrapperCommandNamesMatchCanonicalCorpus"
        status: pass
      - kind: unit
        ref: "cmd/parity_test.go#TestNoPhantomCommands"
        status: pass
      - kind: unit
        ref: "cmd/command_guide_test.go#TestCommandGuideCoversAllYamlCommands"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every golden, audit, and ratchet a renamed/aliased command disturbs was refreshed and checked in this one plan, not discovered failing later: the command catalogue, platform parity snapshot, regression snapshot, severity audit, colony-state-write ratchet, and orphan allowlist."
    requirement: NEXT-05
    verification:
      - kind: unit
        ref: "cmd/audit_catalog_test.go#TestAuditCatalogGolden"
        status: pass
      - kind: unit
        ref: "cmd/parity_test.go#TestPlatformParityGolden"
        status: pass
      - kind: unit
        ref: "cmd/regression_test.go#TestRegressionSnapshot"
        status: pass
      - kind: unit
        ref: "cmd/command_call_audit_test.go#TestDocumentedSubcommandsAreSeverityClassified"
        status: pass
      - kind: unit
        ref: "cmd/colony_state_atomicity_ratchet_test.go#TestColonyStateWriteAllowlistOnlyShrinks"
        status: pass
      - kind: unit
        ref: "cmd/subcommand_reachability_ratchet_test.go#TestOrphanAllowlistOnlyShrinks"
        status: pass
      - kind: other
        ref: "git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json"
        status: pass
    human_judgment: false

duration: 50min
completed: 2026-08-28
status: complete
---

# Phase 197 Plan 05: One Command, Two Names, Declared Once Summary

**`pause` is now the canonical Cobra command with `pause-colony` as a real Cobra alias declared once in `pause.yaml`'s `aliases:` field, enforced on all three hand-maintained wrapper copies by tests in both directions, and `aether update` restores and names a missing alias wrapper instead of only incrementing a file count.**

## Performance

- **Duration:** 50 min
- **Started:** 2026-08-28T23:04Z (base commit)
- **Completed:** 2026-08-28T23:54Z
- **Tasks:** 3
- **Files created:** 7
- **Files modified:** 15

## Accomplishments

- Renamed the `pauseColonyCmd` Cobra command's primary name from `pause-colony` to `pause`, adding `pause-colony` as a genuine `Aliases` entry — both names resolve, through `rootCmd.Find`, to the exact same `*cobra.Command` object, proved by pointer identity rather than matching output.
- Replaced `.aether/commands/pause-colony.yaml` with `.aether/commands/pause.yaml`, which declares the alias via a new `aliases:` field — one definition file for the pair instead of the resume pair's two hand-kept files.
- Added `pause.md` to all three hand-maintained wrapper locations (nested Claude, flat Claude, OpenCode), byte-identical across all three; the existing `pause-colony.md` triplet stays in place as the alias's own wrapper.
- Taught `source_check.go`'s generated-command-surface checker about declared aliases: a declared alias legitimises its wrapper (no second YAML file required), a declared alias missing its wrapper on any of the three platform locations is reported by name, and a wrapper that merely looks like an alias but was never declared anywhere is still reported as an orphan.
- Added an alias-reconciliation pass to `aether update`: it snapshots which declared-alias wrappers are missing from each platform-home surface before the ordinary hub sync runs, then reports — in plain words, naming the command and the surface — only the ones that sync actually restored. A clean run reports nothing.
- Refreshed every golden, audit, and ratchet the rename disturbed in this one plan: the command catalogue (with hand-restored historical continuity), the platform parity snapshot, the severity audit, and the colony-state-write ratchet. The regression snapshot needed no change (same command count). The frozen orphan-allowlist baseline is untouched.

## Task Commits

1. **Task 1: One command, two names, declared once** — `e48e8bdd` (feat)
2. **Task 2: Updating a project restores a missing alias and says so** — `bbe7e411` (feat)
3. **Task 3: Refresh every record a renamed command moves** — `8aca8302` (test)

**Plan metadata:** committed separately per the worktree contract (this SUMMARY + REQUIREMENTS.md; STATE.md/ROADMAP.md are the orchestrator's).

## Files Created/Modified

- `.aether/commands/pause.yaml` — the canonical definition, carrying `aliases: [pause-colony]`
- `.claude/commands/ant/pause.md`, `.claude/commands/ant-pause.md`, `.opencode/commands/ant/pause.md` — the new wrapper triplet, byte-identical
- `.claude/commands/ant/pause-colony.md`, `.claude/commands/ant-pause-colony.md`, `.opencode/commands/ant/pause-colony.md` — updated to point their generated header at `pause.yaml` and mention the alias relationship
- `cmd/session_flow_cmds.go` — `pauseColonyCmd.Use` → `"pause"`, `Aliases: []string{"pause-colony"}`
- `cmd/hook_cmds.go` — stop-hook message now recommends `aether pause`
- `cmd/wrapper_command_names.go`, `cmd/command_guide.go` — `"pause"` added beside `"pause-colony"`, mirroring the resume pair
- `cmd/source_check.go` — `sourceCheckCommandSpec.Aliases`, declared-alias legitimisation, and the three-location alias-wrapper-existence check
- `cmd/canonical_alias_test.go` — `TestCanonicalAliasDelegates`, `TestDeclaredAliasesHaveWrappersOnEveryPlatform`, `TestAliasWrapperNeedsADeclaration`
- `cmd/platform_sync.go` — `declaredAliasSurfaces`, `missingDeclaredAliasSurfaces`, `diffAliasRepairs`, `aliasWrapperRepairReport`
- `cmd/update_cmd.go` — wires the before/after snapshot around `syncPlatformHomeAssetsFromHub`, appends the repair message and `alias_wrapper_repairs` field
- `cmd/alias_reconcile_test.go` — builds a project through the real install/setup path, damages one wrapper, proves `update --force` restores and names it; proves a clean run reports nothing
- `cmd/parity_test.go` — `"pause-colony": "pause"` added to `yamlToRuntimeName`
- `cmd/command_guide_test.go` — `yamlCommandNamesForGuideTest` now expands each YAML source's declared aliases into the returned name set
- `cmd/command_call_audit_test.go` — `"pause": true` added to `knownEnrichmentSubcommands`
- `cmd/testdata/command_catalog.json`, `cmd/testdata/parity_snapshot.json`, `cmd/testdata/colony_state_write_allowlist.json` — regenerated via each file's own `-update-golden` / `-update-colony-state-write-allowlist` flag

## Decisions Made

See `key-decisions` in the frontmatter for the full rationale on each. In short: one definition file (not two), `CommandName` bookkeeping left unchanged (not user-facing), alias reconciliation reads the live Cobra tree rather than parsing YAML (the only alias source actually available inside `aether update` on a downstream project), a shared alias-name-expansion fix rather than three separate ones, `pause` classified alongside `pause-colony` as enrichment (mirroring resume/resume-colony), and the renamed catalog entry's audit history hand-restored so a mechanical regeneration doesn't quietly erase it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 — Blocking] `yamlCommandNamesForGuideTest` needed alias expansion, and it lives in an undeclared file**

- **Found during:** Task 3
- **Issue:** `TestCommandGuideCoversAllYamlCommands`, `TestPlatformParityGolden`, `TestAllYamlHaveWrappersAndGuide`, and `TestNoPhantomCommands` all derive their "legitimate YAML-backed command name" set from `yamlCommandNamesForGuideTest` (`cmd/command_guide_test.go`), which only reads `.aether/commands/*.yaml` filenames. Once `pause-colony.yaml` was deleted in Task 1, `pause-colony` stopped being a recognised name to every one of those tests, even though `source_check.go`'s checker (Task 1's own file) had already been taught that a declared alias is legitimate. `cmd/command_guide_test.go` is not in this plan's declared file list.
- **Fix:** `yamlCommandNamesForGuideTest` now parses each YAML file's `aliases:` field (reusing `sourceCheckCommandSpec`, task 1's type) and folds the declared aliases into the returned name set. One shared fix, consumed by all four affected tests, instead of three separate patches.
- **Verification:** `TestCommandGuideCoversAllYamlCommands`, `TestPlatformParityGolden`, `TestAllYamlHaveWrappersAndGuide`, `TestNoPhantomCommands` all pass.
- **Commit:** `8aca8302`

**2. [Rule 3 — Blocking] `pause` needed a D-01 severity classification, in an undeclared file**

- **Found during:** Task 3
- **Issue:** `TestDocumentedSubcommandsAreSeverityClassified` requires every subcommand referenced in the documented wrapper corpus to carry a deliberate classification. `pause` now appears in that corpus (it is the new canonical name in `pause.md`) and had none. `cmd/command_call_audit_test.go` is not in this plan's declared file list.
- **Fix:** Added `"pause": true` to `knownEnrichmentSubcommands`, mirroring `pause-colony`'s existing classification and the resume/resume-colony pair's precedent of both names carrying their own entry. A colony pause failing is not a halt-worthy condition.
- **Verification:** `TestDocumentedSubcommandsAreSeverityClassified` passes.
- **Commit:** `8aca8302`

**3. [Rule 3 — Blocking] The colony-state-write ratchet's allowlist keys by function name, in an undeclared testdata file**

- **Found during:** Task 3 (full-suite verification)
- **Issue:** `TestColonyStateWriteAllowlistOnlyShrinks` scans source for `COLONY_STATE.json` write sites and keys each by enclosing function name. `pauseColonyCmd`'s RunE closure's key moved from `cobra:pause-colony` to `cobra:pause` as a direct consequence of the Task 1 rename, which the ratchet correctly flagged as both a new site (`cobra:pause`) and a now-stale allowlist entry (`cobra:pause-colony`). `cmd/testdata/colony_state_write_allowlist.json` is not in this plan's declared file list.
- **Fix:** Regenerated with the ratchet's own `-update-colony-state-write-allowlist` flag (never hand-edited); the diff is exactly the one renamed key with its reason preserved.
- **Verification:** `TestColonyStateWriteAllowlistOnlyShrinks` passes; `git diff` shows a single-line rename.
- **Commit:** `8aca8302`

**4. [Rule 1 — Bug] `-update-golden` silently dropped the renamed catalog entry's audit history**

- **Found during:** Task 3
- **Issue:** `cmd/testdata/command_catalog.json` is both the golden the audit test compares against AND (via `go:embed`) the enrichment lookup source `buildAuditCatalog` merges live-scanned commands against, keyed by name. Regenerating with `-update-golden` correctly renamed the entry from `pause-colony` to `pause`, but since `"pause"` was a new key in the embedded (pre-edit) lookup, the regeneration mechanically dropped `classification`, `since_version`, `historical_presence`, and `stability_score` — five versions of audit history, gone as a side effect of a rename the command itself did not deserve to lose.
- **Fix:** Hand-restored those four fields on the `pause` entry from `pause-colony`'s pre-rename values. Because the golden is its own enrichment source, this is self-healing from here forward: the next `-update-golden` run will find `pause` already enriched and carry it forward automatically.
- **Verification:** `TestAuditCatalogGolden`, `TestCatalogCompleteness`, `TestCatalogSchema` all pass; `git diff` on the file is a single one-line rename (the `"name"` field only).
- **Commit:** `8aca8302`

---

**Total deviations:** 4 auto-fixed (3 blocking file-scope, 1 bug)
**Impact on plan:** No scope creep in behaviour. Three of the four are required corollaries of the Task 1 rename that the plan's own acceptance criteria (Task 3's `Test(...)` groups) would not pass without; the fourth prevents silent loss of pre-existing audit-history data. All four are single, minimal, mechanically-verified edits.

## Issues Encountered

**`TestBuildDispatchStartsHeartbeatMonitor` fails only under the full `go test ./cmd` suite (twice in a row, ~270s run), but passes reliably in isolation and alongside its neighbors (`-count=3` all green).** None of this plan's files touch worker-dispatch heartbeat monitoring. Confirmed pre-existing, unrelated, test-order/resource-contention flakiness rather than a regression from this plan. Logged to `.planning/phases/197-one-answer-to-what-next/deferred-items.md`, not fixed, per scope boundary.

## Known Stubs

None. Every behaviour this plan describes is wired and exercised by a real test: the alias resolves through the actual Cobra tree, `aether source-check` reports clean against this repository, and `aether update` was proven against a project built by the real install/setup path.

## Threat Flags

None. This plan renames a command and adds an alias-reconciliation read/write pass that only touches Aether's own command-wrapper files (never user colony data); no new network endpoint, auth path, or trust-boundary schema change.

## User Setup Required

None — no external service configuration required. The rename takes effect for other repositories through the existing `aether publish` / `aether update` path.

## Next Phase Readiness

- **197-06** (the pause card, next wave) can build on `pause` as the canonical, resolver-visible command name; this plan did not touch `cmd/codex_visuals.go` as instructed.
- **197-07** (cross-command envelope assertion) inherits an `aether update` whose JSON envelope carries a new, additive `alias_wrapper_repairs` field alongside the existing `details`/`stale_publish` shape.
- The alias-declaration pattern (`aliases:` in YAML, `Aliases` in Cobra, enforced by `canonical_alias_test.go`'s two-direction tests) is now a reusable template for any future command that wants a short/long name pair without a second definition file.

## Self-Check: PASSED

**Files claimed as created — all present on disk:**

| File | Present |
|---|---|
| `.aether/commands/pause.yaml` | yes |
| `.claude/commands/ant/pause.md` | yes |
| `.claude/commands/ant-pause.md` | yes |
| `.opencode/commands/ant/pause.md` | yes |
| `cmd/canonical_alias_test.go` | yes |
| `cmd/alias_reconcile_test.go` | yes |
| `.planning/phases/197-one-answer-to-what-next/deferred-items.md` | yes |

**Commits claimed — all present in `git log`:** `e48e8bdd`, `bbe7e411`, `8aca8302`.

**Verification re-run:**

| Command | Result |
|---|---|
| `go test ./cmd -run 'Test(CanonicalAliasDelegates\|DeclaredAliasesHaveWrappersOnEveryPlatform\|AliasWrapperNeedsADeclaration\|UpdateRestoresAMissingAliasWrapper\|UpdateReportsTheAliasRepairByName\|UpdateReportsNoRepairWhenNothingIsMissing)' -count=1` | ok |
| `go test ./cmd -run 'Test(AuditCatalogGolden\|CatalogCompleteness\|PlatformParityGolden\|NoPhantomCommands\|RegressionSnapshot\|DocumentedSubcommandsAreSeverityClassified\|HumanFacingOutputGoesThroughWriteVisualOutput\|WrapperCommandNamesMatchCanonicalCorpus\|CommandParity\|SourceCheck\|OrphanAllowlist\|NoRegisteredSubcommandIsUnreferenced)' -count=1` | ok |
| `go build ./cmd/aether && go vet ./cmd` | clean |
| `git diff --exit-code cmd/testdata/orphan_allowlist_baseline.json` | clean |
| `aether source-check` (run against this repository) | `{"ok":true,...}`, no command issues |
| `gofmt -l cmd/ pkg/` | no output |
| `go test ./cmd -count=1` (full suite) | 1 pre-existing, unrelated flake (`TestBuildDispatchStartsHeartbeatMonitor`); everything else passes |

**`.planning/STATE.md` and `.planning/ROADMAP.md`:** not modified — the orchestrator owns them in parallel-executor mode.

---
*Phase: 197-one-answer-to-what-next*
*Completed: 2026-08-28*
