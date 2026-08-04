# 162-05 Layer 2: Recorded Before/After Memory Injection Exhibit

**Date:** 2026-08-04
**Requirement:** LEARN-05 — the learning loop must reach a worker, "measured, not
assumed"
**Status:** One-time recorded exhibit (D-08 Layer 2). Not a repeatable harness —
that is Phase 171's territory. See `cmd/memory_injection_test.go` for the
permanent, deterministic Layer 1 CI proof.

## Why a scratch colony, not this repo's own `.aether/data`

This development repo does not run a self-hosted Aether colony — self-hosting
artifacts were retired in v1.11 (see `CLAUDE.md`'s "Known losses from
shell-to-Go migration" section), and `.aether/data/` is gitignored and, at the
time of this exercise, contained no `COLONY_STATE.json` and no
`instincts.json` in this worktree. There was nothing live here to move aside.

Rather than fabricate a colony inside this dev repo just for one exhibit, the
exercise used the actual compiled `aether` binary — built from this worktree's
own `cmd/` source, so it is the real code this phase changed — against an
isolated scratch colony directory, addressed entirely through the same
environment-variable overrides the runtime already supports
(`COLONY_DATA_DIR`, `AETHER_HUB_DIR`). Every command below is the real,
unmodified CLI; only the working directory is a scratch path instead of this
repo's root. `git status` in this repo was confirmed clean both before and
after (below) — the exercise touched nothing tracked or gitignored in this
repo.

## Exact commands run

```bash
# Build the real binary from this worktree's source
go build -o /tmp/aether-memory-proof/aether ./cmd/aether

# Point the runtime at an isolated scratch colony + hub
export COLONY_DATA_DIR=/tmp/aether-memory-proof/scratch/.aether/data
export AETHER_HUB_DIR=/tmp/aether-memory-proof/scratch/hub
cd /tmp/aether-memory-proof/scratch

# Real colony setup (no fixture files written directly -- everything below
# went through the actual aether CLI, the sanctioned way to write colony state)
aether init "Prove the LEARN-05 memory injection pipeline reaches a real worker brief"
aether plan --synthetic   # local synthesis, no model call, produces a real 4-phase plan
aether instinct-create --trigger "Building a Go CLI subcommand" \
  --action "Check existing flag conventions in codex_workflow_cmds.go before adding new flags" \
  --confidence 0.85 --domain go
aether queen-write-learnings '[{"claim":"When wiring a new CLI subcommand, mirror the flag list of the closest sibling command before inventing new flag names"}]'
aether hive-init
aether hive-store --text "Cobra subcommands that mutate colony state should ship an inspection path like 'aether build --print-brief' (cmd/build_print_brief.go) so the assembled input is reviewable before it is spent" \
  --domain go --source-repo "aether-scratch-proof"

# BEFORE: memory populated
aether build 1 --print-brief --worker Augur-87           # checklist
aether build 1 --print-brief --full --worker Augur-87    # raw prompt

# Move colony memory aside (rename, not delete -- T-162-19)
mv .aether/data/instincts.json .aether/data/instincts.json.bak
mv .aether/QUEEN.md .aether/QUEEN.md.bak

# AFTER: memory wiped
aether build 1 --print-brief --worker Augur-87           # checklist
aether build 1 --print-brief --full --worker Augur-87    # raw prompt

# Restore
mv .aether/data/instincts.json.bak .aether/data/instincts.json
mv .aether/QUEEN.md.bak .aether/QUEEN.md

# Re-verified restoration produced the identical populated output again
aether build 1 --print-brief --worker Augur-87
```

## Output 1: populated (checklist)

```
🔮  Augur-87  (oracle)
────────────────────────────────────────────────────────────────────────
  CONTEXT CHECKLIST
────────────────────────────────────────────────────────────────────────
  Assignment & Task Content          present      258
  Territory Survey                   ABSENT         0
  Phase Research                     present      265
  Codegraph Context                  ABSENT         0
  Pheromone Signals                  ABSENT         0
  Previous Worker Handoffs           ABSENT         0
  Expected Output                    present      101
  Context Capsule (manifest-level)   present      958
  Charter (inside capsule)           ABSENT         0
────────────────────────────────────────────────────────────────────────
  TOTAL (capsule+brief+skills)         3416 / 23700   14.4%
```

## Output 2: populated (`--full`, Context Capsule section verbatim)

```
── Context Capsule (manifest-level) ──

## Colony State

Goal: Prove the LEARN-05 memory injection pipeline reaches a real worker brief
State: READY
Phase: 1
Phase Name: Discovery and boundaries
Tasks:
  - [pending] Read the current implementation paths relevant to the goal
  - [pending] Capture risks, constraints, and a testable target state
Parallel Mode: in-repo
## Active Instincts

- [Building a Go CLI subcommand] Check existing flag conventions in codex_workflow_cmds.go before adding new flags (confidence: 0.85)
## LOCAL QUEEN WISDOM (Repo-Specific)

- When wiring a new CLI subcommand, mirror the flag list of the closest sibling command before inventing new flag names (phase learning, 2026-08-04)
## HIVE WISDOM (Cross-Colony Patterns)

- Cobra subcommands that mutate colony state should ship an inspection path like 'aether build --print-brief' (cmd/build_print_brief.go) so the assembled input is reviewable before it is spent
## Review Depth

Heavy review -- full quality gauntlet
```

## Output 3: wiped (checklist)

```
🔮  Augur-87  (oracle)
────────────────────────────────────────────────────────────────────────
  CONTEXT CHECKLIST
────────────────────────────────────────────────────────────────────────
  Assignment & Task Content          present      258
  Territory Survey                   ABSENT         0
  Phase Research                     present      265
  Codegraph Context                  ABSENT         0
  Pheromone Signals                  ABSENT         0
  Previous Worker Handoffs           ABSENT         0
  Expected Output                    present      101
  Context Capsule (manifest-level)   present      615
  Charter (inside capsule)           ABSENT         0
────────────────────────────────────────────────────────────────────────
  TOTAL (capsule+brief+skills)         3073 / 23700   13.0%
```

## Output 4: wiped (`--full`, Context Capsule section verbatim)

```
── Context Capsule (manifest-level) ──

## Colony State

Goal: Prove the LEARN-05 memory injection pipeline reaches a real worker brief
State: READY
Phase: 1
Phase Name: Discovery and boundaries
Tasks:
  - [pending] Read the current implementation paths relevant to the goal
  - [pending] Capture risks, constraints, and a testable target state
Parallel Mode: in-repo
## HIVE WISDOM (Cross-Colony Patterns)

- Cobra subcommands that mutate colony state should ship an inspection path like 'aether build --print-brief' (cmd/build_print_brief.go) so the assembled input is reviewable before it is spent
## Review Depth

Heavy review -- full quality gauntlet
```

## Diff summary

| Section | Populated | Wiped | Source moved aside |
|---|---|---|---|
| `## Active Instincts` | present, 1 entry (confidence 0.85) | **absent** | `.aether/data/instincts.json` |
| `## LOCAL QUEEN WISDOM (Repo-Specific)` | present, 1 entry | **absent** | `.aether/QUEEN.md` |
| `## HIVE WISDOM (Cross-Colony Patterns)` | present, 1 entry | present (unchanged — this exhibit only moved the two files Task 2 names; hive memory was intentionally left in place) | not moved |
| Context Capsule size | 958 chars | 615 chars | — |
| Total assembled prompt (checklist total) | 3416 chars | 3073 chars | — |

Moving `instincts.json` and `QUEEN.md` aside removed exactly the two sections
backed by those files — `## Active Instincts` and `## LOCAL QUEEN WISDOM` —
from the same worker's assembled prompt, and nothing else changed: task
content, phase research, and hive wisdom (whose backing file was untouched)
are byte-identical between the two runs. Restoring the files reproduced the
original 958-char capsule exactly (re-verified above).

**What a worker would therefore know in one case and not the other:** in the
populated run, Augur-87's assembled brief tells the worker to check existing
CLI flag conventions before adding new ones and to mirror the closest sibling
command's flags (the seeded instinct and local wisdom); in the wiped run,
neither instruction reaches the worker at all — the brief is otherwise
identical.

## Restoration confirmed

```
$ mv .aether/data/instincts.json.bak .aether/data/instincts.json
$ mv .aether/QUEEN.md.bak .aether/QUEEN.md
$ aether build 1 --print-brief --worker Augur-87 | grep "Context Capsule"
  Context Capsule (manifest-level)   present      958
```

958 chars matches the original populated run exactly.

`git status --short` in this repo (`/Users/callumcowie/repos/Aether`, this
worktree) was clean before this exercise began and remained clean afterward —
the scratch colony lived entirely under `/tmp/aether-memory-proof/`, never
touching this repo's tracked files or its own (untouched, pre-existing)
`.aether/QUEEN.md`.

## What this exhibit does not claim

This is a one-time recording, not a repeatable `learning-proof` harness — that
tooling is explicitly out of scope for this plan and is Phase 171's territory
per D-08. For the permanent, CI-enforced version of this same proof, see
`TestWorkerBriefMemoryInjection` and `TestResolveCodexWorkerContextCarriesMemory`
in `cmd/memory_injection_test.go`, which assert the identical mechanism
(populated vs. wiped `instincts.json` / `.aether/QUEEN.md` / hub
`hive/wisdom.json`) deterministically, with no model calls, on every CI run.
