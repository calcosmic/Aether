# External Integrations

**Analysis Date:** 2026-02-17

## APIs & External Services

**LLM Providers (via LiteLLM proxy):**
- **LiteLLM Proxy** - Model routing gateway
  - Endpoint: `http://localhost:4000`
  - Auth: Token-based (`ANTHROPIC_AUTH_TOKEN` or `LITELLM_AUTH_TOKEN`)
  - Health check: `http://localhost:4000/health`
  - Used by: `.aether/utils/spawn-with-model.sh`, `aether-utils.sh` model verification

**Model Providers:**
- **Z.AI** (glm-5) - Planning, coordination, long-horizon tasks
- **Minimax** (minimax-2.5) - Architectural planning, research
- **Kimi** (kimi-k2.5) - Coding, validation, swarm work

**AI Agent Clients:**
- **Claude Code** (`claude` CLI) - Primary worker agent
  - Spawned as colony workers
  - Configured via `ANTHROPIC_MODEL`, `ANTHROPIC_BASE_URL`, `ANTHROPIC_AUTH_TOKEN`
  - Location: `bin/cli.js`, `.aether/aether-utils.sh` spawn functions

- **OpenCode** (`opencode` CLI) - Alternative worker agent
  - Fallback when Claude Code unavailable
  - Used by: `.aether/oracle/oracle.sh`

## Data Storage

**File-based (local):**
- Colony state: `.aether/data/COLONY_STATE.json`
- Pheromones: `.aether/data/pheromones.json`
- Session: `.aether/data/session.json`
- Spawn tree: `.aether/data/spawn-tree.txt`

**No external database required.**

## Authentication & Identity

**Auth Provider:**
- LiteLLM proxy token authentication
  - Environment: `ANTHROPIC_AUTH_TOKEN` or `LITELLM_AUTH_TOKEN`
  - Default: `sk-litellm-local` (local dev)

**No user authentication system.**

## Monitoring & Observability

**Error Tracking:**
- None (local-only operation)

**Logs:**
- Activity log: `.aether/data/activity.log`
- Spawn logging: `.aether/data/spawn-tree.txt`
- Telemetry: `.aether/data/telemetry.json` (if enabled)

## CI/CD & Deployment

**Hosting:**
- npm registry - Package distribution
- GitHub - Source repository

**CI Pipeline:**
- Not detected (no `.github/workflows/`)

## Environment Configuration

**Required env vars:**
- None strictly required (defaults provided)

**Optional env vars:**
- `ANTHROPIC_BASE_URL` - LLM proxy endpoint (default: `http://localhost:4000`)
- `ANTHROPIC_AUTH_TOKEN` - Proxy auth token
- `ANTHROPIC_MODEL` - Default model for workers
- `LITELLM_AUTH_TOKEN` - Alternative proxy auth token

**Secrets location:**
- No secrets stored in codebase
- LiteLLM proxy token is local/dev only

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Key Integration Patterns

**Model Routing Flow:**
```
1. Queen determines caste for task
2. Look up model in .aether/model-profiles.yaml
3. Export ANTHROPIC_MODEL, ANTHROPIC_BASE_URL, ANTHROPIC_AUTH_TOKEN
4. Spawn Claude Code with environment variables
5. LiteLLM proxy routes to correct provider
```

**Spawn Integration:**
```
aether-utils.sh spawn → spawn-with-model.sh → Claude Code CLI
```

**XML Pheromone Exchange:**
```
pheromone-xml.sh → xmlstarlet/xmllint → JSON state
```

---

*Integration audit: 2026-02-17*
