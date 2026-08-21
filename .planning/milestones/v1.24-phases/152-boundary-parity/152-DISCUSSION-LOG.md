# Phase 152: Boundary & Parity - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-22
**Phase:** 152-Boundary & Parity
**Areas discussed:** Visual/ceremony boundary, Classic parity scope, Extraction audit granularity, Doc location and format

---

## Visual/ceremony boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Move to config | Extract caste colours, emojis, labels, and banner templates into YAML/JSON files that Go loads at runtime | ✓ |
| Keep in Go | Visual rendering is runtime spine, not behaviour. The boundary rule targets logic, not presentation strings | |
| Split it | Caste identity moves to config as 'skin' data. Stage separators and banner formatting stay in Go as rendering logic | |

**User's choice:** Move to config
**Notes:** User asked about Ruflo best practice; Claude recommended one shared file across platforms since caste identity is runtime truth.

| Option | Description | Selected |
|--------|-------------|----------|
| One shared file | Single source of truth in .aether/visuals.yaml — all platforms read the same caste identities | ✓ |
| Platform overrides | Base visuals in .aether/visuals.yaml with per-platform overrides | |
| Runtime-native only | Only Codex/Go reads the config. Claude and OpenCode wrappers keep their own markdown-based framing | |

**User's choice:** "Actually i am unsure what the best practice is, you decide"
**Notes:** Claude chose one shared file. Caste identity is consistent across platforms; wrappers add framing on top.

| Option | Description | Selected |
|--------|-------------|----------|
| YAML | Matches existing .aether/commands/*.yaml pattern | |
| JSON | Native Go parsing. Less human-friendly | |
| Markdown with YAML frontmatter | Matches agent definition pattern | ✓ |

**User's choice:** Markdown with YAML frontmatter
**Notes:** Aligns with existing .claude/agents/ pattern.

---

## Classic parity scope

| Option | Description | Selected |
|--------|-------------|----------|
| Gaps only | List only where Go falls short of Classic | |
| Full inventory | Catalog all Classic behaviours, marking each as MATCH, GAP, or DEGRADED | ✓ |
| Hybrid | Full inventory for core workflows, gaps-only for edge cases | |

**User's choice:** Full inventory

| Option | Description | Selected |
|--------|-------------|----------|
| Classic wins | Restore the old shell-based behaviour | |
| Go wins | Keep current Go behaviour | |
| User decides per item | The checklist flags each conflict. You review and choose | ✓ |
| Hybrid wins | Best of both worlds — keep Go improvements, restore Classic soul | |

**User's choice:** User decides per item

| Option | Description | Selected |
|--------|-------------|----------|
| Automated tests | Each item has a golden/snapshot test | |
| Manual checklist | Each item has a concrete manual step | |
| Both | Core workflows get automated tests. Edge cases get manual checklist | ✓ |

**User's choice:** "you decide"
**Notes:** Claude chose Both — matches v1.18 golden test pattern.

---

## Extraction audit granularity

| Option | Description | Selected |
|--------|-------------|----------|
| File-level | Classify each .go file as a whole | ✓ |
| Function-level | Classify each exported and significant unexported function | |
| Hybrid | Package-level for clear-cut packages, file-level for boundary files | |

**User's choice:** File-level

| Option | Description | Selected |
|--------|-------------|----------|
| Automated script | Script greps for keywords and classifies heuristically | |
| Manual review | Human reads each file and decides | |
| Agent-assisted | Automated first pass + agent review for boundary files | ✓ |

**User's choice:** Agent-assisted

| Option | Description | Selected |
|--------|-------------|----------|
| Annotate as MIXED | Mark the file MIXED and list which functions stay vs move | ✓ |
| Split the file | Physically separate spine and behaviour into different files | |
| Favour majority | If >50% is behaviour, classify the whole file as MOVE | |

**User's choice:** "you decide"
**Notes:** Claude chose annotate-as-MIXED. Splitting files is a refactor for Phase 155, not the audit.

---

## Doc location and format

| Option | Description | Selected |
|--------|-------------|----------|
| .aether/docs/ | Distributed to users via aether update | ✓ |
| docs/ | Top-level directory, more visible to external readers | |
| .planning/phases/152-boundary-parity/ | Dev-only, alongside phase artifacts | |
| docs/architecture/ | New subdirectory for architecture docs | |

**User's choice:** "you decide"
**Notes:** Claude chose .aether/docs/ for ARCHITECTURE_BOUNDARY.md and PARITY_CLASSIC_VS_GO.md; .planning/phases/ for BEHAVIOUR_EXTRACTION_AUDIT.md.

| Option | Description | Selected |
|--------|-------------|----------|
| ADR format | Architecture Decision Record template | |
| Reference doc format | Plain markdown with sections | ✓ |
| You decide | Let Claude decide based on what fits each doc best | |

**User's choice:** "You decide"
**Notes:** Claude chose reference doc format — ADR too rigid for a checklist and audit table.

---

## Claude's Discretion

- Visual config format: Markdown+YAML frontmatter (user deferred)
- Verification method: Both automated + manual (user deferred)
- Mixed files handling: Annotate as MIXED (user deferred)
- Doc location: .aether/docs/ for boundary+parity, .planning/phases/ for audit (user deferred)
- Doc format: Reference doc markdown (user deferred)

## Deferred Ideas

- Federation / inter-colony coordination — explicitly deferred in PROJECT.md
- Web UI for event stream — deferred in STATE.md
- Vector backend for memory — deferred in STATE.md

## External References Shared

- `https://github.com/ruvnet/ruflo` — User shared as context for hybrid engine + TS control plane architecture
