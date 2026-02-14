# Technology Stack Research: v3.1 Model Routing & Colony Lifecycle

**Domain:** Model Routing & Colony Lifecycle Management for AI Agent Orchestration
**Milestone:** v3.1 "Open Chambers"
**Researched:** 2026-02-14
**Confidence:** HIGH

## Context

This research is for v3.1 "Open Chambers" milestone of the Aether Colony System. The system already uses LiteLLM proxy for model routing at `localhost:4000`. Current model profiles are defined in `.aether/model-profiles.yaml` with caste-to-model mappings.

**Goal:** Implement intelligent model routing per worker caste and colony lifecycle management (archive/foundation commands).

---

## Recommended Stack

### Core Technologies (Already in Use)

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Node.js | >=16.0.0 | Runtime | Already established in package.json engines |
| commander.js | ^12.1.0 | CLI framework | Already in use, mature, supports subcommands |
| LiteLLM Proxy | latest | Model routing | Already deployed at localhost:4000, industry standard |
| picocolors | ^1.1.1 | Terminal colors | Already in use, lightweight, zero deps |

### New Libraries Required

| Library | Version | Purpose | Why Recommended |
|---------|---------|---------|-----------------|
| js-yaml | ^4.1.0 | YAML parsing | Industry standard (60M+ weekly downloads), fast, YAML 1.2 compliant |
| yaml (eemeli) | ^2.7.0 | Round-trip YAML editing | Preserves comments/formatting when modifying model-profiles.yaml |

### Development Tools (Already Configured)

| Tool | Purpose | Notes |
|------|---------|-------|
| ava | ^6.0.0 | Test runner | Already configured in package.json |
| sinon | ^19.0.5 | Mocking | Already in devDependencies |
| proxyquire | ^2.1.3 | Module mocking | Already in devDependencies |

---

## Installation

```bash
# Core additions for v3.1
npm install js-yaml yaml

# No additional dev dependencies needed
```

---

## Implementation Requirements

### Model Routing Per Worker Caste

**Current State:**
- Model profiles stored in `.aether/model-profiles.yaml`
- LiteLLM proxy running at `localhost:4000`
- Environment variables set: `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `ANTHROPIC_AUTH_TOKEN`
- Model verification module exists at `bin/lib/model-verify.js`

**New Requirements:**

1. **YAML Parsing Module** (`bin/lib/model-profiles.js`)
   - Parse `.aether/model-profiles.yaml`
   - Support reading caste-to-model mappings
   - Support updating model assignments programmatically
   - Preserve comments and formatting on write (use eemeli/yaml)

2. **Model Router Integration** (`bin/lib/model-router.js`)
   - Query LiteLLM proxy for available models via `/models` endpoint
   - Validate model assignments against proxy capabilities
   - Provide fallback model selection
   - Cache model metadata for performance

3. **CLI Commands** (add to `bin/cli.js`)
   - `aether model list` - Show all caste assignments
   - `aether model set <caste> <model>` - Update caste model
   - `aether model validate` - Verify all models available in proxy
   - `aether model reset` - Reset to defaults from hub

### Colony Lifecycle Management

**Current State:**
- Colony state in `.aether/data/COLONY_STATE.json`
- Checkpoints stored in `.aether/checkpoints/`
- Archive directory exists at `.aether/data/archive/`
- Update transaction system in `bin/lib/update-transaction.js`

**New Requirements:**

1. **Archive Command** (`aether archive`)
   - Move current colony state to timestamped archive
   - Preserve: COLONY_STATE.json, constraints.json, flags.json, dreams/, oracle/
   - Create archive manifest with metadata
   - Update COLONY_STATE to reflect archived status
   - Support `--note` flag for archive description

2. **Foundation Command** (`aether foundation`)
   - `aether foundation list` - List available archives
   - `aether foundation restore <archive-id>` - Restore from archive (with safety checks)
   - `aether foundation new` - Create new "foundation" (blank slate) with preserved config
   - Support partial restore (selective state components)

3. **State Versioning**
   - Add `archived_at` field to state schema
   - Add `foundation_version` tracking
   - Migration path for state schema updates

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| js-yaml | yaml (eemeli) | Use eemeli/yaml if heavy modification of YAML files needed |
| yaml (eemeli) | js-yaml | Use js-yaml for simple read-only parsing (faster) |
| LiteLLM proxy | Custom router | Only if LiteLLM becomes unsupported (not recommended) |
| File-based state | SQLite | SQLite if state grows beyond 10MB or needs complex queries |
| YAML config | JSON config | Keep YAML - more human-readable for model profiles |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| yamljs | Unmaintained, security issues | js-yaml or eemeli/yaml |
| node-yaml | Deprecated, no updates | js-yaml |
| Custom YAML parser | Reinventing wheel | js-yaml (battle-tested) |
| Direct file manipulation without atomic writes | Risk of corruption | Use existing `bin/lib/atomic-write.js` pattern |
| TOML for model profiles | Not used elsewhere in codebase | Keep YAML for consistency |

---

## Stack Patterns by Variant

**If heavy model-profiles.yaml editing needed:**
- Use `yaml (eemeli)` for round-trip editing
- Because: Preserves user comments and formatting

**If read-only parsing sufficient:**
- Use `js-yaml` for faster parsing
- Because: 2x faster, smaller footprint

**If LiteLLM proxy unavailable:**
- Fall back to direct model selection
- Because: Graceful degradation already built into system

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| js-yaml@4.x | Node.js >=12 | Safe for Node.js >=16 requirement |
| yaml@2.x | Node.js >=14 | Safe for Node.js >=16 requirement |
| commander@12.x | Node.js >=18 | Already using 12.1.0, compatible |

---

## Architecture Integration

### Model Routing Flow

```
Worker Spawn Request
    |
    v
Caste Identification (builder, watcher, etc.)
    |
    v
Model Profile Lookup (.aether/model-profiles.yaml)
    |
    v
js-yaml parse -> Get model assignment
    |
    v
LiteLLM Proxy Validation (localhost:4000)
    |
    v
Environment Variable Injection
    |
    v
Task Tool Spawn with ANTHROPIC_MODEL set
```

### Colony Lifecycle Flow

```
aether archive --note "Completed phase 3"
    |
    v
Read COLONY_STATE.json
    |
    v
Create timestamped archive directory (.aether/data/archive/<timestamp>/)
    |
    v
Copy state files + manifest.json
    |
    v
Update COLONY_STATE (archived_at, state: ARCHIVED)
    |
    v
Log to activity log

aether foundation --restore <archive-id>
    |
    v
Validate archive exists and is valid
    |
    v
Create checkpoint of current state
    |
    v
Restore from archive
    |
    v
Update COLONY_STATE (restored_from, state: INITIALIZING)
    |
    v
Log to activity log
```

---

## File Structure

```
bin/
  cli.js                    # Add model and foundation commands
  lib/
    model-verify.js         # Already exists - extend for validation
    model-profiles.js       # NEW: YAML parsing for model-profiles.yaml
    model-router.js         # NEW: LiteLLM proxy integration
    lifecycle-archive.js    # NEW: Archive command implementation
    lifecycle-foundation.js # NEW: Foundation command implementation
    state-guard.js          # Already exists - extend for archive state
```

---

## Sources

- [js-yaml GitHub](https://github.com/nodeca/js-yaml) - Industry standard YAML parser
- [eemeli/yaml GitHub](https://github.com/eemeli/yaml) - Modern YAML library with round-trip support
- [LiteLLM Documentation](https://docs.litellm.ai/) - Proxy configuration and API
- [commander.js Documentation](https://github.com/tj/commander.js/) - CLI framework patterns
- Existing codebase: `/Users/callumcowie/repos/Aether/bin/cli.js`
- Existing codebase: `/Users/callumcowie/repos/Aether/.aether/model-profiles.yaml`
- Existing codebase: `/Users/callumcowie/repos/Aether/bin/lib/model-verify.js`

---

*Stack research for: Aether Colony System v3.1 - Open Chambers*
*Researched: 2026-02-14*
