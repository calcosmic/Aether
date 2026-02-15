# Aether Documentation

## Implementation Framework

This directory contains the complete specification and implementation guidance for the Aether colony system.

---

## Priority Documents (Today's Work)

These documents represent the current state of the system and should be read first:

| Document | Size | Purpose |
|----------|------|---------|
| **AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md** | 73KB | Complete pheromone & multi-colony spec (v1.1) |
| **AETHER-2.0-IMPLEMENTATION-PLAN.md** | 36KB | 10 paradigm-shifting features roadmap |
| **PHEROMONE-INJECTION.md** | 8KB | Injection timing and UX flows |
| **PHEROMONE-INTEGRATION.md** | 6KB | Command integration patterns |
| **PHEROMONE-SYSTEM-DESIGN.md** | 13KB | Core philosophy and taxonomy |
| **VISUAL-OUTPUT-SPEC.md** | 6KB | UI/UX standards for all commands |

**Note:** The three `PHEROMONE-*.md` documents have been consolidated into the MASTER-SPEC but are preserved here as reference.

---

## Directory Structure

```
docs/
├── README.md                          # This file
├── AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md     # ← START HERE
├── AETHER-2.0-IMPLEMENTATION-PLAN.md          # ← Then this
├── PHEROMONE-INJECTION.md                     # (consolidated into master)
├── PHEROMONE-INTEGRATION.md                   # (consolidated into master)
├── PHEROMONE-SYSTEM-DESIGN.md                 # (consolidated into master)
├── VISUAL-OUTPUT-SPEC.md                      # Visual standards
│
├── architecture/                      # System design
│   └── MULTI-COLONY-ARCHITECTURE.md
│
├── implementation/                    # Additional implementation guides
│   ├── pheromones.md
│   ├── pathogen-schema.md
│   └── pathogen-schema-example.json
│
└── reference/                         # Supporting materials
    ├── biological-reference.md
    ├── namespace.md
    ├── constraints.md
    ├── command-sync.md
    └── progressive-disclosure.md
```

---

## Quick Start

### For Implementers

1. **Start with:** `AETHER-PHEROMONE-SYSTEM-MASTER-SPEC.md`
   - Complete v1.1 specification
   - 5 implementation phases with agent assignments (A-J)
   - Pheromone taxonomy, injection timing, multi-colony architecture
   - Sections 14-22: hooks, plugins, custom agents, best practices

2. **Then read:** `AETHER-2.0-IMPLEMENTATION-PLAN.md`
   - 10 paradigm-shifting features
   - Research phases and implementation strategy
   - Dependencies and recommended order

3. **Reference:** `VISUAL-OUTPUT-SPEC.md`
   - Visual standards for all commands
   - Worker output rules
   - Caste emoji mappings

### For Context

- `PHEROMONE-INJECTION.md` - Detailed injection timing (now in master spec Section 3.4-3.6)
- `PHEROMONE-INTEGRATION.md` - Command integration patterns (now in master spec Section 10)
- `PHEROMONE-SYSTEM-DESIGN.md` - Original philosophy (now in master spec Section 2-3)

---

## Document Status

| Document | Status | Lines | Content |
|----------|--------|-------|---------|
| Master Spec | Complete | 2,642 | Single source of truth |
| 2.0 Plan | Planning | 1,344 | 10-feature roadmap |
| Visual Spec | Design | ~200 | UI standards |

---

*The colony remembers. The colony learns. The colony evolves.*
