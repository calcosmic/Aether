# Host Command Reference

The `aether host` command delegates to the TypeScript orchestration host for all major workflows. The Go CLI spawns the TS host process; the TS host parses arguments, calls Go CLI subcommands via JSON, and renders output.

## Subcommands

### `aether host plan [flags]`

Run the plan workflow via the TS host.

**Flags:**
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
- `--synthetic` — Skip real worker dispatch (Go CLI flag)
- `--simulate` — Run in simulation mode (TS host flag). No real workers are spawned; synthetic results are produced. Required when no platform CLI is installed.
- `--light` — Force light review
- `--worker-timeout <duration>` — e.g. `15m`
- `--no-dashboard` — Plain text output

**Error:** Without `--simulate`, build requires a platform CLI (claude, opencode, or codex) to be installed.

**Example:**
```bash
aether host build 1 --light
```

---

### `aether host continue [flags]`

Run the continue workflow via the TS host.

**Flags:**
- `--verification-depth <level>` — light | standard | heavy
- `--light` — Force light review
- `--heavy` — Force heavy review
- `--synthetic` — Mark as synthetic (Go CLI flag)
- `--simulate` — Run in simulation mode (TS host flag). No real workers are spawned.
- `--worker-timeout <duration>` — e.g. `15m`
- `--no-dashboard` — Plain text output

**Example:**
```bash
aether host continue --verification-depth heavy
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

## Test Coverage

All subcommands and flags are exercised in the test suite:

- **Flag parsing**: `.aether/ts-host/test/host-flags.test.ts` verifies `parseArgs` handles all documented flags correctly.
- **Integration**: `.aether/ts-host/test/host-integration.test.ts` verifies each subcommand builds the correct Go CLI arguments.

Run tests: `cd .aether/ts-host && npm test`
