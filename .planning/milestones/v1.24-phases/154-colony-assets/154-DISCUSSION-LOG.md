# Phase 154: Colony Assets - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-05-23
**Phase:** 154-colony-assets
**Areas discussed:** Agent completeness, Platform-specific files, Prompt organization, Playbook location

---

## Agent Completeness — 8 Core or All 27 Castes?

| Option | Description | Selected |
|--------|-------------|----------|
| 8 core only | Extract only queen, builder, watcher, scout, oracle, gatekeeper, auditor, probe. The other 19 stay in platform files only. | |
| All 27 castes | Every caste gets a YAML definition. The TS schema gets updated to accept all roles. | ✓ |
| All 27 with tier marker | Extract all 27, but add a tier: core | extended field so the control plane knows which are priority. | |

**User's choice:** All 27 castes
**Notes:** User wants the full colony documented in YAML, not just the milestone's 8-core cap.

---

## Platform-Specific Files — Generated or Parallel?

| Option | Description | Selected |
|--------|-------------|----------|
| Generated outputs | A script reads colony/agents/*.yaml and produces .claude/, .opencode/, .codex/ files automatically. | |
| Parallel tracks | colony/agents/ is for runtime use. Platform files stay manually maintained. | ✓ |
| Hybrid — canonical base + overlays | colony/agents/ defines shared content. Generator produces base platform files, but each platform can add overlays. | |

**User's choice:** "You decide" — deferred to Claude
**Claude's discretion:** Chose parallel tracks. Phase 154 is extraction, not generator building. Platform files already work and differ in meaningful ways (TOML vs Markdown). colony/agents/*.yaml becomes the canonical runtime source; platform files stay as manually-maintained layers. A generator is a promising future enhancement.

---

## Prompt Organization — Whole Prompts or Shared Fragments?

| Option | Description | Selected |
|--------|-------------|----------|
| One complete prompt per agent | Each agent gets one full prompt file. Simple, mirrors current structure. | ✓ |
| Shared fragments | Common sections live as fragments, agent files are lightweight assemblies. DRY but adds assembly. | |
| Hybrid — core + fragments | Core personality in one file, shared conventions as fragments that get appended. | |

**User's choice:** "You decide" — deferred to Claude
**Claude's discretion:** Chose one complete prompt per agent. The AgentSchema already has a prompt_file field — one file per agent. This is pure extraction with no assembly logic needed. Fragment assembly can be optimized in a future phase.

---

## Playbook Location — Move to colony/ or Keep in .aether/?

| Option | Description | Selected |
|--------|-------------|----------|
| Move to colony/playbooks/ | Playbooks become behaviour assets under colony/. Clean boundary. | ✓ |
| Keep in .aether/docs/command-playbooks/ | Less churn, but blurs the line between behaviour and docs. | |
| Copy with deprecation note | Create canonical versions under colony/, leave old ones with a moving notice. | |

**User's choice:** Move to colony/playbooks/
**Notes:** Clean separation between behaviour assets (colony/) and reference documentation (.aether/docs/).

---

## Claude's Discretion

- Platform file strategy: parallel tracks
- Prompt organization: one complete prompt per agent

## Deferred Ideas

- Generator for platform agent files from colony/agents/*.yaml
- Prompt fragment assembly system

---

*Phase: 154-Colony Assets*
*Discussion logged: 2026-05-23*
