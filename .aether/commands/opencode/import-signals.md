<!-- Generated from .aether/commands/import-signals.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant:import-signals
description: "Import pheromone signals from another colony's XML export"
---

### Step -1: Normalize Arguments

Run: `normalized_args=$(bash .aether/aether-utils.sh normalize-args "$@")`

This ensures arguments work correctly in both Claude Code and OpenCode. Use `$normalized_args` throughout this command.

You are the **Queen**. Import pheromone signals from another colony's XML export.

## Instructions

The arguments are: `$normalized_args`

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.
If file missing or `goal: null` -> "No colony initialized. Run /ant:init first.", stop.

Parse `$normalized_args`:
- First argument: path to XML file (required).
- Second argument: colony name/prefix (optional; default: derive from XML filename without extension, or use "imported").

If no arguments provided, show usage and stop:
```
Usage: /ant:import-signals <path-to-signals.xml> [colony-name]

  <path-to-signals.xml>  Path to an exported pheromone XML file
  [colony-name]          Optional prefix for imported signal IDs (prevents collisions)

Example:
  /ant:import-signals .aether/exchange/pheromones.xml partner-colony
```

Verify the XML file exists. If not -> "File not found: <path>", stop.

### Step 2: Import

Run using the Bash tool with description "Importing pheromone signals from XML...":
```bash
bash .aether/aether-utils.sh pheromone-import-xml "<xml_path>" "<colony_prefix>"
```

Parse the returned JSON:
- If `.ok` is `true`: extract `.result.signal_count` and `.result.source`.
- If `.ok` is `false`: check `.error` for details. If error mentions `xmllint` or `E_FEATURE_UNAVAILABLE`, display: "XML import requires xmllint. Install with: xcode-select --install (macOS) or apt-get install libxml2-utils (Linux)." Otherwise display the error message and stop.

### Step 3: Confirm



Output:
```
Pheromone signals imported

  Source: <xml_path>
  Signals imported: <signal_count>
  Colony prefix: <colony_prefix>

Note: On signal ID collision, current colony signals take priority.
```



