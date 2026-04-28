# Phase 72: Smart Init Charter - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-28
**Phase:** 72-smart-init-charter
**Areas discussed:** Charter persistence, Approval flow behavior, Codex ceremony path, Charter document format

---

## Charter Persistence

| Option | Description | Selected |
|--------|-------------|----------|
| Store in COLONY_STATE.json | Add charter fields to existing state file. Downstream commands already read it. | ✓ |
| Separate charter file | Write charter.md in .aether/data/. Cleaner JSON but adds a file to manage. | |
| Both locations | Summary in COLONY_STATE.json, full document as charter.md. | |

**User's choice:** COLONY_STATE.json with a charter sub-object (intent, vision, governance, goals)
**Notes:** User asked for recommendation first. I recommended COLONY_STATE.json because it's the single source of truth, downstream commands already read it, and the charter is structured data that fits naturally as JSON. User confirmed — no separate charter.md needed.

---

## Approval Flow Behavior

### Revise goal behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Re-run research with new goal | User provides new goal, init-research re-runs, fresh charter presented. | ✓ |
| Edit charter fields inline | User edits individual fields without re-scanning. Faster but goal stays same. | |

**User's choice:** Re-run research with new goal — clean restart.

### Reject/cancel behavior

| Option | Description | Selected |
|--------|-------------|----------|
| Clean exit, no state | No files created, no pheromones written, no colony state. | ✓ |
| Save draft charter for later | Save charter as draft so user can resume without re-scanning. | |

**User's choice:** Clean exit — no state left behind.

### Approval sequence

| Option | Description | Selected |
|--------|-------------|----------|
| Keep current order | Charter → pheromones → shelf → final approval. | ✓ |
| Shelf before pheromones | See shelved ideas before deciding on pheromones. | |
| Single combined approval | All three on one screen, approve once. | |

**User's choice:** Keep current order (charter → pheromones → shelf → approval).

---

## Codex Ceremony Path

### Codex ceremony approach

| Option | Description | Selected |
|--------|-------------|----------|
| Skip ceremony on Codex | Codex runs `--no-ceremony`, users who want ceremony use Claude/OpenCode. | |
| Auto-approve flag | Go runtime adds `--approve` flag, Codex shows charter then passes flag. | |
| Full Go-native ceremony | Build terminal-based approval flow in Go for Codex. | ✓ |

**User's choice:** Full Go-native ceremony.

### Wrapper vs Go ceremony

| Option | Description | Selected |
|--------|-------------|----------|
| Dual mode | Go ceremony for Codex/CLI, wrapper ceremony for Claude/OpenCode. | ✓ |
| Go-only | Move everything to Go, wrappers become thin. | |

**User's choice:** Dual mode — best of both worlds.

### Go prompt style

| Option | Description | Selected |
|--------|-------------|----------|
| Numbered list, type to select | Simple, no dependencies, works everywhere. | ✓ |
| Interactive TUI with library | Richer UX but adds dependency (violates zero-new-deps). | |

**User's choice:** Numbered list, no new dependencies.

---

## Charter Document Format

### Section count

| Option | Description | Selected |
|--------|-------------|----------|
| Keep current 4 sections | Intent/Vision/Governance/Goals — proven structure. | |
| Add more sections from scan data | Add Tech Stack, Key Risks, Constraints. | ✓ |

**User's choice:** Add more sections from scan data.

### Additional sections

| Option | Description | Selected |
|--------|-------------|----------|
| Tech Stack | Languages, frameworks, build tools from scan. | ✓ |
| Key Risks | No tests, no CI, large files — set expectations. | ✓ |
| Constraints | Hard limits — no lockfile, missing .gitignore. | ✓ |

**User's choice:** All three selected.

### Phase 72/73 boundary

| Option | Description | Selected |
|--------|-------------|----------|
| 72 uses existing data, 73 adds depth | 72 creates ceremony with current init-research output. 73 adds deeper analysis. | ✓ |
| 72 does everything, 73 is separate | 72 produces fully rich charter, 73 focuses on other features. | |

**User's choice:** 72 uses existing data, 73 adds depth.

---

## Claude's Discretion

No areas explicitly delegated to Claude's discretion — all decisions were made by the user.
