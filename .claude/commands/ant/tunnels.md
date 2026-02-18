---
name: ant:tunnels
description: "🕳️🐜🕳️ Explore tunnels (browse archived colonies, compare chambers)"
---

You are the **Queen**. Browse the colony history.

## Instructions

### Argument Handling

- No arguments: Show timeline view (Step 4)
- One argument: Show single chamber detail with seal document (Step 3)
- Two arguments: Compare two chambers side-by-side (Step 5)
- More than two: "Too many arguments. Use: /ant:tunnels [chamber1] [chamber2]"

### Step 1: Check for Chambers Directory

Check if `.aether/chambers/` exists.

If not:
```
TUNNELS — Colony Timeline

No chambers found.

Archive colonies with /ant:entomb to build the tunnel network.
```
Stop here.

### Step 2: List All Chambers

Run: `bash .aether/aether-utils.sh chamber-list`

Parse JSON result into array of chambers.

If no chambers (empty array):
```
TUNNELS — Colony Timeline

0 colonies archived

The tunnel network is empty.
Archive colonies with /ant:entomb to preserve history.
```
Stop here.

### Step 3: Detail View — Show Seal Document (if one argument provided)

If `$ARGUMENTS` is not empty and contains exactly one argument:
- Treat it as chamber name
- Check if `.aether/chambers/{arguments}/` exists
- If not found:
  ```
  Chamber not found: {arguments}

  Run /ant:tunnels to see available chambers.
  ```
  Stop here.

**If CROWNED-ANTHILL.md exists in the chamber:**

```bash
seal_doc=".aether/chambers/{arguments}/CROWNED-ANTHILL.md"
```

Display the header:
```
CHAMBER DETAILS — {chamber_name}
```

Then display the FULL content of `CROWNED-ANTHILL.md` (read and output the file contents — this IS the seal ceremony record).

After the seal document, check if `colony-archive.xml` exists in the chamber:

```bash
chamber_has_xml=false
[[ -f ".aether/chambers/{chamber_name}/colony-archive.xml" ]] && chamber_has_xml=true
```

**If `colony-archive.xml` exists in the chamber**, show footer with import option:
```
Chamber integrity: {hash_status from chamber-verify}
Chamber location: .aether/chambers/{chamber_name}/
XML Archive: colony-archive.xml found

Actions:
  1. Import signals from this colony into current colony
  2. Return to timeline
  3. Compare with another chamber

Select an action (1/2/3)
```

Use AskUserQuestion with three options.

If option 1 selected: proceed to Step 6 (Import Signals from Chamber).
If option 2 selected: return to timeline (run /ant:tunnels).
If option 3 selected: prompt for second chamber name then run /ant:tunnels {chamber_a} {chamber_b}.

**If `colony-archive.xml` does NOT exist in the chamber**, show the existing footer unchanged:
```
Chamber integrity: {hash_status from chamber-verify}
Chamber location: .aether/chambers/{chamber_name}/

Run /ant:tunnels to return to timeline
Run /ant:tunnels {chamber_a} {chamber_b} to compare chambers
```

**If CROWNED-ANTHILL.md does NOT exist (older chamber):**

Display the header:
```
CHAMBER DETAILS — {chamber_name}

(No seal document — this chamber was created before the sealing ceremony was introduced)
```

Fall back to manifest data display:
- Read `manifest.json` and show: goal, milestone, version, phases_completed, total_phases, entombed_at
- Show decisions count and learnings count from manifest
- Show hash status from `chamber-verify`

Footer with navigation guidance:
```
Run /ant:tunnels to return to timeline
Run /ant:tunnels {chamber_a} {chamber_b} to compare chambers
```

To get the hash status:
- Run `bash .aether/aether-utils.sh chamber-verify .aether/chambers/{chamber_name}`
- If verified: hash_status = "verified"
- If not verified: hash_status = "hash mismatch"
- If error: hash_status = "error"

Stop here.

### Step 4: Timeline View (default, no arguments)

Display header:
```
TUNNELS — Colony Timeline

{count} colonies archived
```

For each chamber in sorted list (already sorted by `chamber-list` — newest first), display as a timeline entry:
```
[{date}] {milestone_emoji} {chamber_name}
           {goal (truncated to 60 chars)}
           {phases_completed} phases | {milestone}
```

Where `milestone_emoji` is:
- Crowned Anthill: crown emoji
- Sealed Chambers: lock emoji
- Other: circle emoji

After the timeline entries, show:
```
Run /ant:tunnels <chamber_name> to view seal document
Run /ant:tunnels <chamber_a> <chamber_b> to compare two colonies
```

Use the entombed_at field from the chamber-list JSON to extract the date (first 10 chars of ISO timestamp).

Stop here.

### Step 5: Chamber Comparison Mode (Two Arguments)

If two arguments provided (chamber names separated by space):
- Treat as: `/ant:tunnels <chamber_a> <chamber_b>`

Check both chambers exist. If either missing:
```
Chamber not found: {chamber_name}

Available chambers:
{list from chamber-list}
```
Stop here.

Run comparison:
```bash
bash .aether/utils/chamber-compare.sh compare <chamber_a> <chamber_b>
bash .aether/utils/chamber-compare.sh stats <chamber_a> <chamber_b>
```

Display comparison header:
```
CHAMBER COMPARISON

{chamber_a}  vs  {chamber_b}
```

Display side-by-side comparison:
```
+---------------------+---------------------+
| {chamber_a}         | {chamber_b}         |
+---------------------+---------------------+
| Goal: {goal_a}      | Goal: {goal_b}      |
|                     |                     |
| {milestone_a}       | {milestone_b}       |
| {version_a}         | {version_b}         |
|                     |                     |
| {phases_a} done     | {phases_b} done     |
| of {total_a}        | of {total_b}        |
|                     |                     |
| {decisions_a}       | {decisions_b}       |
| decisions           | decisions           |
|                     |                     |
| {learnings_a}       | {learnings_b}       |
| learnings           | learnings           |
|                     |                     |
| {date_a}            | {date_b}            |
+---------------------+---------------------+
```

Display growth metrics:
```
Growth Between Chambers:
  Phases: +{phases_diff} ({phases_a} -> {phases_b})
  Decisions: +{decisions_diff} new
  Learnings: +{learnings_diff} new
  Time: {time_between} days apart
```

If phases_diff > 0: show "Colony grew"
If phases_diff < 0: show "Colony reduced (unusual)"
If same_milestone: show "Same milestone reached"
If milestone changed: show "Milestone advanced: {milestone_a} -> {milestone_b}"

Display pheromone trail diff (new decisions/learnings in B):
```bash
bash .aether/utils/chamber-compare.sh diff <chamber_a> <chamber_b>
```

Parse result and show:
```
New Decisions in {chamber_b}:
  {N} new architectural decisions
  {if N <= 5, list them; else show first 3 + "...and {N-3} more"}

New Learnings in {chamber_b}:
  {N} new validated learnings
  {if N <= 5, list them; else show first 3 + "...and {N-3} more"}
```

If both chambers have `CROWNED-ANTHILL.md`, note:
```
Both colonies have seal documents. Run /ant:tunnels <name> to view individually.
```

Footer:
```
Run /ant:tunnels to see all chambers
Run /ant:tunnels <chamber> to view single chamber details
```

Stop here.

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
Stop here (return to timeline).

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

If no: "Import cancelled." Clean up: `rm -rf "$import_tmp_dir"`. Return to timeline.

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

### Edge Cases

**Malformed manifest:** show "Invalid manifest" for that chamber and skip it.

**Missing COLONY_STATE.json:** show "Incomplete chamber" for that chamber.

**Very long chamber list:** display all (no pagination for now).

**Older chambers without CROWNED-ANTHILL.md:** Fall back to manifest data in detail view.

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
