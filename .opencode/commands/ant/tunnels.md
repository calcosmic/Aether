---
name: ant:tunnels
description: "🕳️🐜🕳️ Explore tunnels (browse archived colonies, compare chambers)"
---

You are the **Queen**. Browse the colony history.

## Instructions

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

### Argument Handling

- No arguments: Show chamber list (Step 4)
- One argument: Show single chamber detail (Step 3)
- Two arguments: Compare two chambers (Step 5)
- More than two: "Too many arguments. Use: /ant:tunnels [chamber1] [chamber2]"

### Step 1: Check for Chambers Directory

Check if `.aether/chambers/` exists.

If not:
```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

No chambers found.

Archive colonies with /ant:entomb to build the tunnel network.
```
Stop here.

### Step 2: List All Chambers

Run: `bash .aether/aether-utils.sh chamber-list`

Parse JSON result into array of chambers.

If no chambers (empty array):
```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

Chambers: 0 colonies archived

The tunnel network is empty.
Archive colonies with /ant:entomb to preserve history.
```
Stop here.

### Step 3: Handle Detail View (if argument provided)

If `$normalized_args` is not empty:
- Treat it as chamber name
- Check if `.aether/chambers/{arguments}/` exists
- If not found:
  ```
  Chamber not found: {arguments}

  Run /ant:tunnels to see available chambers.
  ```
  Stop here.

- If found, read manifest.json and display detailed view:
```
🕳️ ═══════════════════════════════════════════════════
   C H A M B E R   D E T A I L S
══════════════════════════════════════════════════ 🕳️

📦 {chamber_name}

👑 Goal:
   {goal}

🏆 Milestone: {milestone} ({version})
📍 Progress: {phases_completed} of {total_phases} phases
📅 Entombed: {entombed_at}

{If decisions exist:}
🧠 Decisions Preserved:
   {N} architectural decisions recorded
{End if}

{If learnings exist:}
💡 Learnings Preserved:
   {N} validated learnings recorded
{End if}

📁 Files:
   - COLONY_STATE.json (verified: {hash_status})
   - manifest.json

Run /ant:tunnels to return to chamber list.
```

To get the counts and hash status:
- Run `bash .aether/aether-utils.sh chamber-verify .aether/chambers/{chamber_name}`
- If verified: hash_status = "✅"
- If not verified: hash_status = "⚠️ hash mismatch"
- If error: hash_status = "⚠️ error"

Check if `colony-archive.xml` exists in the chamber:

```bash
chamber_has_xml=false
[[ -f ".aether/chambers/{chamber_name}/colony-archive.xml" ]] && chamber_has_xml=true
```

**If `colony-archive.xml` exists**, add import option to the detail view footer:
```
📁 Files:
   - COLONY_STATE.json (verified: {hash_status})
   - manifest.json
   - colony-archive.xml (XML Archive)

Actions:
  1. Import signals from this colony into current colony
  2. Return to chamber list

Select an action (1/2)
```

Use AskUserQuestion with two options.

If option 1 selected: proceed to Step 6 (Import Signals from Chamber).
If option 2 selected: return to chamber list (run /ant:tunnels).

**If `colony-archive.xml` does NOT exist**, show existing footer unchanged:
```
Run /ant:tunnels to return to chamber list.
```

Stop here.

### Step 5: Chamber Comparison Mode (Two Arguments)

If two arguments provided (chamber names separated by space):
- Treat as: `/ant:tunnels <chamber_a> <chamber_b>`
- Run comparison: `bash .aether/utils/chamber-compare.sh compare <chamber_a> <chamber_b>`

If either chamber not found:
```
Chamber not found: {chamber_name}

Available chambers:
{list from chamber-list}
```
Stop here.

Display comparison header:
```
🕳️ ═══════════════════════════════════════════════════
   C H A M B E R   C O M P A R I S O N
══════════════════════════════════════════════════ 🕳️

📦 {chamber_a}  vs  📦 {chamber_b}
```

Display side-by-side comparison:
```
┌─────────────────────┬─────────────────────┐
│ {chamber_a}         │ {chamber_b}         │
├─────────────────────┼─────────────────────┤
│ 👑 {goal_a}         │ 👑 {goal_b}         │
│                     │                     │
│ 🏆 {milestone_a}    │ 🏆 {milestone_b}    │
│    {version_a}      │    {version_b}      │
│                     │                     │
│ 📍 {phases_a} done  │ 📍 {phases_b} done  │
│    of {total_a}     │    of {total_b}     │
│                     │                     │
│ 🧠 {decisions_a}    │ 🧠 {decisions_b}    │
│    decisions        │    decisions        │
│                     │                     │
│ 💡 {learnings_a}    │ 💡 {learnings_b}    │
│    learnings        │    learnings        │
│                     │                     │
│ 📅 {date_a}         │ 📅 {date_b}         │
└─────────────────────┴─────────────────────┘
```

Display growth metrics:
```
📈 Growth Between Chambers:
   Phases: +{phases_diff} ({phases_a} → {phases_b})
   Decisions: +{decisions_diff} new
   Learnings: +{learnings_diff} new
   Time: {time_between} days apart
```

If phases_diff > 0: show "📈 Colony grew"
If phases_diff < 0: show "📉 Colony reduced (unusual)"
If same_milestone: show "🏆 Same milestone reached"
If milestone changed: show "🏆 Milestone advanced: {milestone_a} → {milestone_b}"

Display pheromone trail diff (new decisions/learnings in B):
```bash
bash .aether/utils/chamber-compare.sh diff <chamber_a> <chamber_b>
```

Parse result and show:
```
🧠 New Decisions in {chamber_b}:
   {N} new architectural decisions
   {if N <= 5, list them; else show first 3 + "...and {N-3} more"}

💡 New Learnings in {chamber_b}:
   {N} new validated learnings
   {if N <= 5, list them; else show first 3 + "...and {N-3} more"}
```

Display knowledge preservation:
```
📚 Knowledge Preservation:
   {preserved_decisions} decisions carried forward
   {preserved_learnings} learnings carried forward
```

Footer:
```
Run /ant:tunnels to see all chambers
Run /ant:tunnels <chamber> to view single chamber details
```

Stop here.

### Step 4: Display Chamber List (default view)

```
🕳️ ═══════════════════════════════════════════════════
   T U N N E L S   (Colony History)
══════════════════════════════════════════════════ 🕳️

Chambers: {count} colonies archived

{For each chamber in sorted list:}
📦 {chamber_name}
   👑 {goal (truncated to 50 chars)}
   🏆 {milestone} ({version})
   📍 {phases_completed} phases | 📅 {date}

{End for}

Run /ant:tunnels <chamber_name> to view details
```

**Formatting details:**
- Sort by entombed_at descending (newest first) - already sorted by chamber-list
- Truncate goal to 50 characters with "..." if longer
- Format date as YYYY-MM-DD from ISO timestamp (extract first 10 chars of entombed_at)
- Show chamber count at top

**Edge cases:**
- Malformed manifest: show "⚠️  Invalid manifest" for that chamber and skip it
- Missing COLONY_STATE.json: show "⚠️  Incomplete chamber" for that chamber
- Very long chamber list: display all (no pagination for now)

### Step 6: Import Signals from Chamber

When user selects "Import signals" from Step 3:

**Step 6.1: Check XML tools**
```bash
if command -v xmllint >/dev/null 2>&1; then
  xmllint_available=true
else
  xmllint_available=false
fi
```

If xmllint not available:
```
Import requires xmllint. Install it first:
  macOS: xcode-select --install
  Linux: apt-get install libxml2-utils
```
Stop here (return to chamber list).

**Step 6.2: Extract source colony name**
```bash
chamber_xml=".aether/chambers/{chamber_name}/colony-archive.xml"
# Extract colony_id from the archive root element
source_colony=$(xmllint --xpath "string(/*/@colony_id)" "$chamber_xml" 2>/dev/null)
[[ -z "$source_colony" ]] && source_colony="{chamber_name}"
```

**Step 6.3: Extract pheromone section and show import preview**

The combined `colony-archive.xml` contains pheromones, wisdom, and registry sections. Extract the pheromone section to a temp file before counting or importing. This prevents over-counting signals from wisdom/registry sections and ensures `pheromone-import-xml` receives the format it expects (`<pheromones>` as root element).

```bash
# Extract the <pheromones> section from the combined archive into a standalone temp file
import_tmp_dir=$(mktemp -d)
import_tmp_pheromones="$import_tmp_dir/pheromones-extracted.xml"

# Use xmllint to extract the pheromones element (with its namespace)
xmllint --xpath "//*[local-name()='pheromones']" "$chamber_xml" > "$import_tmp_pheromones" 2>/dev/null

# Add XML declaration to make it a standalone well-formed document
if [[ -s "$import_tmp_pheromones" ]]; then
  # Portable approach: prepend declaration via temp file (avoids macOS/Linux sed -i differences)
  { echo '<?xml version="1.0" encoding="UTF-8"?>'; cat "$import_tmp_pheromones"; } > "$import_tmp_dir/tmp_decl.xml"
  mv "$import_tmp_dir/tmp_decl.xml" "$import_tmp_pheromones"
fi

# Count pheromone signals in extracted pheromone-only XML
# Scoped to pheromone section only — no over-counting from wisdom/registry sections
pheromone_count=$(xmllint --xpath "count(//*[local-name()='signal'])" "$import_tmp_pheromones" 2>/dev/null || echo "unknown")
```

Display:
```
IMPORT FROM COLONY: {source_colony}

Source: .aether/chambers/{chamber_name}/colony-archive.xml
Signals available: ~{pheromone_count} pheromone signals

Import behavior:
  - Signals tagged with prefix "{source_colony}:" to identify origin
  - Additive merge — your current signals are never overwritten
  - On conflict, your current colony wins

Import these signals? (yes/no)
```

Use AskUserQuestion with yes/no options.

If no: "Import cancelled." Clean up: `rm -rf "$import_tmp_dir"`. Return to chamber list.

**Step 6.4: Perform import**

Pass the extracted pheromone-only temp file (NOT the combined `colony-archive.xml`) to `pheromone-import-xml`, along with `$source_colony` as the second argument. This ensures:
1. `pheromone-import-xml` receives XML with `<pheromones>` as root element (the format it expects)
2. The prefix-tagging logic prepends `${source_colony}:` to each imported signal's ID before the merge

```bash
# Import the EXTRACTED pheromone-only XML (NOT the combined colony-archive.xml)
# $import_tmp_pheromones has <pheromones> as root — the format pheromone-import-xml expects
# Second argument triggers prefix-tagging — imported signal IDs become "{source_colony}:original_id"
import_result=$(bash .aether/aether-utils.sh pheromone-import-xml "$import_tmp_pheromones" "$source_colony" 2>&1)
import_ok=$(echo "$import_result" | jq -r '.ok // false' 2>/dev/null)

if [[ "$import_ok" == "true" ]]; then
  imported_count=$(echo "$import_result" | jq -r '.result.signal_count // 0' 2>/dev/null)
else
  imported_count=0
  import_error=$(echo "$import_result" | jq -r '.error // "Unknown error"' 2>/dev/null)
fi

# Clean up temp files
rm -rf "$import_tmp_dir"
```

**Step 6.5: Display result**

If import succeeded:
```
SIGNALS IMPORTED

Source: {source_colony}
Imported: {imported_count} pheromone signals
Tagged with: "{source_colony}:" prefix

Your colony now carries wisdom from {source_colony}.
Run /ant:status to see current colony state.
```

If import failed:
```
Import failed: {import_error}

The archive may be malformed. Check:
  .aether/chambers/{chamber_name}/colony-archive.xml
```

## Implementation Notes

The `chamber-list` utility returns JSON in this format:
```json
{
  "ok": true,
  "result": [
    {
      "name": "add-user-auth-20260214-153022",
      "goal": "Add user authentication",
      "milestone": "Sealed Chambers",
      "phases_completed": 5,
      "entombed_at": "2026-02-14T15:30:22Z"
    }
  ]
}
```

Parse with jq: `jq -r '.result[] | "\(.name)|\(.goal)|\(.milestone)|\(.phases_completed)|\(.entombed_at)"'`

For detail view, read manifest.json directly:
```bash
jq -r '.goal, .milestone, .version, .phases_completed, .total_phases, .entombed_at, (.decisions | length), (.learnings | length)' .aether/chambers/{name}/manifest.json
```
