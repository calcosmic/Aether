# Phase 55: Agent Definition Updates - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-04-26
**Phase:** 55-agent-definition-updates
**Areas discussed:** Write-scope guardrail design, Dispatch prompt injection, Tracker's role boundary, Mirror sync strategy

---

## Write-scope guardrail design

| Option | Description | Selected |
|--------|-------------|----------|
| Agent body text only (soft) | Guardrails in agent definition body, no runtime enforcement | |
| Body text + dispatch injection | Guardrails in body + findings-path at dispatch time | ✓ |
| Body text + runtime validation | Guardrails in body + Go runtime validates writes after agent returns | |

**User's choice:** Body text + dispatch injection
**Notes:** Two-layer approach keeps it simple — no Go runtime code needed for validation. Agent body has permanent boundaries, dispatch adds concrete paths.

---

## Dispatch prompt injection

| Option | Description | Selected |
|--------|-------------|----------|
| Embed in agent body | Static domain paths in agent definition, no Go code changes | |
| Inject at dispatch | Go code appends findings-path to task prompts at dispatch time | ✓ |
| Both layers | Agent body has generic instructions, dispatch adds concrete paths | |

**User's choice:** Inject at dispatch (Recommended)
**Notes:** Agent body keeps generic guardrails, dispatch adds specific paths. Clean separation — body says "only write to reviews/", dispatch says "write to security domain at .aether/data/reviews/security/".

---

## Tracker's role boundary

| Option | Description | Selected |
|--------|-------------|----------|
| Specific carve-out | Replace "no Write" with restricted Write, update boundary text | ✓ |
| Generic guardrails only | Remove "no Write" language, rely on generic write-scope guardrails | |
| New dedicated section | Add a "Findings Persistence" section to preserve diagnose-only identity | |

**User's choice:** Specific carve-out (Recommended)
**Notes:** Minimal text change. Replace "no Write tool" with "Write restricted to .aether/data/reviews/", update ".aether/data/" boundary to exclude reviews/bugs/. Preserves Tracker's identity.

---

## Mirror sync strategy

| Option | Description | Selected |
|--------|-------------|----------|
| Single plan | All 7 agents × 4 surfaces = 28 files in one plan | ✓ |
| Split by surface priority | First .claude + mirror, then .opencode + .codex | |

**User's choice:** Single plan (Recommended)
**Notes:** All changes are mechanical (add Write to tools frontmatter, add findings section, add guardrails). One coherent pass easier to verify parity.

---

## Claude's Discretion

- Exact wording of findings write instructions in agent bodies
- Exact format of dispatch injection text in Go code
- Whether to add findings_section to agent return format
- Error handling for wrong-path writes

## Deferred Ideas

None — discussion stayed within phase scope.
