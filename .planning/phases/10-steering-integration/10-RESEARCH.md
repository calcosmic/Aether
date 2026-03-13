# Phase 10: Steering Integration - Research

**Researched:** 2026-03-13
**Domain:** Pheromone signal integration into oracle research loop, configurable search strategy, focus area prioritization, bash orchestration, AI prompt engineering
**Confidence:** HIGH

## Summary

Phase 10 adds mid-session steering to the oracle research loop via the existing pheromone signal system (FOCUS/REDIRECT/FEEDBACK) and introduces configurable search strategy (breadth-first, depth-first, adaptive) set in the wizard before research begins. The core insight is that all the infrastructure already exists on both sides: the pheromone system has `pheromone-read` returning JSON arrays of active signals with type, content, and effective_strength, and the oracle loop has a clear between-iteration gap (lines 630-714 of oracle.sh) where signal reading naturally fits. The work is wiring these two existing systems together -- not building either from scratch.

The oracle loop currently checks three things between iterations: stop file, phase transition, and convergence. Phase 10 adds a fourth check: read pheromone signals from `.aether/data/pheromones.json` via the `pheromone-read` subcommand and inject them into the prompt. FOCUS signals adjust question prioritization (targeting focused areas instead of lowest-confidence). REDIRECT signals act as hard constraints prepended to the phase directive. FEEDBACK signals provide gentle behavioral guidance. The signal content is injected into the prompt via a new `build_steering_directive` function that formats active signals into prompt text, prepended to the existing phase directive in `build_oracle_prompt`.

The wizard gains two new questions: search strategy selection (breadth-first / depth-first / adaptive) and optional focus areas. Strategy is stored in state.json and used by `build_oracle_prompt` to adjust the phase directive emphasis. Focus areas are stored in state.json and also emitted as FOCUS pheromone signals so they work through the same channel as runtime steering. This means the wizard's initial focus and runtime `/ant:focus` commands converge to the same mechanism -- clean and composable.

**Primary recommendation:** Add a `read_steering_signals` function to oracle.sh that calls `pheromone-read` and formats active signals as prompt directives. Insert the call between stop-file check and AI invocation. Extend the wizard with strategy and focus questions. Store strategy in state.json. Inject steering into the prompt via `build_oracle_prompt` modifications.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
None explicitly locked -- all decisions delegated to Claude's discretion.

### Claude's Discretion
- **Signal timing and delivery** -- When the oracle checks for signals between iterations, how quickly they take effect, and how in-progress work is handled when a signal arrives
- **Strategy selection UX** -- How the user picks breadth-first / depth-first / adaptive in the wizard, whether strategy can change mid-session, and what each strategy feels like in practice
- **Focus area behavior** -- How specific focus areas can be, how visibly the oracle shifts priorities, and how conflicts between focus signals and current progress are resolved
- **Signal feedback to user** -- How the user knows their signal was received and acted on, what confirmation or status changes they see between iterations

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| STRC-01 | User can steer research mid-session via pheromone signals (FOCUS/REDIRECT/FEEDBACK) read between iterations | The `read_steering_signals` function in oracle.sh calls `pheromone-read` between iterations and formats active signals as a steering directive prepended to the phase prompt. FOCUS signals adjust question targeting, REDIRECT signals add hard constraints, FEEDBACK signals provide behavioral guidance. The AI receives the directive before its next iteration and acts on it. Signal acknowledgment is logged to the iteration output. |
| STRC-02 | Configurable search strategy in wizard: breadth-first, depth-first, or adaptive | The oracle wizard (oracle.md commands for Claude and OpenCode) adds a new question asking the user to select a search strategy. The choice is stored in state.json as `strategy` field. The `build_oracle_prompt` function reads the strategy and adjusts phase directive emphasis accordingly. Breadth-first extends survey phase behavior, depth-first extends investigate behavior, adaptive uses the existing phase-driven approach. |
| STRC-03 | Configurable focus areas to prioritize certain aspects of the research | The wizard adds an optional focus areas question. Focus areas are stored in state.json as `focus_areas` array and also emitted as FOCUS pheromone signals via `pheromone-write`. This means initial focus and runtime focus signals converge to the same mechanism. The steering directive includes focus areas in the prompt, instructing the AI to prioritize questions related to the focused areas. |
</phase_requirements>

## Standard Stack

### Core
| Library/Tool | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| oracle.sh | Bash script (723 lines) | Add read_steering_signals, modify build_oracle_prompt, add strategy handling | Existing orchestrator; all steering logic lives here |
| oracle.md | Prompt file (155 lines) | Add steering response instructions | Existing prompt; AI must know how to respond to steering directives |
| aether-utils.sh | ~9,808 lines | pheromone-read subcommand (existing), validate-oracle-state updates | Existing; pheromone-read returns JSON with active signals and effective_strength |
| jq | 1.6+ | Signal formatting, state.json field access, strategy field validation | Project standard; already used throughout oracle.sh and pheromone subcommands |
| ava | ^6.0.0 | Unit tests for steering functions | Project standard test runner |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| pheromone-write | aether-utils.sh | Emit FOCUS signals from wizard focus areas | During wizard setup (Step 2) to convert initial focus areas into pheromone signals |
| pheromone-read | aether-utils.sh | Read active signals with decay-adjusted strength | Called by read_steering_signals between iterations |
| bash test framework | tests/bash/test-helpers.sh | Bash integration tests for steering functions | Testing read_steering_signals and strategy handling |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Reading pheromones.json via pheromone-read | Reading pheromones.json directly with jq | pheromone-read handles decay calculation, expiry filtering, and strength adjustment; direct jq would duplicate this logic |
| Storing strategy in state.json | Separate strategy.json file | Extra file; strategy is session metadata that belongs with other session metadata in state.json |
| Emitting focus areas as FOCUS pheromones | Separate focus_areas field only in state.json | Using pheromones means wizard focus and runtime focus converge to one mechanism; the oracle reads signals from one source |
| Formatting signals as prompt text | Injecting raw JSON signal data into prompt | Formatted text is more reliable for AI comprehension; raw JSON wastes tokens and risks misinterpretation |

## Architecture Patterns

### Recommended Project Structure

No new source files created -- all changes modify existing files:

```
.aether/oracle/
  oracle.sh           # MODIFY: add read_steering_signals, modify build_oracle_prompt for strategy/steering
  oracle.md           # MODIFY: add steering response instructions (how AI should respond to steering directives)
  state.json          # SCHEMA EXTEND: add strategy and focus_areas fields

.aether/aether-utils.sh  # MODIFY: update validate-oracle-state to accept new state.json fields

.claude/commands/ant/oracle.md     # MODIFY: add wizard questions for strategy and focus areas
.opencode/commands/ant/oracle.md   # MODIFY: mirror wizard changes (parity)

tests/
  unit/oracle-steering.test.js     # NEW: ava tests for steering signal reading and strategy handling
  bash/test-oracle-steering.sh     # NEW: bash integration tests for steering functions
```

### Pattern 1: Between-Iteration Signal Reading

**What:** A function that reads pheromone signals between iterations and formats them as a steering directive prepended to the AI prompt.

**When to use:** Called at the start of each iteration, after the stop-file check and before the AI invocation.

**Design:**

The oracle loop currently has this structure at each iteration:
1. Check stop file
2. Pre-iteration backup
3. Build prompt and run AI
4. Validate JSON
5. Increment iteration, check phase transition
6. Update convergence metrics
7. Check diminishing returns and convergence

Steering signal reading inserts between steps 1 and 2 (or equivalently, between 1 and 3). The function:

1. Calls `pheromone-read` to get active signals as JSON
2. Filters for oracle-relevant signals (all types are relevant, but REDIRECT is highest priority)
3. Formats signals into a steering directive block
4. Returns the directive text, which `build_oracle_prompt` incorporates before the phase directive

The placement before the AI call means signals take effect on the very next iteration after they are emitted. There is no delay -- if a user runs `/ant:focus "security"` while the oracle is between iterations, the next iteration will see it.

**Insertion point in oracle.sh main loop (line ~640, after stop-file check):**

```bash
# Read steering signals from pheromones
STEERING_DIRECTIVE=$(read_steering_signals "$AETHER_ROOT")

# Run AI with phase-aware prompt (directive + steering + oracle.md)
OUTPUT=$(build_oracle_prompt "$STATE_FILE" "$SCRIPT_DIR/oracle.md" "$STEERING_DIRECTIVE" | $AI_CMD 2>&1 | tee /dev/stderr) || true
```

### Pattern 2: Steering Directive Formatting

**What:** Converting raw pheromone JSON into prompt-friendly text that the AI can act on.

**When to use:** Inside `read_steering_signals` to produce the directive text.

**Design:**

The directive follows the same pattern as the existing phase directives (survey/investigate/synthesize/verify) -- a markdown block prepended to the prompt. This keeps the prompt architecture consistent: phase directive + steering directive + oracle.md base prompt.

**Output format:**

```markdown
## Active Steering Signals

**REDIRECT (Hard constraints -- MUST follow):**
- [0.9] "Do not use deprecated API endpoints -- use v3 API only"

**FOCUS (Prioritize these areas):**
- [0.8] "Security implications of token storage"
- [0.7] "Performance under high concurrency"

**FEEDBACK (Adjust approach):**
- [0.7] "Findings are too abstract -- include concrete code examples"

When selecting your target question, PRIORITIZE questions related to focus areas
above. If a REDIRECT signal conflicts with your planned approach, the REDIRECT
takes precedence -- adjust your research direction accordingly.

---
```

The format mirrors the existing `pheromone-prime` output (see aether-utils.sh lines 7470-7530) but is simpler -- no instincts section, just the three signal types relevant to oracle steering.

### Pattern 3: Strategy-Modified Phase Directives

**What:** Adjusting the phase directive text based on the configured search strategy.

**When to use:** In `build_oracle_prompt` when emitting the phase directive.

**Design:**

The three strategies map to behavioral modifications of the existing phase system:

| Strategy | survey phase | investigate phase | synthesize phase | verify phase |
|----------|-------------|-------------------|------------------|-------------|
| **breadth-first** | Extended: aim for ALL questions touched before moving on; lower per-question confidence target (15-25%) | Targets breadth: cycle through questions round-robin rather than deepening one | Normal | Normal |
| **depth-first** | Shortened: touch each question briefly, then move to investigate faster | Extended: deep single-question focus; push to 85%+ before moving to next question | Normal | Normal |
| **adaptive** | Normal (current behavior) | Normal (current behavior) | Normal (current behavior) | Normal (current behavior) |

The "adaptive" strategy is the current behavior -- phase transitions driven purely by structural metrics. Breadth-first and depth-first modify the phase directive emphasis by appending a strategy clause to the phase heredoc.

**Implementation approach:** Rather than duplicating the four phase directives three times (12 variants), add a strategy-specific suffix line to the existing phase directive:

```bash
# In build_oracle_prompt, after phase directive:
case "$strategy" in
  breadth-first)
    echo "STRATEGY: Breadth-first -- prioritize covering ALL questions before going deep on any single one."
    ;;
  depth-first)
    echo "STRATEGY: Depth-first -- pick ONE question and investigate it exhaustively before moving to the next."
    ;;
  *)
    # adaptive: no modifier needed, phase system handles it
    ;;
esac
```

### Pattern 4: Signal Acknowledgment in Iteration Output

**What:** Showing the user that their signals were received and acted on.

**When to use:** In the iteration header output, between "Iteration X of Y" and the AI call.

**Design:**

When active steering signals exist, print a brief acknowledgment:

```
---------------------------------------------------------------
  Iteration 5 of 15
  Steering: 1 FOCUS, 1 REDIRECT, 0 FEEDBACK signals active
  Strategy: depth-first
---------------------------------------------------------------
```

This gives the user watching the tmux session (or reading the log) immediate confirmation that their signals are being read. No additional confirmation mechanism is needed -- the pheromone-read call itself is the acknowledgment, and the iteration output is the feedback channel.

### Anti-Patterns to Avoid

- **Reading pheromones.json directly with jq in oracle.sh:** The pheromone-read subcommand handles decay calculation, expiry filtering, and strength normalization. Duplicating this logic in oracle.sh would be fragile and would diverge from the canonical pheromone reading behavior.

- **Making steering override phase transitions:** Steering adjusts emphasis within a phase, not the phase itself. Phase transitions remain driven by structural metrics (determine_phase). A FOCUS signal saying "go deeper on question 3" should not prevent the oracle from transitioning to synthesize phase when metrics indicate it.

- **Polling pheromones during the AI call:** The AI call can take minutes. Do not read pheromones during the call or try to interrupt it. Read once before the call, inject into the prompt, and let the iteration complete. The next iteration will read any new signals.

- **Storing signal state in the oracle directory:** Pheromone signals live in `.aether/data/pheromones.json` (the colony pheromone store). The oracle reads from there but does not maintain its own signal state. This keeps the system composable -- the same signals that guide colony builds also guide oracle research.

- **Strategy that fights phase transitions:** Do not let strategy selection prevent phase advancement. A depth-first strategy should modify how investigate phase works (deeper single-question focus) but should not block the transition to synthesize when metrics warrant it. Strategy is emphasis, not override.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Signal decay and expiry filtering | Custom jq decay queries in oracle.sh | `pheromone-read` subcommand | pheromone-read already handles decay rates (FOCUS=30d, REDIRECT=60d, FEEDBACK=90d), expiry checking, and strength normalization; reimplementing this would diverge from colony behavior |
| Signal writing for wizard focus areas | Direct JSON manipulation of pheromones.json | `pheromone-write` subcommand | pheromone-write handles ID generation, validation, locking, and backward-compatibility with constraints.json |
| JSON state file validation | Ad-hoc jq checks | `validate-oracle-state` subcommand extension | Existing validation infrastructure; just add the new fields (strategy, focus_areas) |
| Signal formatting for prompts | New formatting function from scratch | Follow `pheromone-prime` output pattern | pheromone-prime already formats signals into prompt-ready text; use the same structure but simplified for oracle context |

**Key insight:** The pheromone infrastructure is mature (500+ lines across pheromone-read, pheromone-write, pheromone-prime, pheromone-display). Oracle steering should consume this infrastructure, not recreate it.

## Common Pitfalls

### Pitfall 1: Signal Stacking Overwhelms the Prompt
**What goes wrong:** User emits 10+ FOCUS signals, flooding the steering directive and confusing the AI with competing priorities.
**Why it happens:** No limit on how many signals are injected into the prompt.
**How to avoid:** Cap the number of signals injected. Use the same approach as `pheromone-prime --compact` which limits to top N signals by priority and strength. Recommend max 3 FOCUS + 2 REDIRECT + 2 FEEDBACK signals in the steering directive. Take highest-strength signals when count exceeds limits.
**Warning signs:** Steering directive exceeds 500 tokens; AI output ignores focus signals; research becomes unfocused.

### Pitfall 2: Strategy Prevents Phase Advancement
**What goes wrong:** Depth-first strategy keeps the oracle in investigate phase forever because it pushes each question to 85%+ before moving on, but the structural metrics for phase transition are based on average confidence across all questions.
**Why it happens:** Strategy modifies AI behavior but phase transitions are driven by structural metrics in oracle.sh.
**How to avoid:** Strategy is an emphasis modifier on the phase directive, not an override of the phase transition function (`determine_phase`). The investigate phase directive with depth-first emphasis tells the AI to go deep on one question, but the oracle still transitions to synthesize when avg confidence >= 60% or fewer than 2 questions below 50%.
**Warning signs:** Oracle stuck in one phase for many iterations despite high per-question confidence on touched questions.

### Pitfall 3: Focus Signals Conflict with Convergence
**What goes wrong:** User emits FOCUS on area A, but the oracle is close to convergence on everything except area B. The FOCUS signal pulls the AI toward A (which is already well-researched) instead of B (which needs work).
**Why it happens:** FOCUS signals override the default lowest-confidence-first targeting.
**How to avoid:** The steering directive should instruct the AI to "prioritize focus areas among open/partial questions" not "only work on focus areas." If no open questions match the focus area, fall back to default lowest-confidence targeting. This makes focus a preference, not a hard constraint. Only REDIRECT is a hard constraint.
**Warning signs:** AI repeatedly targets already-answered questions because they match focus area.

### Pitfall 4: Wizard Focus Areas Persist as Stale Pheromones
**What goes wrong:** Focus areas set in the wizard persist as pheromone signals across oracle sessions because pheromones.json is not session-scoped.
**Why it happens:** Wizard emits FOCUS pheromones with default TTL (phase_end), but oracle sessions are not colony phases.
**How to avoid:** Wizard-emitted focus pheromones should use a session-specific TTL or a tag identifying them as oracle-session signals. Alternatively, the oracle can emit signals with a custom source (e.g., "oracle:wizard") and the steering reader can filter for these specifically. The simplest approach: emit wizard focus signals with a wall-clock TTL matching the expected session duration (e.g., `--ttl 24h`) and `--source "oracle:wizard"`.
**Warning signs:** New oracle session picks up focus areas from a previous unrelated session.

### Pitfall 5: pheromone-read Failure Crashes the Oracle
**What goes wrong:** If `pheromone-read` fails (pheromones.json missing, corrupted, etc.), the oracle loop crashes.
**Why it happens:** Signal reading is not graceful about failures.
**How to avoid:** Wrap the pheromone-read call in error handling. If it fails, proceed with an empty steering directive. Signal reading is optional enhancement, not required for the oracle loop to function. Use `|| echo '[]'` pattern matching how other optional reads work in oracle.sh.
**Warning signs:** Oracle crashes on first iteration in a fresh project without pheromones.json.

## Code Examples

Verified patterns from the existing codebase:

### Calling pheromone-read from bash (existing pattern)
```bash
# Source: .aether/aether-utils.sh lines 7095-7173
# pheromone-read returns JSON with active signals including effective_strength
signals=$(bash "$AETHER_ROOT/.aether/aether-utils.sh" pheromone-read 2>/dev/null || echo '[]')
# Result format: [{"id":"sig_focus_...", "type":"FOCUS", "content":{"text":"..."}, "effective_strength":0.75, ...}]
```

### Formatting signals for prompt injection (following pheromone-prime pattern)
```bash
# Source: .aether/aether-utils.sh lines 7470-7530 (pheromone-prime)
# Extract FOCUS signal texts sorted by strength
focus_lines=$(echo "$signals" | jq -r '
  map(select(.type == "FOCUS"))
  | sort_by(-.effective_strength)
  | .[:3]
  | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
' 2>/dev/null || echo "")
```

### Extending state.json schema (following Phase 6/9 pattern)
```json
{
  "version": "1.1",
  "topic": "...",
  "scope": "both",
  "strategy": "adaptive",
  "focus_areas": ["security implications", "performance"],
  "phase": "survey",
  "iteration": 0,
  "max_iterations": 15,
  "target_confidence": 95,
  "overall_confidence": 0,
  "started_at": "2026-03-13T00:00:00Z",
  "last_updated": "2026-03-13T00:00:00Z",
  "status": "active"
}
```

### Emitting wizard focus areas as pheromone signals (existing pattern)
```bash
# Source: .aether/aether-utils.sh line 3455 (phase-insert uses same pattern)
# For each focus area from the wizard:
bash "$AETHER_ROOT/.aether/aether-utils.sh" pheromone-write FOCUS "$focus_area" \
  --strength 0.8 --source "oracle:wizard" \
  --reason "Focus area set in oracle wizard" --ttl "24h" 2>/dev/null || true
```

### Strategy modifier in build_oracle_prompt (new, following existing phase directive pattern)
```bash
# Source pattern: oracle.sh lines 133-224 (build_oracle_prompt)
# After the phase directive case block, add strategy modifier:
strategy=$(jq -r '.strategy // "adaptive"' "$state_file" 2>/dev/null || echo "adaptive")
case "$strategy" in
  breadth-first)
    cat <<'STRATEGY'

STRATEGY NOTE: Breadth-first mode is active. Prioritize covering ALL questions
with initial findings before going deep on any single question. Aim for broad
coverage across the research plan. When multiple questions are untouched, target
the easiest-to-answer first for maximum coverage.

STRATEGY
    ;;
  depth-first)
    cat <<'STRATEGY'

STRATEGY NOTE: Depth-first mode is active. Pick the single most important open
question and investigate it exhaustively. Push confidence to 80%+ before moving
to the next question. Prefer thorough, well-sourced answers over broad coverage.

STRATEGY
    ;;
  *)
    # adaptive: no strategy modifier needed
    ;;
esac
```

### Updating validate-oracle-state for new fields (following Phase 9 pattern)
```bash
# Source: .aether/aether-utils.sh lines 1210-1231 (validate-oracle-state state)
# Add optional fields for strategy and focus_areas:
# After existing enum("status";["active","complete","stopped"]) check:
#   if has("strategy") then enum("strategy";["breadth-first","depth-first","adaptive"]) else "pass" end,
#   if has("focus_areas") then (if (.focus_areas | type) == "array" then "pass" else "fail: focus_areas not array" end) else "pass" end
```

### read_steering_signals function design
```bash
# New function for oracle.sh (place after build_oracle_prompt, before main loop)
read_steering_signals() {
  local aether_root="$1"
  local utils="$aether_root/.aether/aether-utils.sh"

  # Gracefully handle missing pheromone system
  if [ ! -f "$utils" ]; then
    echo ""
    return 0
  fi

  # Read active signals via pheromone-read (handles decay, expiry, filtering)
  local signals
  signals=$(bash "$utils" pheromone-read 2>/dev/null || echo '{"signals":[]}')

  # Extract the signals array from the json_ok wrapper
  local signal_array
  signal_array=$(echo "$signals" | jq -c '.result.signals // .signals // []' 2>/dev/null || echo '[]')

  local count
  count=$(echo "$signal_array" | jq 'length' 2>/dev/null || echo "0")

  if [ "$count" -eq 0 ]; then
    echo ""
    return 0
  fi

  # Format steering directive
  local directive=""

  # REDIRECT signals (highest priority -- hard constraints)
  local redirects
  redirects=$(echo "$signal_array" | jq -r '
    map(select(.type == "REDIRECT"))
    | sort_by(-.effective_strength)
    | .[:2]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [ -n "$redirects" ]; then
    directive+="**REDIRECT (Hard constraints -- MUST follow):**"$'\n'"$redirects"$'\n\n'
  fi

  # FOCUS signals (prioritization)
  local focuses
  focuses=$(echo "$signal_array" | jq -r '
    map(select(.type == "FOCUS"))
    | sort_by(-.effective_strength)
    | .[:3]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [ -n "$focuses" ]; then
    directive+="**FOCUS (Prioritize these areas):**"$'\n'"$focuses"$'\n\n'
  fi

  # FEEDBACK signals (behavioral adjustment)
  local feedbacks
  feedbacks=$(echo "$signal_array" | jq -r '
    map(select(.type == "FEEDBACK"))
    | sort_by(-.effective_strength)
    | .[:2]
    | .[] | "- [" + ((.effective_strength * 100 | floor | tostring)) + "%] \"" + (.content.text // "") + "\""
  ' 2>/dev/null || echo "")

  if [ -n "$feedbacks" ]; then
    directive+="**FEEDBACK (Adjust approach):**"$'\n'"$feedbacks"$'\n\n'
  fi

  if [ -n "$directive" ]; then
    echo "## Active Steering Signals"
    echo ""
    echo "$directive"
    echo "When selecting your target question, PRIORITIZE questions related to FOCUS areas."
    echo "REDIRECT signals are hard constraints -- adjust your approach to comply."
    echo "FEEDBACK signals are suggestions -- incorporate where appropriate."
    echo ""
    echo "---"
    echo ""
  fi
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| No mid-session control | .stop file for halting only | Phase 6 (oracle rewrite) | User can stop but not steer |
| Single research behavior | Phase-aware prompts (survey/investigate/synthesize/verify) | Phase 7 | AI behavior changes with research stage |
| Fixed iteration loop | Convergence detection with diminishing returns | Phase 8 | Loop stops intelligently |
| No source tracking | Source registry with trust scoring | Phase 9 | Claims are verifiable |
| **No steering (current)** | **Pheromone-based mid-session steering + configurable strategy** | **Phase 10 (this phase)** | **User can redirect, focus, and adjust research mid-flight** |

**Current state of oracle.sh functions (post Phase 8 + 9):**
- `generate_research_plan` -- generates research-plan.md from state/plan
- `determine_phase` -- structural phase transitions
- `build_oracle_prompt` -- phase directive + oracle.md
- `compute_convergence` -- gap resolution, coverage, novelty metrics
- `update_convergence_metrics` -- writes convergence to state.json
- `compute_trust_scores` -- source tracking metrics
- `check_convergence` -- composite score threshold check
- `detect_diminishing_returns` -- low-change iteration detection
- `validate_and_recover` -- JSON recovery from backups
- `build_synthesis_prompt` -- final synthesis pass prompt
- `run_synthesis_pass` -- synthesis invocation and cleanup
- `cleanup_and_synthesize` -- trap handler

**Phase 10 adds:**
- `read_steering_signals` -- new function reading pheromones for prompt injection
- `build_oracle_prompt` modifications -- accepts steering directive, adds strategy modifier
- Wizard extensions -- strategy and focus area questions
- state.json schema extension -- strategy and focus_areas fields

## Open Questions

1. **Should strategy be changeable mid-session?**
   - What we know: The wizard sets strategy before research begins. state.json stores it. build_oracle_prompt reads it each iteration.
   - What's unclear: Should the user be able to change strategy mid-session (e.g., `/ant:oracle strategy depth-first`)?
   - Recommendation: Allow it. Reading strategy from state.json each iteration means the user could manually edit state.json or a new subcommand could update it. But a dedicated command is out of scope for Phase 10 -- the field being in state.json makes it extensible later. For now, strategy is set once in the wizard.

2. **How to handle oracle-specific vs colony-wide pheromones?**
   - What we know: `pheromone-read` returns all active signals. Some may be colony-wide (e.g., "avoid runtime/ edits") and not relevant to oracle research. Others are oracle-specific (e.g., FOCUS on "security").
   - What's unclear: Should oracle.sh filter signals by source or tag?
   - Recommendation: Include all active signals. Colony-wide REDIRECTs are valid constraints even during research. FOCUS signals emitted by the wizard should use `--source "oracle:wizard"` for traceability, but filtering by source is unnecessary -- all active signals are relevant guidance.

3. **What happens when focus areas conflict with the research plan?**
   - What we know: The user might focus on "database performance" but the research topic is "React rendering optimization" with no database-related questions.
   - What's unclear: Should the oracle ignore irrelevant focus signals or try to incorporate them?
   - Recommendation: The steering directive tells the AI to "prioritize questions RELATED to focus areas." If no questions are related, the AI naturally falls back to default targeting (lowest-confidence). The prompt wording handles this gracefully without special logic.

## Sources

### Primary (HIGH confidence)
- **oracle.sh** (`.aether/oracle/oracle.sh`, 723 lines) -- Complete main loop, all existing functions, iteration lifecycle. Verified line by line.
- **oracle.md** (`.aether/oracle/oracle.md`, 155 lines) -- Current AI iteration prompt with phase awareness, confidence rubric, source tracking.
- **aether-utils.sh** (`.aether/aether-utils.sh`, ~9,808 lines) -- pheromone-read (lines 7095-7173), pheromone-write (lines 6771-6954), pheromone-prime (lines 7364-7540), pheromone-display (lines 6980-7093), validate-oracle-state (lines 1203-1289).
- **pheromones.json** (`.aether/data/pheromones.json`) -- Live pheromone data showing signal structure.
- **pheromones.md** (`.aether/docs/pheromones.md`) -- Pheromone user guide with signal types, TTL, decay.
- **oracle wizard** (`.claude/commands/ant/oracle.md`, `.opencode/commands/ant/oracle.md`) -- Wizard flow, questions, file creation.

### Secondary (MEDIUM confidence)
- **Phase 6 research** (`.planning/phases/06-state-architecture-foundation/06-RESEARCH.md`) -- State file schemas, validation patterns.
- **Phase 7 plans** (`.planning/phases/07-iteration-prompt-engineering/07-01-PLAN.md`) -- Phase directive patterns, build_oracle_prompt design.
- **Phase 8 research** (`.planning/phases/08-orchestrator-upgrade/08-RESEARCH.md`) -- Convergence detection, main loop architecture.
- **Phase 9 research/plans** (`.planning/phases/09-source-tracking-and-trust-layer/09-RESEARCH.md`, `09-01-PLAN.md`) -- Schema extension patterns, backward compatibility approach.

### Tertiary (LOW confidence)
None -- all findings come from direct codebase inspection.

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH -- all tools and libraries are existing codebase components directly verified
- Architecture: HIGH -- integration points clearly identified in oracle.sh main loop and build_oracle_prompt; pheromone-read API verified
- Pitfalls: HIGH -- derived from understanding of signal decay, prompt token limits, and phase transition mechanics verified in code
- Code examples: HIGH -- patterns extracted from existing codebase (pheromone-prime, phase-insert, validate-oracle-state)

**Research date:** 2026-03-13
**Valid until:** 2026-04-13 (30 days -- stable codebase, patterns well-established)
