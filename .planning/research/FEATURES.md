# Feature Research: Colony Context Enhancement

**Domain:** AI-assisted development with session continuity
**Researched:** 2026-02-21
**Confidence:** HIGH (based on existing system analysis + domain expertise)

---

## Feature Landscape

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| **Goal Preservation** | Core of why colony exists | LOW | Already in COLONY_STATE.json, session.json |
| **Current Phase/Position** | "Where am I?" is fundamental | LOW | session.json has current_phase, current_milestone |
| **Recent Activity Log** | What just happened | LOW | activity.log exists, needs summarization |
| **Active Constraints** | What to avoid (REDIRECTs) | LOW | pheromones.json already captures these |
| **Next Action Clarity** | What to do now | MEDIUM | session.json has suggested_next, needs enhancement |

**Assessment:** Aether already has 4/5 table stakes. The gap is "Next Action Clarity" — session.json has `suggested_next: "verify"` but lacks context about *why* verify, *what* to verify, and *how*.

---

### Differentiators (Competitive Advantage)

Features that set Aether apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| **Decision Archaeology** | Show not just *what* was decided but *why* — the rationale chain | MEDIUM | QUEEN.md has evolution log pattern, extend to all decisions |
| **Pheromone Persistence** | User signals survive session breaks with TTL awareness | LOW | Already implemented — signals have expires_at |
| **Wisdom Inheritance** | Cross-colony learning via QUEEN.md promotion pipeline | MEDIUM | learning-observe exists, promotion thresholds configured |
| **Ant-Themed Naming** | NEST.md, TRAILS/, BROOD/ vs generic "context", "history" | LOW | Brand differentiation, memorable mental model |
| **Phase-Aware Resumption** | Resume knows if you're mid-phase, between phases, or ad-hoc | MEDIUM | Requires tracking phase boundaries in session.json |
| **Colony Priming (colony-prime)** | Unified context injection for workers | LOW | Already implemented — combines QUEEN.md + pheromones + instincts |

**Assessment:** Aether already has significant differentiators implemented. The opportunity is in "Decision Archaeology" — surfacing the *why* behind decisions, not just the *what*.

---

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|-----------------|-------------|
| **Full Session Replay** | "I want to see everything that happened" | Overwhelming, low signal-to-noise | Summarized activity with drill-down to activity.log |
| **Auto-Resume on Session Start** | "Just continue where I left off" | Violates user autonomy, may be unwanted | Detect stale session, offer `/ant:resume` explicitly |
| **Infinite Decision History** | "Keep every decision forever" | Performance degradation, context window bloat | Keep last N decisions + promoted wisdom in QUEEN.md |
| **Real-Time Sync Across Sessions** | "Multiple Claude windows" | Complexity, conflict resolution nightmares | Single colony per repo, explicit handoff between sessions |
| **Full File Content Snapshots** | "Restore exact file state" | Git already does this, duplication | Reference git commits, track modified files list |
| **Nested Colony Sessions** | "Colony within a colony" | Cognitive overhead, state management hell | One active colony per repo, archive to chambers |

**Assessment:** Aether's current design avoids these anti-features well. The session freshness detection (already implemented) prevents auto-resume. The chamber system (archive) prevents infinite history growth.

---

## Feature Dependencies

```
NEST.md (Session Restoration Document)
    └──requires──> session.json (session tracking)
    └──requires──> COLONY_STATE.json (colony goal)
    └──requires──> QUEEN.md (wisdom context)
    └──enhances──> CONTEXT.md (current context summary)

TRAILS/ (Decision History)
    └──requires──> learning-observations.json (observation tracking)
    └──requires──> pheromones.json (signal history)
    └──requires──> activity.log (action log)
    └──enhances──> QUEEN.md (wisdom promotion)

BROOD/ (Phase Documentation)
    └──requires──> COLONY_STATE.json (phase tracking)
    └──requires──> session.json (position tracking)
    └──enhances──> .planning/phases/ (existing phase docs)

ROYAL-CHAMBER/ (Decision Archive)
    └──requires──> chambers/ (existing archive system)
    └──requires──> TRAILS/ (decision history)
    └──enhances──> QUEEN.md (promoted wisdom)

/ant:resume Command
    └──requires──> NEST.md (restoration doc)
    └──requires──> Session Freshness Detection (already implemented)
    └──conflicts──> Auto-resume (explicit is better)
```

### Dependency Notes

- **NEST.md requires session.json:** Session freshness detection already implemented, provides baseline for restoration
- **TRAILS/ requires learning-observations.json:** learning-observe command exists, observation tracking functional
- **BROOD/ requires COLONY_STATE.json:** Phase tracking exists but sparse — needs enhancement for position awareness
- **ROYAL-CHAMBER/ enhances QUEEN.md:** Wisdom promotion pipeline exists (0.8 threshold), needs decision archaeology integration

---

## Categorized by Context Type

### 1. Project Context (What are we building?)

| Feature | Status | Location | Enhancement Needed |
|---------|--------|----------|-------------------|
| Colony Goal | EXISTS | COLONY_STATE.json `goal` | Add to NEST.md header |
| Milestone | EXISTS | session.json `current_milestone` | Add milestone description |
| Phase Overview | EXISTS | .planning/phases/ | Link from NEST.md |
| Success Criteria | PARTIAL | Phase docs | Summarize in NEST.md |

### 2. Current Position (Where am I?)

| Feature | Status | Location | Enhancement Needed |
|---------|--------|----------|-------------------|
| Current Phase | EXISTS | session.json `current_phase` | Add phase name + description |
| Current Task | MISSING | — | Track active task in session.json |
| Files Modified | PARTIAL | git status | List in NEST.md |
| Blockers | MISSING | — | Add blockers section to session.json |

### 3. Decision Logging (Why did we choose X?)

| Feature | Status | Location | Enhancement Needed |
|---------|--------|----------|-------------------|
| Recent Decisions | PARTIAL | CONTEXT.md | Formalize in TRAILS/decisions.json |
| Decision Rationale | MISSING | — | Add rationale field to decision records |
| Alternative Considered | MISSING | — | Add alternatives field |
| Decision Maker | EXISTS | CONTEXT.md `Made By` | Preserve in structured format |

### 4. Phase Documentation (What happened in each phase?)

| Feature | Status | Location | Enhancement Needed |
|---------|--------|----------|-------------------|
| Phase Summaries | EXISTS | .planning/phases/*/NN-XX-SUMMARY.md | Link from BROOD/ |
| Phase Learnings | EXISTS | learning-observations.json | Promote to QUEEN.md |
| Phase Errors | EXISTS | COLONY_STATE.json `errors` | Summarize in BROOD/ |
| Phase Completion | EXISTS | Phase SUMMARY.md | Track in BROOD/completed.json |

### 5. Session Continuity (How do I resume?)

| Feature | Status | Location | Enhancement Needed |
|---------|--------|----------|-------------------|
| Session Detection | EXISTS | session freshness check | Already implemented |
| Resume Command | EXISTS | `/ant:resume` | Enhance to read NEST.md |
| Session Timestamp | EXISTS | session.json `started_at` | Calculate "away time" |
| Next Action | PARTIAL | session.json `suggested_next` | Add context + rationale |

---

## MVP Definition

### Launch With (v1)

Minimum viable product — what's needed to validate the concept.

- [ ] **NEST.md** — Single file containing: goal, position, last 5 decisions, next action with context
- [ ] **Enhanced `/ant:resume`** — Reads NEST.md, presents restoration summary, confirms with user
- [ ] **Decision Logging** — Extend existing CONTEXT.md pattern with structured rationale field
- [ ] **Session Position Awareness** — Track "mid-phase" vs "between-phases" in session.json

**Why these are essential:**
- NEST.md = the 2-3 file promise — one file gives full context
- Enhanced resume = the activation mechanism
- Decision logging = the "why" that differentiates from generic session restore
- Position awareness = prevents "what was I in the middle of?" confusion

### Add After Validation (v1.x)

Features to add once core is working.

- [ ] **TRAILS/decisions.json** — Structured decision history (last 20 with full rationale)
- [ ] **BROOD/phase-index.json** — Quick-reference phase completion status
- [ ] **Auto-NEST-update** — Update NEST.md on significant events (phase complete, decision made)
- [ ] **Away-time calculation** — "You've been gone 3 days, here's what changed"

**Trigger for adding:**
- TRAILS/ when users ask "why did we decide that?"
- BROOD/ when users lose track of phase progress
- Auto-update when NEST.md becomes stale
- Away-time when users return after long breaks

### Future Consideration (v2+)

Features to defer until product-market fit is established.

- [ ] **ROYAL-CHAMBER/ cross-colony search** — Search decisions across all archived colonies
- [ ] **Decision confidence scoring** — Track which decisions were "tentative" vs "firm"
- [ ] **Visual timeline** — ASCII or markdown timeline of colony lifecycle
- [ ] **Integration with external docs** — Link to Notion, Confluence, etc.

**Why defer:**
- Cross-colony search requires multiple archived colonies to be valuable
- Confidence scoring adds UI complexity without clear user demand
- Visual timeline is nice-to-have, not need-to-have
- External integration requires auth, APIs, maintenance burden

---

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| NEST.md | HIGH | LOW | P1 |
| Enhanced `/ant:resume` | HIGH | LOW | P1 |
| Decision rationale field | HIGH | LOW | P1 |
| Session position awareness | MEDIUM | LOW | P1 |
| TRAILS/decisions.json | MEDIUM | MEDIUM | P2 |
| BROOD/phase-index.json | MEDIUM | LOW | P2 |
| Auto-NEST-update | MEDIUM | MEDIUM | P2 |
| Away-time calculation | LOW | LOW | P3 |
| ROYAL-CHAMBER/ search | LOW | HIGH | P3 |
| Decision confidence | LOW | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch — core promise of instant session restoration
- P2: Should have, add when possible — enhance the core experience
- P3: Nice to have, future consideration — polish and advanced features

---

## Ant-Themed Naming Suggestions

| Concept | Generic Name | Ant-Themed Name | Rationale |
|---------|--------------|-----------------|-----------|
| Session restoration doc | CONTEXT.md | **NEST.md** | The colony's home, contains everything needed to resume |
| Decision history | decisions/ | **TRAILS/** | Pheromone trails left by decisions |
| Phase documentation | phases/ | **BROOD/** | Where colony's work is nurtured |
| Decision archive | archive/ | **ROYAL-CHAMBER/** | Where queen's wisdom is preserved |
| Session snapshot | checkpoint | **CHAMBER/** | Already implemented, chamber system |
| Resume command | resume | **/ant:resume** | Already implemented |
| Context injection | prime | **colony-prime** | Already implemented |
| Observation tracking | telemetry | **learning-observe** | Already implemented |

**Naming principles:**
1. Use ant colony metaphors consistently
2. Prefer concrete nouns (NEST, TRAIL, BROOD) over abstract (CONTEXT, HISTORY)
3. Align with existing pheromone metaphor (signals = chemical trails)
4. Keep it pronounceable — "check the NEST" not "check the .aether/data/session-restoration-document"

---

## Competitor Feature Analysis

| Feature | Claude Projects | Cursor | Aether Approach |
|---------|-----------------|--------|-----------------|
| Session continuity | Manual (user re-describes) | Limited (file-based) | **Structured NEST.md with auto-detection** |
| Decision history | None | None | **TRAILS/ with rationale** |
| Cross-session learning | None | None | **QUEEN.md wisdom promotion** |
| Position awareness | None | None | **Phase-aware with mid-phase detection** |
| User signals | None | Rules files | **Pheromone system (FOCUS/REDIRECT/FEEDBACK)** |
| Resume mechanism | Manual | Manual | **Explicit `/ant:resume` with freshness check** |

**Key differentiators:**
1. **Structured restoration** — Not just "here are the files" but "here's what we were doing and why"
2. **Decision archaeology** — The *why* behind choices, not just the *what*
3. **Cross-colony learning** — Wisdom persists across projects via QUEEN.md
4. **Explicit resume** — User controls when to restore, not auto-resume surprises

---

## Backwards Compatibility Assessment

| Existing Feature | Compatibility Risk | Mitigation |
|------------------|-------------------|------------|
| COLONY_STATE.json | NONE | Read-only, NEST.md supplements |
| session.json | LOW | Add new fields, don't remove old |
| CONTEXT.md | MEDIUM | NEST.md replaces, keep CONTEXT.md as human-readable |
| QUEEN.md | NONE | NEST.md references, doesn't replace |
| pheromones.json | NONE | NEST.md summarizes, doesn't replace |
| learning-observations.json | NONE | TRAILS/ will reference, not replace |
| `/ant:resume` | LOW | Enhance existing, don't break |

**Compatibility strategy:**
- NEST.md is additive — doesn't replace existing files
- New fields in session.json are optional — old code ignores them
- CONTEXT.md becomes "human-readable summary" — NEST.md is "machine-readable restoration"
- All existing commands continue working — new features are opt-in

---

## Sources

- Existing Aether codebase analysis:
  - `/Users/callumcowie/repos/Aether/.aether/data/COLONY_STATE.json` — colony state structure
  - `/Users/callumcowie/repos/Aether/.aether/data/session.json` — session tracking
  - `/Users/callumcowie/repos/Aether/.aether/CONTEXT.md` — current context document
  - `/Users/callumcowie/repos/Aether/.aether/docs/QUEEN.md` — wisdom system
  - `/Users/callumcowie/repos/Aether/.aether/docs/pheromones.md` — signal system
  - `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` — colony-prime, learning-observe implementations
  - `/Users/callumcowie/repos/Aether/.aether/templates/handoff.template.md` — existing archive pattern

- Domain expertise in session management and context restoration patterns
- Analysis of existing Aether milestone/chamber system

---

*Feature research for: Colony Context Enhancement*
*Researched: 2026-02-21*
