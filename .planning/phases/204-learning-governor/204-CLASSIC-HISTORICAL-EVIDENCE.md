# Phase 204: Classic-Era Learning Mechanics — Historical Evidence

This file exists so `204-CLASSIC-SYNTHESIS.md` can cite, rather than restate, direct quotable evidence for the seven Classic-era learning mechanics SYNTH-06 names. Every excerpt below was obtained with `git show <commit>:<path>`, reproduced beside the command that produced it.

## Migration boundary

Commit `6732b0ba` ("contextual(phase-06): Deprecate dead shell scripts and complete migration to Go binary", 2026-04-07) is the commit RESEARCH.md's Open Question 1 names as the shell-to-Go migration boundary. Direct inspection this session (`git show --stat 6732b0ba`) shows this commit did **not** delete the shell scripts — it prepended a 7-line deprecation banner to each of 43 files under `.aether/utils/` and `.aether/scripts/`, changing no executable line. `git show 6732b0ba:.aether/utils/learning.sh | head -20` confirms the banner sits above the unmodified `#!/usr/bin/env bash` shebang and the original file body.

The shell scripts were physically removed from git tracking later, across four separate commits reachable from the current HEAD: `0063be8b` (2026-04-05, "feat(04): remove shell scripts and Node.js runtime"), `4e9377db` (2026-04-07, "feat(02-02): remove all deprecated commands, shell scripts, and audit infrastructure"), `92d6c8d6` (2026-04-08, "feat(07-01): remove all 58 dead shell scripts from git tracking"), and `9de7083e` (2026-04-08, a merge-conflict resolution). Their non-linear dates relative to `6732b0ba` (one is dated two days *before* the deprecation commit) indicate this repository went through more than one parallel rewrite branch before the final merge to a single Go binary — not a single clean cutover. Since `6732b0ba`'s own diff shows no functional change to the shell bodies, **`6732b0ba` itself is the fullest surviving state of every mechanism below** and is used as the single citation anchor throughout this file, per the plan's own instruction. All seven mechanics below are recovered from this one commit (i.e., from *before* the shell-to-Go migration point at which the Go binary took over); none required a commit *after* `6732b0ba`, because `6732b0ba` changed no executable shell content relative to its immediate parent — it only marked the files deprecated.

`.aether/learning.md` at `3a5b81c2` (v3.1, 2026-02-15, already cited in RESEARCH.md) is a markdown *design spec*. It describes the same wisdom-type taxonomy (`philosophy/pattern/redirect/stack/decree/failure`) and per-instinct `instinct_outcomes: [{"id":..., "success": true/false}]` self-report shape the shell code below implements. Where the spec and the executable code agree, both are cited; no disagreement was found between them for any of the seven mechanics.

---

## 1. Observation capture

**Source:** `git show 6732b0ba:.aether/utils/learning.sh` (function `_learning_observe`, lines 104–320)

A worker or command called `learning-observe <content> <wisdom_type> [colony_name] [source_type] [evidence_type]`. The runtime validated `wisdom_type` against a closed enum (`philosophy|pattern|redirect|stack|decree|failure`), then computed a SHA-256 content hash for deduplication:

```bash
# git show 6732b0ba:.aether/utils/learning.sh (lines 104-127, 208-213)
_learning_observe() {
    content="${1:-}"
    wisdom_type="${2:-}"
    colony_name="${3:-unknown}"
    local source_type="${4:-observation}"
    local evidence_type="${5:-anecdotal}"
    ...
    valid_types=("philosophy" "pattern" "redirect" "stack" "decree" "failure")
    ...
    content_hash="sha256:$(echo -n "$content" | sha256sum | cut -d' ' -f1)"
    observations_file="$COLONY_DATA_DIR/learning-observations.json"
```

On a repeat observation of the same content hash, the entry's `observation_count` incremented, `last_seen` updated, the reporting colony name was appended to a `colonies` array (deduplicated via `unique`), and a `trust_score` was recomputed via a separate `trust-calculate` subprocess call keyed on `source_type`/`evidence_type`/days-since-first-seen (lines 242–246, 255–268). A brand-new observation created a fresh entry (schema includes `content_hash`, `wisdom_type`, `observation_count: 1`, `colonies: [colony_name]`, `trust_score`, `first_seen`, `last_seen`). The write path rotated three numbered `.bak` backups before every mutation and had an explicit corruption-recovery path (retry against `.bak.1`/`.bak.2`/`.bak.3`, else reset from an empty template) — lines 141–174.

**Judgement:** Causally real. This is genuine, executed shell arithmetic against a real, atomically-written, backed-up JSON file with real deduplication and a real (if simplistic) trust-scoring subprocess call — not presentation and not honour-system.

**Limitation that must not return:** The trust score recompute on a re-observation is a full subprocess round-trip (`bash "$0" trust-calculate ...`) inside the hot write path, on every single repeated observation — an unbounded-cost pattern later replaced in Go by `pkg/memory/trust.go`'s in-process calculation.

**Before/after migration:** Unchanged in content across `6732b0ba` (deprecation-only commit, confirmed above). The Go rewrite's `learn.NewObservationService(...).CaptureWithTrust(...)` (`pkg/memory/observe.go`, cited in RESEARCH.md Census item 13/current audit) is architecturally the direct descendant of this function — same dedup-by-hash, same trust-score-on-write shape, moved in-process.

---

## 2. Instinct formation and confidence adjustment

**Source:** `git show 6732b0ba:.aether/utils/learning.sh` (functions `_instinct_create`, lines 1566–1744, and `_instinct_apply`, lines 1745–1811)

Instinct creation (`instinct-create --trigger T --action A --confidence C --domain D --source S --evidence E`) first checked for an exact `(trigger, action)` match; if found, it boosted confidence **+0.1 capped at 1.0** and incremented `applications` (lines 1596–1608). If no exact match existed, it ran a **fuzzy-dedup pass**: Jaccard similarity independently computed for `trigger` and `action` text against every existing instinct; if both exceeded **0.80**, the two instincts were merged — confidences averaged, the longer of the two trigger/action strings kept, evidence arrays concatenated (lines 1614–1672). Only if neither an exact nor fuzzy match existed was a wholly new instinct created, seeded with:

```bash
# git show 6732b0ba:.aether/utils/learning.sh (lines 1699-1717)
.memory.instincts = (
  ((.memory.instincts // []) + [{
    id: env.IC_ID, trigger: env.IC_TRIGGER, action: env.IC_ACTION,
    confidence: (env.IC_CONFIDENCE | tonumber),
    status: "hypothesis", domain: env.IC_DOMAIN, source: env.IC_SOURCE,
    evidence: [env.IC_EVIDENCE], tested: false, created_at: env.IC_NOW,
    last_applied: null, applications: 0, successes: 0, failures: 0
  }])
  | sort_by(-.confidence) | .[:30]
)
```

New instincts started with an explicit `status: "hypothesis"` field and a `tested: false` flag — the vocabulary the Go rewrite's `learn.StatusHypothesis` directly descends from. The store was capped at 30 entries, sorted by descending confidence (lowest-confidence entries silently evicted past the cap).

**Confidence moves down on failure — the step size:** `_instinct_apply --id <id> --outcome success|failure` recorded the actual application outcome:

```bash
# git show 6732b0ba:.aether/utils/learning.sh (lines 1770-1796)
if [[ "$ia_outcome" == "success" ]]; then
  _state_mutate '... .confidence = ([(.confidence + 0.05), 1.0] | min) ...'
else
  _state_mutate '... .confidence = ([(.confidence - 0.1), 0.1] | max) ...'
fi
```

Success: `applications += 1`, `successes += 1`, confidence **+0.05, capped at 1.0**. Failure: `applications += 1`, `failures += 1`, confidence **−0.1, floored at 0.1** (never driven to zero — a failed instinct stays visible, just weaker). This is the exact "confidence decreases when a pattern leads to errors" behavior RESEARCH.md's Code Examples section quotes from the `3a5b81c2` design spec, now confirmed against the executable code itself, not just the spec.

**Judgement:** Causally real, with an honour-system input. The confidence arithmetic itself (creation boost, fuzzy merge, apply success/failure step) is genuine, executed, atomically-persisted state mutation — not presentation. But `_instinct_apply`'s `--outcome success|failure` argument was supplied by whatever caller invoked it (a worker self-reporting per the `3a5b81c2` design spec's `instinct_outcomes` shape) with **no independent verification of the claimed outcome anywhere in this function** — the arithmetic that acts on the outcome is real; the honesty of the outcome itself was never checked. This is the exact "honour-system" input RESEARCH.md's Anti-Patterns section already warns the modern replacement must not revive unverified.

**Limitation that must not return:** Self-reported, unverified per-instinct outcome as the sole trigger for a real state mutation (confidence up OR down) with no evidence requirement of any kind — contrast with the current Go `recordRecruitmentCredit`'s `ChangedDecisionID`/`EffectEvidenceID` evidence-gating, which has no Classic ancestor (see mechanism 2's current-Go audit in `204-CLASSIC-SYNTHESIS.md`).

**Before/after migration:** Unchanged in content across `6732b0ba`. **This is a genuine "current is thinner than Classic's own design intent" finding**: the modern Go `recordInstinctApplicationsForPhase` (RESEARCH.md Pitfall 2) has **no negative branch at all** — every recorded application is unconditionally `success: true`. Classic's `_instinct_apply` had a real, if unverified, negative branch (`-0.1`, floored at `0.1`) that the current Go mechanism has fully lost, not merely left unverified.

---

## 3. Queen memory promotion

**Source:** `git show 6732b0ba:.aether/utils/queen.sh` (function `_queen_promote`, lines 326–420+) and `.aether/utils/learning.sh` (function `_learning_promote_auto`, lines 149–232)

A learning became eligible for promotion into `QUEEN.md` (the shared instruction file every worker's prompt is primed from) purely on an **observation-count threshold**, with one exception:

```bash
# git show 6732b0ba:.aether/utils/queen.sh (lines 364-380)
threshold=$(get_wisdom_threshold "$wisdom_type" "propose")
if [[ "$wisdom_type" != "decree" ]] && [[ -f "$observations_file" ]]; then
  observation_data=$(jq ... '.observations[] | select(.content_hash == $hash) | {count: .observation_count, colonies: .colonies}' ...)
  if [[ -n "$observation_data" ]] ...; then
    obs_count=$(echo "$observation_data" | jq -r '.count // 0')
    if [[ "$obs_count" -lt "$threshold" ]]; then
      json_err ... "Threshold not met: $obs_count/$threshold observations" ...
    fi
  else
    json_err ... "No observations found for this content" ...
  fi
fi
```

`decree`-type wisdom (an owner-issued directive) always promoted immediately regardless of observation count — the sole type with an implicit threshold of 0. Every other type (`philosophy`, `pattern`, `redirect`, `stack`, `failure`) required its observation count to meet a per-type threshold read from a shared policy table (`get_wisdom_threshold`). On success, the content was appended verbatim under one of six section headers (`## 📜 Philosophies`, `## 🧭 Patterns`, `## ⚠️ Redirects`, `## 🔧 Stack Wisdom`, `## 🏛️ Decrees`, in the "v1" emoji format, or the four-section "v2" format — `## Codebase Patterns`, `## User Preferences`, `## Build Learnings`, `## Instincts` — detected by probing for the literal line `## Build Learnings` in the target file, lines 383–398).

The auto-promotion entry point (`_learning_promote_auto`, `learning.sh:149-232`) additionally computed a **recurrence-calibrated confidence** for the instinct it created alongside the QUEEN.md promotion:

```bash
# git show 6732b0ba:.aether/utils/learning.sh (lines 191-197)
# LRN-01: Recurrence-calibrated confidence
# Formula: min(0.7 + (observation_count - 1) * 0.05, 0.9)
lp_confidence=$(awk -v c="${observation_count:-1}" 'BEGIN {
  v = 0.7 + (c - 1) * 0.05
  if (v > 0.9) v = 0.9
  if (v < 0.7) v = 0.7
  printf "%.2f", v
}')
```

**Judgement:** Causally real. The threshold check, the section-header mapping, and the confidence formula are genuine executed logic against real files, not presentation.

**Limitation that must not return:** No "was this ever genuinely delivered to a worker and acted on" gate existed anywhere in this promotion path — a learning reaching the observation-count threshold promoted purely on repetition count, never on any evidence that the promoted content had actually helped anyone. This is precisely the gap RESEARCH.md's Pitfall 1/2 describe as still open in the current Go mechanism, but for a different reason: current Go's `learn.Entry` promotion path is even weaker in one respect (Pitfall 1: unverified hypothesis-status entries are shown to workers under a header claiming "Verified Outcomes") while being stricter in another (`TestQueenPromotionNeverHappensWithoutRecordedUse`, per CLAUDE.md's Wisdom Pipeline table, requires genuine recorded use before a lesson reaches QUEEN.md — a real-use gate Classic never had at all).

**Before/after migration:** Unchanged in content across `6732b0ba`.

---

## 4. Hive cross-colony sharing

**Source:** `git show 6732b0ba:.aether/utils/hive.sh` (functions `_hive_store`, lines 71–253, and `_hive_read`, lines 254–394)

A wisdom entry crossed a project boundary via `hive-store --text T --domain CSV --source-repo PATH --confidence C --category CAT`, written to a single cross-colony file (`~/.aether/hive/wisdom.json`), content-hash-deduplicated (first 12 hex chars of SHA-256), and capped at 200 entries with oldest-by-`last_accessed` eviction:

```bash
# git show 6732b0ba:.aether/utils/hive.sh (lines 220-236)
hs_updated=$(jq --argjson entry "$hs_entry" --arg now "$hs_now_iso" '
  .entries = (.entries + [$entry]) |
  if (.entries | length) > 200 then
    .entries = (.entries | sort_by(.last_accessed) | .[-200:])
  else . end | ...
' "$hs_wisdom_file")
```

A same-content-hash entry from a **new** source repo did not create a duplicate — it merged, incrementing `validated_count`, adding the new repo to a `source_repos` array, and boosting confidence via a **repo-count confidence tier, never downgraded**:

```bash
# git show 6732b0ba:.aether/utils/hive.sh (lines 178-186)
(.source_repos | length) as $repo_count |
(if $repo_count >= 4 then 0.95
 elif $repo_count == 3 then 0.85
 elif $repo_count == 2 then 0.7
 else .confidence end) as $tier_confidence |
.confidence = ([.confidence, $tier_confidence] | max)
```

This is the exact `2 repos → 0.70 / 3 repos → 0.85 / 4+ repos → 0.95, never downgraded` tier table CLAUDE.md's "Multi-Repo Confidence Boosting" section documents for the current Go implementation — confirmed here to be a **faithful, near-verbatim restoration**, not a Go-era reinvention.

The receiving colony saw a `source_repos` array on every hive entry (which repos had confirmed it), so provenance was visible, though not necessarily surfaced to the worker's own prompt text (`_hive_read`'s `--format text` output, lines 366–378, renders `[confidence] [category] text (validated: N, domains: ...)` — no `source_repos` in the rendered text form, only in the raw JSON).

**Critically, `_hive_read` updated `access_count`/`last_accessed` on every returned entry as part of the read itself:**

```bash
# git show 6732b0ba:.aether/utils/hive.sh (lines 333-345)
hr_updated=$(jq --argjson returned_ids "$hr_returned_ids" --arg now "$hr_now_iso" '
  .entries = [.entries[] |
    if (.id as $id | $returned_ids | index($id)) != null then
      .access_count = ((.access_count // 0) + 1) | .last_accessed = $now
    else . end
  ] | .last_updated = $now
' "$hr_wisdom_file")
```

**Judgement:** Causally real, and the write side is already a faithful modern restoration. But this is a genuine **regression, not a gap** in the current Go implementation: RESEARCH.md's Census item 9 confirms `cmd/context_weighting.go:18-73` (the current Go read path feeding worker briefs) reads hive wisdom but **never bumps `access_count`/`last_accessed`**, so the 200-cap LRU eviction today is driven only by the orphaned manual `hive-read` CLI. Classic's shell implementation updated access tracking on every genuine read, unconditionally, in the same function that returned the data. The current Go implementation has silently dropped this half of the mechanism.

**Limitation that must not return:** None specific to this mechanism beyond the sanitization discipline already restored faithfully (`_hive_store`'s XML-tag/prompt-injection/shell-injection rejection at lines 96–110 is the direct ancestor of `colony.SanitizeSignalContent`).

**Before/after migration:** Unchanged in content across `6732b0ba`.

---

## 5. Midden failure reinforcement

**Source:** `git show 6732b0ba:.aether/utils/midden.sh` (functions `_midden_write`, lines 34–106, and `_midden_review`, lines 134–193); cross-checked against `.aether/utils/council.sh` (`git grep -n "midden\|recur" 6732b0ba -- .aether/utils/council.sh` returns zero hits).

`midden-write <category> <message> <source>` appended one entry (`id`, `timestamp`, `category`, `source`, `message`, `reviewed: false`) to `midden.json` under a lock, with a documented no-lock fallback that logged a warning rather than losing the entry (lines 71–106). `midden-review [--category] [--limit] [--include-acknowledged]` filtered unacknowledged entries, grouped counts by category, and returned them sorted by timestamp descending — a manual review surface, not an automatic trigger.

**No auto-reinforcement or auto-REDIRECT-on-recurrence mechanism was found anywhere in this file, and a targeted search of `council.sh` (the other plausible location for a recurrence-triggered signal) returned zero matches for "midden" or "recur".** Every recurrence of a failure category simply accumulated as additional unacknowledged `midden.json` entries, surfaced to a human via `midden-review` at `/ant-patrol` time — there was no code path that counted N recurrences of the same category and automatically emitted a REDIRECT pheromone.

**Judgement:** Causally real for the write/review mechanism itself (a genuine, atomically-written, lockable append-only log, reviewed by category) — but the specific behavior CLAUDE.md's "The Core Insight" section describes for the CURRENT system ("three unacknowledged failures of the same kind produce one 'don't do this' note... two produce none", `TestThreeFailuresOfOneKindProduceOneRedirect`) has **no historical evidence recoverable**. This is genuinely new to the Go rewrite, not a restoration.

**Limitation that must not return:** N/A — the write/review mechanism itself carries no unsafe shortcut; its only gap (no automatic reinforcement) is closed by new, not restored, work.

**Before/after migration:** Unchanged in content across `6732b0ba`. The automatic recurrence-to-REDIRECT behavior exists only after the migration, in Go — it has no shell-era ancestor at all.

---

## 6. Signal strength reinforcement and decay

**Source:** `git show 6732b0ba:.aether/utils/pheromone.sh` (functions `_pheromone_write`, lines 51–225ish; `_pheromone_read`/decay formula, lines 470–554; `_pheromone_expire`, lines 2180–2385)

**Write-time defaults:** `pheromone-write <type> <content> [--strength N] [--ttl T]` defaulted strength by type — **REDIRECT 0.9, FOCUS 0.8, FEEDBACK 0.7** (lines 118–124) — and defaulted TTL to `phase_end` unless an explicit duration (`Nm`/`Nh`/`Nd`) was given (lines 145–166). Content was sanitized against XML-tag injection, shell-metacharacter injection, and prompt-override phrases *before* the angle-bracket-escaping step (lines 79–99) — the direct, near-verbatim ancestor of the current Go `colony.SanitizeSignalContent`.

**Decay arithmetic (read-time, never baked into the stored value):**

```bash
# git show 6732b0ba:.aether/utils/pheromone.sh (lines 513-530, function _pheromone_read)
def decay_days(t):
  if t == "FOCUS" then 30
  elif t == "REDIRECT" then 60
  else 90
  end;
...
(decay_days(.type)) as $dd |
... effective_strength: (($eff * 100 | round) / 100) ...
```

`effective_strength = strength * (1 - elapsed_days / decay_days(type))`, computed fresh on every read — never mutating the stored `strength` field. A signal whose decayed `effective_strength` fell below **0.1** was treated as inactive (the exact floor cited at `.aether/utils/pheromone.sh` lines 350–416's display logic and RESEARCH.md's own citation of `v5.4.0`'s equivalent). This decay-days table (FOCUS 30 / REDIRECT 60 / everything-else 90) is reused verbatim inside `_pheromone_expire`'s own eternal-promotion check (line 2313–2319), confirming one shared formula, not two.

**Eternal-memory promotion on expiry — the actual gate, more selective than "any expiring signal":**

```bash
# git show 6732b0ba:.aether/utils/pheromone.sh (lines 2324-2331)
((.strength // 0) * (1 - ($elapsed / $dd))) as $eff_raw |
(if $eff_raw < 0 then 0 else $eff_raw end) as $eff |
(($eff * 100) | floor) as phe_strength_int
...
if [[ "$phe_strength_int" -gt 80 ]]; then
  bash "$0" eternal-store "$phe_text" --type "$phe_type" --source "$phe_source" \
    --strength "$(echo "$phe_signal" | jq -r '.strength // 0')" --signal-id "$phe_id" \
    --reason "promoted_on_expire" >/dev/null 2>&1
fi
```

**A signal was promoted to eternal (cross-session, undecaying) memory on expiry ONLY if its own decayed `effective_strength` exceeded 80% at the moment it expired** — not "every signal that expires," as CLAUDE.md's summary prose implies and as the current Go `signalIsWorthKeeping`/`appendEternalMemoryEntry` gate (`cmd/phase_end_signals.go:233`, RESEARCH.md Census item 10) must be checked against directly before the synthesis assumes parity. This is a genuine, non-trivial quantitative gate this study must not silently drop when comparing to the current implementation.

The expired signal was additionally archived into `midden.json` (lines 2358–2374) with `active: false` set on the original `pheromones.json` record (never deleted, only deactivated — lines 2343–2351), and reinforcement of a duplicate write (same content hash) was not found as a distinct function in this file; RESEARCH.md's own citation (CLAUDE.md's "Content Deduplication (v2.0)" section describing SHA-256 content-hash reinforcement to `max(old, new)` strength) was not independently re-verified against the shell code in this pass — flagged here as an assumption the synthesis should either re-confirm from `_pheromone_write`'s full body or record as unresolved, since the excerpt read in this session (lines 51–225) shows write-time defaulting but this session did not trace the full dedup/reinforcement branch beyond that point.

**Judgement:** Causally real. Real, executed arithmetic; the 0.1 floor and the decay-days table are not presentation.

**Limitation that must not return:** None found beyond the quantitative-gate precision noted above (>80%, not "any expiry") which the synthesis must carry forward accurately.

**Before/after migration:** Unchanged in content across `6732b0ba`. The Go rewrite's `writePheromoneSignal` (`NOW-08` in Phase 203's own synthesis) is already confirmed there as "near-identical to Classic's own real write discipline." The 0.1-floor decay convention is the exact historical precedent Phase 203's synthesis (`SYN-203-10`) already used to convict `codex_plan.go:4274`'s `>0` REDIRECT scan of drifting from Classic's own established convention — reconfirmed here directly against the executable shell source, not merely against `v5.4.0` as Phase 203 cited.

---

## 7. Learning presentation

**Source:** `git show 6732b0ba:.aether/utils/queen.sh` (function `_extract_wisdom_sections`, lines 20–133, and `_queen_read`, lines 185–265+); `.aether/utils/learning.sh` (function `_learning_inject`, lines 76–103)

`queen-read` assembled worker-priming context from `QUEEN.md` (global `~/.aether/QUEEN.md` first, then local `.aether/QUEEN.md`, combined by simple string concatenation — "local wisdom extends global," lines 226–241) into exactly four named sections in the "v2" format: `user_prefs`, `codebase_patterns`, `build_learnings`, `instincts` — sourced from the literal Markdown headers `## User Preferences`, `## Codebase Patterns`, `## Build Learnings`, `## Instincts` (lines 27–30). An older "v1" six-emoji-heading format (`## 📜 Philosophies`, `## 🧭 Patterns`, `## ⚠️ Redirects`, `## 🔧 Stack Wisdom`, `## 🏛️ Decrees`, `## ... User Preferences`) was detected and mapped onto the same four v2 keys for backward compatibility (lines 64–131).

**A targeted search for the exact current-Go phrase this study must compare against — `git grep -n "LEARNED MEMORY\|Verified Outcomes" 6732b0ba -- .aether/utils` — returns zero matches.** Neither "LEARNED MEMORY" nor "Verified Outcomes" appears anywhere in the Classic shell corpus. **The header phrase `"## LEARNED MEMORY (Verified Outcomes)"` that RESEARCH.md's Pitfall 1 identifies as currently, actively mislabeling hypothesis-status content is a Go-era invention, not a regression from a previously-honest Classic label.** Classic's own section headers (`## Instincts`, `## Build Learnings`) made no verification claim at all, honest or otherwise — they were plain content headers.

Separately, `_learning_inject <tech_keywords_csv>` filtered the global cross-colony `learnings.json` (capped at 50 entries via `_learning_promote`, lines 23–68) by tag/keyword substring match for injection into worker context, independent of the QUEEN.md wisdom path — a second, narrower injection surface with no status/verification concept of its own either.

**Judgement:** Presentation-only for the labeling question specifically (no code in either function computed or asserted "verified" — the sections were rendered as plain named categories), but causally real for the underlying section-extraction and combination mechanism itself (line-numbered `awk` extraction between detected section headers, real string concatenation of global+local content — not fabricated or inferred).

**Limitation that must not return:** None to carry forward from Classic on this specific point, since Classic never made the false claim the current system makes. The limitation to fix is entirely current-Go's own invention (Pitfall 1), not a Classic-era regression being restored.

**Before/after migration:** Unchanged in content across `6732b0ba`.

---

## Summary table

| Mechanism | Recovered from | Causally real / presentation / honour-system | Must-not-return limitation | Differed materially across migration? |
|---|---|---|---|---|
| 1. Observation capture | `6732b0ba:.aether/utils/learning.sh` | Causally real | Unbounded subprocess cost per re-observation | No — content unchanged by the migration commit |
| 2. Instinct confidence | `6732b0ba:.aether/utils/learning.sh` | Causally real arithmetic, honour-system input (self-reported outcome) | Unverified self-report as sole trigger for a real state mutation | No — but current Go has REGRESSED (dropped the negative branch entirely) |
| 3. Queen memory promotion | `6732b0ba:.aether/utils/queen.sh`, `.aether/utils/learning.sh` | Causally real | No "genuinely delivered and used" gate existed at all | No — current Go is now STRICTER on this axis (requires recorded use) |
| 4. Hive sharing | `6732b0ba:.aether/utils/hive.sh` | Causally real, faithfully restored on write; REGRESSED on read | Access-tracking-on-read must not stay dropped | No — current Go has REGRESSED (drops access_count/last_accessed bump) |
| 5. Midden reinforcement | `6732b0ba:.aether/utils/midden.sh` (write/review only) | Causally real for write/review; **no historical evidence recoverable for automatic recurrence-to-REDIRECT** | N/A | Yes — the recurrence-triggered auto-REDIRECT is Go-era-only, no Classic ancestor |
| 6. Signal strength/decay | `6732b0ba:.aether/utils/pheromone.sh` | Causally real | Eternal promotion is a >80%-effective-strength gate, not "any expiry" | No — content unchanged by the migration commit |
| 7. Learning presentation | `6732b0ba:.aether/utils/queen.sh`, `.aether/utils/learning.sh` | Presentation-only (plain headers, no verification claim); causally real extraction mechanism | None — the "(Verified Outcomes)" mislabeling is a Go-era-only defect | No — content unchanged by the migration commit |
