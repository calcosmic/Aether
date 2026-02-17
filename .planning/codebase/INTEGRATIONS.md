# External Integrations

**Analysis Date:** 2026-02-17

## APIs & External Services

**AI Provider (Anthropic):**
- Anthropic API - Primary AI model provider
  - SDK/Client: Native fetch (Node 18+) via `bin/lib/model-verify.js` and `bin/lib/proxy-health.js`
  - Auth: `ANTHROPIC_API_KEY` environment variable
  - Configuration: `ANTHROPIC_MODEL` (default model), `ANTHROPIC_BASE_URL` (proxy endpoint, defaults to Anthropic)
  - Used by: Worker spawning, oracle research, colony intelligence

**Proxy Health Check:**
- Local proxy endpoint - Optional middleware for model routing
  - Endpoint: `http://localhost:4000/health` (checked in `bin/lib/proxy-health.js:36`)
  - Endpoint: `http://localhost:4000/models` (checked in `bin/lib/proxy-health.js:152`)
  - Used for: Verifying model routing configuration

## Data Storage

**Local State:**
- Filesystem-based JSON storage
  - Location: `.aether/data/COLONY_STATE.json`
  - Location: `.aether/data/constraints.json`
  - Location: `.aether/data/checkpoint-allowlist.json`
  - Client: Native fs module (Node.js)

**Session/Checkpoint Storage:**
- Filesystem-based
  - Location: `.aether/checkpoints/` - Session snapshots
  - Location: `.aether/locks/` - File-based locking
  - Location: `.aether/chambers/` - Archived colonies

**No external database** - All data stored locally in the repo

## Version Control & Distribution

**Git:**
- Repository hosting: GitHub
  - URL: `git+https://github.com/calcosmic/Aether.git`
  - Used for: Version control, branching strategy
  - Integration: Full git integration in `.aether/aether-utils.sh`

**npm Registry:**
- Package distribution
  - Package name: `aether-colony`
  - Published from: Aether repo `runtime/` directory
  - Distribution flow: `.aether/` (source) -> `runtime/` (staging) -> `~/.aether/` (hub) -> target repos

## Authentication & Identity

**Auth Provider:**
- Not applicable - This is a CLI tool, not a service requiring auth

**Environment-based Configuration:**
- `HOME` / `USERPROFILE` - User home directory for path resolution
- `NO_COLOR` - Disable colored output

## CI/CD & Deployment

**Hosting:**
- npm Registry - Package distribution
- GitHub - Source code hosting

**CI Pipeline:**
- GitHub Actions - CI/CD automation
  - Config: `.github/workflows/ci.yml`
  - Runs: Tests, linting, shellcheck

## Monitoring & Observability

**Error Tracking:**
- Custom structured error system in `bin/lib/errors.js`
  - Error codes: E_* constants
  - JSON-formatted error output

**Logs:**
- Activity log: `.aether/data/activity.log`
- Implementation: `bin/lib/logger.js`

## Environment Configuration

**Required env vars:**
- `HOME` or `USERPROFILE` - User home directory (required for CLI initialization)
- `ANTHROPIC_API_KEY` - AI API key (required for worker spawning)
- `ANTHROPIC_MODEL` - Model name (optional, for model routing)
- `ANTHROPIC_BASE_URL` - Custom API endpoint (optional, for proxy support)

**Optional env vars:**
- `NO_COLOR` - Disable colored output
- `WORKER_NAME` - Override worker name (testing)
- `CASTE` - Override caste assignment (testing)

## Webhooks & Callbacks

**Incoming:**
- Not applicable - No HTTP server, purely CLI-based

**Outgoing:**
- Not applicable - No external webhook calls made

## Cross-Repo Integration

**Hub Distribution Model:**
- Aether acts as a "hub" distributing to target repos
- Flow: Aether repo -> `npm install -g .` -> `~/.aether/` hub -> target repo `aether update`
- Files synced: `.aether/workers.md`, `.aether/aether-utils.sh`, `.aether/utils/`, `.aether/docs/`

---

*Integration audit: 2026-02-17*
