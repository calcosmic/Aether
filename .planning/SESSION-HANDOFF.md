# Session Handoff — 26 July 2026 (evening)

> Supersedes the 25/26 July handoff. Read this first; verify claims by
> execution, not grep — that rule caught real defects twice this session.

## Where things stand: v1.25 "Close the Loops" is COMPLETE and published

All five phases of the approved plan
(`~/.claude/plans/optimized-noodling-prism.md`) shipped, were reviewed by four
execution-based agents, and every review finding was fixed. Binary and hub
agree at **1.0.42**. Working tree clean at every publish.

The commits, in order:

| Commit | What |
|---|---|
| `281dd34a` | Phase 1 — build workers receive the real brief (pheromones/handoffs/survey); continue stops blocking on nonexistent commands; help.md is a front door |
| `c5c90df9` | Phase 2 — security gate executes, dry-runs are pure, re-init confirms + backs up, swarm-cleanup honest, learning fires on default continue, typed phase modes (prose can never steer dispatch) |
| `a2c8288e` | Phase 3 — Scout-per-phase research at plan time (wave 1, before Route-Setter), finalize preserves worker research, briefs carry Phase Research |
| `99226d4c` | publish bug — `go run` publish installed the fresh binary into a temp dir while reporting success. Both earlier publishes had shipped stale binaries. Fixed + regression test |
| `c7030eab` | Phase 4 — Research Brief gate in oracle.md, `aether oracle promote` (through admissibility gate), revision evidence CONTENTS inlined into planning briefs, oracle-phase-directives distributed to hub |
| `713090fc` | Phase 5 — honest "Aether-managed" headers repo-wide, dead code deleted (survey-load, playbook residue), hive cleared, stale REDIRECTs purged, next-step guidance, publish preflight documented |
| `caa636bb` | Multi-agent review fixes (see below) |
| `758dbd85` | Final hive-pollution vector killed |

## The multi-agent review (user-requested) and what it caught

Four agents verified by EXECUTION: plan-completeness audit, downstream
update-flow test in scratch repos, runtime-truth verification in a fixture
colony, wrapper/docs consistency. Verdict: 30/33 items genuinely done. The
defects they caught, all fixed and test-locked:

1. **check-antipattern half-fixed** (`cmd/security_cmds.go`): positional form
   emitted a spurious `--file is required` error + exit 1 on every call
   (mustGetString ran before the positional fallback), and the secret regex
   missed `aws_secret_access_key = "…"` (keyword had to sit immediately before
   `=`). The drift test had passed because it never asserted exit code/stderr —
   now it pins all three surfaces (`cmd/security_gate_drift_test.go`).
2. **Test suite polluted the real hive**: cmd tests wrote fixtures into the
   user's actual `~/.aether/hive/wisdom.json` on every full run. Two vectors:
   no suite-wide hub isolation, and tests that `Unsetenv("AETHER_HUB_DIR")` on
   cleanup, erasing isolation for every test after them. Fixes: TestMain in
   `cmd/testing_main_test.go` sets a temp AETHER_HUB_DIR (minimally valid hub
   mirroring the source version); ~60 tests that manage their own hub via
   `--home-dir` opt out with `t.Setenv("AETHER_HUB_DIR", "")`; `runSealCmd`
   pins a per-test hub unconditionally. Proven: hive stays 0 across full+race.
3. **Every hive write now passes the admissibility gate** at the
   `storeHiveWisdomEntry` chokepoint (`cmd/hive.go`) — the seal-promotion path
   had bypassed it ("Always write tests first" got in with no anchor).
4. **First `aether update` in a fresh repo corrupted JSON output**: npm ci ran
   with stdout ahead of the envelope. npm now goes to stderr with
   `--no-audit --no-fund` (`cmd/update_cmd.go` ensureTsHostBuilt).
5. **`.aether/.gitignore` now covers `ts-host/node_modules/`** (60 MB), with
   idempotent append for already-scaffolded repos (`cmd/platform_sync.go`).
6. **`build --plan-only` re-entry**: an idle plan-only attempt (awaiting
   external, no workers recorded) is superseded automatically — an aborted
   /ant-build no longer jams the next one. Dispatching attempts still require
   `--force`. Help text stops claiming plan-only mutates nothing.
7. **Playbook residue fully gone**: `manifest.playbooks` field,
   `codexBuildPlaybooks()`, TS `playbook-loader`, false retention comment.
   CLAUDE.md's "Split Playbooks" section rewritten to current reality.
8. Smaller: recovery-menu JSON errors exit 1; re-init guard keys on COMPLETED
   state too; continue.yaml `--skip-watchers` contradiction resolved; 3 flat
   `ant-reference-*` mirrors added; keystone composition test
   (`cmd/build_manifest_brief_composition_test.go`) pins plan-only briefs
   non-empty + carrying pheromones.

## What's left

**The acceptance run — user-driven.** Full lifecycle
`/ant-init → /ant-plan → /ant-build → /ant-continue → /ant-oracle → /ant-seal`
in `M4L-AnalogWave-System` (or any repo) on a cheap model, scored against the
owner's charter at `.aether/oracle/brief.md`. Everything is published and
waiting. The "learnings non-zero after one default /ant-continue" live check
also lands during this run.

Deferred (plan's cut list, revisit after acceptance): YAML→md generator,
vital-signs hardcoded zeros, hive auto-injection, ceremony beyond the charter
list, full colony/ policy distribution.

## Operational knowledge that will bite if forgotten

- **Publish**: `go run ./cmd/aether publish` now correctly lands the binary in
  `~/.local/bin` (ephemeral-path fix). ALWAYS verify after publish:
  `aether version --check` AND that new strings are in the binary
  (`strings ~/.local/bin/aether | grep <new-symbol>`). Clean tree first —
  publish builds from the working tree with no guard.
- **`aether update` in other repos** syncs companion files + global
  `~/.claude/commands|agents`; it does NOT update the binary (by design,
  loudly documented now). First run in a repo installs ts-host node_modules
  (~60 MB, gitignored, npm noise on stderr only).
- **Test hygiene**: never `os.Unsetenv("AETHER_HUB_DIR")` in a test — use
  `t.Setenv`. Tests simulating a hub-less machine or their own `--home-dir`
  hub must `t.Setenv("AETHER_HUB_DIR", "")` explicitly.
- **Goldens**: `cmd/testdata/{command_catalog,parity_snapshot,regression}` need
  `-update-golden` after adding/removing commands or flags — three gate runs
  failed on stale goldens this session.
- **Test-order trap**: `renderedCommandExitCode` is a global atomic;
  `rootCmd.Execute()` alone doesn't reset it — store 0 first if asserting it.
- The `.aether/CONTEXT.md` in this repo was once clobbered by a test run
  (writer via git-toplevel fallback); watch `git status` on `.aether/` before
  committing. Restored via `git checkout` when seen.
- Hive backups if ever needed: `~/.aether-backup-20260725-2328` (67 old
  entries), `~/.aether-backup-20260726-1250`.

## Key architectural facts (stable)

- Typed phase modes: declared by planner, backfilled by `aether migrate-state`,
  `effectiveQueenPhaseMode` never infers from prose; Oracle in build flow only
  for discovery mode (`cmd/caste_relevance.go` isCasteSuppressed).
- Phase research: `cmd/phase_research.go` — wave-1 Scouts per drafted phase,
  fast preset skips, once per phase (template marker
  "no dedicated research worker ran for this phase" = re-researchable);
  finalize preserves worker files, prunes orphans; briefs carry
  `## Phase Research` (3.5k char budget).
- Oracle promote: `aether oracle promote [--dry-run] [--min-confidence 80]` —
  findings → learnings + instincts through admissibility; pheromones stay a
  user decision (wizard offers a FOCUS write with approval).
- Wrapper headers: `<!-- Aether-managed: runtime spec at
  .aether/commands/<name>.yaml. Synced by aether update. -->` — enforced by
  source_check; update/prune recognizes legacy "Generated from" headers too.
