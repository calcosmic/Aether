# Stack Research

**Project:** Aether Integration Gap Fixes
**Researched:** 2026-03-14
**Confidence:** HIGH

---

## Context: What This Milestone Is

This is NOT a greenfield build and NOT a new dependency milestone. The four integration
gaps are wiring problems inside an existing bash system. Every required capability
already exists as an `aether-utils.sh` subcommand or in a command playbook. The work
is connecting them in the right order, in the right place, with correct argument shapes.

**The stack does not change.** No new tools, no new languages, no new libraries.

---

## Existing Stack (Confirmed Verified)

All of the below is already present and working in the codebase.

### Core Runtime

| Technology | Version | Purpose | Status |
|------------|---------|---------|--------|
| Bash | 3.2+ | Orchestration, all subcommand dispatch | In use everywhere |
| jq | 1.6+ | JSON read/write for all state files | Required by 150+ subcommands |
| awk | macOS BSD / GNU | CONTEXT.md parsing (Recent Decisions table) | Used in colony-prime, context-update |
| sed | macOS BSD / GNU | Inline file updates for CONTEXT.md | Used in context-update subcommand |

**Confidence: HIGH** — Verified by reading aether-utils.sh source. These are the only
runtime dependencies across the entire system.

### Key Subcommands (Already Implemented, Gaps Are in Callers)

The gaps are in the playbooks and agent definitions that should be calling these
subcommands but are not wired correctly. The subcommands themselves are correct.

| Subcommand | Signature | What It Does | Gap Connection |
|------------|-----------|--------------|----------------|
| `context-update decision` | `<description> [rationale] [who]` | Writes decision row to CONTEXT.md AND auto-emits `system:decision` FEEDBACK pheromone (strength 0.65, 30d TTL) | PHER-01 gap: the `decision` action already emits pheromones — callers just need to call it |
| `pheromone-write` | `<TYPE> <content> [--strength N] [--ttl TTL] [--source SOURCE] [--reason REASON]` | Writes signal to pheromones.json with deduplication | PHER-01: called in continue-advance Step 2.1b but needs correct AWK to read CONTEXT.md |
| `instinct-create` | `--trigger <str> --action <str> --confidence <0-1> --domain <str> --source <str> --evidence <str>` | Writes instinct to COLONY_STATE.json `.memory.instincts[]`, deduplicates by trigger+action match, boosts confidence +0.1 on match, evicts lowest at cap 30 | PHER-01/learnings gap: called in continue-advance Step 3 and 3a — check caller context is correct |
| `memory-capture` | `<event_type> <content> [wisdom_type] [source]` | Full pipeline: `learning-observe` → `pheromone-write` → `learning-promote-auto` → `activity-log` → `rolling-summary`. event_type must be one of: `learning\|failure\|redirect\|feedback\|success\|resolution` | Gap 4: should be called at decision and failure points during build |
| `midden-recent-failures` | `[limit]` | Returns `{"count": N, "failures": [{timestamp, category, source, message}]}` from `.aether/data/midden/midden.json` | PHER-02 gap: already called in continue-advance Step 2.1c, threshold logic needs validation |
| `midden-write` | `<category> <message> <source>` | Appends entry to `.aether/data/midden/midden.json` | Gap 3: write-side — should be called when failures occur during build, currently only partially wired |
| `learning-check-promotion` | `[observations_file]` | Returns `{"proposals": [...]}` of observations meeting threshold | MEM-01: already called in continue finalize, check it runs after memory-capture |
| `learning-promote-auto` | `<wisdom_type> <content> [colony_name] [event_type]` | Checks recurrence threshold, calls `queen-promote` if met, also calls `instinct-create` on promotion | Runs inside memory-capture automatically |
| `colony-prime` | `[--compact]` | Combines QUEEN.md wisdom + pheromone signals + instincts into `prompt_section` for worker injection | Already called in build-context.md Step 4 — but only picks up pheromones that already exist |

**Confidence: HIGH** — Verified by reading all subcommand implementations directly in aether-utils.sh.

---

## The Four Integration Gaps: What Exists vs What's Missing

### Gap 1: Decisions → Pheromones via CONTEXT.md (PHER-01)

**What exists:**
- `context-update decision` already auto-emits a `system:decision` FEEDBACK pheromone
  (source `"system:decision"`, strength 0.65, TTL 30d) — line 508 of aether-utils.sh
- continue-advance.md Step 2.1b already has AWK to parse CONTEXT.md "Recent Decisions"
  table and call `pheromone-write` with source `"auto:decision"` for up to 3 decisions,
  with deduplication check against both `auto:decision` and `system:decision` sources

**What is the actual gap:**
The deduplication in Step 2.1b checks `.content.text | contains($text)` — this relies on
the pheromone content text containing the decision text. The `context-update decision`
pheromone has content `"Decision: $decision — $rationale"`, while Step 2.1b emits
`"[decision] $dec"`. These two formats don't overlap, so deduplication may fail silently
and produce duplicate signals. The gap is format alignment in the deduplication check,
not missing infrastructure.

**Fix location:** `.aether/docs/command-playbooks/continue-advance.md` Step 2.1b
**Fix type:** Pure playbook markdown change — adjust dedup check OR normalize to same
source name so only one path emits

**No aether-utils.sh changes needed.**

---

### Gap 2: Learnings → Instincts via instinct-create

**What exists:**
- `instinct-create` is fully implemented (lines 7252-7369 aether-utils.sh)
- continue-advance.md Steps 3, 3a, 3b already call `instinct-create` with correct
  argument shapes for patterns from phase work, midden error patterns, and success patterns
- `memory-capture` calls `learning-promote-auto` which also calls `instinct-create` on
  promotion (line 5381)

**What is the actual gap:**
The instructions in Steps 3, 3a, 3b are written as guidance for the Queen LLM agent to
interpret and translate into bash calls ("Review the completed phase for repeating
patterns"). This is ambiguous — the LLM must decide what constitutes a pattern.
High-confidence instincts that SHOULD be created from explicit phase learnings stored in
`COLONY_STATE.json memory.phase_learnings` are not extracted programmatically. The
pipeline from `memory.phase_learnings[].learnings[].claim` → `instinct-create` requires
the LLM to bridge this gap interpretively, which produces inconsistent output.

**Fix location:** `.aether/docs/command-playbooks/continue-advance.md` Step 3
**Fix type:** Add explicit bash loop over `memory.phase_learnings` to feed claims directly
into `instinct-create`, replacing (or supplementing) the open-ended LLM interpretation.
No new subcommands needed — just a tighter bash block.

**No aether-utils.sh changes needed.**

---

### Gap 3: Midden → Behavior via Threshold Tuning

**What exists:**
- `midden-write` implemented and called in build-wave.md at worker failure (line 414)
- `midden-recent-failures` implemented and returns structured data
- continue-advance.md Step 2.1c reads midden and emits REDIRECT pheromones for categories
  with 3+ occurrences
- continue-advance.md Step 3a reads midden and calls `instinct-create` for patterns with
  2+ occurrences

**What is the actual gap:**
Two separate thresholds exist for the same midden data (3 for REDIRECT pheromones in
2.1c, 2 for instinct creation in 3a) — this is intentional but undocumented, causing
confusion about which threshold is correct. More critically, `midden-write` is called
in build-wave.md only for builder worker failures — it is NOT called for:
- Watcher failures
- Chaos failures
- Verification loop failures in continue-verify.md

So the midden underrepresents failures, and the threshold (3) may never be reached in
practice for non-builder failure categories.

**Fix location:**
- `.aether/docs/command-playbooks/build-wave.md` — add `midden-write` at Watcher and
  Chaos failure points
- `.aether/docs/command-playbooks/continue-verify.md` — add `midden-write` at
  verification failures (build fail, test fail, success criteria not met)

**No aether-utils.sh changes needed.** Threshold values are in the playbook markdown
as literals (2 and 3) — they can be changed as markdown edits.

---

### Gap 4: Memory-Capture at Decision and Failure Points

**What exists:**
- `memory-capture` is fully implemented with all six event types
- continue-advance.md Step 2.5 calls it for each learning extracted from the phase
- build-wave.md calls it for builder worker failures (line 414)
- `phase-insert` calls it for inserted phases (line 3462 of aether-utils.sh)

**What is the actual gap:**
`memory-capture` is NOT called at:
1. **Decision points** — when `context-update decision` is called (build or continue), no
   corresponding `memory-capture "feedback" "$decision" "pattern" "worker:continue"` fires
2. **Non-builder failures** — Watcher failures and Chaos failures in build-wave.md
   don't call `memory-capture "failure"` (only `midden-write` is called there for builder
   failures, and `memory-capture` is only at line 414 for builders)
3. **Verification failures** — continue-verify.md's gate decision (NOT READY path) does
   not call `memory-capture` at all

The result: the `learning-observations.json` accumulator never sees decision events or
non-builder failures. These never reach `learning-check-promotion` thresholds and never
auto-promote to QUEEN.md.

**Fix location:**
- `.aether/docs/command-playbooks/build-wave.md` — add `memory-capture "failure"` at
  Watcher and Chaos failure handling
- `.aether/docs/command-playbooks/continue-verify.md` — add `memory-capture "failure"`
  in the NOT READY gate path
- `.aether/docs/command-playbooks/continue-advance.md` or
  `.aether/docs/command-playbooks/continue-finalize.md` — add `memory-capture "feedback"`
  for each decision recorded via `context-update decision`

**No aether-utils.sh changes needed.**

---

## JSON Structures to Validate Before Calling Each Subcommand

These are the shapes the bash caller must conform to. Callers that produce wrong shapes
fail silently (the subcommands use `|| true` everywhere).

### pheromones.json (read by dedup checks)

```json
{
  "signals": [
    {
      "id": "sig_...",
      "type": "FOCUS|REDIRECT|FEEDBACK",
      "active": true,
      "source": "auto:decision|system:decision|auto:error|auto:success|worker:continue",
      "content": { "text": "..." },
      "strength": 0.6,
      "ttl": "30d",
      "created_at": "ISO-8601"
    }
  ]
}
```

**Dedup check pattern (correct):**
```bash
existing=$(jq -r --arg text "$dec" '
  [.signals[] | select(.active == true
    and (.source == "auto:decision" or .source == "system:decision")
    and (.content.text | contains($text)))] | length
' .aether/data/pheromones.json 2>/dev/null || echo "0")
```

**The bug:** `context-update decision` emits content `"Decision: $decision — $rationale"`.
The dedup check uses `contains($text)` where `$text` is the raw `$dec` (no prefix). This
WILL match because `contains` does substring match, not exact match. The dedup should work
— but only if `$dec` is the exact string that appears in the `context-update decision`
pheromone's content. Verify the AWK extraction in Step 2.1b produces the same text used
as input to `context-update decision`.

### COLONY_STATE.json memory.instincts[] (written by instinct-create)

```json
{
  "memory": {
    "instincts": [
      {
        "id": "instinct_<epoch>",
        "trigger": "when this situation arises",
        "action": "what to do",
        "confidence": 0.7,
        "status": "hypothesis",
        "domain": "testing|architecture|code-style|debugging|workflow",
        "source": "phase-{id}",
        "evidence": "specific observation",
        "applications": 0,
        "created_at": "ISO-8601",
        "last_applied": null
      }
    ]
  }
}
```

**Cap enforcement:** `instinct-create` automatically evicts the lowest-confidence instinct
when count exceeds 30. No caller needs to handle this.

### COLONY_STATE.json memory.phase_learnings[] (read to drive Gap 2 fix)

```json
{
  "memory": {
    "phase_learnings": [
      {
        "id": "learning_<epoch>",
        "phase": 3,
        "phase_name": "phase-name",
        "learnings": [
          {
            "claim": "specific actionable learning",
            "status": "hypothesis|validated|disproven",
            "tested": false,
            "evidence": "what observation led to this",
            "disproven_by": null
          }
        ],
        "timestamp": "ISO-8601"
      }
    ]
  }
}
```

**The correct bash loop for Gap 2 fix:**
```bash
current_phase_num=$(jq -r '.current_phase' .aether/data/COLONY_STATE.json)
jq -r --argjson phase "$current_phase_num" '
  .memory.phase_learnings[]?
  | select(.phase == $phase)
  | .learnings[]?
  | select(.status != "disproven")
  | .claim
' .aether/data/COLONY_STATE.json 2>/dev/null | while IFS= read -r claim; do
  [[ -z "$claim" ]] && continue
  bash .aether/aether-utils.sh instinct-create \
    --trigger "When working on patterns from phase $current_phase_num" \
    --action "$claim" \
    --confidence 0.7 \
    --domain "workflow" \
    --source "phase-$current_phase_num" \
    --evidence "$claim" 2>/dev/null || true
done
```

### midden.json (written by midden-write, read by midden-recent-failures)

```json
{
  "version": "1.0.0",
  "entry_count": 5,
  "entries": [
    {
      "id": "midden_<epoch>_<pid>",
      "timestamp": "ISO-8601",
      "category": "security|test_failure|build|coverage|edge_cases|integration|general",
      "source": "gatekeeper|watcher|chaos|builder|probe",
      "message": "...",
      "reviewed": false
    }
  ]
}
```

**midden-recent-failures return shape:**
```json
{
  "count": 5,
  "failures": [
    {"timestamp": "...", "category": "...", "source": "...", "message": "..."}
  ]
}
```

Note: `count` is the TOTAL entry count, not the count of returned entries. The limit
parameter controls how many are returned in `failures[]`.

### learning-observations.json (written by memory-capture via learning-observe)

```json
{
  "observations": [
    {
      "content_hash": "sha256:<hash>",
      "content": "the claim text",
      "wisdom_type": "pattern|failure|redirect",
      "observation_count": 2,
      "colonies": ["colony-name"],
      "first_seen": "ISO-8601",
      "last_seen": "ISO-8601"
    }
  ]
}
```

**Dedup mechanism:** `memory-capture` calls `learning-observe`, which hashes content and
increments `observation_count` if the hash already exists. This means calling
`memory-capture` with the same claim text twice (across phases) correctly increments
the count. The auto-promotion threshold for `pattern` type is 2 (propose) / 3 (auto).

---

## What Requires aether-utils.sh Changes vs Pure Playbook Changes

### Pure Playbook Changes (markdown edits only)

| Gap | File(s) to Edit | Change |
|-----|-----------------|--------|
| Gap 1 (PHER-01) | `continue-advance.md` Step 2.1b | Align dedup check: use same source name `"system:decision"` OR change `context-update decision` to not auto-emit so Step 2.1b is sole emitter |
| Gap 2 (learnings→instincts) | `continue-advance.md` Step 3 | Replace open-ended LLM instruction with explicit bash loop over `memory.phase_learnings` |
| Gap 3 (midden→behavior) | `build-wave.md`, `continue-verify.md` | Add `midden-write` calls at Watcher/Chaos failure points and verification NOT READY path |
| Gap 4 (memory-capture) | `build-wave.md`, `continue-verify.md`, `continue-advance.md` or `continue-finalize.md` | Add `memory-capture` calls at missing trigger points |

### Potential aether-utils.sh Changes (evaluate during implementation)

| Change | Why | Priority |
|--------|-----|----------|
| `context-update decision` — suppress auto-pheromone OR change source to `"auto:decision"` | Eliminates the dedup conflict between `system:decision` and `auto:decision` formats | Low — dedup may work correctly as-is; verify first |
| Add `midden-write` call inside `spawn-complete` for failed status | Centralizes failure tracking so it fires regardless of which playbook handles the failure | Medium — reduces duplication across multiple playbook files |

**Recommendation:** Start with pure playbook changes only. Resist adding aether-utils.sh
subcommands — each new subcommand adds to an already 9,808-line file and requires
syncing with `.aether/agents-claude/` mirrors.

---

## Alternatives Considered and Rejected

| Category | Recommended | Alternative | Why Not |
|----------|-------------|-------------|---------|
| Gap 1 fix | Align dedup check in playbook | Add new `decisions-to-pheromones` subcommand | New subcommand adds complexity for a one-line bash fix |
| Gap 2 fix | Explicit bash loop over phase_learnings | New `phase-to-instincts` subcommand | Logic is simple enough for a bash block; no new infrastructure needed |
| Gap 3 fix | Add midden-write calls at additional failure points | Lower threshold from 3 to 1 | Threshold of 1 defeats the purpose of recurrence detection; better to capture more failures |
| Gap 4 fix | Add memory-capture calls at missing trigger points | Centralize all memory-capture in a single `build-complete` hook | Single hook can't distinguish failure type (decision vs failure vs success) for correct event_type mapping |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Python/Node for any gap fix | Breaks portability; entire system is bash | Bash + jq |
| New JSON state files | Four gaps are about calling existing subcommands, not storing new state | Existing `pheromones.json`, `COLONY_STATE.json`, `midden.json`, `learning-observations.json` |
| New agent definitions | Gaps are in orchestration playbooks, not agent behavior | Edit existing playbook markdown files |
| `--no-verify` on git | Would hide pre-commit failures | Fix the failing hook, don't skip it |

---

## Sources

All findings are HIGH confidence based on direct source code inspection:

- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 482-514 (context-update decision + auto-pheromone)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 5402-5501 (memory-capture full implementation)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 5286-5331 (learning-check-promotion)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 7252-7369 (instinct-create)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 8211-8270 (midden-write)
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` lines 9581-9605 (midden-recent-failures)
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-advance.md` (Steps 2.1b, 2.1c, 3, 3a, 3b)
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/continue-finalize.md` (Step 2.1.6 QUEEN-01)
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-wave.md` lines 400-419 (failure tracking)
- `/Users/callumcowie/repos/Aether/.aether/docs/command-playbooks/build-context.md` Step 4 (colony-prime)

---

*Stack research for: Aether integration gap fixes (bash-only wiring changes)*
*Researched: 2026-03-14*
