# Phase 55: Agent Definition Updates - Research

**Researched:** 2026-04-26
**Domain:** Agent definition files (markdown + TOML), Go dispatch code, review ledger CLI
**Confidence:** HIGH

## Summary

Phase 55 updates 7 review agent definitions across 4 surface mirrors (28 files total) to add Write tool access and findings persistence instructions. Each agent gains a scoped Write capability restricted to its domain review ledger under `.aether/data/reviews/`. The Go dispatch code in `codex_build.go` and `codex_continue.go` must inject concrete findings-path instructions at task assignment time. The review ledger CLI (`review-ledger-write`) already exists and is fully tested from Phase 53.

The core pattern is established: Builder already has unrestricted Write with boundary declarations. This phase replicates that pattern but with tighter scoping -- agents may ONLY write to their designated ledger files. The agent-to-domain mapping is enforced both in the agent body text (permanent guardrails) and in the Go CLI (runtime validation via `agentAllowedDomains`).

**Primary recommendation:** Edit 7 canonical agent files in `.claude/agents/ant/`, copy each to `.aether/agents-claude/` (byte-identical), apply structural parity to `.opencode/agents/` (change `write: false` to `write: true` in frontmatter), and update `.codex/agents/` TOML files. Then modify `codexBuildSpecialistDispatch` and `renderCodexContinueReviewBrief` to append findings-path injection text.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- **D-01:** Two-layer guardrails: agent body text carries permanent boundary instructions ("only write to .aether/data/reviews/{domain}/"), dispatch injection adds concrete path at task time
- **D-02:** No runtime validation in Go code -- relies on LLM following instructions, same trust model as existing "no Bash" rules for Gatekeeper
- **D-03:** Guardrails explicitly list what agents MUST NOT write to (source code, tests, colony state, dreams, .env) and the single exception (review ledger files)
- **D-04:** Findings-path instructions injected at dispatch time in Go code, NOT embedded in agent body
- **D-05:** Agent body has generic guardrails (what NOT to write), dispatch adds the specific path (e.g., "Write findings to `.aether/data/reviews/security/` using `review-ledger-write`")
- **D-06:** Go code changes required in build dispatch (`codexBuildSpecialistDispatch` for Watcher, Chaos, Measurer, Archaeologist) and continue dispatch (`codexContinueReviewSpecs` for Gatekeeper, Auditor, Probe)
- **D-07:** Specific carve-out for Tracker: replace "no Write tool" with "Write tool restricted to `.aether/data/reviews/` only"
- **D-08:** Update Tracker boundary from "Do not write to `.aether/data/`" to "`.aether/data/` except `reviews/bugs/`"
- **D-09:** Preserve Tracker's diagnose-only identity -- Write is for persisting findings only, never for applying fixes
- **D-10:** Single plan handles all 7 agents across all 4 surfaces (28 files) -- changes are mechanical, one coherent pass easier to verify parity
- **D-11:** Surfaces: `.claude/agents/ant/` (canonical), `.aether/agents-claude/` (mirror), `.opencode/agents/` (mirror), `.codex/agents/` (TOML mirror)

### Claude's Discretion
- Exact wording of findings write instructions in agent bodies
- Exact format of dispatch injection text in Go code
- Whether to add a `findings_section` to the agent return format or rely on dispatch instructions
- Error handling when agents write to wrong paths

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| AGENT-01 | Gatekeeper agent definition includes Write tool, findings write instructions for security domain, write-scope guardrails | Gatekeeper currently has `tools: Read, Grep, Glob` -- add Write. Agent body references `<boundaries>` section for guardrails. Domain mapping: gatekeeper -> security |
| AGENT-02 | Auditor agent definition includes Write tool, findings write instructions for quality/security/performance domains, write-scope guardrails | Auditor currently has `tools: Read, Grep, Glob` -- add Write. Multi-domain agent (3 domains). Domain mapping: auditor -> quality, security, performance |
| AGENT-03 | Chaos agent definition includes Write tool, findings write instructions for resilience domain, write-scope guardrails | Chaos currently has `tools: Read, Bash, Grep, Glob` -- add Write. Domain mapping: chaos -> resilience |
| AGENT-04 | Watcher agent definition includes Write tool, findings write instructions for testing/quality domains, write-scope guardrails | Watcher currently has `tools: Read, Bash, Grep, Glob` -- add Write. Domain mapping: watcher -> testing, quality |
| AGENT-05 | Archaeologist agent definition includes Write tool, findings write instructions for history domain, write-scope guardrails | Archaeologist currently has `tools: Read, Bash, Grep, Glob` -- add Write. Domain mapping: archaeologist -> history |
| AGENT-06 | Measurer agent definition includes Write tool, findings write instructions for performance domain, write-scope guardrails | Measurer currently has `tools: Read, Bash, Grep, Glob` -- add Write. Domain mapping: measurer -> performance |
| AGENT-07 | Tracker agent definition includes Write tool, findings write instructions for bugs domain, write-scope guardrails | Tracker currently has `tools: Read, Bash, Grep, Glob` -- add Write. Domain mapping: tracker -> bugs. SPECIAL CASE: existing "no Write" and "do not write to .aether/data/" boundaries must be carved out |
| AGENT-08 | All 7 agent definitions synced across 4 surfaces: 28 file edits total | 4 surfaces verified: `.claude/agents/ant/` (canonical), `.aether/agents-claude/` (byte-identical mirror), `.opencode/agents/` (structural parity with YAML tools map), `.codex/agents/` (TOML format) |
| AGENT-09 | Write-scope guardrails explicitly restrict agents to ONLY write to designated review ledger files, never source code, tests, or colony state | Pattern exists in Builder `<boundaries>` section -- replicate with tighter scoping. Explicit "MUST NOT" language as used in existing agent critical_rules |
| AGENT-10 | Build and continue dispatch flows inject findings-path instructions into review agent task prompts | Build dispatch at `codexBuildSpecialistDispatch` (line ~660) and continue dispatch at `renderCodexContinueReviewBrief` (line ~907) need appended findings-path text per agent caste |
</phase_requirements>

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Agent definition edits (markdown) | Configuration / Static Assets | -- | Agent .md files are static configuration loaded by the platform at spawn time |
| Agent definition edits (TOML) | Configuration / Static Assets | -- | Codex TOML files are equivalent static config |
| Frontmatter tools list | Configuration / Static Assets | -- | Single-line YAML frontmatter change |
| Go dispatch injection | Go Binary (cmd/) | -- | Go code constructs task prompts at runtime |
| Review ledger CLI (target for writes) | Go Binary (cmd/) | pkg/colony/ | Already implemented and tested in Phase 53 |
| Mirror sync verification | Build / Packaging | -- | Byte-identical copy and structural parity checks |

## Standard Stack

### Core (existing -- no new dependencies)
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib (strings, fmt) | go.mod current | String building for dispatch injection | Already used throughout cmd/ |
| cobra | go.mod current | CLI command framework | Already used for review-ledger-write |
| pkg/colony | local | ReviewLedgerEntry types, DomainOrder, agent-domain mapping | Phase 53 implementation |
| pkg/storage | local | JSON file locking for ledger writes | Phase 53 implementation |

### Supporting (no install needed)
| Tool | Purpose | When to Use |
|------|---------|-------------|
| `diff` | Verify byte-identical mirrors | After copying canonical to agents-claude mirror |
| `go test ./...` | Verify Go changes don't break existing tests | After dispatch code changes |

**Installation:** No new packages required -- zero new dependencies per established decision.

## Architecture Patterns

### Agent Definition Structure (Canonical - Claude)

All 7 agents follow this exact structure [VERIFIED: codebase read of all 7 files]:

```
---
name: aether-{agent}
description: "{description}"
tools: Read, Bash, Grep, Glob     <-- ADD Write HERE
color: {color}
model: {sonnet|opus}
---

<role>...</role>
<glm_safety>...</glm_safety>        <-- opus models only
<execution_flow>...</execution_flow>
<critical_rules>...</critical_rules>
<return_format>...</return_format>
<success_criteria>...</success_criteria>
<failure_modes>...</failure_modes>
<escalation>...</escalation>
<boundaries>...</boundaries>
```

Key changes per agent:
1. **Frontmatter `tools:` line**: Add `Write` to the comma-separated list
2. **`<role>` section**: Update "no Write" / "read-only" language to reflect scoped Write
3. **`<boundaries>` section**: Add write-scope guardrails with explicit MUST NOT / ONLY MAY declarations
4. **`<critical_rules>` section**: Update any "no Write" rules to reflect scoped Write exception
5. **`<return_format>` section**: Optionally add findings write command example (Claude's discretion)

### Agent Definition Structure (OpenCode Mirror)

OpenCode uses YAML map for tools instead of comma-separated string [VERIFIED: `.opencode/agents/aether-watcher.md`]:

```yaml
---
name: aether-watcher
description: "..."
mode: subagent
tools:
  write: false          <-- CHANGE TO true
  edit: false
  bash: true
  grep: true
  glob: true
  task: false
color: "#2ecc71"
---
```

Key difference: OpenCode body content is identical to Claude canonical (same `<role>`, `<boundaries>`, etc.).

### Agent Definition Structure (Codex TOML Mirror)

Codex uses TOML format with `developer_instructions` block [VERIFIED: `.codex/agents/aether-chaos.toml`]:

```toml
name = "aether-chaos"
description = "..."
nickname_candidates = ["chaos", "breaker"]

developer_instructions = '''
... abbreviated instructions ...
'''
```

Codex TOML does NOT have an explicit tools list in the same way -- tools are typically controlled by the platform. The `developer_instructions` body must reflect the scoped Write capability in the instructions text.

### Dispatch Injection Points

Two dispatch paths need findings-path injection [VERIFIED: codebase read of `codex_build.go` and `codex_continue.go`]:

**Build dispatch** (for Watcher, Chaos, Measurer, Archaeologist):
- Function: `codexBuildSpecialistDispatch` at line ~660
- Currently constructs a `codexBuildDispatch` with `.Task` field set to a generic description
- Injection point: Append findings-path text to the `task` parameter for these castes

**Continue dispatch** (for Gatekeeper, Auditor, Probe):
- Function: `renderCodexContinueReviewBrief` at line ~907
- Already constructs a detailed task brief with phase info, evidence, etc.
- Injection point: Append findings-path text before the closing of the brief builder
- Note: Continue review brief already says "This is a read-only review" -- this language must be updated for agents that now have scoped Write

**Expected dispatch outcomes** (for reference at line ~1485):
- `expectedDispatchOutcome` returns outcome descriptions per caste
- These may need updating to mention findings persistence

### Tracker Special Case

Tracker has the most complex boundary changes [VERIFIED: `.claude/agents/ant/aether-tracker.md` lines 252-270]:

Current boundaries to modify:
1. `<role>`: "you have no Write or Edit tools" -> scoped Write exception
2. `<critical_rules>` "Diagnose Only": "You have no Write or Edit tools" -> scoped Write exception
3. `<critical_rules>` "Never Modify Files": Carve-out for findings persistence
4. `<boundaries>` "Tracker Is Diagnose-Only": "Tracker has no Write or Edit tools by design" -> scoped Write exception
5. `<boundaries>` "Do not write to `.aether/data/`": Change to "except `reviews/bugs/`"
6. Tracker's `<role>` line 14: "You return structured analysis. No activity logs. No side effects." -- still true, writing findings is not a side effect in the same sense

Tracker is NOT dispatched in build or continue -- invoked on-demand via `/ant-swarm`. No dispatch code changes needed for Tracker specifically. The findings-path instructions would come from whatever invokes Tracker (swarm command or manual invocation).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Writing findings to ledger files | Raw file I/O in agent instructions | `aether review-ledger-write --domain <d> --phase <N> --findings <json>` | CLI already handles ID generation, summary recomputation, file locking, domain validation, agent-domain mapping |
| Validating agent-domain mapping | Custom validation in agent body | Existing `agentAllowedDomains` map in `cmd/review_ledger.go` | Already enforced server-side -- duplicate enforcement in agent body is unnecessary per D-02 |
| Mirror sync | Custom copy script | Manual copy + `diff` verification | 7 files, one-time operation -- script overhead not justified |

**Key insight:** The review ledger CLI from Phase 53 is the complete CRUD layer. Agents just need to know the CLI command and JSON format -- no raw file manipulation needed.

## Common Pitfalls

### Pitfall 1: Inconsistent Mirror Updates
**What goes wrong:** Canonical `.claude/agents/ant/` edited but `.aether/agents-claude/` not copied, or `.opencode/agents/` frontmatter not updated.
**Why it happens:** 28 files across 4 directories -- easy to miss one surface.
**How to avoid:** Edit canonical first, then copy to mirror, then update OpenCode frontmatter, then update Codex TOML. Use `diff` to verify byte-identical for agents-claude mirror.
**Warning signs:** Publish workflow complains about mirror mismatch; `aether integrity` fails.

### Pitfall 2: Contradictory Language in Agent Body
**What goes wrong:** `<role>` says "no Write" but `<boundaries>` says "Write restricted to reviews" -- confusing for the LLM.
**Why it happens:** Partial edits where some sections are updated but others still reference the old "no Write" constraint.
**How to avoid:** For each agent, search for ALL occurrences of "no Write", "read-only", "Write or Edit", "no file modifications" and update each one consistently.
**Warning signs:** Agent returns blocked because it thinks it cannot write despite having the Write tool.

### Pitfall 3: Gatekeeper/Auditor Still Listed as "Read-Only" in Continue Brief
**What goes wrong:** `renderCodexContinueReviewBrief` still appends "This is a read-only review" for Gatekeeper and Auditor, contradicting their new Write capability.
**Why it happens:** The continue brief function is shared across all review castes; the "read-only" text was added before agents had Write access.
**How to avoid:** Make the "read-only" line conditional on whether the caste has scoped Write access, or update it to "This is a review task. You may persist findings to the review ledger using Write, but do not modify repo source files."
**Warning signs:** Gatekeeper or Auditor returns findings in JSON only, without writing to ledger.

### Pitfall 4: OpenCode Frontmatter Only Changes `write:` Flag
**What goes wrong:** OpenCode mirror gets `write: true` in frontmatter but the body text still says "no Write tool".
**Why it happens:** Frontmatter and body are separate; updating one without the other.
**How to avoid:** OpenCode body must match Claude canonical body exactly. Since OpenCode mirrors copy the full body from canonical, the body will be correct IF canonical is edited first.
**Warning signs:** OpenCode agents behave differently from Claude agents.

### Pitfall 5: Dispatch Injection Missing for Specific Castes
**What goes wrong:** Archaeologist gets findings injection in build dispatch but Tracker gets none (since it is not dispatched in build/continue).
**Why it happens:** Tracker is invoked via `/ant-swarm`, not through the standard dispatch paths.
**How to avoid:** Per D-04/D-06, dispatch injection only needed for build dispatch (Watcher, Chaos, Measurer, Archaeologist) and continue dispatch (Gatekeeper, Auditor, Probe). Tracker does not need dispatch injection -- its findings-path instruction comes from swarm invocation context or is embedded in its body.
**Warning signs:** Tracker invoked via swarm but doesn't know where to write findings.

## Code Examples

### Current Agent Frontmatter (to be changed)

**Claude canonical** (`aether-chaos.md`):
```yaml
tools: Read, Bash, Grep, Glob
```
Changes to:
```yaml
tools: Read, Bash, Grep, Glob, Write
```

**OpenCode** (`aether-chaos.md`):
```yaml
tools:
  write: false
  edit: false
  bash: true
  grep: true
  glob: true
  task: false
```
Changes to:
```yaml
tools:
  write: true
  edit: false
  bash: true
  grep: true
  glob: true
  task: false
```

### New Write-Scope Guardrail Section (for `<boundaries>`)

Pattern to add to each agent's `<boundaries>` section:

```markdown
### Write-Scope Restriction
You have Write tool access for ONE purpose only: persisting findings to your domain review ledger. You MUST use `aether review-ledger-write` to write findings.

**You MAY write to:**
- `.aether/data/reviews/{domain}/ledger.json` (via `review-ledger-write`)

**You MUST NOT write to:**
- Source code files (any `*.go`, `*.js`, `*.ts`, `*.py`, etc.)
- Test files
- Colony state (`.aether/data/COLONY_STATE.json`, `.aether/data/pheromones.json`, etc.)
- User notes (`.aether/dreams/`)
- Environment files (`.env*`)
- CI configuration (`.github/workflows/`)
- Any file not in `.aether/data/reviews/`

If you need a file modified to address a finding, report it in your return and route to Builder.
```

### Dispatch Injection Text Pattern

For build dispatch (`codexBuildSpecialistDispatch`), append to task string:

```go
// After the existing task description, append for review castes:
func findingsInjectionForCaste(caste string) string {
    domainMap := map[string]string{
        "watcher":      "testing and quality",
        "chaos":        "resilience",
        "measurer":     "performance",
        "archaeologist": "history",
    }
    domains, ok := domainMap[caste]
    if !ok {
        return ""
    }
    return fmt.Sprintf("\n\nPersist your %s findings to the domain review ledger using: aether review-ledger-write --domain <domain> --phase <N> --findings '<json>'", domains)
}
```

For continue dispatch (`renderCodexContinueReviewBrief`), append findings instruction after existing evidence listing:

```go
// Per D-06: Gatekeeper, Auditor, Probe are the continue review castes
// Add domain-specific findings injection based on spec.Caste
```

### Review-Ledger-Write Command Format (for agent body examples)

```bash
aether review-ledger-write \
  --domain security \
  --phase 55 \
  --findings '[{"severity":"HIGH","file":"cmd/example.go","line":42,"category":"Input Validation","description":"Missing null check","suggestion":"Add guard"}]' \
  --agent gatekeeper \
  --agent-name "Warden-12"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Review agents have no Write tool | Review agents gain scoped Write for ledger persistence | Phase 55 (this phase) | Agents can persist findings across phases; colony-prime can surface them |
| Findings only in agent JSON return | Findings in both JSON return AND domain ledger | Phase 55 (this phase) | Structured persistence enables cross-phase tracking |
| "This is a read-only review" in continue brief | Conditional text based on caste capabilities | Phase 55 (this phase) | Prevents contradiction between dispatch instructions and agent capabilities |

**Deprecated/outdated:**
- Agent body text claiming "no Write tool" for the 7 review agents -- will be replaced with scoped Write language
- OpenCode frontmatter `write: false` for the 7 agents -- will be changed to `write: true`

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Codex TOML agents do not have a tools configuration field -- the `developer_instructions` body is the only thing to update | Agent Definition Structure | If Codex TOML has a tools field we missed, agents won't get Write access on Codex |
| A2 | Tracker's swarm invocation context provides sufficient information for findings-path -- no separate dispatch injection needed | Dispatch Injection Points | If swarm invocation doesn't provide domain/phase context, Tracker won't know where to write |
| A3 | The `expectedDispatchOutcome` function at line ~1485 does NOT need updating for this phase | Dispatch Injection Points | If it does need updating, the planner should add a task for it |

## Open Questions (RESOLVED)

1. **Should the `<return_format>` section of each agent include a findings-write example?** — RESOLVED
   - Decision: Add a brief `findings_persistence` note to the return format showing the CLI command, rather than a full JSON example. The dispatch injection provides the concrete path at task time. Implemented in Plan 01 Task 1 step 5.

2. **How should Tracker receive findings-path instructions without dispatch injection?** — RESOLVED
   - Decision: Include generic findings instructions in Tracker's agent body (since it has no dispatch injection point), referencing "your domain (bugs)" rather than a concrete path. Implemented in Plan 01 Task 1 steps 7-8 (Tracker special case).

## Environment Availability

Step 2.6: SKIPPED (no external dependencies identified -- all changes are to markdown files, TOML files, and Go source code within the existing project).

## Validation Architecture

> Config check: `workflow.nyquist_validation` is absent from `.planning/config.json`, so this section is included.

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing (stdlib) |
| Config file | None (conventional `go test`) |
| Quick run command | `go test ./cmd/... -run TestReviewLedger -count=1` |
| Full suite command | `go test ./... -count=1` |

### Phase Requirements to Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AGENT-01 through AGENT-07 | Agent body text contains Write tool, guardrails, findings instructions | Manual verification | `grep -c "Write" .claude/agents/ant/aether-{agent}.md` | N/A (content verification) |
| AGENT-08 | All 4 surfaces in sync for all 7 agents | Integration | `diff .claude/agents/ant/aether-chaos.md .aether/agents-claude/aether-chaos.md` | N/A (file comparison) |
| AGENT-09 | Write-scope guardrails present in boundaries section | Manual verification | `grep -c "MUST NOT write to" .claude/agents/ant/aether-chaos.md` | N/A (content verification) |
| AGENT-10 | Dispatch injection text appears for review castes | Unit | `go test ./cmd/... -run TestFindingsInjection -count=1` | Wave 0 (new test) |

### Sampling Rate
- **Per task commit:** `go test ./cmd/... -count=1`
- **Per wave merge:** `go test ./... -count=1`
- **Phase gate:** Full suite green + manual mirror parity verification

### Wave 0 Gaps
- [ ] Unit test for `findingsInjectionForCaste` (or equivalent function name) -- covers AGENT-10
- [ ] Integration test verifying dispatch text contains findings-path for review castes -- covers AGENT-10
- [ ] Existing review ledger tests cover the CLI commands agents will call -- no gap

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | no | N/A -- no auth changes |
| V3 Session Management | no | N/A -- no session changes |
| V4 Access Control | yes | Agent-domain mapping enforced by `agentAllowedDomains` in Go CLI |
| V5 Input Validation | yes | `review-ledger-write` already validates domain, agent, findings JSON, max 50 findings |
| V6 Cryptography | no | N/A -- no crypto changes |

### Known Threat Patterns for Agent Definition Updates

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Agent writes to unauthorized path | Tampering | Write-scope guardrails in agent body + LLM trust model (D-02) |
| Malicious findings JSON injection | Tampering | `review-ledger-write` validates JSON structure and caps at 50 entries |
| Cross-domain agent writes | Elevation | `agentAllowedDomains` map in Go CLI enforces domain membership |

## Sources

### Primary (HIGH confidence)
- Codebase read: All 7 canonical agent definitions in `.claude/agents/ant/`
- Codebase read: `cmd/codex_build.go` lines 570-670 (build dispatch)
- Codebase read: `cmd/codex_continue.go` lines 788-960 (continue dispatch and review specs)
- Codebase read: `cmd/review_ledger.go` full file (CLI commands)
- Codebase read: `pkg/colony/review_ledger.go` full file (types)
- Codebase read: `.opencode/agents/aether-watcher.md` and `aether-tracker.md` (OpenCode format)
- Codebase read: `.codex/agents/aether-chaos.toml` and `aether-tracker.toml` (Codex TOML format)
- Byte-identical verification: `diff` between `.claude/agents/ant/` and `.aether/agents-claude/` for chaos and tracker

### Secondary (MEDIUM confidence)
- CONTEXT.md canonical references (confirmed via codebase reads above)

### Tertiary (LOW confidence)
- Assumption A1: Codex TOML lacks tools field -- based on read of 2 TOML files only

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - all existing code read and verified
- Architecture: HIGH - all 4 surface formats verified, dispatch injection points identified
- Pitfalls: HIGH - derived from reading actual agent bodies and identifying specific text that would conflict

**Research date:** 2026-04-26
**Valid until:** 30 days (stable -- agent definition format unlikely to change)
