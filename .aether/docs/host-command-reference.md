# Host Command Reference

The `aether host` command delegates selected host-backed workflows to the
TypeScript orchestration host. The Go CLI spawns the TS host process; the TS
host parses arguments, calls Go CLI subcommands via JSON, and renders output.
Go remains the source of truth for state, finalizers, provider diagnostics, and
canonical ceremony output.

## Subcommands

### `aether host colonize [flags]`

Run the colonize survey manifest path via the TS host.

**Flags:**
- `--force-resurvey` — Refresh existing survey artifacts
- `--force` — Alias for `--force-resurvey`
- `--worker-timeout <duration>` — e.g. `5m`
- `--no-dashboard` — Plain text output

**Example:**
```bash
aether host colonize --force-resurvey
```

---

### `aether host plan [flags]`

Run the plan workflow via the TS host.

**Flags:**
- `--refresh` — Regenerate the plan when one already exists
- `--force` — Alias for `--refresh`
- `--depth <level>` — fast | balanced | deep | exhaustive
- `--planning-depth <level>` — light | standard | deep
- `--verification-depth <level>` — light | standard | heavy
- `--synthetic` — Skip real worker dispatch (Go CLI flag)
- `--worker-timeout <duration>` — e.g. `5m`, `15m`
- `--no-dashboard` — Plain text output (no live dashboard)

**Note:** `--synthetic` is the Go CLI flag that skips real worker dispatch during planning. It is distinct from `--simulate` (TS host flag) which gates simulation in the build/lifecycle/orchestration layer.

**Example:**
```bash
aether host plan --depth balanced --planning-depth standard
```

---

### `aether host build <phase> [flags]`

Run the build workflow for a phase via the TS host.

**Flags:**
- `--task <id>` — Redispatch only the specified task ID; repeat for multiple IDs
- `--force` — Force redispatch of an interrupted active phase
- `--synthetic` — Skip real worker dispatch (Go CLI flag)
- `--simulate` — Run in simulation mode (TS host flag). No real workers are spawned; synthetic results are produced. Required when no platform CLI is installed.
- `--light` — Force light review
- `--heavy` — Force heavy review
- `--verification-depth <level>` — light | standard | heavy
- `--worker-timeout <duration>` — e.g. `15m`
- `--circuit-breaker-threshold <n>` — Consecutive failures before a worker circuit breaker trips
- `--no-suggest` — Skip pheromone suggestion analysis during build
- `--verbose` — Show full worker output
- `--no-dashboard` — Plain text output

**Error:** Without `--simulate`, build requires a platform CLI (claude, opencode, or codex) to be installed.

**Example:**
```bash
aether host build 1 --light
```

---

### `aether host continue [flags]`

Run the heavy external-review continue manifest path via the TS host. The
default continue path remains Go-owned through
`aether continue --skip-watchers --verification-depth standard`; use
`aether host continue --classic-ceremony` only when the older visible review
ritual, heavy review, or runtime guidance asks for wrapper-spawned reviewers.

**Flags:**
- `--reconcile-task <id>` — Mark task reconciliation before continue gating; repeat for multiple IDs
- `--verification-depth <level>` — light | standard | heavy
- `--verification-timeout <duration>` — Override deterministic verification timeout, e.g. `30m`
- `--light` — Force light review
- `--heavy` — Force heavy review
- `--skip-watchers` — Skip watcher agent spawn when Go allows it
- `--synthetic` — Mark as synthetic (Go CLI flag)
- `--simulate` — Run in simulation mode (TS host flag). No real workers are spawned.
- `--worker-timeout <duration>` — e.g. `15m`
- `--no-learn` — Skips only the legacy learning-entry capture; observations and failure records are still written so a blocked check still records what broke
- `--classic-ceremony` — Named shortcut for the real heavy-review manifest path
- `--no-dashboard` — Plain text output

**Example:**
```bash
aether host continue --classic-ceremony
```

---

### `aether host seal [flags]`

Run the final seal review manifest path via the TS host.

**Flags:**
- `--force` — Forward the runtime force flag when blockers are intentionally accepted
- `--no-dashboard` — Plain text output

**Example:**
```bash
aether host seal
```

---

### `aether host oracle [topic]`

Run the Oracle RALF lifecycle loop via the TS host.

**Flags:**
- `--simulate` — Run in simulation mode (no real worker spawning). Required when no platform CLI is installed.
- `--no-dashboard` — Plain text output

**Error:** Without `--simulate`, oracle requires a platform CLI (claude, opencode, or codex) to be installed.

**Example:**
```bash
aether host oracle "security audit"
```

---

### `aether host lifecycle [phase] [oracle-topic]`

Run the full plan→build→continue sequence via the TS host.

**Flags:**
- `--simulate` — Run in simulation mode (no real worker spawning). Required when no platform CLI is installed.
- `--no-dashboard` — Plain text output
- `--skip-midden-check` — Skip pre-build midden threshold check

**Error:** Without `--simulate`, lifecycle requires a platform CLI (claude, opencode, or codex) to be installed.

**Example:**
```bash
aether host lifecycle 1 "security audit"
```

---

### `aether host watch [flags]`

Show colony status via the TS host.

**Flags:**
- `--no-dashboard` — Plain text output instead of live dashboard

**Example:**
```bash
aether host watch --no-dashboard
```

---

### `aether host swarm [target]`

Show swarm plan for a target problem via the TS host.

**Flags:**
- `--no-dashboard` — Plain text output instead of live dashboard

**Example:**
```bash
aether host swarm "test bug" --no-dashboard
```

---

## Global Options

- `--cwd <path>` — Working directory (default: current directory)

## Provider Availability And Auth Diagnostics

Before real worker dispatch, the Go runtime selects a platform provider and
performs an availability preflight. That preflight checks whether the platform
CLI exists and whether its auth probe reports usable credentials. It returns a
structured `AvailabilityStatus` category such as `binary_missing`,
`auth_probe_failed`, `auth_inactive`, `invalid_auth_output`,
`credentials_missing`, `probe_skipped`, or `available`.

In plain terms: preflight only says Aether can try to launch a worker. It does
not prove that the provider account, model, proxy, or upstream API will accept
the later worker request.

Host and wrapper surfaces must show only the sanitized provider, cause, and
next action from the Go-owned diagnostic. Do not print raw provider
stdout/stderr, tokens, or auth probe output. If a worker launches but the
provider later returns an API/auth payload instead of worker claims JSON, treat
that as a post-launch provider/API/auth failure, not as provider availability
preflight failure.

## Command Spine Boundary

The TypeScript host is the command spine for host-backed workflows, not a second
runtime. Direct host commands parse host flags and call Go in JSON mode for
manifests. The wrapper/lifecycle orchestration layer built on those manifests
invokes Go-owned ceremony commands and returns worker completion packets to Go
finalizers.

For beginners: TypeScript lines up the work; Go remains the engine that changes
state and decides the official result.

Current host-backed orchestration surfaces:

- `aether host colonize` delegates to `aether colonize --plan-only`; Go still
  owns `aether colonize-finalize`.
- `aether host plan` delegates to `aether plan --plan-only`; Go still owns
  `aether plan-finalize`.
- `aether host build` delegates to `aether build <phase> --plan-only`; Go still
  owns `aether build-finalize`.
- `continue` uses the host only for heavy external review; default continue is
  Go-owned through `aether continue --skip-watchers --verification-depth standard`.
- `aether host continue` delegates to `aether continue --plan-only` for that
  heavy-review manifest path. `--classic-ceremony` is the named shortcut for
  that visible review ritual; Go still owns `aether continue-finalize`.
- `aether host seal` delegates to `aether seal --plan-only`; Go still owns
  `aether seal-finalize`.
- `aether host oracle`, `aether host watch`, and `aether host swarm` expose
  lifecycle/display surfaces. `watch` displays Go-owned status facts. `swarm`
  fetches and displays the Go swarm plan for problem runs; `swarm --watch`
  remains dashboard-style runtime facts. Canonical wrapper finalization remains
  Go-owned.

All lifecycle manifest-generation surfaces that need wrapper worker ceremony are
now on the TS host spine: `colonize`, `plan`, `build`, heavy/classic
`continue`, and `seal`.

The machine-readable parity contract is
`.aether/commands/classic-command-parity.json`; the human-readable companion is
`.aether/docs/classic-command-parity-matrix.md`.

## Ceremony Classes Outside The Host Spine

Some important commands are intentionally not host-backed. `update`, `publish`,
`install`, `lay-eggs`, `porter`, `source-check`, `run`, `continue`, and
`bump-version` are progress/delivery surfaces: the Go runtime owns the work and
the wrapper should show progress without inventing worker dispatch. Finalizers,
`command-guide`, spawn log helpers, ceremony helper commands, shell completion,
version helpers, and generated progress helpers are quiet/internal surfaces.

For beginners: these commands can be important, but they are not a colony worker
show. They are the engine doing setup, checks, release chores, or wrapper
plumbing.

Release-surface note: `.opencode/package.json` and
`.opencode/package-lock.json`, when present, are ignored local OpenCode install
artifacts. Do not treat them as release-surface package files unless that scope
changes. The tracked TS-host package manifests live under `.aether/ts-host/`.

## Flag Distinction: `--synthetic` vs `--simulate`

| Flag | Layer | Purpose |
|------|-------|---------|
| `--synthetic` | Go CLI | Skips real worker dispatch in Go-owned commands (plan, build, continue) |
| `--simulate` | TS host | Gates simulation in the TypeScript orchestration layer (build, oracle, lifecycle) |

The TS host forwards `--synthetic` to the Go CLI when constructing subcommand arguments. `--simulate` is consumed by the TS host itself to decide whether to spawn real platform workers or produce synthetic results.

## Architecture Note

```
User → aether host <subcommand> [flags]
         ↓ (Go CLI)
       node dist/host.js <subcommand> [flags]
         ↓ (TS host parses flags)
       aether <subcommand> --plan-only [flags] → JSON
         ↓ (TS host renders)
       stdout (JSON or formatted text)
```

The TS host is the authoritative flag parser. Go CLI host subcommands use `DisableFlagParsing: true` to forward raw arguments to the TS host.

## Worker Spawning

Workers can request additional child workers mid-build by returning a `spawns` array in their claims JSON:

```json
{
  "status": "completed",
  "spawns": [
    { "caste": "scout", "task": "Research API options", "reason": "Need context" }
  ]
}
```

### Spawn Rules

- **Budget**: Total workers (manifest + spawned) cannot exceed the Queen spawn budget (`max_workers` from manifest). Excess spawn requests are logged and skipped.
- **Depth**: Maximum spawn depth is 2. Manifest workers (depth 1) can spawn children (depth 2). Children cannot spawn further.
- **Parent tracking**: Child spawns are recorded in the spawn tree with their actual parent worker name, not "Queen".
- **Results**: Child worker results are attached to the parent worker's handoff, visible to downstream workers.

### Spawn Lifecycle

1. Worker completes and returns claims with `spawns` array
2. Wave orchestrator validates spawns against budget and depth
3. Accepted spawns are synthesized into child dispatch entries
4. Child workers dispatched in a spawn wave after the parent wave
5. Child results attached to parent handoff

## Test Coverage

All subcommands and flags are exercised in the test suite:

- **Flag parsing**: `.aether/ts-host/test/host-flags.test.ts` verifies `parseArgs` handles all documented flags correctly.
- **Integration**: `.aether/ts-host/test/host-integration.test.ts` verifies each subcommand builds the correct Go CLI arguments.

Run tests: `cd .aether/ts-host && npm test`
