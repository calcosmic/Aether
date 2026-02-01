# External Integrations

**Analysis Date:** 2025-02-01

## APIs & External Services

**None detected** - The Aether system is entirely self-contained with no external API integrations.

**Note:** The system is designed to be Claude-native, meaning it operates through Claude's tool interface rather than direct API calls.

## Data Storage

**Databases:**
- **File-based JSON storage** (No traditional database)
  - Working memory: `.aether/memory/` directory
  - Short-term memory: `.aether/memory/short_term_memory.py` (in-memory)
  - Long-term memory: `.aether/memory/long_term.json`
  - Meta-learning: `.aether/memory/meta_learning.json`
  - Checkpoints: `.aether/checkpoints/` directory
  - Colony state: `.aether/data/` directory
  - Error logs: `.aether/errors/` directory

**File Storage:**
- Local filesystem only (`.aether/` directory tree)
- No cloud storage integration
- No object storage (S3, GCS, etc.)

**Caching:**
- In-memory caching for:
  - Working memory token budget
  - Pheromone signals (in-memory list)
  - Semantic embeddings (optional numpy arrays)
- No external cache (Redis, Memcached)

## Authentication & Identity

**Auth Provider:**
- **None** (Local CLI tool, no authentication)
- Implementation: CLI commands via argparse
- No user accounts, sessions, or authentication tokens

## Monitoring & Observability

**Error Tracking:**
- **Custom error ledger** (`error_prevention.py`)
  - Error categorization (ErrorCategory enum)
  - Severity tracking (ErrorSeverity enum)
  - Pattern detection for recurring errors
  - JSON file storage: `.aether/errors/`
  - No external error tracking (Sentry, Rollbar, etc.)

**Logs:**
- **Console output** (print statements)
- JSON file logging for errors: `.aether/errors/err_*.json`
- Error patterns tracked: `.aether/errors/patterns.json`
- No structured logging (no loguru, structlog)

## CI/CD & Deployment

**Hosting:**
- **None** (Local CLI tool)
- Runs as Python module: `python -m aether`
- No web server, no hosting platform

**CI Pipeline:**
- **None detected** (no .github/, .gitlab-ci.yml, etc.)
- Manual execution only

## Environment Configuration

**Required env vars:**
- **None** (All configuration through CLI arguments and JSON files)

**Secrets location:**
- No secrets management (no API keys, tokens)
- All state stored locally in `.aether/`

## Webhooks & Callbacks

**Incoming:**
- **None** (CLI tool, no HTTP server)

**Outgoing:**
- **None** (No external HTTP requests)

## Optional Integrations

**Semantic Layer (Optional):**
- **sentence-transformers** - Text embeddings
  - Model: all-MiniLM-L6-v2 (384 dimensions)
  - Purpose: Semantic pheromone communication
  - Fallback: Hash-based embeddings if unavailable
  - No API calls (local model)
  - Storage: `.cache/embeddings.json` (cached embeddings)

**Testing Frameworks (Referenced):**
- **pytest** - Test execution (referenced in patterns, not integrated)
- **hypothesis** - Property-based testing (referenced, not integrated)
- Usage: Pattern references in `worker_ants.py` for test generation templates

## Claude Integration

**Claude Native Design:**
- The system is designed to be invoked through Claude's tool interface
- Commands: `/ant:init`, `/ant:plan`, `/ant:execute`, etc.
- State persistence between Claude interactions
- No direct Anthropic API integration (relies on Claude's tool calling)

---

*Integration audit: 2025-02-01*
