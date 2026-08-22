# Decision: The Queen decides, the program checks (2026-08-22)

**Ruled by the owner, 2026-08-22.** Supersedes the "Watcher is always required on
build" rule (CLAUDE.md "Queen-Owned Orchestration", `TestWatcherIsAlwaysRequiredOnBuild`)
and the "Probe is required wherever the phase produces testable code" floor
(`TestProbeIsRequiredOnlyWhereItCanFindSomething` keeps its *negative* half —
no Probe where nothing is testable — but loses its positive half).

## The ruling

> "I don't even necessarily want a watcher every time… I thought the idea was
> that the queen orchestrates based on what's necessary. It's meant to use the
> model's intelligence to delegate roles."

1. **Every worker that costs tokens is the Queen's call**, with a one-line
   reason the operator can read. That includes sending *zero* reviewer workers
   on low-risk work. The model's judgement (the chat proposing `--castes` with
   `--caste-reason`) is the primary source of the team; the keyword engine is
   the fallback for autopilot and for proposals that never arrive.
2. **The floor is deterministic, not agentic.** The program's own free checks —
   build, types, lint, tests, "do the files the worker claims exist", "is
   there evidence for each success criterion" — run on every phase and cannot
   be skipped by any depth flag, review policy, or proposal. A phase with zero
   reviewer workers advances when those checks pass and blocks when they fail.
   No gate may synthesize a hidden checker-worker requirement.
3. **The program forces a reviewer only for a named high-risk signal** —
   credentials/auth, payments, data deletion, migrations, release sign-off —
   and says which signal, every time. This is the only case where the runtime
   adds a token-costing worker the Queen did not ask for.
4. **Each phase is verified once.** The build-side verification stage and the
   follow-up (`continue`) review are one pass, not two. The same caste is not
   dispatched at both boundaries for the same phase unless the Queen asks.

## Why

- The spec the owner ratified on 2026-08-21 already says it: *"QUICK means
  zero reviewer agents, not zero deterministic validation."* The code
  contradicted the spec.
- Measured on 2026-08-22 with today's Queen at standard settings (nothing
  spawned): a one-task bug fix was sent **8 workers** (build: builder, watcher,
  auditor, probe, tracker; continue: watcher, auditor, probe). v5.4.0 sent 3–4.
  The CSV-export phase judged "production" got 9; v5.4.0 sent 5–6. Most of the
  gap is the same files reviewed two or three times.
- The mandatory Watcher existed because this repo's history is full of work
  "declared done" that never worked. That fear is answered by the free checks
  and criteria evidence being unskippable — not by a ~100k-token reviewer
  agent on a copy tweak.

## What this does NOT change

- Deterministic verification remains mandatory everywhere (rule 2 is the
  stronger floor, not a weaker one).
- `queenPhaseHasSecuritySignal`-class forcing stays, now with a reason string
  attached and visible; what goes is "production mode ⇒ auditor" and
  "watcher always".
- Owner override in both directions (`--castes`, `--heavy`, `--light`, the
  check-in card) is unchanged.

## Locked by (to be written in milestone v1.27)

- `TestDeterministicChecksCannotBeSkipped`
- `TestGateAcceptsDeterministicEvidenceWithoutReviewer`
- `TestOneTaskBugFixIsOneWorkerPlusChecks`
- `TestNoWorkerWithoutStatedReason`
- `TestReviewerForcedOnlyByNamedRisk`
- `TestPhaseVerifiedOnce`
- Retirement of `TestWatcherIsAlwaysRequiredOnBuild` via the test-deletion
  ledger, citing this file.
