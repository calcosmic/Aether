# Priority Spec v3 — Governing Backlog (owner-ratified order)

**Ruling (D1, 2026-08-21):** the spec's own dependency-aware order governs
("The spec should have an order of importance" — and the spec's stated purpose
is exactly that). This file is the working projection of that order: the
spec's slices/milestones, repaired per the Codex audit
(`.aether/dreams/AETHER_PRIORITY_SPEC_V3_ANALYSIS_2026-08-21.md`), merged with
the in-flight programme, with live status. Owner rulings D1–D10:
`.planning/decisions/2026-08-21-owner-rulings-priority-spec-v3.md`.

**Audit repairs applied to the spec's order:**
1. The milestone list (M0–M12) omitted the spec's own two P0 families —
   execution accounting (§2) and canonical commands (§3). Slice A covers
   them; they are inserted explicitly as stage 1 below.
2. Seal (M3) reconciles SPEC requirements but the SPEC layer was scheduled at
   M5 — minimal SPEC identity (stable requirement/criterion IDs) moves ahead
   of stage 3.
3. Model routing was gated on an owner reversal — granted (D2): automatic
   routing approved, after the cost line, reasons always displayed,
   per-build override.

## Order of work

### Stage 0 — Baseline / anti-regression (spec M0) — LARGELY DONE
- Golden-path fixtures, measurement libraries, acceptance scripts,
  harness: ✅ done (186-01..06).
- ⏳ 186-07: ONE live benchmark run, operator at keyboard — **needs the
  owner present**; schedule any time, blocks nothing.
- Phase 192 (full 3-repeat showdown) is the finish-line rerun of this
  baseline — deliberately AFTER stages 1–2 so it measures the improved
  system.

### Stage 1 — Slice A: execution accounting + command contract (spec §2, §3, §22A)
The proof chain: stable identity task→dispatch→attempt→result→evidence;
lossless `covers_task_ids` aggregation; proof-obligation preflight (never
demand evidence the dispatch topology cannot produce); `completed_no_change`
/ `verified_existing` as honest terminal outcomes (ruling D6); append-only
attempts + explicit supersession (never overwrite valid evidence); task-scoped
Builder–Probe unlock; recovery recommendations ranked reconcile-before-rerun;
one canonical command registry with generated aliases (ruling D5).
- Already landed: marginal-value spawn gate + per-worker justification
  display + owner check-in (2026-08-21, v1.0.62); swarm/seal/colonize spawn
  floors; honest `completed_no_change`/`verified_existing` outcomes with
  mandatory evidence + resumable `interrupted` (2026-08-21 evening, commit
  969a2bda — UNPUBLISHED pending fresh-eyes review of 043c2745..HEAD).
- Codex grades this the largest correctness gap (C−/D). Extend the existing
  attempt journals/manifest — no parallel stores (ruling D3).

### Stage 2 — Slice B: recovery + quota guardrails + model economy (spec §4, §5, §22B)
- Honest cost line (in-flight Phase 185; prior art on the three
  `worktree-agent-*` 174 branches): every build ends with one plain
  sentence of tokens/cost.
- Budget controller: context-pressure vs allowance-pressure as separate
  states; pre-dispatch budget gate; bounded Builder slices with durable
  checkpoints; rate-limit interruption = resumable, never a code failure
  (ruling D7); large tool-output externalization.
- Automatic model routing (ruling D2): Sonnet routine Builders, no
  accidental `inherit`, recorded Opus escalation reasons, reviewer
  count/model independent, host-override detection, per-build override,
  reasons on every dispatch card.
- Recovery hardening: write-ahead intent, checkpoint boundaries, the two
  worktree fan-in defects from the audit (merged-flag not stamped; fan-in
  can check out the wrong branch).

### Stage 3 — Slice C: completion integrity, seal, lineage (spec §6, §22C)
Whole-colony seal reconciliation (PASS / PASS_WITH_BACKLOG / BLOCKED /
FORCE_SEALED); flag dispositions with provenance — no blocker silently
expires; immutable hashed seal packets in Chambers (ruling D10). Preceded by
minimal SPEC identity (audit repair 2).

### Stage 4 — Slice D: self-guiding UX (spec §7, §8, §22D)
One next-action resolver every surface renders (never hard-coded command
strings); context-health / safe-clear guidance; full pheromone lifecycle
(edit/strengthen/reactivate/convert/review-stale + "what this signal
changed"); typed user preferences with project overrides.
- Already landed: team check-in, mid-build owner decisions, CLARIFIED INTENT
  relay (2026-08-21).

### Stage 5 — SPEC-first initialization + planning dimensions (spec §9, §10; M5–M6)
Adaptive `/ant-init`, `.aether/SPEC.md` lifecycle (`/ant-spec`), revision
impact on plans; independent planning dimensions (engine / confidence /
decomposition / review / economy); Queen evidence-based risk scoring with
inspectable proposals (ruling D3: propose; Go validates/persists).

### Stage 6 — Context-efficient execution (spec §11; M7)
Phase DAG → coherent workstreams (never one-agent-per-task); Context
Compiler with bounded inputs; fresh Sonnet handoffs; worktree fan-in with
durable integration identity.

### Stage 7 — Capability router + Claude-native reliability (spec §12, §13; M8–M9)
Progressive skill disclosure, trust tiers, conflict order; slim agent
definitions only after native enforcement exists; lifecycle hooks;
integration doctor.

### Stage 8 — Oracle planning, Portal, parity, learning (spec §14–§17; M10–M12)
Oracle hypothesis ledger and 10/25/50 budgets; read-only Portal over
canonical state; structured ceremony events with cross-platform semantic
parity; effectiveness learning and calibration — instrumented on the stable
identities from stage 1.

## Standing constraints (spec §21 + rulings)
No second state stores. No Portal-first. No all-reviewers-on-Opus. No
monolithic unbounded Builder runs. Safety outranks SPEC, preferences,
skills and learned policy (ruling D4). Legacy colonies run in compatibility
mode with explicit draft-SPEC proposals (ruling D9).
