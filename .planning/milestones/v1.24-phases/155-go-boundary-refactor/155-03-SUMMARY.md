# 155-03 Summary: Go Boundary Refactor — Prompt Section Extraction

## What Changed

This plan extracted all hardcoded prompt section text from `cmd/colony_prime_context.go` into an editable `colony/prompts/colony-prime.md` file, while keeping the context assembly algorithm (trimming, scoring, budget management) entirely in Go.

### Files Created

- `colony/prompts/colony-prime.md` — YAML frontmatter containing all 16 section templates (state, review_depth, pheromones, instincts, decisions, learnings, worker_handoffs, hive_wisdom, learned_memory, global_queen_md, user_preferences, prior_reviews, local_queen_wisdom, clarified_intent, blockers, medic_health)
- `cmd/prompt_template_loader.go` — `loadColonyPrimeTemplates()`, `getSectionTemplate()`, `writeSectionHeader()`, `sectionString()`, `fmtOrFallback()` with `sync.Once` caching and fallback to hardcoded strings
- `cmd/colony_prime_context_test.go` — 10 tests covering template load, fallback, partial fallback, review depth, blocker format, pheromone format, worker handoffs, prior reviews, medic health, and learned memory

### Files Modified

- `cmd/colony_prime_context.go` — Refactored all 16 section builders to use `writeSectionHeader()` + `fmtOrFallback()` / `sectionString()` helpers. Algorithmic logic (budget, scoring, trimming, ranking) is unchanged. Hardcoded strings remain as fallbacks.
- `cmd/codex_dispatch_contract.go` — `renderWorkerHandoffSection()` now uses template-driven format strings via `fmtOrFallback()` instead of hardcoded Sprintf calls.

### Verification Results

```
$ go build ./cmd/aether
# success

$ go test ./cmd/... -run ColonyPrime -v
ok      github.com/calcosmic/Aether/cmd 0.882s
(all ColonyPrime tests pass)

$ go test ./cmd/... -count=1
ok      github.com/calcosmic/Aether/cmd 92.498s
(no regressions)
```

### Key Design Decisions

- **Fallback-first**: If `colony/prompts/colony-prime.md` is missing, malformed, or a section is absent, the runtime falls back to the exact hardcoded strings that existed before extraction. No user-facing behaviour change.
- `sync.Once` caches the parsed templates so file I/O only happens once per process.
- `colonyPrimeTemplatesPathOverride` + `resetColonyPrimeTemplatesCache()` enable isolated tests without touching the real prompts file.
- The plan’s threat model (T-155-07, T-155-08, T-155-09) is addressed: file is loaded read-only, parse errors fallback safely, and templates have no more privilege than the hardcoded text they replaced.

## Status

All deliverables created and verified. Build passes, all tests pass, no regressions.
