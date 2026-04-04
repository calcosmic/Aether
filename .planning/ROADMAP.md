# Roadmap: Aether v5.4 Shell-to-Go Conversion

## Overview

This milestone replaces all shell script invocations with Go binary calls. The Go binary already has 153 commands overlapping with shell, but slash commands, playbooks, and 101 shell-only commands still need work. We port the missing commands first (so Go equivalents exist), then wire everything to call the Go binary, verify parity, and ship.

## Phases

**Phase Numbering:**
- Integer phases (05, 06, ...): Planned milestone work (continues from previous milestone ending at Phase 04)
- Decimal phases (05.1, 05.2): Urgent insertions (marked with INSERTED)

- [x] **Phase 05: Structural Learning & Curation Ants** - Port the most complex self-contained shell modules to Go (25 commands) -- completed 2026-04-03
- [x] **Phase 06: XML, Display & Semantic Search** - Port XML exchange, display helpers, and suggestion commands to Go (15 commands) -- completed 2026-04-03
- [x] **Phase 07: Error Handling & Miscellaneous** - Port remaining error handling and misc commands to Go (~38 commands) (completed 2026-04-04)
- [x] **Phase 08: Slash Command Wiring** - Rewire all 42 Claude + 45 OpenCode slash commands to call Go binary (completed 2026-04-04)
- [x] **Phase 09: Playbook Wiring** - Rewire all 11 build/continue playbooks to call Go binary (completed 2026-04-04)
- [ ] **Phase 10: Integration Parity Tests** - Build shell vs Go comparison framework, verify all 254 commands
- [ ] **Phase 11: Distribution** - Cross-platform builds, npm bridge, CI pipeline, ship Go-primary

## Phase Details

### Phase 05: Structural Learning & Curation Ants
**Goal**: The structural learning stack and curation ant pipeline run entirely through Go, replacing the shell-based trust scoring, event bus, instinct storage, graph layer, consolidation, and 9 curation ant commands.
**Depends on**: Phase 04 (session-context commands complete)
**Requirements**: PORT-01, PORT-02
**Success Criteria** (what must be TRUE):
  1. `aether trust-score-compute` produces identical tier assignments as the shell version for the same observation inputs
  2. `aether event-bus-publish` and `event-bus-read` operate on JSONL files with the same TTL and subscription semantics as the shell event bus
  3. `aether instinct-create` stores instincts with full provenance and they appear in COLONY_STATE.json exactly as the shell version writes them
  4. `aether consolidation-seal` runs the full curation pipeline (8 ants + orchestrator) and produces the same archive output as the shell pipeline
  5. All 25 ported commands pass `go test ./cmd/...` with no regressions in existing tests
**Plans**: TBD

### Phase 06: XML, Display & Semantic Search
**Goal**: XML exchange commands use native Go XML handling instead of xmllint/xmlstarlet, display helpers render swarm trees and spawn trees in Go, and suggestion commands run entirely through Go.
**Depends on**: Phase 05
**Requirements**: PORT-03, PORT-04, PORT-05
**Success Criteria** (what must be TRUE):
  1. `aether pheromone-export-xml` produces byte-identical XML output to the shell `xml-compose` pipeline
  2. `aether pheromone-import-xml` reads XML exported by the shell version and loads all signals correctly
  3. `aether swarm-display` renders the same ASCII tree layout as the shell version for identical colony state
  4. `aether suggest-analyze` produces actionable suggestions with the same deduplication behavior as the shell version
  5. `aether suggest-approve` applies approved suggestions to pheromones.json with the same format as the shell version
**Plans**: TBD

### Phase 07: Error Handling & Miscellaneous
**Goal**: All remaining shell-only commands have Go equivalents -- error handling, midden operations, state mutations, flag management, session commands, and every other subcommand that slash commands or playbooks call.
**Depends on**: Phase 06
**Requirements**: PORT-06, PORT-07
**Success Criteria** (what must be TRUE):
  1. `aether error-add` and `error-flag-pattern` record errors with the same classification and suppression behavior as the shell versions
  2. Every subcommand listed in `aether-utils.sh` that does not yet have a Go equivalent now has one registered in the Go binary
  3. `aether help` lists all 254+ commands with no "shell-only" gaps remaining
  4. Running `aether <cmd> --help` for any newly ported command shows usage, flags, and description without errors
  5. All newly ported commands pass unit tests confirming they parse arguments and produce output without panics
**Plans**: 4 plans

Plans:
- [x] 07-01-PLAN.md -- Error handling, midden-write, security scanning (8 commands)
- [x] 07-02-PLAN.md -- Core build-flow utilities: name gen, commit msg, progress bar, version, milestone, progress update (8 commands)
- [x] 07-03-PLAN.md -- XML exchange aliases, pheromone display, context update, eternal init (10 commands)
- [x] 07-04-PLAN.md -- Learning pipeline + remaining internal utilities (20 commands)

### Phase 08: Slash Command Wiring
**Goal**: Every slash command in Claude Code and OpenCode invokes `aether <cmd>` instead of `bash .aether/aether-utils.sh <cmd>`, with shell fallback only for any remaining gaps.
**Depends on**: Phase 07
**Requirements**: WIRE-01, WIRE-02, WIRE-03
**Success Criteria** (what must be TRUE):
  1. Running `/ant:status` in Claude Code invokes the Go binary and displays the colony dashboard (no shell dispatcher call)
  2. Running `/ant:pheromones` in Claude Code displays signals via Go with identical output format to the shell version
  3. Running the OpenCode equivalent of `/ant:build 1` invokes the Go binary for all subcommand calls in the build flow
  4. Any command that lacks a Go equivalent falls back to shell with a visible deprecation notice on stderr
  5. The 42 Claude commands and 45 OpenCode commands all have their `bash .aether/aether-utils.sh` calls replaced with `aether` calls in their YAML sources
**Plans**: 3 plans

Plans:
- [x] 08-01-PLAN.md -- Go prerequisites: --json flags for table commands + normalize-args command
- [x] 08-02-PLAN.md -- YAML wiring: update all 45 YAML sources + generator preamble
- [ ] 08-03-PLAN.md -- Regenerate + verify: produce 90 .md files, validate zero shell calls

### Phase 09: Playbook Wiring
**Goal**: All 11 build and continue playbooks call the Go binary for every subcommand invocation, making the full build-verify-advance cycle Go-native.
**Depends on**: Phase 08
**Requirements**: MIGRATE-04, MIGRATE-05
**Success Criteria** (what must be TRUE):
  1. A complete `/ant:build` cycle (prep, context, wave, verify, complete) executes using only Go binary calls with no shell dispatcher invocations
  2. A complete `/ant:continue` cycle (verify, gates, advance, finalize) executes using only Go binary calls with no shell dispatcher invocations
  3. Build playbooks correctly call `aether generate-ant-name`, `aether context-capsule`, and all other subcommands that workers invoke during execution
  4. Continue playbooks correctly call `aether update-progress`, `aether midden-write`, and all other subcommands called during verification and learning extraction
**Plans**: 2 plans

Plans:
- [x] 09-01-PLAN.md -- Convert 5 large playbooks (204 shell calls): build-full, continue-full, build-wave, continue-advance, build-verify
- [x] 09-02-PLAN.md -- Convert 6 remaining playbooks (67 shell calls) + full verification across all 11 files

### Phase 10: Integration Parity Tests
**Goal**: A systematic test framework verifies that every Go command produces the same output as its shell counterpart for the same inputs, and the full test suite passes at the same or higher rate.
**Depends on**: Phase 09
**Requirements**: TEST-01, TEST-02, TEST-03, TEST-04
**Success Criteria** (what must be TRUE):
  1. A shell-vs-Go comparison test harness exists that can run any subcommand through both paths and diff the output
  2. All 142 overlapping commands pass behavioral parity tests with semantic comparison
  3. All 99 Go-only commands pass functional smoke tests (no panic + valid JSON envelope)
  4. The full test suite (879 tests) passes at the same or higher rate (at most 9 pre-existing failures)
  5. The parity test framework runs in CI as part of the Go test job
**Plans**: 3 plans

Plans:
- [x] 10-01-PLAN.md -- Parity harness + semantic comparators + 142 overlapping command parity tests (TEST-01, TEST-02)
- [ ] 10-02-PLAN.md -- Go-only command functional tests: 99 commands smoke tests + 5 deeper functional tests (TEST-03)
- [ ] 10-03-PLAN.md -- Fix parity breaks, full suite verification, CI integration (TEST-04)

### Phase 11: Distribution
**Goal**: Users install Aether via `npm install -g aether-colony` and get a Go binary -- the npm package becomes a thin downloader, shell scripts are retained as fallback only, and CI validates both Go and Node paths.
**Depends on**: Phase 10
**Requirements**: DIST-01, DIST-02, DIST-03, DIST-04, DIST-05
**Success Criteria** (what must be TRUE):
  1. `npm install -g aether-colony` on macOS (amd64 and arm64), Linux (amd64), and Windows (amd64) downloads the correct Go binary and the `aether` command works immediately
  2. goreleaser produces cross-platform artifacts (tar.gz, checksums) for all four platform/architecture combinations
  3. CI runs both Go build/test/vet and Node lint/test jobs, and PRs must pass both to merge
  4. The npm postinstall script downloads the platform-correct binary from GitHub Releases (no bundled binary in the npm tarball)
  5. Shell scripts remain in `.aether/` as fallback but are not the primary execution path
**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 05 -> 06 -> 07 -> 08 -> 09 -> 10 -> 11

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 05. Structural Learning & Curation Ants | 2/2 | Complete | 2026-04-03 |
| 06. XML, Display & Semantic Search | 2/2 | Complete | 2026-04-03 |
| 07. Error Handling & Miscellaneous | 4/4 | Complete    | 2026-04-04 |
| 08. Slash Command Wiring | 2/2 | Complete   | 2026-04-04 |
| 09. Playbook Wiring | 2/2 | Complete    | 2026-04-04 |
| 10. Integration Parity Tests | 1/3 | In Progress|  |
| 11. Distribution | 0/? | Not started | - |
