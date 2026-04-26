<!-- Generated from .aether/commands/import-signals.yaml - DO NOT EDIT DIRECTLY -->
---
name: ant-import-signals
description: "📥 Import pheromone signals from another colony's XML export"
---

> **Important:** This is a pure prompt command. Do NOT attempt to run `aether import-signals`. Follow the instructions below directly.

You are the **Queen**. Import pheromone signals from another colony's XML export.

## Instructions

The arguments are: `$ARGUMENTS`

### Step 1: Validate

Read `.aether/data/COLONY_STATE.json`.
If file missing or `goal: null` -> "No colony initialized. Run /ant-init first.", stop.

Parse `$ARGUMENTS`:
- First argument: path to XML file (required).

If no arguments provided, show usage and stop:
```
Usage: /ant-import-signals <path-to-signals.xml>

  <path-to-signals.xml>  Path to an exported pheromone XML file

Example:
  /ant-import-signals .aether/exchange/pheromones.xml
```

Verify the XML file exists. If not -> "File not found: <path>", stop.

### Step 2: Import

Run using the Bash tool with description "Importing pheromone signals from XML...":
```bash
aether import-signals --file "<xml_path>"
```

Parse the returned JSON:
- If `.ok` is `true`: extract `.result.imported` and `.result.source`.
- If `.ok` is `false`: check `.error` for details. If error mentions `xmllint` or `E_FEATURE_UNAVAILABLE`, display: "XML import requires xmllint. Install with: xcode-select --install (macOS) or apt-get install libxml2-utils (Linux)." Otherwise display the error message and stop.

### Step 3: Confirm


Output (4-5 lines, no banners):
```
Pheromone signals imported
  Source: <xml_path>
  Signals imported: <imported>
```




### Step 4: Next Up

Generate the state-based Next Up block by running using the Bash tool with description "Generating Next Up suggestions...":
```bash
state=$(jq -r '.state // "IDLE"' .aether/data/COLONY_STATE.json)
current_phase=$(jq -r '.current_phase // 0' .aether/data/COLONY_STATE.json)
total_phases=$(jq -r '.plan.phases | length' .aether/data/COLONY_STATE.json)
aether print-next-up
```
