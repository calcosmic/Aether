<!-- Generated from .aether/commands/data-clean.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:data-clean
description: "Scan and remove test artifacts from colony data files"
---

You are the **Queen Ant Colony**. Run the data cleaner to scan for and remove test/synthetic artifacts from colony data files.


> **Note:** `$ARGUMENTS` is unused. This command always scans all data files.



## Instructions

### Step 1: Scan

Run using the Bash tool with description "Scanning colony data for test artifacts...":
```bash
aether data-clean --dry-run
```

Display the output to the user. This shows artifact counts per data file without modifying anything.

### Step 2: Decision Gate

Parse the scan output for "Total artifacts: N".

**If total is 0:**
Display:
```
Colony data is clean. No artifacts found.
```
Skip to Step 5.

**If total is greater than 0:**
Ask the user:
```
Found {N} test artifacts across colony data files.
Remove these artifacts? (yes/no)
```

If user says no, display "No changes made." and skip to Step 5.

### Step 3: Clean

If user confirmed, run using the Bash tool with description "Removing test artifacts...":
```bash
aether data-clean --confirm
```

### Step 4: Summary

Display the cleanup results showing what was removed from each file.

For example:
```
Data Clean Complete
===================
Removed {total} artifacts:
  - pheromones.json: {N} test signals
  - QUEEN.md: {N} test entries
  - learning-observations.json: {N} test observations
  - midden.json: {N} test entries
  - spawn-tree.txt: {N} test worker lines
  - constraints.json: {N} test focus entries

Run /ant:status to verify colony state.
```


### Step 5: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
```

