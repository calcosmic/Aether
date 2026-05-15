# Host Command Reference

The `aether host` command delegates to the TypeScript orchestration host for all major workflows. The Go CLI spawns the TS host process; the TS host parses arguments, calls Go CLI subcommands via JSON, and renders output.

## Subcommands

### `aether host plan [flags]`

Run the plan workflow via the TS host.

**Flags:**
- `--depth <level>` — fast | balanced | deep | exhaustive
- `--planning-depth <level>` — light | standard | deep
- `--verification-depth <level>` — light | standard | heavy
- `--synthetic` — Skip real worker dispatch
- `--worker-timeout <duration>` — e.g. `5m`, `15m`
- `--no-dashboard` — Plain text output (no live dashboard)

**Example:**
```bash
aether host plan --depth balanced --planning-depth standard
```

---

### `aether host build <phase> [flags]`

Run the build workflow for a phase via the TS host.

**Flags:**
- `--synthetic` — Skip real worker dispatch
- `--light` — Force light review
- `--worker-timeout <duration>` — e.g. `15m`
- `--no-dashboard` — Plain text output

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
- `--synthetic` — Mark as synthetic
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
- `--simulate` — Simulation mode (no real worker spawning)
- `--no-dashboard` — Plain text output

**Example:**
```bash
aether host oracle "security audit"
```

---

### `aether host lifecycle [phase] [oracle-topic]`

Run the full plan→build→continue sequence via the TS host.

**Flags:**
- `--simulate` — Simulation mode
- `--no-dashboard` — Plain text output
- `--skip-midden-check` — Skip pre-build midden threshold check

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
