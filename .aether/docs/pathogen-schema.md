# Pathogen Signature Schema

## Overview

Pathogen signatures extend the existing error pattern format (`~/.aether/error-patterns.json`) with fields for automated detection and escalation. The schema is backward-compatible: existing jq queries on core fields (`name`, `description`, `severity`, `occurrences`, etc.) work unchanged on pathogen entries.

## Storage

- **File:** `.aether/data/pathogens.json` (project-local)
- **Format:** JSON, jq-compatible
- **Relationship:** Extends `error-patterns.json` field set; stored separately to avoid polluting global pattern data

## Schema

### Root Object

| Field | Type | Description |
|-------|------|-------------|
| `pathogens` | array | Array of pathogen signature objects |
| `metadata` | object | File-level metadata |

### Metadata Object

| Field | Type | Description |
|-------|------|-------------|
| `version` | string | Schema version (currently `"1.0"`) |
| `last_updated` | string\|null | ISO 8601 UTC timestamp of last modification |

### Pathogen Signature Object

#### Inherited Fields (from error-patterns format)

These fields match the existing `error-patterns.json` schema exactly. Any jq query that works on error patterns will work on pathogen entries.

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Unique identifier for the pathogen |
| `description` | string | Human-readable description |
| `severity` | string | `"warning"` \| `"critical"` |
| `first_seen` | string | ISO 8601 UTC timestamp |
| `last_seen` | string | ISO 8601 UTC timestamp |
| `occurrences` | number | Count of times detected |
| `projects` | array[string] | Project names where detected |
| `resolved` | boolean | Whether the pathogen has been resolved |

#### New Fields (pathogen-specific)

| Field | Type | Description |
|-------|------|-------------|
| `signature_type` | string | `"exact"` \| `"fuzzy"` — matching strategy |
| `pattern_string` | string | The literal string or regex to match against |
| `confidence_threshold` | number | Float 0.0–1.0, minimum confidence for a match |
| `escalation_level` | string | `"log"` \| `"elevated"` \| `"swarm"` — response tier |

## Escalation Thresholds (Hardcoded)

| Confidence | Escalation Level | Action |
|------------|-----------------|--------|
| >= 0.9 | `"swarm"` | Immediate swarm response |
| 0.7–0.89 | `"elevated"` | Elevated scrutiny |
| < 0.7 | `"log"` | Log only |

## Backward Compatibility

### Existing queries that still work

```bash
# Select by name (works on both error patterns and pathogens)
jq '.pathogens[] | select(.name == "some_name")' pathogens.json

# Filter by severity
jq '.pathogens[] | select(.severity == "critical")' pathogens.json

# Filter recurring (2+ occurrences, unresolved)
jq '[.pathogens[] | select(.occurrences >= 2 and .resolved == false)]' pathogens.json

# Get all names
jq '[.pathogens[].name]' pathogens.json
```

### New queries for pathogen-specific fields

```bash
# Filter by confidence threshold
jq '.pathogens[] | select(.confidence_threshold >= 0.7)' pathogens.json

# Get all exact-match pathogens
jq '[.pathogens[] | select(.signature_type == "exact")]' pathogens.json

# Get pathogens requiring swarm escalation
jq '[.pathogens[] | select(.escalation_level == "swarm")]' pathogens.json

# Search for a specific pattern string
jq --arg pat "error_string" '.pathogens[] | select(.pattern_string == $pat)' pathogens.json
```

## Validation

Validate the file structure:

```bash
# Check file is valid JSON
jq empty pathogens.json

# Verify required fields exist on all entries
jq '.pathogens[] | has("name", "signature_type", "pattern_string", "confidence_threshold", "escalation_level")' pathogens.json

# Verify confidence is in range
jq '.pathogens[] | select(.confidence_threshold < 0 or .confidence_threshold > 1)' pathogens.json
# (should return empty if all valid)
```
