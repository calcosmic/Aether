### Step 5.4: Spawn Watcher for Verification

**MANDATORY: Always spawn a Watcher — testing must be independent.**

**Announce the verification wave:**
```
━━━ 👁️🐜 V E R I F I C A T I O N ━━━
──── 👁️🐜 Spawning {watcher_name} ────
```

Spawn the Watcher using Task tool with `subagent_type="aether-watcher"`, include `description: "👁️ Watcher {Watcher-Name}: Independent verification"` (DO NOT use run_in_background - task blocks until complete):

Run using the Bash tool with description "Dispatching watcher...": `bash .aether/aether-utils.sh spawn-log "Queen" "watcher" "{watcher_name}" "Independent verification" && bash .aether/aether-utils.sh swarm-display-update "{watcher_name}" "watcher" "observing" "Verification in progress" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "nursery" 50`

**Watcher Worker Prompt (CLEAN OUTPUT):**
```
You are {Watcher-Name}, a 👁️🐜 Watcher Ant.

Verify all work done by Builders in Phase {id}.

Files to verify:
- Created: {list from builder results}
- Modified: {list from builder results}

{ prompt_section }

**IMPORTANT:** When using the Bash tool for activity calls, always include a description parameter:
- activity-log calls → "Logging {action}..."
- swarm-display-update calls → "Updating build display..."
- pheromone-read calls → "Checking colony signals..."
- spawn-log calls → "Dispatching sub-worker..."

Use colony-flavored language, 4-8 words, trailing ellipsis.

Verification:
1. Check files exist (Read each)
2. Run build/type-check
3. Run tests if they exist
4. Check success criteria: {list}

Spawn sub-workers if needed:
- Log spawn using Bash tool with description
- Announce: "🐜 Spawning {child} to investigate {issue}"

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON:
{"ant_name": "{Watcher-Name}", "verification_passed": true|false, "files_verified": [], "issues_found": [], "quality_score": N, "tool_count": 0, "recommendation": "proceed|fix_required"}
```

### Step 5.5: Process Watcher Results

**Task call returns results directly (no TaskOutput needed).**

**Parse the Watcher's JSON response:** verification_passed, issues_found, quality_score, recommendation

**Display Watcher completion line:**

For successful verification:
```
👁️ {Watcher-Name}: Independent verification ({tool_count} tools) ✓
```

For failed verification:
```
👁️ {Watcher-Name}: Independent verification ✗ ({issues_found count} issues after {tool_count} tools)
```

**Store results for synthesis in Step 5.7**

**Update swarm display when Watcher completes:**
Run using the Bash tool with description "Recording watcher completion...": `bash .aether/aether-utils.sh swarm-display-update "{watcher_name}" "watcher" "completed" "Verification complete" "Queen" '{"read":3,"grep":2,"edit":0,"bash":1}' 100 "nursery" 100`

### Step 5.5.1: Measurer Performance Agent (Conditional)

**Conditional step — only runs for performance-sensitive phases.**

1. **Check if phase is performance-sensitive:**

   Extract phase name from COLONY_STATE.json (already loaded in Step 1). Check for performance keywords (case-insensitive):
   - "performance", "optimize", "latency", "throughput", "benchmark", "speed", "memory", "cpu", "efficiency"

   Run using the Bash tool with description "Checking phase for performance sensitivity...":
   ```bash
   phase_name="{phase_name_from_state}"
   performance_keywords="performance optimize latency throughput benchmark speed memory cpu efficiency"
   is_performance_sensitive="false"
   for keyword in $performance_keywords; do
     if [[ "${phase_name,,}" == *"$keyword"* ]]; then
       is_performance_sensitive="true"
       break
     fi
   done
   echo "{\"is_performance_sensitive\": \"$is_performance_sensitive\", \"phase_name\": \"$phase_name\"}"
   ```

   Parse the JSON result. If `is_performance_sensitive` is `"false"`:
   - Display: `📊 Measurer: Phase not performance-sensitive — skipping baseline measurement`
   - Skip to Step 5.6 (Chaos Ant)

2. **Check Watcher verification status:**

   Only spawn Measurer if Watcher verification passed (`verification_passed: true`). If Watcher failed:
   - Display: `📊 Measurer: Watcher verification failed — skipping performance measurement`
   - Skip to Step 5.6 (Chaos Ant)

3. **Generate Measurer name and dispatch:**

   Run using the Bash tool with description "Naming measurer...": `bash .aether/aether-utils.sh generate-ant-name "measurer"` (store as `{measurer_name}`)
   Run using the Bash tool with description "Dispatching measurer...": `bash .aether/aether-utils.sh spawn-log "Queen" "measurer" "{measurer_name}" "Performance baseline measurement" && bash .aether/aether-utils.sh swarm-display-update "{measurer_name}" "measurer" "benchmarking" "Performance baseline measurement" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "fungus_garden" 20`

   Display:
   ```
   ━━━ 📊🐜 M E A S U R E R ━━━
   ──── 📊🐜 Spawning {measurer_name} — establishing performance baselines ────
   📊 Measurer {measurer_name} spawning — Establishing performance baselines for {phase_name}...
   ```

4. **Get files to measure:**

   Use `files_created` and `files_modified` from builder results (already collected in synthesis preparation). Filter for source files only:
   - Include: `.js`, `.ts`, `.sh`, `.py` files
   - Exclude: `.test.js`, `.test.ts`, `.spec.js`, `.spec.ts`, `__tests__/`, config files

   Store filtered list as `{source_files_to_measure}`.

5. **Spawn Measurer using Task tool:**

   Spawn the Measurer using Task tool with `subagent_type="aether-measurer"`, include `description: "📊 Measurer {Measurer-Name}: Performance baseline measurement"` (DO NOT use run_in_background - task blocks until complete):

   # FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Measurer Ant - performance profiler that benchmarks and identifies bottlenecks."

   **Measurer Worker Prompt (CLEAN OUTPUT):**
   ```
   You are {Measurer-Name}, a 📊 Measurer Ant.

   Mission: Performance baseline measurement for Phase {id}

   Phase: {phase_name}
   Keywords that triggered spawn: {matched_keywords}

   Files to measure:
   - {list from source_files_to_measure}

   Work:
   1. Read each source file to understand operation patterns
   2. Analyze algorithmic complexity (Big O) for key functions
   3. Identify potential bottlenecks (loops, recursion, I/O)
   4. Document current baseline metrics for comparison
   5. Recommend optimizations with estimated impact

   **IMPORTANT:** You are strictly read-only. Do not modify any files.

   Log activity: bash .aether/aether-utils.sh activity-log "BENCHMARKING" "{Measurer-Name}" "description"

   Return ONLY this JSON (no other text):
   {
     "ant_name": "{Measurer-Name}",
     "caste": "measurer",
     "status": "completed" | "failed" | "blocked",
     "summary": "What you measured and found",
     "metrics": {
       "response_time_ms": 0,
       "throughput_rps": 0,
       "cpu_percent": 0,
       "memory_mb": 0
     },
     "baselines_established": [
       {"operation": "name", "complexity": "O(n)", "file": "path", "line": 0}
     ],
     "bottlenecks_identified": [
       {"description": "...", "severity": "high|medium|low", "location": "file:line"}
     ],
     "recommendations": [
       {"priority": 1, "change": "...", "estimated_improvement": "..."}
     ],
     "tool_count": 0
   }
   ```

6. **Parse Measurer JSON output:**

   Extract from response: `baselines_established`, `bottlenecks_identified`, `recommendations`, `tool_count`

   Log completion and update swarm display:
   Run using the Bash tool with description "Recording measurer completion...": `bash .aether/aether-utils.sh spawn-complete "{measurer_name}" "completed" "Baselines established, bottlenecks identified" && bash .aether/aether-utils.sh swarm-display-update "{measurer_name}" "measurer" "completed" "Performance measurement complete" "Queen" '{"read":5,"grep":3,"edit":0,"bash":0}' 100 "fungus_garden" 100`

   **Display Measurer completion line:**
   ```
   📊 {Measurer-Name}: Performance baseline measurement ({tool_count} tools) ✓
   ```

7. **Log findings to midden:**

   For each baseline established, run using the Bash tool with description "Logging baseline...":
   ```bash
   bash .aether/aether-utils.sh midden-write "performance" "Baseline: {baseline.operation} ({baseline.complexity}) at {baseline.file}:{baseline.line}" "measurer"
   ```

   For each bottleneck identified, run using the Bash tool with description "Logging bottleneck...":
   ```bash
   bash .aether/aether-utils.sh midden-write "performance" "Bottleneck: {bottleneck.description} ({bottleneck.severity}) at {bottleneck.location}" "measurer"
   ```

   For each recommendation, run using the Bash tool with description "Logging recommendation...":
   ```bash
   bash .aether/aether-utils.sh midden-write "performance" "Recommendation (P{rec.priority}): {rec.change} - {rec.estimated_improvement}" "measurer"
   ```

8. **Display summary and store for synthesis:**

   Display:
   ```
   📊 Measurer complete — {baseline_count} baselines, {bottleneck_count} bottlenecks logged to midden
   ```

   Store Measurer results in synthesis data structure:
   - Add `performance` object to synthesis JSON with: `baselines_established`, `bottlenecks_identified`, `recommendations`
   - Include in BUILD SUMMARY display: `📊 Measurer: {baseline_count} baselines established, {bottleneck_count} bottlenecks identified`

9. **Continue to Chaos Ant:**

   Proceed to Step 5.6 (Chaos Ant) regardless of Measurer results — Measurer is strictly non-blocking.

### Step 5.6: Spawn Chaos Ant for Resilience Testing

**After the Watcher completes, spawn a Chaos Ant to probe the phase work for edge cases and boundary conditions.**

Generate a chaos ant name and dispatch:
Run using the Bash tool with description "Naming chaos ant...": `bash .aether/aether-utils.sh generate-ant-name "chaos"` (store as `{chaos_name}`)
Run using the Bash tool with description "Loading existing flags...": `bash .aether/aether-utils.sh flag-list --phase {phase_number}`
Parse the result and extract unresolved flag titles into a list: `{existing_flag_titles}` (comma-separated titles from `.result.flags[].title`). If no flags exist, set `{existing_flag_titles}` to "None".
Run using the Bash tool with description "Dispatching chaos ant...": `bash .aether/aether-utils.sh spawn-log "Queen" "chaos" "{chaos_name}" "Resilience testing of Phase {id} work" && bash .aether/aether-utils.sh swarm-display-update "{chaos_name}" "chaos" "probing" "Resilience testing" "Queen" '{"read":0,"grep":0,"edit":0,"bash":0}' 0 "refuse_pile" 75`

**Announce the resilience testing wave:**
```
──── 🎲🐜 Spawning {chaos_name} — resilience testing ────
```

Spawn the Chaos Ant using Task tool with `subagent_type="aether-chaos"`, include `description: "🎲 Chaos {Chaos-Name}: Resilience testing"` (DO NOT use run_in_background - task blocks until complete):
# FALLBACK: If "Agent type not found", use general-purpose and inject role: "You are a Chaos Ant - resilience tester that probes edge cases and boundary conditions."

**Chaos Ant Prompt (CLEAN OUTPUT):**
```
You are {Chaos-Name}, a 🎲🐜 Chaos Ant.

Test Phase {id} work for edge cases and boundary conditions.

Files to test:
- {list from builder results}

Skip these known issues: {existing_flag_titles}

**IMPORTANT:** When using the Bash tool for activity calls, always include a description parameter:
- activity-log calls → "Logging {action}..."
- swarm-display-update calls → "Updating build display..."
- pheromone-read calls → "Checking colony signals..."

Use colony-flavored language, 4-8 words, trailing ellipsis.

Rules:
- Max 5 scenarios
- Read-only (don't modify code)
- Focus: edge cases, boundaries, error handling

Count your total tool calls (Read + Grep + Edit + Bash + Write) and report as tool_count.

Return ONLY this JSON:
{"ant_name": "{Chaos-Name}", "scenarios_tested": 5, "findings": [{"id": 1, "category": "edge_case|boundary|error_handling", "severity": "critical|high|medium|low", "title": "...", "description": "..."}], "overall_resilience": "strong|moderate|weak", "tool_count": 0, "summary": "..."}
```

### Step 5.7: Process Chaos Ant Results

**Task call returns results directly (no TaskOutput needed).**

**Parse the Chaos Ant's JSON response:** findings, overall_resilience, summary

**Display Chaos completion line:**
```
🎲 {Chaos-Name}: Resilience testing ({tool_count} tools) ✓
```

**Store results for synthesis in Step 5.9**

**Flag critical/high findings:**

If any findings have severity `"critical"` or `"high"`:
Run using the Bash tool with description "Flagging {finding.title}...": `bash .aether/aether-utils.sh flag-add "blocker" "{finding.title}" "{finding.description}" "chaos-testing" {phase_number} && bash .aether/aether-utils.sh activity-log "FLAG" "Chaos" "Created blocker: {finding.title}"`

**Log resilience finding to midden (MEM-02):**

For each critical/high finding, run using the Bash tool with description "Logging resilience finding...":
```bash
colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")
phase_num=$(jq -r '.phase.number // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

# Append to build-failures.md
cat >> .aether/midden/build-failures.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: ${phase_num}
  colony: "${colony_name}"
  worker: "${chaos_name}"
  test_context: "resilience"
  what_failed: "${finding.title}"
  why: "${finding.description}"
  what_worked: null
  severity: "${finding.severity}"
EOF

# Capture resilience failure in memory pipeline (observe + pheromone + auto-promotion)
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Resilience issue found: ${finding.title} (${finding.severity})" \
  "failure" \
  "worker:chaos" 2>/dev/null || true
```

Log chaos ant completion and update swarm display:
Run using the Bash tool with description "Recording chaos completion...": `bash .aether/aether-utils.sh spawn-complete "{chaos_name}" "completed" "{summary}" && bash .aether/aether-utils.sh swarm-display-update "{chaos_name}" "chaos" "completed" "Resilience testing done" "Queen" '{"read":2,"grep":1,"edit":0,"bash":0}' 100 "refuse_pile" 100`

### Step 5.8: Create Flags for Verification Failures

If the Watcher reported `verification_passed: false` or `recommendation: "fix_required"`:

For each issue in `issues_found`:
Run using the Bash tool with description "Flagging {issue_title}...": `bash .aether/aether-utils.sh flag-add "blocker" "{issue_title}" "{issue_description}" "verification" {phase_number} && bash .aether/aether-utils.sh activity-log "FLAG" "Watcher" "Created blocker: {issue_title}"`

**Log verification failure to midden (MEM-02):**

After flagging each issue, run using the Bash tool with description "Logging verification failure...":
```bash
colony_name=$(jq -r '.session_id | split("_")[1] // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")
phase_num=$(jq -r '.phase.number // "unknown"' .aether/data/COLONY_STATE.json 2>/dev/null || echo "unknown")

# Append to test-failures.md
cat >> .aether/midden/test-failures.md << EOF
- timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  phase: ${phase_num}
  colony: "${colony_name}"
  worker: "${watcher_name}"
  test_context: "verification"
  what_failed: "${issue_title}"
  why: "${issue_description}"
  what_worked: null
  severity: "high"
EOF

# Capture verification failure in memory pipeline (observe + pheromone + auto-promotion)
bash .aether/aether-utils.sh memory-capture \
  "failure" \
  "Verification failed: ${issue_title} - ${issue_description}" \
  "failure" \
  "worker:watcher" 2>/dev/null || true
```

This ensures verification failures are persisted as blockers that survive context resets. Chaos Ant findings are flagged in Step 5.7.
