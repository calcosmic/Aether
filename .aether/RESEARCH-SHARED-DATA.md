# Aether Shared Data Layer — Multi-Agent Research Synthesis

**Status:** Research Complete
**Date:** 2026-02-15
**Objective:** Design a hub-level shared data tier (`~/.aether/shared/`) that distributes cross-repo knowledge while maintaining local colony isolation.

---

## 🎯 Executive Summary

Three scout ants (Alpha, Beta, Gamma) conducted parallel research on the shared data layer proposal. This document synthesizes their findings into a unified architecture with implementation roadmap.

**Key Finding:** The existing Aether architecture already anticipates this need through:
- The pheromone system (eternal vs ephemeral trails)
- Chamber archives with completion reports
- Instinct lifecycle (proposed → validated → core)
- Nestmate discovery in `bin/lib/nestmate-loader.js`

**Recommendation:** Implement a `~/.aether/shared/` tier with 5 categories, distributed via existing `syncDirWithCleanup()` mechanism, with read-only repos and hub-validated writes.

---

## 📊 Synthesized Data Taxonomy

### Shared Data Categories (Distributed to All Repos)

| Category | Lifetime | Sync Strategy | Source of Truth | Privacy Level |
|----------|----------|---------------|-----------------|---------------|
| **instincts/** | Eternal | Bidirectional with validation | Hub (promoted from colonies) | Low (abstract patterns) |
| **patterns/** | 90 days | Hub→repo only, colonies suggest | Hub | Low (code patterns) |
| **chambers/** | Eternal | Hub→repo read-only mirror | Hub (colonies publish) | Medium (project metadata) |
| **telemetry/** | 30 days | Aggregated anonymized | Hub | Low (metrics only) |
| **nestmates/** | Session | Lazy federation (on-demand) | Distributed (each colony) | Low (project names only) |

### Local-Only Data (Never Distributed)

| Category | Reason for Isolation |
|----------|---------------------|
| **active colony state** | Project-specific, ephemeral |
| **current phase progress** | Work-in-progress, not shareable |
| **active pheromones** | Context-dependent steering |
| **session events** | Timeline-specific activity |
| **project constraints** | May contain sensitive business logic |
| **view state** | UI preferences, user-specific |
| **checkpoints** | Repo-state specific, transient |

---

## 🏗️ Unified Architecture

### Directory Structure

```
~/.aether/ (THE HUB)
├── system/              # System files (runtime/) - existing
├── commands/            # Slash commands - existing
├── agents/              # Agent definitions - existing
├── visualizations/      # ASCII art - existing
├── data/                # Hub's private state - existing
│   └── registry.json    # Repo registrations
│
└── shared/              # NEW: Distributed shared data
    ├── instincts/       # Validated cross-repo instincts
    │   ├── instincts.json
    │   └── index/by-domain/
    ├── patterns/        # Reusable code patterns
    │   ├── patterns.json
    │   └── signatures/
    ├── chambers/        # Archive of completed colonies
    │   ├── index.json   # Chamber metadata index
    │   └── {chamber-id}/  # Full chamber data
    ├── telemetry/       # Aggregated performance data
    │   └── model-rankings.json
    └── nestmates/       # Cross-project awareness
        └── registry.json
```

### Repo Structure (After Update)

```
any-repo/
└── .aether/
    ├── data/                    # LOCAL: Colony state (untouched)
    │   ├── COLONY_STATE.json
    │   ├── constraints.json
    │   └── pheromones.json
    │
    ├── shared/                  # NEW: Read-only mirror of hub/shared/
    │   ├── instincts/           # Sym-link or copy from hub
    │   ├── patterns/
    │   ├── chambers/
    │   └── telemetry/
    │
    └── (system files)           # From hub/system/
```

### Sync Mechanism

**Hub→Repo (Distribution via `/ant:update`):**
```javascript
// In bin/cli.js, extend existing syncDirWithCleanup()
const SHARED_FILES = [
  'shared/instincts/*.json',
  'shared/patterns/*.json',
  'shared/chambers/index.json',
  'shared/telemetry/aggregated.json'
];

// Sync with same hash-based comparison as system files
syncDirWithCleanup(HUB_SHARED, REPO_SHARED, SHARED_FILES);
```

**Repo→Hub (Contribution via validation):**
```bash
# Colonies propose instincts via completion
/ant:seal → Extracts validated instincts → Proposes to hub

# Hub validates before accepting
- Confidence >= 0.9 required
- 5+ applications required
- Manual review for private-repo content
- Schema validation
```

### Conflict Resolution

| Scenario | Resolution Strategy |
|----------|-------------------|
| Hub ahead of repo | Standard update: overwrite local shared/ |
| Repo modified shared/ | Stash to `.aether/shared-diverged/`, notify user |
| Conflicting instincts | Hub wins; conflicting local instinct marked "diverged" |
| Schema version mismatch | Migration function upgrades on read |

---

## 📋 Schema Definitions

### Instinct Schema

```json
{
  "schema_version": "1.0",
  "instincts": [
    {
      "id": "instinct_{uuid}",
      "trigger": "pattern or condition",
      "action": "recommended action",
      "confidence": 0.95,
      "status": "validated",
      "domain": "workflow|code|testing|architecture",
      "source": {
        "type": "colony|manual",
        "repo_hash": "sha256_of_path",
        "chamber_id": "chamber_{uuid}"
      },
      "evidence": ["source references"],
      "metrics": {
        "created_at": "2026-02-15T10:00:00Z",
        "last_applied": "2026-02-15T14:30:00Z",
        "applications": 12,
        "successes": 11,
        "failures": 1
      },
      "content_hash": "sha256_of_content"
    }
  ]
}
```

### Chamber Index Schema

```json
{
  "schema_version": "1.0",
  "chambers": [
    {
      "id": "chamber_{uuid}",
      "repo_name": "project name (hashed)",
      "repo_path_hash": "sha256_of_full_path",
      "goal": "chamber goal",
      "milestone": "Crowned Anthill",
      "phases_completed": 6,
      "total_phases": 6,
      "version": "v3.1.7",
      "created_at": "2026-02-13T20:40:00Z",
      "sealed_at": "2026-02-14T02:39:00Z",
      "stats": {
        "decisions_count": 24,
        "learnings_count": 18,
        "instincts_contributed": 3
      },
      "key_decisions": [
        {"id": "d1", "summary": "Use sync-with-cleanup pattern"}
      ],
      "tags": ["nodejs", "cli", "distribution"],
      "location": "~/.aether/shared/chambers/{id}/"
    }
  ],
  "indexes": {
    "by_tag": {"nodejs": ["chamber_001", "chamber_003"]},
    "by_milestone": {"Crowned Anthill": ["chamber_001"]},
    "by_tech": {"npm": ["chamber_001"]}
  }
}
```

### Pattern Schema

```json
{
  "schema_version": "1.0",
  "patterns": [
    {
      "id": "pattern_{uuid}",
      "name": "sync-with-cleanup",
      "description": "Two-phase file sync with orphan removal",
      "type": "implementation",
      "applies_to": ["bash", "nodejs"],
      "signature": {
        "type": "regex|exact|ast",
        "pattern": "syncDirWithCleanup|copy.*then.*remove"
      },
      "context": {
        "before": ["backup", "checkpoint"],
        "after": ["verify", "commit"]
      },
      "confidence": 0.92,
      "source_chambers": ["chamber_001"],
      "expires_at": "2026-05-15T00:00:00Z"
    }
  ]
}
```

---

## 🚀 Implementation Phases

### Phase 1: Foundation (1 week)

**Deliverables:**
- [ ] Create `~/.aether/shared/` directory structure
- [ ] Implement `shared-sync` command in `aether-utils.sh`
- [ ] Add shared data to `package.json` files array
- [ ] Update `/ant:update` to sync shared tier
- [ ] Create JSON schemas for validation

**Files Modified:**
- `bin/cli.js` - Add shared sync
- `runtime/aether-utils.sh` - Add shared-* commands
- `package.json` - Include shared/ in distribution

### Phase 2: Instinct Harvesting (1 week)

**Deliverables:**
- [ ] Extend `/ant:seal` to extract validated instincts
- [ ] Create instinct validation pipeline
- [ ] Build `aether shared publish` command
- [ ] Implement content-addressable storage (SHA hashing)

**Files Modified:**
- `.claude/commands/ant/seal.md` - Add instinct extraction
- `runtime/utils/shared-data.sh` - New utility

### Phase 3: Chamber Archive (1 week)

**Deliverables:**
- [ ] Extend `/ant:entomb` to publish chamber summary
- [ ] Create chamber index with searchable metadata
- [ ] Implement `/ant:tunnels --global` to browse all chambers
- [ ] Add tag indexing

**Files Modified:**
- `.claude/commands/ant/entomb.md` - Add chamber publishing
- `.claude/commands/ant/tunnels.md` - Add global search

### Phase 4: Query Interface (1 week)

**Deliverables:**
- [ ] Create `aether query` CLI command
- [ ] Implement instinct lookup by domain/confidence
- [ ] Add pattern matching for current project
- [ ] Build chamber search by tags/tech

**New Commands:**
- `aether query instincts --domain workflow --min-confidence 0.9`
- `aether query patterns --tech nodejs --context "file-sync"`
- `aether query chambers --tag "authentication"`

### Phase 5: Automatic Inheritance (1 week)

**Deliverables:**
- [ ] Extend `/ant:init` to inherit from similar colonies
- [n] Implement CHC fingerprinting (repo similarity detection)
- [ ] Add automatic instinct injection as FEEDBACK pheromones
- [ ] Create `~/.aether/shared/queen-will.md` for global directives

**Files Modified:**
- `bin/lib/init.js` - Add shared data inheritance
- `.claude/commands/ant/init.md` - Inject relevant instincts

### Phase 6: Telemetry Aggregation (Optional, 1 week)

**Deliverables:**
- [ ] Anonymize and aggregate model performance data
- [ ] Build model ranking by caste/task type
- [ ] Optimize worker routing based on collective data

---

## 🔒 Privacy & Security

### Data Protection Measures

| Risk | Mitigation |
|------|-----------|
| Code leakage | Only patterns (regex/abstract) shared, never file contents |
| Project identification | Repo paths hashed (SHA-256), names optional |
| Business logic exposure | Constraints/project-specific signals stay local |
| Vulnerability disclosure | Error signatures abstracted, severity-limited sharing |
| Size bloat | 10MB cap per category, TTL eviction, compression |

### Validation Gates

Before data enters shared tier:
1. **Schema validation** - Must match shared schema
2. **Confidence threshold** - Instincts >= 0.9, patterns >= 0.8
3. **Content scanning** - No secrets, API keys, or PII
4. **Origin verification** - Crowned Anthill colonies only
5. **Manual review option** - Flag for review if uncertain

---

## 🐜 Biological Analogies Applied

| Real Ant Behavior | Aether Implementation |
|-------------------|----------------------|
| **Pheromone evaporation** | TTL on patterns (90 days default) |
| **Tandem running** | `/ant:init` inherits from parent colony |
| **Trophallaxis** | Cross-colony instinct sharing via hub |
| **Cuticular hydrocarbons** | CHC fingerprinting for repo similarity |
| **Budding** | Colony lineage carries context forward |
| **Cemetery aggregation** | Independently-discovered patterns promoted |
| **Alarm pheromones** | Critical error signatures (high priority) |
| **Trail consolidation** | Pattern merging and deduplication |

---

## 📚 Precedents & Patterns

### From Other Systems

| System | Pattern Applied |
|--------|----------------|
| **npm/cargo** | Local-first with global cache |
| **Git** | Content-addressable immutable objects |
| **CRDTs** | Conflict-free merging for instinct updates |
| **IntelliJ Indexes** | Pre-computed shared knowledge |
| **Federated Learning** | Privacy-preserving collective intelligence |
| **Gossip Protocols** | Organic pheromone propagation (Phase 2+) |

---

## ⚠️ Open Questions

1. **Should we implement gossip protocol for pheromone propagation, or stick to hub-centric?**
   - Gossip: More scalable, works offline
   - Hub: Simpler, easier to control

2. **How do we handle conflicting instincts from different colonies?**
   - Option A: Hub curation (manual review)
   - Option B: Confidence-weighted automatic resolution
   - Option C: Context-aware (both exist, colony chooses)

3. **What's the retention policy for chamber archives?**
   - Keep forever (storage grows)
   - Keep last N per user (configurable)
   - Keep only Crowned Anthill (milestone filter)

4. **Should nestmates be explicit (opt-in) or automatic (opt-out)?**
   - Privacy vs. utility tradeoff

---

## 🎯 Success Criteria

The shared data layer is successful when:

1. ✅ New colonies automatically inherit relevant instincts from similar projects
2. ✅ Chamber archives are searchable across all repos via `/ant:tunnels --global`
3. ✅ Model routing improves through aggregated telemetry
4. ✅ No sensitive data leaks between repos (privacy audit pass)
5. ✅ `/ant:update` syncs shared data in <5 seconds
6. ✅ Hub storage remains under 100MB per user

---

## 🔗 Related Documents

- `/runtime/docs/PHEROMONE-SYSTEM-DESIGN.md` - Eternal memory architecture
- `/runtime/docs/MULTI-COLONY-ARCHITECTURE.md` - Colony registry system
- `/runtime/docs/pheromones.md` - Pheromone signal guide
- `/runtime/learning.md` - Instinct model documentation
- `/bin/lib/nestmate-loader.js` - Cross-project discovery

---

## 📝 Next Steps

1. **Review this research** - User approval of architecture
2. **Create Phase 1 plan** - Detailed task breakdown
3. **Implement foundation** - Directory structure + sync mechanism
4. **Test with existing colonies** - Backfill chamber archives
5. **Version bump to 3.2.0** - New feature release

---

*Synthesized from three parallel scout ant investigations.*
*The colony remembers. The mound grows. 🐜*
