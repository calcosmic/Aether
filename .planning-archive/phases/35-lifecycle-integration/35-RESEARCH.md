# Phase 35: Lifecycle Integration - Research

**Researched:** 2026-02-21
**Domain:** Colony Lifecycle Boundaries (seal.md, entomb.md) with Wisdom Extraction
**Confidence:** HIGH (direct source inspection of existing implementation)

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

#### Approval Flow
- **Require approval** — same tick-to-approve UI as continue.md
- **Block until approved** — wisdom must be handled before seal/entomb ceremony proceeds
- **Same full UI** — checkboxes, threshold bars, preview/confirm (not abbreviated)
- **Both boundaries require approval** — seal and entomb have identical approval requirements

#### What to Extract
- **All pending proposals** from learning-observations.json
- **Proposals only** — no auto-generated colony summary
- **Keep deferred items** — persist for future sessions (don't clear at lifecycle boundary)
- **Show message if empty** — "No wisdom proposals to review" then proceed with ceremony

#### Integration Timing
- **Before ceremony** — wisdom extraction first, celebration second
- **AI prompts user within command** — not a separate command the user runs
- **Both continue and lifecycle boundaries show proposals** — continue handles phase-end, seal/entomb do final check
- **All pending with highlighting** — show new vs deferred status visually

#### Consistency
- **Same code path** — shared function for both seal.md and entomb.md
- **Reuse existing functions** where possible (learning-display-proposals, learning-approve-proposals)

### Claude's Discretion
- Exact implementation approach (refactor vs new wrapper vs direct reuse)
- Highlighting format for new vs deferred proposals
- Error handling if wisdom extraction fails

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INT-04 | seal.md promotes final colony wisdom (before sealing) | Use learning-approve-proposals with blocking approval UI; integrate before Step 4 (milestone update) in seal.md |
| INT-05 | entomb.md promotes wisdom before archiving (before entomb) | Use learning-approve-proposals with blocking approval UI; integrate before Step 4 (chamber creation) in entomb.md |
</phase_requirements>

---

## Summary

Phase 35 integrates wisdom extraction into the colony lifecycle boundaries: `seal.md` (final milestone) and `entomb.md` (archival). Both commands currently have wisdom promotion logic, but they operate silently without user approval. This phase adds the tick-to-approve UX (same as `continue.md`) to ensure wisdom is validated by the user before becoming permanent.

**Key insight:** The infrastructure already exists — `learning-approve-proposals` in `aether-utils.sh` provides the full approval workflow including display, selection, promotion, and deferral. The work is wiring this into `seal.md` and `entomb.md` at the right integration points.

**Primary recommendation:** Use `learning-approve-proposals` directly in both commands, placed BEFORE the ceremonial steps (milestone award in seal, chamber creation in entomb). Block progression until wisdom is handled.

---

## Standard Stack

### Core (Already Implemented)
| Component | Location | Purpose |
|-----------|----------|---------|
| `learning-approve-proposals` | `aether-utils.sh:4723` | Full approval workflow: display, select, promote, defer, undo |
| `learning-display-proposals` | `aether-utils.sh:4286` | Checkbox UI with threshold bars |
| `learning-check-promotion` | `aether-utils.sh:4232` | Filter observations meeting thresholds |
| `learning-defer-proposals` | `aether-utils.sh:4680` | Move unselected to deferred file |
| `learning-select-proposals` | `aether-utils.sh:4451` | Interactive selection with confirmation |
| `queen-promote` | `aether-utils.sh:3516` | Add wisdom to QUEEN.md |

### Data Files
| File | Purpose | Location |
|------|---------|----------|
| `learning-observations.json` | Pending wisdom proposals | `.aether/data/` |
| `learning-deferred.json` | User-deferred proposals | `.aether/data/` |
| `QUEEN.md` | Eternal wisdom storage | `.aether/docs/` |

---

## Architecture Patterns

### Pattern 1: Blocking Approval Flow
**What:** Pause command execution for user approval before proceeding with irreversible ceremony steps.

**When to use:** Lifecycle boundaries where wisdom promotion is permanent and should be validated.

**Implementation in seal.md:**
```
Step 3: Confirmation (existing)
Step 3.5: Wisdom Approval (NEW - blocking)
  - Call learning-approve-proposals
  - Block until user approves/deferred
  - Show "No proposals" message if empty
Step 4: Promote Colony Wisdom (existing - now streamlined)
Step 5: Update Milestone (existing)
```

**Implementation in entomb.md:**
```
Step 3: User Confirmation (existing)
Step 3.5: Wisdom Approval (NEW - blocking)
  - Same flow as seal.md
Step 3.75: Check XML Tools (existing)
Step 4: Promote Wisdom to QUEEN.md (existing - now streamlined)
Step 5: Generate Chamber Name (existing)
```

### Pattern 2: Shared Function Approach
**What:** Both seal.md and entomb.md call the same approval workflow.

**Why:** Consistent UX, single code path to maintain.

**Code:**
```bash
# Check for pending proposals
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  # Run approval workflow
  bash .aether/aether-utils.sh learning-approve-proposals
fi
```

### Pattern 3: Graceful Empty State
**What:** If no proposals exist, show message and continue.

**Why:** Don't block ceremony for colonies with no accumulated wisdom.

**Code:**
```bash
if [[ "$proposal_count" -eq 0 ]]; then
  echo "No wisdom proposals to review."
  echo "Proceeding with ceremony..."
fi
```

### Pattern 4: Deferred Persistence
**What:** Unselected proposals remain in `learning-deferred.json` for future sessions.

**Why:** User can defer decisions without losing the proposals.

**Note:** `learning-approve-proposals` handles this automatically — deferred items are written to `learning-deferred.json` and can be reviewed later with `--deferred` flag.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Approval UI | Custom prompt code | `learning-approve-proposals` | Already has checkbox UI, threshold bars, preview/confirm, undo support |
| Proposal display | Echo statements | `learning-display-proposals` | Consistent formatting, color support, threshold visualization |
| Threshold checking | Manual jq queries | `learning-check-promotion` | Centralized threshold logic per wisdom type |
| Deferral tracking | Custom file format | `learning-defer-proposals` | Integrated with approval workflow |
| QUEEN.md updates | Direct file edits | `queen-promote` | Maintains metadata, stats, evolution log |

---

## Common Pitfalls

### Pitfall 1: Non-Blocking Wisdom Flow
**What goes wrong:** Wisdom approval runs but doesn't block ceremony progression. User can accidentally skip wisdom review.

**Why it happens:** Integration placed after ceremony steps or without checking return status.

**How to avoid:**
- Place wisdom approval BEFORE milestone update (seal) and chamber creation (entomb)
- Check return status of `learning-approve-proposals`
- Don't proceed to next step until approval workflow completes

**Warning signs:** User reports wisdom "appeared but ceremony continued anyway"

### Pitfall 2: Duplicate Promotion
**What goes wrong:** Same wisdom promoted twice — once in continue.md, again in seal.md.

**Why it happens:** Observations not cleared after successful promotion.

**How to avoid:**
- `learning-approve-proposals` removes promoted items from `learning-observations.json`
- Verify this cleanup happens correctly
- Check that promoted items don't reappear

### Pitfall 3: Missing Deferred Highlighting
**What goes wrong:** User can't distinguish new proposals from previously deferred ones.

**Why it happens:** `learning-display-proposals` doesn't mark deferred items visually.

**How to avoid:**
- Check if proposal exists in `learning-deferred.json`
- Add visual indicator (e.g., "[deferred]" tag or different color)
- Document the highlighting scheme

**Claude's discretion area:** Exact highlighting format is open — could be emoji, color, or text tag.

### Pitfall 4: Silent Failures in Wisdom Extraction
**What goes wrong:** `learning-approve-proposals` fails but ceremony continues.

**Why it happens:** Error output not checked, command wrapped in `|| true` or similar.

**How to avoid:**
- Capture exit status explicitly
- On failure: display error, pause ceremony, offer retry/skip options
- Log error to activity log

### Pitfall 5: Breaking Existing Seal/Entomb Flow
**What goes wrong:** New wisdom step interferes with existing ceremony logic.

**Why it happens:** Integration step numbering conflicts, variable name collisions.

**How to avoid:**
- Use decimal step numbers (3.5, 3.75) to avoid renumbering
- Don't rename existing variables
- Test full ceremony flow end-to-end

---

## Code Examples

### Example 1: Wisdom Approval Integration (seal.md)

```bash
### Step 3.5: Wisdom Approval (NEW)

# Check for pending proposals
proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   🧠 WISDOM REVIEW"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "Before sealing, review wisdom proposals from this colony."
  echo ""

  # Run approval workflow (blocking)
  bash .aether/aether-utils.sh learning-approve-proposals

  echo ""
  echo "Wisdom review complete. Proceeding with sealing ceremony..."
  echo ""
else
  echo "No wisdom proposals to review."
fi
```

### Example 2: Wisdom Approval Integration (entomb.md)

```bash
### Step 3.5: Wisdom Approval (NEW)

# Same implementation as seal.md
# Placed after user confirmation, before XML tool check

proposals=$(bash .aether/aether-utils.sh learning-check-promotion 2>/dev/null || echo '{"proposals":[]}')
proposal_count=$(echo "$proposals" | jq '.proposals | length')

if [[ "$proposal_count" -gt 0 ]]; then
  echo ""
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo "   🧠 FINAL WISDOM REVIEW"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
  echo "Before archiving, review wisdom proposals from this colony."
  echo ""

  bash .aether/aether-utils.sh learning-approve-proposals

  echo ""
  echo "Wisdom review complete. Proceeding with entombment..."
  echo ""
else
  echo "No wisdom proposals to review."
fi
```

### Example 3: Error Handling Pattern

```bash
# Run approval workflow with error capture
approval_result=$(bash .aether/aether-utils.sh learning-approve-proposals 2>&1)
approval_exit=$?

if [[ $approval_exit -ne 0 ]]; then
  echo "⚠️  Wisdom review encountered an error:"
  echo "$approval_result"
  echo ""
  echo "You can:"
  echo "  1. Retry wisdom review"
  echo "  2. Skip wisdom review and continue with ceremony"
  echo "  3. Cancel and investigate"

  # Ask user choice...
fi
```

---

## State of the Art

### Current Implementation (Pre-Phase 35)

| Command | Current Wisdom Handling | Gap |
|---------|------------------------|-----|
| `continue.md` | Calls `learning-approve-proposals` at Step 2.1.5 | Has approval UX |
| `seal.md` | Auto-promotes validated learnings at Step 4 | No user approval |
| `entomb.md` | Auto-promotes validated learnings at Step 4 | No user approval |

### What Changes

**seal.md Step 4** (existing auto-promotion):
- Currently extracts and promotes without approval
- Will be preceded by Step 3.5 with approval workflow
- Existing Step 4 can be simplified or removed (redundant)

**entomb.md Step 4** (existing auto-promotion):
- Same pattern as seal.md
- Preceded by Step 3.5 with approval workflow

---

## Open Questions

1. **Deferred highlighting format**
   - What we know: Need to distinguish new vs deferred proposals
   - What's unclear: Exact visual format (emoji, color, text)
   - Recommendation: Use "[deferred]" text tag for clarity, or 🕐 emoji

2. **Error handling strategy**
   - What we know: Should handle failures gracefully
   - What's unclear: Whether to block ceremony on wisdom errors
   - Recommendation: Log error, notify user, offer skip option (don't block permanently)

3. **Existing auto-promotion logic**
   - What we know: seal.md and entomb.md have Step 4 auto-promotion
   - What's unclear: Whether to remove or keep as fallback
   - Recommendation: Remove auto-promotion — approval workflow replaces it

---

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh:4723-4919` — `learning-approve-proposals` implementation
- `.aether/aether-utils.sh:4286-4449` — `learning-display-proposals` implementation
- `.aether/aether-utils.sh:4232-4284` — `learning-check-promotion` implementation
- `.claude/commands/ant/continue.md:678-725` — Existing approval integration (Step 2.1.5)
- `.claude/commands/ant/seal.md:121-196` — Current auto-promotion logic (Step 4)
- `.claude/commands/ant/entomb.md:171-253` — Current auto-promotion logic (Step 4)

### Secondary (MEDIUM confidence)
- `.planning/phases/35-lifecycle-integration/35-CONTEXT.md` — User decisions and constraints
- `.planning/RESEARCH/ARCHITECTURE.md` — Wisdom system architecture overview
- `.planning/RESEARCH/PITFALLS.md` — Known pitfalls for wisdom system

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All functions exist and are tested
- Architecture: HIGH — Clear integration points identified
- Pitfalls: MEDIUM-HIGH — Based on existing patterns, but new integration always carries risk

**Research date:** 2026-02-21
**Valid until:** 2026-03-21 (30 days — stable system)

---

## RESEARCH COMPLETE

**Phase:** 35 - Lifecycle Integration
**Confidence:** HIGH

### Key Findings

1. **Infrastructure exists:** `learning-approve-proposals` provides complete approval workflow — display, selection, promotion, deferral, undo

2. **Integration points clear:**
   - seal.md: Add Step 3.5 between confirmation (Step 3) and milestone update (Step 4)
   - entomb.md: Add Step 3.5 between confirmation (Step 3) and XML check (Step 3.75)

3. **Existing auto-promotion redundant:** seal.md and entomb.md Step 4 auto-promotion can be removed — approval workflow replaces it

4. **Blocking behavior required:** User must approve/defer before ceremony proceeds — this is the core requirement

5. **Empty state handling:** Show "No wisdom proposals to review" and continue — don't block ceremony for empty colonies

### File Created
`.planning/phases/35-lifecycle-integration/35-RESEARCH.md`

### Confidence Assessment
| Area | Level | Reason |
|------|-------|--------|
| Standard stack | HIGH | All functions implemented and tested in continue.md |
| Architecture | HIGH | Clear integration points, existing patterns to follow |
| Pitfalls | MEDIUM-HIGH | Known patterns, but integration testing required |

### Open Questions
- Deferred highlighting format (Claude's discretion)
- Exact error handling strategy (Claude's discretion)
- Whether to remove or modify existing auto-promotion (recommend: remove)

### Ready for Planning
Research complete. Planner can now create PLAN.md files.
