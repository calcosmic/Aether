---
phase: 10
plan: 01
name: Chamber Management Utilities
subsystem: colony-lifecycle
tags: [bash, chamber, lifecycle, manifest, sha256]

requires:
  - 09-05  # Auto-Load Context (previous phase complete)

provides:
  - chamber_create    # Create entombment chamber with manifest
  - chamber_verify    # Verify chamber integrity via SHA256
  - chamber_list      # List chambers sorted by timestamp
  - chamber_sanitize_goal  # Sanitize goal for directory names

affects:
  - 10-02  # Entomb command (uses chamber_create)
  - 10-03  # Lay eggs command (uses chamber_list)
  - 10-04  # Tunnels command (uses chamber_list)

tech-stack:
  added: []
  patterns:
    - Copy-then-verify safety pattern
    - JSON output helpers (json_ok/json_err)
    - SHA256 cross-platform hashing (sha256sum/shasum)

key-files:
  created:
    - .aether/utils/chamber-utils.sh
    - .aether/chambers/.gitkeep
  modified:
    - .aether/aether-utils.sh

decisions:
  - id: D10-01-001
    text: Use jq -Rs '.[:-1]' to strip trailing newlines from JSON string output
    rationale: jq -Rs adds a trailing newline which pollutes JSON output; .[:-1] removes it

metrics:
  duration: 30m
  completed: 2026-02-14
---

# Phase 10 Plan 01: Chamber Management Utilities Summary

## One-Liner
Created chamber management utilities (create, verify, list) with SHA256 integrity checking and integrated into aether-utils.sh CLI.

## What Was Built

### chamber-utils.sh Module
A new utility module at `.aether/utils/chamber-utils.sh` providing four functions:

1. **chamber_create()** - Creates an entombment chamber
   - Creates directory structure
   - Copies COLONY_STATE.json
   - Generates manifest.json with metadata and SHA256 hash
   - Returns JSON with chamber info

2. **chamber_verify()** - Verifies chamber integrity
   - Re-computes SHA256 hash of COLONY_STATE.json
   - Compares with stored hash in manifest
   - Returns verification result (pass/fail with details)

3. **chamber_list()** - Lists all chambers
   - Scans chambers directory
   - Reads manifest.json from each chamber
   - Returns JSON array sorted by entombed_at (descending)

4. **chamber_sanitize_goal()** - Sanitizes goal strings
   - Lowercases, replaces special chars with hyphens
   - Limits length to 50 chars
   - Suitable for directory names

### aether-utils.sh Integration
Added three new subcommands to the utility layer:
- `chamber-create` - Wrapper around chamber_create
- `chamber-verify` - Wrapper around chamber_verify
- `chamber-list` - Wrapper around chamber_list (optional chambers_root arg)

### Chambers Directory
Created `.aether/chambers/` with `.gitkeep` for git tracking.

## Verification Results

All verification checks passed:
- [x] chamber-utils.sh exists with 4 functions
- [x] chamber_create generates valid manifest.json
- [x] chamber_verify detects hash mismatches (tampering detection works)
- [x] chamber_list returns sorted JSON array
- [x] aether-utils.sh includes chamber-* subcommands
- [x] .aether/chambers/ directory exists and is git-tracked

## Deviations from Plan

None - plan executed exactly as written.

## Key Implementation Details

### Cross-Platform SHA256
```bash
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "$file_path" | cut -d' ' -f1
elif command -v shasum >/dev/null 2>&1; then
  shasum -a 256 "$file_path" | cut -d' ' -f1
fi
```

### Manifest Schema
```json
{
  "entombed_at": "2026-02-14T18:02:58Z",
  "goal": "Test Goal",
  "phases_completed": 5,
  "total_phases": 5,
  "milestone": "Sealed Chambers",
  "version": "v1.0.0",
  "decisions": [],
  "learnings": [],
  "files": {
    "COLONY_STATE.json": "sha256:..."
  }
}
```

## Next Phase Readiness

This plan provides the foundation for:
- **10-02**: `/ant:entomb` command (uses chamber_create)
- **10-03**: `/ant:lay-eggs` command (uses chamber_list for prior knowledge)
- **10-04**: `/ant:tunnels` command (uses chamber_list for browsing)

All chamber operations are ready for use.

## Commits

1. `d422bd3` - fix(10-01): chamber-utils.sh JSON output - remove trailing newlines
2. `eb55f58` - feat(10-01): integrate chamber commands into aether-utils.sh
3. `6194612` - chore(10-01): create chambers directory structure
