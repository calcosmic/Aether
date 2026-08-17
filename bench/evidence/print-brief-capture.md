# Committed capture: what a worker is told

## What this is

This is the exact set of instructions one of Aether's helper agents (a
"worker" — an AI helper assigned one job, like writing code or checking
someone else's work) receives before it starts work on a real, in-progress
project ("colony"). It was printed by a command that only reads project
state and prints it back — it never runs any of the actual work, and it
never changes anything. This capture exists because later work in this
programme deliberately changes what workers are told; without a dated
snapshot from before that change, there is nothing to measure the change
against.

## Provenance

- **Colony repo captured:** `agency-agents` (a separate real project on this
  machine, not this repo — capturing from this repo would risk mixing this
  benchmark's own tooling into the evidence)
- **Colony goal:** "Rebrand all agents for the Aether ant colony using
  emojis and different names and ant metaphors"
- **Phase captured:** Phase 6, "Update Strategy Documentation" (in progress
  at capture time)
- **Total phases in the plan:** 7
- **Phases complete at capture time:** 5 (phases 1–5 all `completed`; phase 6
  `in_progress`; phase 7 not yet started) — this is the "mid-project"
  condition: more than one phase, and the current phase is not the first.
- **Worker captured:** `Guard-55`, a "keeper" (the knowledge-preservation
  caste — the worker whose job on this phase is to write down reusable
  patterns for later workers). Five workers were dispatched to this phase in
  total; this capture is scoped to one of them via the command's own
  `--worker` flag, so the committed file is exactly one worker's assembled
  prompt rather than a concatenation of several — closer to "the exact
  prompt an Aether worker receives," singular, as this evidence is meant to
  show.
- **Capture date (UTC):** 2026-08-17T20:13:49Z
- **Binary version:** 1.0.58
- **This repo's git SHA at capture time:** `261f2f5083b4b25fd890fb7a97739c277015bfa1`
- **Command run:** `aether build 6 --print-brief --full --worker Guard-55`
  (from inside the `agency-agents` repo, using a binary built fresh from
  this repo's source)

## Read-only proof

The capture script fingerprints the colony's saved state file before and
after running the command and refuses to write the capture if they differ
even by one byte.

- **Before checksum** (SHA-256 of `.aether/data/COLONY_STATE.json`):
  `51afff05b00bc3a5eb5bbbbabdadeb442914211470b8212eb9ca36ee26be72fe`
- **After checksum** (same file, same algorithm):
  `51afff05b00bc3a5eb5bbbbabdadeb442914211470b8212eb9ca36ee26be72fe`
- These two values are identical.
- The recursive listing of every other file under `.aether/data/` (name,
  modification time, size) was also identical before and after, with one
  documented exception: a `.cache_COLONY_STATE.json` file. That file is a
  read-side parse cache the runtime is allowed to write on any read
  (`pkg/cache/session_cache.go` — keyed by the source file's own
  modification time, purely to avoid re-parsing JSON that has not changed).
  It never changes what the colony believes, so it is excluded from this
  proof the same way a browser's disk cache would be excluded from a proof
  that visiting a page didn't edit it.

## Composition summary

The command's `--full` output prints one worker's assembled prompt, then
breaks it into named parts and shows what share of the whole each part
occupies. The table below is this capture's own composition table, taken
directly from the committed raw output — these are the reference figures a
later phase will compare against.

| Section | Characters | Share of total |
|---|---|---|
| Skill Section | 6,244 | 68.3% |
| Context Capsule (manifest-level) | 2,247 | 24.6% |
| Phase Success Criteria | 198 | 2.2% |
| (preamble) | 188 | 2.1% |
| Phase Objective | 112 | 1.2% |
| Expected Output | 93 | 1.0% |
| Assignment | 66 | 0.7% |
| **Grand total** | **9,147** | **100.0%** |

(The seven rows above sum to 9,148 — one character off the command's own
stated 9,147 total, a rounding artifact of the underlying character count;
the command's own total, not a re-derived one, is what is recorded here.)

## How to retake

```
bench/evidence/capture-print-brief.sh <path-to-a-real-mid-project-colony> <phase-number> [worker-name]
```

The worker name is optional; omitting it captures every worker planned for
the phase in one file (each with its own composition table), the same way
the underlying command behaves by default. This baseline was taken with a
worker name (`Guard-55`) so the committed file is one clean, single-worker
prompt.

This overwrites `bench/evidence/print-brief-capture.txt` only after the
read-only proof and the secret scan both pass.

To check whether the assembled prompt has changed since this baseline
without overwriting it, run the same script with `--check` prepended:

```
bench/evidence/capture-print-brief.sh --check <path-to-the-same-colony> <same-phase-number> [worker-name]
```

`--check` exits 0 when the composition is unchanged (ignoring the one line
that always differs — the run's start timestamp) and exits non-zero with a
line-count and section-heading diff when it has changed. This is what later
phases in this programme will run to show, with evidence, exactly how much
the worker's prompt changed.
