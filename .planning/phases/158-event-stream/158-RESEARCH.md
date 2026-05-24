---
phase: 158-event-stream
generated_at: "2026-05-24"
source: inline (planner skill stalled)
---

# Phase 158 Research: Event Stream

## Context

Phase 158 builds the NDJSON event stream that serves as the shared observable truth between the Go runtime spine and the TypeScript control plane. Both platforms must agree on the event format, and both must be able to read and write events concurrently.

## Technical Approach

### NDJSON Format

- One JSON object per line, newline-delimited
- Each line: `{ "type": "...", "timestamp": "...", "payload": { ... } }`
- No trailing commas, no pretty-printing (to keep lines compact)
- UTF-8 encoding

### File Location

- `.aether/events/current.ndjson` — the active event stream
- `.aether/events/` directory created if it doesn't exist
- File opened with append mode for atomic line writes

### Concurrency Model

- Go: mutex-protected file writes, fsnotify or polling for reads
- TypeScript: async file operations, fs.watch or polling for reads
- Both sides treat the file as append-only log
- No locks across processes (OS-level file append is atomic per line)

### Event Types

Both platforms share these event types:
- `phase_start` — colony phase beginning
- `agent_selected` — worker chosen for task
- `task_planned` — task breakdown created
- `worker_spawned` — agent launched
- `tool_call` — tool invocation
- `verification_result` — quality gate outcome
- `memory_update` — learning/instinct recorded
- `run_seal` — colony completion
- `custom` — extensible event type

### Go Patterns

- Use `encoding/json` for JSON marshaling
- Use `os.OpenFile` with `O_APPEND|O_WRONLY|O_CREATE` for writes
- Use `bufio.Scanner` for line-by-line reading
- Use `fsnotify` or polling for tail behavior

### TypeScript Patterns

- Use native `JSON.stringify`/`JSON.parse`
- Use `fs.promises.appendFile` for writes
- Use `readline` or custom stream parser for reads
- Use `fs.watch` or polling for tail behavior

## Pitfalls to Avoid

1. **Partial line reads** — always read complete lines before parsing JSON
2. **Corrupted JSON** — handle parse errors gracefully, skip bad lines
3. **Race conditions** — use atomic append operations, not read-modify-write
4. **File rotation** — consider `.aether/events/YYYY-MM-DD.ndjson` pattern for long-running colonies
5. **Memory growth** — don't load entire event stream into memory; stream it

## Dependencies

- Phase 155 (Go Boundary Refactor) — event types must match Go runtime shapes
- Phase 156 (TS Control Plane Core) — event module lives in control-ts
- Phase 157 (TS Adapters & Oracle) — adapters may emit events
