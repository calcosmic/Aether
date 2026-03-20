# Phase 44: Suggest Pheromones - Research

**Researched:** 2026-02-22
**Domain:** Aether Colony System / Code Analysis / Pheromone Integration
**Confidence:** HIGH

## Summary

This phase integrates **code pattern analysis** with the existing **pheromone system** to suggest FOCUS, REDIRECT, and FEEDBACK signals at build start. The implementation reuses established patterns: the tick-to-approve UI from `learning-approve-proposals`, pheromone writing via `pheromone-write`, and code analysis heuristics from the surveyor-pathogens agent.

**Primary recommendation:** Build a lightweight analysis engine that runs at build start, generates up to 5 pheromone suggestions based on code patterns, and presents them using the existing one-at-a-time approval UI. Approved suggestions are written as FOCUS signals; dismissed suggestions are dropped without logging.

---

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Suggestion Timing:**
- Show suggestions on **every build** (no skipping based on existing pheromones)
- Quick dismiss option to proceed to build without approving
- Exact timing point: Claude's Discretion (choose non-disruptive point)

**Suggestion Types:**
- All three pheromone types: FOCUS, REDIRECT, FEEDBACK
- Up to 5 suggestions at once
- One-by-one approval (same pattern as learning proposals)
- Dismissed suggestions just disappear (no logging or deferral)

**Analysis Inputs:**
- Analyze **code patterns** (not git changes or phase context)
- Look for: complexity hotspots, anti-patterns, change frequency — multiple signals
- File scope: Claude's Discretion
- Analysis sophistication: Claude's Discretion (recommend heuristic scoring)

**Frequency Control:**
- Always show suggestions (every build)
- Cap at 5 suggestions maximum per build
- Avoid duplicate suggestions within same session

### Claude's Discretion

- Exact timing point in build flow (non-disruptive)
- Which files to analyze (colony files, project code, or source only)
- Analysis algorithm sophistication (simple thresholds vs heuristic scoring)
- How to track "already suggested this session" state

### Deferred Ideas (OUT OF SCOPE)

None — discussion stayed within phase scope.

</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SUGG-01 | Show suggested pheromones with tick-to-approve at build start | Use `learning-approve-proposals` UI pattern from aether-utils.sh:5057-5328 |
| SUGG-02 | Suggestions based on codebase analysis | Reuse surveyor-pathogens analysis patterns; implement heuristic scoring |

</phase_requirements>

---

## Standard Stack

### Core (Aether System)
| Component | Location | Purpose | Why Standard |
|-----------|----------|---------|--------------|
| `pheromone-write` | `.aether/aether-utils.sh:5897-6057` | Write FOCUS/REDIRECT/FEEDBACK signals | Existing utility, handles TTL, strength, JSON format |
| `pheromone-count` | `.aether/aether-utils.sh:6059-6081` | Count active signals by type | Used for "Active signals" display |
| `learning-approve-proposals` | `.aether/aether-utils.sh:5057-5328` | One-at-a-time approval UI | Exact pattern to replicate for tick-to-approve |
| `session.json` | `.aether/data/session.json` | Session tracking | Add `suggested_pheromones` array to track "already suggested this session" |

### Supporting
| Component | Location | Purpose | When to Use |
|-----------|----------|---------|-------------|
| `pheromones.json` | `.aether/data/pheromones.json` | Signal storage | Read to avoid duplicating existing active signals |
| `surveyor-pathogens` | `.claude/agents/ant/aether-surveyor-pathogens.md` | Code analysis patterns | Reference for complexity heuristics |

---

## Architecture Patterns

### Pattern 1: Build Hook Integration
**What:** Insert suggestion check at a non-disruptive point in build.md flow
**When to use:** Before worker spawning but after state validation
**Recommended location:** After Step 4 (Load Colony Context) and before Step 5 (Initialize Swarm Display)

**Rationale:**
- State is loaded, pheromones are already displayed to user
- User sees current signals before getting suggestions for new ones
- Non-blocking: dismissed suggestions allow immediate continuation to Step 5

### Pattern 2: Heuristic Scoring for Code Analysis
**What:** Simple threshold-based analysis for complexity hotspots and anti-patterns
**When to use:** When analyzing files for suggestion candidates

**Signals to detect:**
| Signal | Detection Method | Suggestion Type |
|--------|------------------|-----------------|
| Large files | `wc -l > 300` | FOCUS "Large file: consider refactoring" |
| TODO/FIXME comments | `grep -rn "TODO\|FIXME\|XXX"` | FEEDBACK "N pending TODOs in modified files" |
| High complexity | `grep -c "^function\|^def" > 20` | FOCUS "Complex module: test carefully" |
| Anti-patterns | `grep -rn "console.log\|debugger"` | REDIRECT "Remove debug artifacts before commit" |
| Type safety gaps | `grep -rn ": any\|: unknown"` | FEEDBACK "Type safety gaps detected" |
| Test coverage gaps | Files without corresponding `.test.` files | FOCUS "Add tests for uncovered modules" |

### Pattern 3: One-At-A-Time Approval UI
**What:** Display suggestions individually with [A]pprove/[R]eject/[S]kip options
**When to use:** Replicate from `learning-approve-proposals`

**Key implementation details from source:**
```bash
# From aether-utils.sh:5167-5218
echo "───────────────────────────────────────────────────"
echo "Proposal $((i+1)) of $proposal_count"
echo "───────────────────────────────────────────────────"
echo ""
echo "$emoji $name (observed $count time(s), threshold: $threshold)"
echo ""
echo "$content"
echo ""
echo "───────────────────────────────────────────────────"
echo -n "[A]pprove  [R]eject  [S]kip  Your choice: "
read -r choice
```

**Adaptation for pheromone suggestions:**
- Emoji per pheromone type: FOCUS=🎯, REDIRECT=🚫, FEEDBACK=💬
- Show suggestion text and rationale
- On Approve: call `pheromone-write` with appropriate type
- On Reject/Skip: continue to next suggestion (no persistence)

### Pattern 4: Session-Based Deduplication
**What:** Track suggested items in session.json to avoid duplicates within session
**When to use:** Prevent showing same suggestion multiple times in one build session

**Implementation:**
```json
// Add to session.json structure
{
  "suggested_pheromones": [
    {
      "suggestion_hash": "sha256_of_content",
      "suggested_at": "2026-02-22T10:00:00Z",
      "type": "FOCUS"
    }
  ]
}
```

**Hash generation:** Combine file path + suggestion type + brief content for deduplication key.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Pheromone JSON handling | Custom JSON manipulation | `pheromone-write` utility | Handles schema, TTL parsing, backward compatibility with constraints.json |
| Approval UI | New interactive prompt | `learning-approve-proposals` pattern | Already handles [A]pprove/[R]eject/[S]kip, dry-run, verbose modes |
| Session tracking | Custom file | Extend `session.json` | Session system already manages lifecycle, stale detection |
| Code complexity analysis | Full AST parsing | Shell heuristics (wc -l, grep patterns) | Fast, no dependencies, sufficient for suggestions |

---

## Common Pitfalls

### Pitfall 1: Blocking the Build Flow
**What goes wrong:** Suggestion UI waits indefinitely for user input, breaking automated builds
**Why it happens:** Interactive prompts in headless environments hang
**How to avoid:**
- Add `--yes` flag support (auto-approve all)
- Add `--no-suggest` flag to skip suggestions entirely
- Check `stdin` is tty before interactive prompts

### Pitfall 2: Duplicate Suggestions Across Sessions
**What goes wrong:** Same "large file" suggestion appears every build, becoming noise
**Why it happens:** No persistence of what was already suggested
**How to avoid:**
- Hash suggestions and store in session.json
- Clear session suggestions on `/ant:init` or context clear
- Respect existing pheromones (don't suggest what's already active)

### Pitfall 3: Suggesting on Wrong Files
**What goes wrong:** Suggestions target node_modules, .aether system files, or generated code
**Why it happens:** Greedy file globbing without exclusion filters
**How to avoid:**
- Exclude: `node_modules/`, `.aether/`, `dist/`, `build/`, `*.min.js`
- Focus on: `src/`, project source directories
- Respect `.gitignore` patterns

### Pitfall 4: Too Many Suggestions
**What goes wrong:** 15+ suggestions overwhelm the user
**Why it happens:** No cap on suggestion generation
**How to avoid:**
- Hard cap at 5 suggestions (per user decision)
- Score suggestions by severity/priority, show top 5
- Prioritize: REDIRECT > FOCUS > FEEDBACK

---

## Code Examples

### Reading Active Pheromones (Avoid Duplicates)
```bash
# From aether-utils.sh:6069-6074
pc_result=$(jq -c '{
  focus: ([.signals[] | select(.active == true and .type == "FOCUS")] | length),
  redirect: ([.signals[] | select(.active == true and .type == "REDIRECT")] | length),
  feedback: ([.signals[] | select(.active == true and .type == "FEEDBACK")] | length)
}' "$pheromones_file")
```

### Writing a Suggested Pheromone (On Approve)
```bash
# Adapted from aether-utils.sh:5897-6057
bash .aether/aether-utils.sh pheromone-write FOCUS "Large file detected: $file" \
  --strength 0.7 \
  --source "system:suggestion" \
  --reason "Auto-suggested: file exceeds 300 lines, consider refactoring" \
  --ttl "phase_end"
```

### Heuristic Analysis Pattern
```bash
# Complexity detection (from surveyor-pathogens)
find src/ -name "*.ts" -o -name "*.js" | while read -r file; do
  lines=$(wc -l < "$file")
  if [[ $lines -gt 300 ]]; then
    echo "{\"type\":\"FOCUS\",\"file\":\"$file\",\"reason\":\"Large file ($lines lines)\"}"
  fi
done
```

### Session Tracking for Deduplication
```bash
# Check if already suggested this session
suggestion_hash=$(echo "$file:$type:$reason" | sha256sum | cut -d' ' -f1)
already_suggested=$(jq --arg hash "$suggestion_hash" \
  '.suggested_pheromones // [] | map(select(.suggestion_hash == $hash)) | length' \
  .aether/data/session.json)

if [[ "$already_suggested" -eq 0 ]]; then
  # Add to suggestions list
  suggestions+=("$suggestion_json")
fi
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual pheromone emission only | Auto-suggest based on code analysis | Phase 44 (this phase) | Reduces cognitive load, catches patterns user might miss |
| Learning proposals (post-build) | Pheromone suggestions (pre-build) | v1.1.0+ | Different timing, same UI pattern |

---

## Open Questions

1. **Should suggestions consider the current phase's goal?**
   - What we know: User decided "analyze code patterns, not phase context"
   - What's unclear: Whether to weight suggestions by phase relevance
   - Recommendation: Start with pure code analysis; phase context can be added later

2. **How to handle suggestion persistence across context clears?**
   - What we know: session.json survives context clears
   - What's unclear: Whether to clear suggestions on `/clear` or keep them
   - Recommendation: Clear on new `/ant:init` only; context clear preserves session

3. **Should users configure which heuristics are active?**
   - What we know: No user request for configuration yet
   - What's unclear: Whether hardcoded heuristics will satisfy all projects
   - Recommendation: Start with fixed set; add config if users request

---

## Sources

### Primary (HIGH confidence)
- `.aether/aether-utils.sh:5057-5328` — `learning-approve-proposals` implementation
- `.aether/aether-utils.sh:5897-6057` — `pheromone-write` implementation
- `.aether/aether-utils.sh:6083-6196` — `pheromone-display` implementation
- `.aether/docs/pheromones.md` — Pheromone system documentation
- `.claude/agents/ant/aether-surveyor-pathogens.md` — Code analysis patterns

### Secondary (MEDIUM confidence)
- `.claude/commands/ant/build.md` — Build flow for integration point selection
- `.claude/commands/ant/continue.md` — Reference for tick-to-approve UI usage
- `.aether/data/session.json` — Session structure for deduplication tracking

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — All components exist and are tested
- Architecture: HIGH — Clear integration point in build.md
- Pitfalls: MEDIUM — Based on similar feature patterns, not direct testing

**Research date:** 2026-02-22
**Valid until:** 2026-03-22 (30 days for stable Aether system)
