---
phase: 01-infrastructure
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - runtime/data/signatures.json
autonomous: true

must_haves:
  truths:
    - "signatures.json template exists at runtime/data/signatures.json"
    - "Template has valid JSON structure with signatures array"
    - "signature-scan and signature-match commands can read the file without error"
  artifacts:
    - path: "runtime/data/signatures.json"
      provides: "Default signatures template for pattern matching"
      contains: '[{"pattern": "...", "name": "...", "confidence": ...}]'
  key_links:
    - from: ".aether/aether-utils.sh signature-scan (source of truth)"
      to: "runtime/data/signatures.json"
      via: "file read at $DATA_DIR/signatures.json"
---

<objective>
Create the missing signatures.json template file that aether-utils.sh references.

Purpose: The signature-scan and signature-match subcommands in aether-utils.sh expect a signatures.json file at $DATA_DIR/signatures.json, but this file does not exist. This causes the commands to return empty results or errors.
Output: A default signatures.json template with example patterns for common code signatures.
</objective>

<execution_context>
@~/.claude/cosmic-dev-system/workflows/execute-plan.md
@~/.claude/cosmic-dev-system/templates/summary.md
</execution_context>

<context>
@/Users/callumcowie/repos/Aether/.aether/aether-utils.sh (source of truth, auto-synced to runtime/)

The .aether/aether-utils.sh file contains two subcommands that reference signatures.json:
1. signature-scan (line 583-587): Reads signatures to scan for code patterns
2. signature-match (line 636-637): Reads signatures for matching against targets

Both commands expect the file at $DATA_DIR/signatures.json and handle the missing file case gracefully, but having a default template improves the user experience and makes the feature functional out of the box.
</context>

<tasks>

<task type="auto">
  <name>Create runtime/data directory and signatures.json template</name>
  <files>runtime/data/signatures.json</files>
  <action>
    Create the directory runtime/data/ if it does not exist.

    Create runtime/data/signatures.json with a default template containing example signatures for common code patterns. The file should have this structure:
    {
      "signatures": [
        {
          "pattern": "TODO|FIXME|XXX|HACK",
          "name": "todo-marker",
          "description": "Code markers indicating pending work",
          "confidence": 0.9,
          "category": "maintenance"
        },
        {
          "pattern": "console\\.(log|warn|error|debug)",
          "name": "debug-logging",
          "description": "Console logging statements",
          "confidence": 0.8,
          "category": "debugging"
        },
        {
          "pattern": "describe\\s*\\(|it\\s*\\(|test\\s*\\(",
          "name": "test-definition",
          "description": "Test case definitions",
          "confidence": 0.95,
          "category": "testing"
        },
        {
          "pattern": "function\\s+\\w+\\s*\\(|const\\s+\\w+\\s*=\\s*\\(|async\\s+function",
          "name": "function-definition",
          "description": "Function declarations",
          "confidence": 0.85,
          "category": "structure"
        },
        {
          "pattern": "import\\s+|require\\s*\\(",
          "name": "module-import",
          "description": "Module import statements",
          "confidence": 0.9,
          "category": "dependencies"
        }
      ],
      "version": "1.0.0",
      "last_updated": ""
    }

    Ensure the JSON is properly formatted with 2-space indentation.
  </action>
  <verify>
    cat runtime/data/signatures.json | jq empty && echo "Valid JSON"
  </verify>
  <done>
    runtime/data/signatures.json exists with valid JSON structure containing a signatures array with at least 5 example patterns, each having pattern, name, description, confidence, and category fields.
  </done>
</task>

</tasks>

<verification>
- [ ] File exists at runtime/data/signatures.json
- [ ] JSON is valid and parseable by jq
- [ ] Contains signatures array with pattern objects
- [ ] Each signature has required fields: pattern, name, confidence
- [ ] aether-utils.sh signature-scan command can read the file
</verification>

<success_criteria>
- signatures.json template exists and is valid JSON
- signature-scan and signature-match commands work without file-not-found warnings
- Template provides useful default patterns for common code signatures
</success_criteria>

<output>
After completion, create `.planning/phases/01-infrastructure/01-infrastructure-01-SUMMARY.md`
</output>
