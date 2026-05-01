# Phase 90: Learning Foundation - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-01
**Phase:** 90-learning-foundation
**Areas discussed:** Learning trigger points, Memory store architecture, Evidence and classification, Learned context injection

---

## Learning Trigger Points

### Q1: When should learning capture fire?

| Option | Description | Selected |
|--------|-------------|----------|
| After continue verification passes | Learning fires only after /ant-continue verifies gates pass AND provenance is valid | ✓ |
| After build AND after continue | Two learning events per phase, different evidence | |
| After build-complete only | Simpler but means failed gates don't block learning | |

**User's choice:** After continue verification passes
**Notes:** Build can succeed but gates can fail. Only after continue verifies both provenance and gates is learning valid.

### Q2: What about partial worker success?

| Option | Description | Selected |
|--------|-------------|----------|
| All workers must succeed | Any failure blocks the whole phase from durable memory | ✓ |
| Per-worker success filtering | Successful workers contribute, failed excluded | |
| All workers + all gates must pass | Most conservative | |

**User's choice:** All workers must succeed
**Notes:** Strictest interpretation of LRN-01. Any failure blocks the whole phase from durable memory.

### Q3: What other colony activities should produce durable learning?

| Option | Description | Selected |
|--------|-------------|----------|
| Build+continue only | Simplest model, other commands stay transient | ✓ |
| Build+continue + Oracle with approval | Oracle can optionally produce memory | |
| All commands with evidence | Maximum learning surface | |

**User's choice:** Build+continue only
**Notes:** Oracle, chaos, archaeology, dreams all stay transient.

### Q4: How should the system determine verified outcomes?

| Option | Description | Selected |
|--------|-------------|----------|
| Continue gates + provenance = verified | Existing SAFE-03/04 is sufficient | ✓ |
| Dedicated evidence verification step | Additional checks beyond provenance | |
| Gate results JSON as evidence | Leverages Phase 88 gate-results directly | |

**User's choice:** Continue gates + provenance = verified
**Notes:** Existing provenance + gate pass status IS the verification. No additional layer needed.

---

## Memory Store Architecture

### Q1: How should the unified memory API relate to existing infrastructure?

| Option | Description | Selected |
|--------|-------------|----------|
| New pkg/learn/ with unified API | Clean separation, pkg/memory becomes internal | ✓ |
| Extend pkg/memory in place | Less disruption but package gets bigger | |
| New orchestration layer over pkg/memory | Lightweight wrapper | |

**User's choice:** New pkg/learn/ with unified API
**Notes:** Clean separation. pkg/memory becomes internal implementation detail.

### Q2: Where should repo-local colony memory live?

| Option | Description | Selected |
|--------|-------------|----------|
| .aether/data/ with existing JSON patterns | Same storage layer | |
| .aether/data/learn/ subdirectory | Physically separate from colony state | ✓ |
| Existing learning files, new API | No new storage location | |

**User's choice:** .aether/data/learn/ subdirectory
**Notes:** Physically separate from colony state. Clean deletion/export.

### Q3: What's the relationship between pkg/learn/ and pkg/memory?

| Option | Description | Selected |
|--------|-------------|----------|
| pkg/memory becomes internal to pkg/learn/ | Cleanest long-term, requires call site updates | ✓ |
| Both coexist, pkg/learn/ calls pkg/memory | Zero disruption but some duplication | |
| Shared interface package | DRY but more indirection | |

**User's choice:** pkg/memory becomes internal to pkg/learn/
**Notes:** Most disruptive but cleanest long-term. Existing call sites need updating.

### Q4: How does colony memory relate to hive memory?

| Option | Description | Selected |
|--------|-------------|----------|
| Unified API with scope parameter | Clean but couples them | |
| Separate APIs, explicit promotion step | Decoupled but some duplication | |
| Interface with two implementations | Testable, decoupled, Phase 91 can swap SQLite | ✓ |

**User's choice:** You decide
**Notes:** Claude chose Interface with two implementations (MemoryStore, ColonyStore, HiveStore).

---

## Evidence and Classification

### Q1: What should evidence look like structurally?

| Option | Description | Selected |
|--------|-------------|----------|
| Full structured evidence (all fields) | Maximum traceability, all LRN-02 fields | ✓ |
| Required + optional fields | Lighter entries possible | |
| References to existing data | DRY but depends on file longevity | |

**User's choice:** Full structured evidence (all fields)
**Notes:** Every entry includes run ID, phase, workers, files, gates, confidence, timestamp, scope.

### Q2: When does classification happen?

| Option | Description | Selected |
|--------|-------------|----------|
| Default repo-local, classify at promotion | Classification deferred | |
| Automatic classification at creation | No user involvement initially | ✓ |
| User reviews all classifications | Most control, highest friction | |

**User's choice:** Automatic classification at creation
**Notes:** No user involvement for initial classification.

### Q3: Classification rules?

| Option | Description | Selected |
|--------|-------------|----------|
| Path/pattern heuristics | Simple detection | |
| Extend existing privacy scanner | Reuses Phase 88 infrastructure | ✓ |
| Heuristics + privacy scanner two-pass | Belt and suspenders | |

**User's choice:** You decide
**Notes:** Claude chose extend existing privacy scanner. Scanner runs first, classification layer on top.

### Q4: Export/import user flow?

| Option | Description | Selected |
|--------|-------------|----------|
| JSON manifest + preview-before-apply | Full audit trail | ✓ |
| Single bundled JSON file | Simpler format | |
| Defer export/import to later phase | Focus on core lifecycle first | |

**User's choice:** JSON manifest + preview-before-apply
**Notes:** aether learn export/import with preview step.

---

## Learned Context Injection

### Q1: How should learned memory integrate with existing context ranking?

| Option | Description | Selected |
|--------|-------------|----------|
| Feed into existing context_ranking.go | Shares token budget, least disruptive | ✓ |
| Separate dedicated budget | Simpler but another budget to manage | |
| Replace existing instinct injection | Most unified, biggest migration | |

**User's choice:** You decide
**Notes:** Claude chose feed into existing context_ranking.go. Learning entries become ContextCandidates.

### Q2: How do workers receive learned context?

| Option | Description | Selected |
|--------|-------------|----------|
| Via colony-prime context assembly | Frozen snapshot pattern | ✓ |
| Workers fetch directly | More targeted | |
| Snapshot + on-demand fetch | Hybrid | |

**User's choice:** Via colony-prime context assembly
**Notes:** Workers don't call pkg/learn/ directly. Receive as part of context section.

### Q3: Default learning behavior?

| Option | Description | Selected |
|--------|-------------|----------|
| Default enabled, config + flag to disable | PRIV-05 compliant | ✓ |
| Default disabled, opt-in | More cautious | |
| Enabled for hive-shareable only by default | Balanced | |

**User's choice:** Default enabled, config + flag to disable
**Notes:** Learning on by default. --no-learn flag and config to disable.

### Q4: Frozen snapshot or dynamic refresh?

| Option | Description | Selected |
|--------|-------------|----------|
| Frozen at colony-prime assembly | Simplest, most predictable | |
| Snapshot with between-wave refresh | More dynamic, later waves see updates | ✓ |

**User's choice:** Snapshot with between-wave refresh
**Notes:** Frozen at assembly, refreshed between waves in parallel execution.

---

## Claude's Discretion

Three decisions were deferred to Claude:
1. **Hive relationship** (D-07): Chose MemoryStore interface with ColonyStore + HiveStore implementations. Rationale: decoupled, testable, Phase 91 can swap SQLite into ColonyStore without touching HiveStore.
2. **Classification rules** (D-11): Chose extend existing privacy scanner. Rationale: reuses Phase 88 infrastructure, one scanner + one classification layer, aligns with established pattern of building on existing code.
3. **Context injection** (D-13): Chose feed into existing context_ranking.go. Rationale: least disruptive, shares token budget, learning entries map naturally to ContextCandidate struct.

## Deferred Ideas

None — discussion stayed within phase scope.
