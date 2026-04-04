<!-- Generated from .aether/commands/export-signals.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:export-signals
description: "Export colony pheromone signals to portable XML format"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen**. Export colony pheromone signals to portable XML format.

## Instructions

The optional output path is: `$normalized_args`

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.
If file missing or `goal: null` -> "No colony initialized. Run /ant:init first.", stop.

Parse `$normalized_args`:
- If a path is provided, use it as the output path.
- If empty, default to `.aether/exchange/pheromones.xml`.

### Step 2: Export

Run using the Bash tool with description "Exporting pheromone signals to XML...":
```bash
aether pheromone-export-xml --output "<output_path>"
```

Parse the returned JSON:
- If `.ok` is `true`: extract `.result.path` and `.result.validated` (if present).
- If `.ok` is `false`: check `.error` for details. If error mentions `xmllint` or `E_FEATURE_UNAVAILABLE`, display: "XML export requires xmllint. Install with: xcode-select --install (macOS) or apt-get install libxml2-utils (Linux)." Otherwise display the error message and stop.

### Step 3: Confirm



Output:
```
Export colony pheromone signals to portable XML format

  Path: <output_path>
  Validated: <yes/no based on .result.validated>

Share this file with another colony using /ant:import-signals.
```



