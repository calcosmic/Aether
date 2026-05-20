# Feature Research: Historical Workflow Baselines for Reliability Restoration

**Domain:** AI colony framework (Aether) -- flagship workflow behavioral baselines from Classic era through modern versions
**Researched:** 2026-05-20
**Confidence:** HIGH (based on direct git tag inspection of v5.4.0 source and milestone audit trail)

## Feature Landscape

This research defines behavioral baselines for each flagship workflow by inspecting the actual command source code at the Classic v5.4.0 git tag, tracking evolution through v1.10-v1.21 milestones, and identifying regressions from the shell-to-Go migration and subsequent TS host cutover.

### Table Stakes: Flagship Workflows

These are the core workflows users rely on. A workflow is "working" when it completes the full lifecycle from user invocation through state mutation with honest output and no silent failures.

| Workflow | "Good" at v5.4.0 (Classic) | Status at v1.22 (Current) | Key Regression |
|----------|---------------------------|--------------------------|----------------|
| **init** | Rich charter ceremony: scan repo, generate charter, ask approval, write QUEEN.md, set state v3.0 | Runtime-owned via `aether init-research`; wrapper does intent refinement | Lost: charter approval ceremony replaced by AI synthesis; lost: auto-suggest pheromones |
| **colonize** | 4 parallel Surveyor agents (nest, disciplines, pathogens, provisions) writing 7 survey docs; stale detection | Runtime manifest via `aether host colonize`; wrapper spawns from manifest | Surveyors still work but ceremony is runtime-driven instead of playbook-driven |
| **plan** | Iterative research loop (scout + route-setter), confidence scoring, stall detection, territory survey injection, auto-finalize | Go-owned `aether plan` with worker dispatch; wrapper is thin passthrough | Lost: interactive confidence display during planning; gained: Go-managed state |
| **oracle/RALF** | Rich wizard (7 questions), in-session loop, template system (5 types), diminishing returns, promote findings to colony | Go-owned `aether oracle` with `--depth`, `--template`, `--background`; wrapper is thin | Lost: wizard ceremony (replaced by shorter intent refinement); gained: background mode, Go state |
| **build** | 5-stage playbook (prep, context, wave, verify, complete); Queen spawns workers directly; wave-based parallel execution | Manifest via `aether host build --dry-run`; wrapper spawns from manifest; Go owns finalizer | Architecture fundamentally changed: was playbook-driven, now manifest-driven |
| **continue** | 4-stage playbook (verify, gates, advance, finalize); 6-phase verification loop; command resolution; learning extraction | Default path: `aether continue` (fast, Go-only); heavy path: manifest + wrapper-spawned reviewers | Default path is faster but lost: manual verification loop, learning extraction ceremony |
| **run** | Full autopilot: loads build+continue playbooks inline, pause conditions, replan trigger, headless mode, elapsed tracking | `aether run` CLI subcommand, wrapper just delegates | Lost: inline playbook execution; gained: Go-native autopilot with same pause conditions |
| **swarm** | 4 parallel scouts (archaeologist, pattern hunter, error analyst, web researcher); cross-compare; auto-fix with rollback | `aether swarm` CLI subcommand | Lost: rich 4-scout ceremony; gained: Go-native dispatch with same logic |
| **seal** | Multi-step ceremony: Sage analytics, wisdom approval, hive promotion, Chronicler audit, CROWNED-ANTHILL.md, XML export, commit suggestion | Go-owned `aether seal`; wrapper shows ceremony output | Lost: interactive wisdom review, Sage spawn, Chronicler spawn; gained: Go-native ceremony |
| **entomb** | Full archive: chamber creation, XML export (hard-stop on failure), state reset, HANDOFF.md, eternal memory | Go-owned `aether entomb`; wrapper shows ceremony output | Lost: interactive wisdom review; gained: Go-native archive pipeline |

---

## Detailed Workflow Behavioral Baselines

### 1. Init Workflow

**Classic v5.4.0 (GOOD):**
- User provides goal string
- `aether queen-init` initializes QUEEN.md
- Charter ceremony: scan repo surface, present charter for approval, write to QUEEN.md
- Auto-upgrade old state to v3.0
- Write COLONY_STATE.json with goal, state=IDLE
- Write session file for `/ant-resume`
- Display: colony name, goal, territory status, next steps

**v1.10 Polish (HIGH WATER MARK):**
- Rich init-research: tech stack, directory analysis, colony context, governance detection, 10 pheromone patterns
- Charter approval ceremony with MAX 2 revision rounds
- Suggest-analyze: automatic pheromone suggestions during builds (618 lines of pattern detection)

**v1.11 Unification (Restored):**
- Smart Init ceremony: charter approval flow, repo scanning, governance detection ported to Go
- Rich init-research restored: tech stack, directory analysis, colony context, governance detection, pheromone suggestions
- Suggest-analyze restored

**Current (v1.22) Architecture:**
- `aether init-research --goal "$ARGUMENTS" --target .` does the deterministic scan
- Wrapper does AI intent refinement (4-7 questions when goal is broad)
- Wrapper calls `aether init-create` with refined goal + charter
- Lost: explicit charter approval ceremony (user approves the charter itself)
- Lost: suggest-analyze pheromone auto-detection (not called from init flow)

**Restoration Target:** Charter approval ceremony and suggest-analyze integration need verification. The intent refinement is a reasonable replacement but users should see the charter before it is committed.

### 2. Colonize Workflow

**Classic v5.4.0 (GOOD):**
- Validates colony state exists and is not sealed
- Quick surface scan: package manifests, README, entry points, config files
- Dispatches 4 parallel Surveyor agents via Task tool:
  - `aether-surveyor-provisions` -- PROVISIONS.md + TRAILS.md
  - `aether-surveyor-nest` -- BLUEPRINT.md + CHAMBERS.md
  - `aether-surveyor-disciplines` -- DISCIPLINES.md + SENTINEL-PROTOCOLS.md
  - `aether-surveyor-pathogens` -- PATHOGENS.md
- Session freshness check: auto-clears stale surveys
- Updates COLONY_STATE.json: state=IDLE, territory_surveyed=<timestamp>
- Produces 7 survey documents in `.aether/data/survey/`
- Next steps: route to `/ant-plan` or signal injection

**v1.22 Current:**
- Runtime manifest via `aether host colonize`
- Wrapper spawns surveyors from manifest using platform Agent tool
- Surveyors still produce the same 7 documents
- Lost: stale survey detection is now Go-managed (same behavior, different code path)

**Restoration Target:** Functional parity achieved. Verify stale survey detection works. Verify 7 documents are always produced.

### 3. Plan Workflow

**Classic v5.4.0 (GOOD):**
- Iterative research loop: scout + route-setter per iteration
- User selects depth: fast/balanced/deep/exhaustive
- Territory survey loaded into planning context
- Hive wisdom retrieved for research priming
- Phase domain research spawned before planning loop
- Gap-focused research on iterations 2+
- Stall detection: < 5% improvement for 2 consecutive iterations
- Auto-finalize when confidence >= target or stall detected
- Plan persisted with verification (read-back check)
- Watch files updated for tmux visibility
- Session updated for `/ant-resume`
- No user confirmation needed -- plan auto-finalizes

**v1.12 Safe Colony (Added):**
- Independent planning depth (light/standard/deep)
- Smart depth defaults based on phase position + code change risk
- User depth selection UI at plan start
- Depth persistence from plan through build to continue

**Current (v1.22):**
- Go-owned `aether plan` with worker dispatch
- Scout surveys repo, Route-Setter creates phases
- Confidence scoring preserved
- Territory survey integration preserved
- Lost: watch-status.txt / watch-progress.txt (tmux visibility files)
- Lost: explicit iterative loop visible to user (now hidden in Go runtime)
- Gained: Go-managed state, deterministic execution

**Restoration Target:** Verify the planning loop produces plans with real file references (v1.22 grounding gate). Verify depth controls persist through build/continue. Watch files are obsolete (replaced by Go ceremony output).

### 4. Oracle / RALF Workflow

**Classic v5.4.0 (GOOD):**
- Rich 7-question wizard:
  1. Research topic (or use $ARGUMENTS)
  1.5. Research Brief formulation and approval (MAX 2 revision rounds)
  2. Research template (tech-eval, architecture-review, bug-investigation, best-practices, custom)
  3. Research depth (5/15/30/50 iterations)
  4. Confidence target (80/90/95/99%)
  5. Research scope (codebase only / codebase+web / web only)
  6. Search strategy (adaptive/breadth-first/depth-first)
  7. Focus areas (optional, comma-separated)
- In-session loop: reads `.aether/utils/oracle/oracle.md`, iterates through phases (survey/investigate/synthesize/verify)
- State tracked in `.aether/oracle/state.json` and `.aether/oracle/plan.json`
- Sub-questions with per-question confidence tracking
- Diminishing returns detection
- Template-specific question sets
- Promote findings to colony (instincts, learnings, observations)
- Non-invasive: only writes to `.aether/oracle/`

**v1.10 Polish (Fixed):**
- Oracle loop fix with research formulation, depth selection, and state persistence

**v1.17 Classic Restoration:**
- Phase-aware prompts, diminishing returns detection, template-specific synthesis

**Current (v1.22):**
- Go-owned `aether oracle` with `--depth`, `--template`, `--background`
- Wrapper does short AI intent refinement (3-6 questions) then delegates
- Template mapping preserved: tech-eval, architecture-review, bug-investigation, research-brief, custom, prd
- Depth presets preserved: quick, balanced, deep, exhaustive
- Confidence target preserved
- Background mode added
- Lost: rich 7-question wizard (replaced by 3-6 question scoping)
- Lost: research brief formulation with approval rounds
- Lost: interactive in-session loop visible to user (now Go-managed)

**Restoration Target:** The Go oracle loop is the correct architecture. The wrapper intent refinement is adequate but should include the research brief formulation step. Verify diminishing returns detection and promote-to-colony pipeline work end-to-end.

### 5. Build Workflow

**Classic v5.4.0 (GOOD):**
- 5-stage modular playbook (short wrapper, long playbooks):
  1. `build-prep.md` -- load state, determine visual mode, select colony depth, pheromone suggestions
  2. `build-context.md` -- context capsule, survey context, active signals, skill injection
  3. `build-wave.md` -- THE BIG ONE: analyze tasks, group by dependencies into waves, assign castes, generate ant names, spawn workers via Task tool, Oracle research step (non-blocking), Builder-Probe Lock
  4. `build-verify.md` -- post-build verification, gate evaluation (Probe, Gatekeeper, Auditor, Measurer)
  5. `build-complete.md` -- synthesis, learning extraction, activity log, next steps
- Queen spawns workers directly (not delegated to Prime Worker)
- Wave-based parallel execution with dependency ordering
- Oracle research step at depth "deep" or "full" (non-blocking)
- Caste identity: colored labels, emojis, deterministic names
- Pheromone suggestions analyzed at build start
- State checkpoint before build wave

**v1.14 Queen Authority (Added):**
- Queen decision layer: pure-function coordinator
- Queen wave lifecycle: always-advance, dependency injection, recovery

**Current (v1.22):**
- Manifest via `aether host build --dry-run`
- Wrapper spawns from manifest using platform Agent tool
- Go owns state mutation via `build-finalize`
- Ceremony via `aether ceremony spawn-plan`, `wave-start`, `worker-complete`, `closeout`
- Lost: playbook-driven execution (replaced by manifest protocol)
- Lost: Queen directly spawning workers (now wrapper interprets manifest)
- Gained: Go-owned ceremony rendering, boundary enforcement
- Gained: Builder-Probe Lock in Go runtime

**Restoration Target:** The manifest protocol is the correct architecture. The key question is whether workers receive the same quality of context (signals, skills, pheromones, survey data) that the Classic playbooks assembled. Verify prompt injection is complete.

### 6. Continue Workflow

**Classic v5.4.0 (GOOD):**
- 4-stage modular playbook:
  1. `continue-verify.md` -- load state, staleness detection, command resolution (CLAUDE.md > codebase.md > heuristic), 6-phase verification loop
  2. `continue-gates.md` -- gate classification, recovery templates, per-gate skip, Watcher Veto
  3. `continue-advance.md` -- mark phase completed, extract learnings (with hypothesis/validated/disproven status), deterministic fallback extraction, memory pipeline, instinct promotion, hive promotion
  4. `continue-finalize.md` -- changelog append, registry update, session update, wisdom summary
- "Iron Law": no phase advancement without fresh verification evidence
- Learning starts as hypothesis until verified by testing
- Deterministic fallback: git-diff-based learning extraction when AI produces none
- Memory pipeline: observation capture, auto-pheromone, auto-promotion check
- Confidence-driven depth resolution

**v1.10 Polish (Fixed):**
- Gate failure recovery with skip logic
- Smart review depth (auto/light/heavy)

**Current (v1.22):**
- Default path: `aether continue --skip-watchers --verification-depth standard` (Go-only, fast)
- Heavy path: manifest + wrapper-spawned reviewers (classic ceremony)
- Lost: 4-stage playbook execution visible to user
- Lost: explicit hypothesis/validated/disproven learning lifecycle
- Lost: deterministic fallback learning extraction
- Gained: Go-native verification, faster default path
- Gained: `--reconcile-task` for manual task reconciliation

**Restoration Target:** The fast default path is correct for daily use. The heavy path preserves classic ceremony. Critical question: does the Go runtime still extract learnings with the hypothesis lifecycle? This is the most important behavioral regression to verify.

### 7. Run (Autopilot) Workflow

**Classic v5.4.0 (GOOD):**
- Inline playbook execution: loads build-prep through build-complete, then continue-verify through continue-finalize
- Variables/results carried forward between stages
- Pause conditions:
  1. Watcher verification_failed
  2. Critical/high Chaos findings
  3. New blocker flags
  4. Verification loop NOT READY
  5. Gatekeeper critical CVEs
  6. Auditor critical findings or score < 60
  7. Unresolved blockers
  8. Runtime verification needed
  9. All phases complete
  10. Replan trigger
- Replan trigger: configurable interval (default 2 phases)
- Headless mode: queues decisions instead of pausing
- Elapsed time tracking
- Dry-run preview mode
- Max-phases cap

**v1.10 Polish:**
- Autopilot loop complete with smart pausing

**Current (v1.22):**
- Go-owned `aether run` CLI subcommand
- Wrapper is 15 lines: just delegates to CLI
- Same flags: --max-phases, --replan-interval, --continue, --dry-run, --headless, --verbose
- Lost: inline playbook loading (now Go manages the loop)
- Gained: Go-native execution, cleaner separation

**Restoration Target:** Go autopilot should have feature parity with the Classic playbook version. Verify all 10 pause conditions are implemented. Verify replan trigger fires correctly.

### 8. Swarm Workflow

**Classic v5.4.0 (GOOD):**
- Two modes: Quick View (live swarm display) and Bug Destruction
- Bug Destruction mode:
  - Validate input, read state, generate swarm ID
  - Git checkpoint before investigation (auto-fix-checkpoint)
  - Read context: blockers, activity log, recent git commits
  - Deploy 4 parallel scouts:
    1. Archaeologist (git history)
    2. Pattern Hunter (working patterns)
    3. Error Analyst (stack trace analysis)
    4. Web Researcher (external solutions)
  - Cross-compare findings, rank solutions by confidence
  - Apply best fix, verify, rollback on failure
  - 3-attempt limit before escalating to architectural concern
  - Cleanup: archive findings, inject learnings as FOCUS/REDIRECT

**Current (v1.22):**
- Go-owned `aether swarm` CLI subcommand
- Wrapper is thin passthrough
- Lost: rich 4-scout ceremony visible to user
- Gained: Go-native dispatch with same logic
- Verify: git checkpoint, rollback, 3-attempt limit, learning injection

**Restoration Target:** Verify the Go implementation preserves all 4 scout types, cross-comparison ranking, auto-fix with rollback, and 3-attempt architectural escalation.

### 9. Seal Workflow

**Classic v5.4.0 (GOOD):**
- Multi-step ceremony:
  1. Read state, maturity gate (not executing, handle incomplete phases)
  2. User confirmation
  3. Sage spawn: colony analytics review (velocity, bug density, review turnaround)
  4. Wisdom approval: batch auto-promotion + interactive review
  5. Hive promotion: high-confidence instincts to cross-colony hive (non-blocking)
  6. Chronicler spawn: documentation coverage audit
  7. Log activity, checkpoint state, increment colony version
  8. Update milestone to Crowned Anthill
  9. Update changelog
  10. Update registry (silent)
  11. Write CROWNED-ANTHILL.md from template
  12. Export XML archives (colony-archive.xml, pheromones.xml, queen-wisdom.xml, colony-registry.xml)
  13. Display ASCII art ceremony
  14. Commit suggestion (non-blocking)
  15. Push suggestion (non-blocking)
- Every spawn is non-blocking except wisdom approval

**v1.10 Polish (Fixed):**
- Hive Brain wiring: seal auto-promotes high-confidence instincts

**Current (v1.22):**
- Go-owned `aether seal` CLI
- Lost: Sage spawn (colony analytics)
- Lost: Chronicler spawn (documentation coverage audit)
- Lost: interactive wisdom approval ceremony
- Lost: commit/push suggestions
- Gained: Go-native ceremony, faster execution
- Preserved: CROWNED-ANTHILL.md, XML export, hive promotion, changelog

**Restoration Target:** Sage analytics and Chronicler audit were valuable ceremony steps that provided data-driven insights before sealing. These should be restored as optional Go subcommands or brought back as non-blocking agent spawns. Wisdom approval ceremony needs investigation.

### 10. Entomb Workflow

**Classic v5.4.0 (GOOD):**
- Seal-first enforcement (hard gate)
- User confirmation
- Wisdom approval (blocking)
- xmllint check (required for XML archiving)
- Ensure QUEEN.md exists
- Generate chamber name (date-first, collision handling)
- Create chamber, archive all data files
- Export XML archive (HARD STOP on failure -- chamber cleaned up)
- Verify chamber integrity
- Record in eternal memory
- Reset colony state (backup, reset via jq template, verify, remove backup)
- Clear session, seal document, exchange XML
- Write HANDOFF.md
- Display result, offer next steps

**Current (v1.22):**
- Go-owned `aether entomb` CLI
- Lost: interactive wisdom approval ceremony
- Lost: HANDOFF.md generation (verify)
- Preserved: seal-first gate, chamber creation, XML archive, state reset, eternal memory
- Preserved: chamber integrity verification

**Restoration Target:** Verify XML archive hard-stop behavior. Verify chamber integrity verification. Verify HANDOFF.md is written. Verify eternal memory recording.

---

## Feature Dependencies

```
Init
  └──requires──> Colony state initialized
                  └──requires──> QUEEN.md

Colonize
  └──requires──> Colony state initialized
  └──produces──> Survey documents (consumed by Plan)

Plan
  └──requires──> Colony state initialized
  └──enhances──> Survey documents (better plans with survey)
  └──produces──> Phase plan (consumed by Build)

Build
  └──requires──> Phase plan exists
  └──produces──> Completed work (consumed by Continue)

Continue
  └──requires──> Build completed
  └──produces──> Learnings, advanced state

Run
  └──requires──> Phase plan exists
  └──wraps──> Build + Continue in a loop

Oracle
  └──independent──> Can run anytime
  └──produces──> Research findings (promotable to colony)

Swarm
  └──requires──> Colony state initialized
  └──independent──> Bug investigation, not phase-bound

Seal
  └──requires──> Colony at any milestone
  └──produces──> CROWNED-ANTHILL.md, hive wisdom

Entomb
  └──requires──> Colony sealed (Crowned Anthill milestone)
  └──produces──> Chamber archive, state reset
```

---

## MVP Definition

### Launch With (Daily Driver)

Minimum for Aether to work reliably as a daily-driver development tool:

- [ ] **init** -- Creates colony with goal, sets state, initializes QUEEN.md
- [ ] **plan** -- Generates phases with real file references and confidence scoring
- [ ] **build** -- Dispatches workers, produces working code, finalizes state
- [ ] **continue** -- Verifies work, extracts learnings, advances phase
- [ ] **run** -- Chains build+continue across phases with smart pausing

### Restore After Validation

Workflows that complete the lifecycle but can be validated after core flows work:

- [ ] **colonize** -- Survey territory before planning (enhances plan quality)
- [ ] **seal** -- Crown the colony and promote wisdom
- [ ] **entomb** -- Archive completed colony

### Restore After Core Proof

Workflows that add power-user value:

- [ ] **oracle/RALF** -- Deep research loop with diminishing returns
- [ ] **swarm** -- Parallel bug investigation with 4-scout cross-comparison

---

## Feature Prioritization Matrix

| Workflow | User Value (Daily Driver) | Restoration Cost | Priority |
|----------|--------------------------|------------------|----------|
| build | HIGH -- core value prop | MEDIUM -- manifest protocol works, need context injection verification | P1 |
| continue | HIGH -- learnings + advancement | MEDIUM -- fast path works, need learning extraction verification | P1 |
| plan | HIGH -- phase generation | LOW -- Go-owned and working | P1 |
| run | HIGH -- autopilot | LOW -- Go-owned with same flags | P1 |
| init | MEDIUM -- ceremony quality | LOW -- Go-owned, intent refinement works | P2 |
| seal | MEDIUM -- lifecycle completion | MEDIUM -- ceremony simplified, need Sage/Chronicler investigation | P2 |
| entomb | MEDIUM -- cleanup | LOW -- Go-owned, verify XML hard-stop | P2 |
| colonize | MEDIUM -- plan quality | LOW -- survey works via manifest | P2 |
| oracle | HIGH (power users) | MEDIUM -- Go loop works, need diminish-returns + promote verification | P2 |
| swarm | MEDIUM (power users) | MEDIUM -- Go dispatch works, need 4-scout + rollback verification | P3 |

---

## Regression Points (Version References)

| Regression | When | Evidence | Severity |
|-----------|------|----------|----------|
| Shell-to-Go migration lost ceremony richness | v1.0-v1.5 (April 2026) | CLAUDE.md "Known losses" section lists 11 lost features | HIGH |
| Playbook-driven execution replaced by manifest protocol | v1.16-v1.18 (May 2026) | v1.16 established Go boundary, v1.17 restored ceremony via Go events | HIGH |
| Wrapper commands lost 3786 lines of behavioral spec | v5.4.0 to HEAD | `git diff` shows 3786 deletions, 818 additions | HIGH |
| Oracle wizard reduced from 7 questions to 3-6 | v1.19-v1.20 (May 2026) | Current oracle.md is 85 lines vs v5.4.0's 700+ lines | MEDIUM |
| Seal lost Sage/Chronicler spawns | v1.19-v1.20 (May 2026) | Current seal.md delegates to Go; no agent spawns | MEDIUM |
| Continue lost hypothesis/validated learning lifecycle | v1.16-v1.18 (May 2026) | Classic had explicit hypothesis/validated/disproven status tracking | HIGH |
| Entomb lost interactive wisdom approval | v1.19-v1.20 (May 2026) | Classic had blocking wisdom approval; current is Go-owned | LOW |

## What "Good" Looked Like

The best version of Aether (v5.4.0 Classic + v1.10 Polish) had these qualitative properties:

1. **Rich ceremony at every step.** Every command had visual headers, progress bars, caste identity, and clear next steps. Users felt the colony was "alive."

2. **Playbook-driven execution.** Build and continue were 5-stage and 4-stage playbooks respectively. Each stage was a readable markdown file that anyone could inspect, modify, or debug. The wrapper was thin; the playbooks were the brain.

3. **Learning was first-class.** Continue extracted learnings as hypotheses, tracked evidence, promoted to instincts, and piped through memory. The hypothesis/validated/disproven lifecycle ensured only tested knowledge became wisdom.

4. **Oracle was a deep research partner.** 7-question wizard, 5 template types, diminishing returns, phase-aware prompts, and promote-to-colony made Oracle genuinely useful for domain research.

5. **Seal was a ceremony, not a checkpoint.** Sage analytics, Chronicler audit, wisdom approval, hive promotion, XML export, commit suggestion -- seal was a 15-step ritual that made colony completion feel meaningful.

6. **Swarm was nuclear bug destruction.** 4 parallel scouts with cross-comparison ranking, auto-fix with rollback, 3-attempt architectural escalation. Swarm felt like deploying an army.

7. **State was honest and inspectable.** Watch files, activity logs, spawn trees, timing data -- everything was visible. Users could see what the colony was doing in real time.

## What to Keep From Modern Architecture

The Go runtime migration was correct. These changes should NOT be reverted:

1. **Go owns state mutation.** The Frankenstein state corruption bug proved LLMs cannot safely reconstruct JSON. Go finalizers are the right pattern.

2. **Manifest protocol for worker dispatch.** The wrapper-spawns-from-manifest pattern provides clean boundary enforcement and cross-platform consistency.

3. **Go ceremony rendering.** ANSI-colored banners, caste identity, stage markers -- Go renders these deterministically.

4. **Golden workflow tests.** v1.17-v1.18 established automated behavioral tests. These are the correctness anchor.

5. **Loop safety.** v1.12 added 6 loop-breaking requirements. These prevent the worst failure modes.

6. **Depth controls.** v1.12 added independent planning/verification depth with smart defaults. These give users appropriate control.

## What Needs Restoration

1. **Learning extraction in continue.** Verify Go runtime implements hypothesis/validated/disproven lifecycle. If not, this is the single highest-priority behavioral regression.

2. **Worker context quality.** Verify workers receive: active pheromone signals, skill injection, survey context, colony goal, phase description, success criteria. Classic playbooks assembled this manually; the manifest protocol must carry it.

3. **Oracle promote-to-colony pipeline.** Verify findings can flow: oracle research > instincts > learnings > QUEEN.md > hive brain.

4. **Sage analytics at seal.** Restore as optional Go subcommand or non-blocking ceremony step.

5. **Swarm 4-scout cross-comparison.** Verify Go implementation preserves all scout types and confidence-based ranking.

6. **Autopilot pause conditions.** Verify all 10 Classic pause conditions are implemented in Go `aether run`.

---

## Sources

- v5.4.0 git tag source code (direct inspection of .claude/commands/ant/*.md)
- v1.10-MILESTONE-AUDIT.md -- 35/35 requirements, 6/6 E2E flows
- v1.12-REQUIREMENTS.md -- loop safety + depth controls
- v1.16-MILESTONE-AUDIT.md -- Classic baseline selection, boundary contract
- v1.17-ROADMAP.md -- Classic restoration phases
- CLAUDE.md "Known losses from shell-to-Go migration"
- cmd/testdata/golden_*.txt -- behavioral snapshots
- .claude/commands/ant/*.md (current) -- comparison baseline
- PROJECT.md v1.23 milestone requirements -- active requirements list
- Confidence: HIGH -- all findings from direct source code inspection
