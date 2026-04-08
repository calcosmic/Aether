<!-- Generated from .aether/commands/export-signals.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:export-signals
description: "Export colony pheromone signals to portable XML format"
---

You are the **Queen**. Export colony pheromone signals to portable XML format.

## Instructions

The optional output path is: `$ARGUMENTS`

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.
If file missing or `goal: null` -> "No colony initialized. Run /ant:init first.", stop.

Parse `$ARGUMENTS`:
- If a path is provided, use it as the output path.
- If empty, default to `.aether/exchange/pheromones.xml`.

### Step 2: Export

Run using the Bash tool with description "Exporting pheromone signals to XML...":
```bash
bash .aether/aether-utils.sh pheromone-export-xml "<output_path>"
```

Parse the returned JSON:
- If `.ok` is `true`: extract `.result.path` and `.result.validated` (if present).
- If `.ok` is `false`: check `.error` for details. If error mentions `xmllint` or `E_FEATURE_UNAVAILABLE`, display: "XML export requires xmllint. Install with: xcode-select --install (macOS) or apt-get install libxml2-utils (Linux)." Otherwise display the error message and stop.

### Step 3: Confirm


Output (3-5 lines, no banners):
```
Pheromone signals exported to XML
  Path: <output_path>
  Validated: <yes/no based on .result.validated>

Share this file with another colony using /ant:import-signals.
```




### Step 4: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
bash .aether/aether-utils.sh print-next-up "$state" "$current_phase" "$total_phases"
```

