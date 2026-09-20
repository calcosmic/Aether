# Phase 149 Research: Queen Execution Policy

## Question
Does documentation (CLAUDE.md, playbooks, wrappers) accurately reflect what the Go runtime does for verification depth, caste relevance, and agent spawning?

## Method
Read `cmd/review_depth.go`, `cmd/caste_relevance.go`, `cmd/queen_spawn_budget.go`, and compare against `CLAUDE.md`, `.claude/commands/ant/build.md`, `.claude/commands/ant/continue.md`.

## Findings

### Finding 1: CLAUDE.md execution mode documentation is incomplete

**What CLAUDE.md says (v1.0.41):**
> `fast`: low-risk work; light continue verification, watcher subprocess skipped, no continue review subprocesses.
> `standard`: moderate-risk/refactor work; standard verification, watcher subprocess skipped, focused Probe review allowed.
> `final-review`: final, release, security, or core runtime/state/dispatch work; heavy verification with Watcher and specialist review enabled.

**What the Go runtime actually does (`cmd/review_depth.go`):**
- `resolveVerificationDepth()` has a 5-level priority chain: heavy flag > light flag > explicit `--verification-depth` string > keyword match > smart default
- Smart default considers: phase mode (discovery/production), position (early/late/final), risk level (high/medium/low)
- Discovery mode → always light
- Production mode → at least standard, heavy for final/high-risk
- Security keywords in phase name → heavy
- Final phase → heavy (unless light flag explicitly set)

**Gap:** CLAUDE.md only documents 3 human-facing modes (fast/standard/final-review) but doesn't explain the actual algorithm. The runtime's behavior is more nuanced.

### Finding 2: Continue wrapper contradicts Go runtime on watcher spawning

**What the continue wrapper says:**
> Normal path: `aether continue --skip-watchers --verification-depth standard $ARGUMENTS`

**What the Go runtime says (`cmd/caste_relevance.go:240-245`):**
```go
case "continue":
    switch stateVerificationDepth(state) {
    case colony.VerificationDepthLight:
        return caste == "watcher"
    case colony.VerificationDepthHeavy:
        return caste == "watcher" || caste == "gatekeeper" || caste == "auditor" || caste == "probe"
    default:
        return caste == "watcher" || caste == "probe"
    }
```

**Gap:** The wrapper runs `--skip-watchers` for standard continue, but the Go runtime's `isAlwaysRequired()` says `watcher` is ALWAYS required for continue at any depth. This is a direct contradiction.

### Finding 3: Playbook spawning instructions don't explain the caste relevance system

**What playbooks say:** "Use `dispatch_manifest`" / "spawn matching dispatches"

**What the Go runtime does (`cmd/caste_relevance.go`):**
- `casteRelevanceScore()` scores each caste 0-100 based on keyword matching against phase text
- `spawnThreshold()` varies by flow: build/continue=30, plan=40, colonize/swarm=35, seal=50
- `isAlwaysRequired()` hardcodes required castes per flow+depth
- `isCasteSuppressed()` removes castes that don't belong (e.g., builder/weaver/tracker for continue/seal)
- `applyQueenSpawnBudget()` caps total workers (continue light=3, standard=4, heavy=6)

**Gap:** Playbooks tell users to "spawn from manifest" but never explain HOW the manifest decides which castes appear. Users can't predict which agents will spawn.

### Finding 4: Deterministic commands need verification

Commands classified as "display, signal, session, admin" in the command catalog (Phase 146) should never spawn agents. Need to verify this by checking their implementations.

### Finding 5: Continue at standard depth spawns more than documented

**Wrapper says:** standard = "watcher subprocess skipped, focused Probe review allowed"

**Runtime says:** standard continue always requires watcher+probe, plus any scored castes >= 25 threshold, capped at 4 workers total.

**Gap:** The wrapper claims watcher is "skipped" at standard, but the runtime requires it.

## Recommendations

1. **Update CLAUDE.md** to match Go runtime VerificationDepth behavior exactly
2. **Fix continue wrapper** to remove `--skip-watchers` for standard depth (or fix Go runtime to match)
3. **Document caste relevance** in playbooks so users understand spawn decisions
4. **Verify deterministic commands** never spawn agents
5. **Add tests** proving the documented behavior matches the runtime
