# Pitfalls Research: v1.11 Aether Unification

**Domain:** Restoring deleted intelligence features, cleaning self-hosting artifacts, hardening multi-platform CLI
**Researched:** 2026-04-28
**Confidence:** HIGH (based on direct codebase analysis of git history, Go stubs, agent mirrors, and tracked artifacts)

---

## Critical Pitfalls

### Pitfall 1: Restoring Shell Logic Without Adapting to Go Concurrency Model

**What goes wrong:**
The 58 deleted shell scripts (31,411 lines, deleted in commits `92d6c8d6` and `0063be8b`) used sequential bash execution with global state, `source`-based dependency injection, and `jq` for JSON manipulation. The current Go runtime uses goroutines, structured error handling, and the `pkg/storage` locking layer. Porting shell logic line-by-line into Go produces code that works in tests but deadlocks under concurrent access or panics when file locks contend.

The shell scripts relied on bash-level behaviors that have no Go equivalent:
- `trap '' ERR` to disable error trapping in suggest.sh (line 26 of deleted file) -- Go has no equivalent of trap disabling
- `source`-based function loading from `aether-utils.sh` -- Go requires explicit imports and dependency injection
- `jq` streaming pipelines for JSON transforms -- Go needs manual struct unmarshaling

**Why it happens:**
The shell scripts were marked DEPRECATED but their logic was the most complete implementation. The current Go stubs (e.g., `init_research.go` at 597 lines vs `scan.sh` at 867 lines + `suggest.sh` at 618 lines) reimplemented the *surface* (same output JSON shape) but simplified the *behavior* (fewer edge cases, no error recovery, no retry logic). When restoring the full intelligence, developers naturally look at the shell version as the reference implementation and try to match it exactly, ignoring that Go's concurrency model requires fundamentally different control flow.

**Consequences:**
- Deadlocks when suggest-analyze and colony-prime both read pheromones concurrently
- Panic in file locking when the shell script's "retry with backoff" pattern is replaced by a single Go call without timeout
- Subtle behavioral differences between restored Go code and the shell reference that only manifest under load

**Prevention:**
- Treat the shell scripts as *specification documents*, not reference implementations. Extract the *what* (inputs, outputs, edge cases) and rewrite the *how* in Go idioms.
- For each restored feature, write the Go implementation first, then diff the output JSON against what the shell version would have produced using a golden-file test.
- Use `context.Context` with timeouts on all storage operations, especially suggest-analyze which reads pheromones, constraints, session state, and file patterns.
- Never port bash `trap` logic -- use Go's `defer` and explicit error returns instead.

**Detection:**
- `go test -race ./...` reveals data races during suggest-analyze
- Colony-prime hangs when suggest-analyze holds a lock that colony-prime needs
- Golden-file tests show output mismatch with shell reference

---

### Pitfall 2: Deleting Self-Hosting Artifacts That Downstream Repos Depend On

**What goes wrong:**
Aether was used to develop itself, producing artifacts that exist in the repo but are also consumed by downstream repos via `aether update`. The most dangerous artifacts are:

1. **`.aether/agents/`** -- A third copy of agent definitions (26 files) that differs from both `.claude/agents/ant/` and `.aether/agents-claude/`. Every single file differs (verified via `diff -rq`). The install command (`install_cmd.go` line 608) explicitly skips `.aether/agents/` with the comment "agents/ is opencode-only, agents-claude/ is the packaging mirror." But if any downstream repo was installed from a version that copied `.aether/agents/`, deleting it breaks that repo's `aether update`.

2. **`.aether/chambers/`** -- 241 tracked files (54,832 lines) of entombed colony data from Aether developing itself. These are historical artifacts with no functional purpose, but the `.gitkeep` in chambers/ suggests the directory structure is intentional.

3. **`.aether/CONTEXT.md`** -- A self-hosting colony context file (phase 1, "Assumptions and gap audit", First Mound milestone) tracked in git. This is runtime data that was committed accidentally.

4. **`.aether/CROWNED-ANTHILL.md`** -- A seal output from a previous self-hosting colony, tracked in git.

5. **`.aether/QUEEN.md`** -- Contains wisdom accumulated during self-hosting, including patterns specific to the Aether codebase that would be irrelevant or misleading for downstream repos.

6. **`.aether/data/COLONY_STATE.json`** -- Explicitly tracked in git despite `.aether/data/` being gitignored (the gitignore has a negation exception for this one file). This is self-hosting state.

**Why it happens:**
The `.aether/.gitignore` only excludes `data/`, `dreams/`, `checkpoints/`, and `locks/`. It does not exclude `CONTEXT.md`, `CROWNED-ANTHILL.md`, `chambers/`, or `QUEEN.md`. During self-hosting colonies, these files were created by the runtime and committed as part of normal development workflow. The install/publish system copies companion files to the hub, so self-hosting artifacts were published to `~/.aether/system/` and then distributed to downstream repos via `aether update`.

**Consequences:**
- Downstream repos receive irrelevant Aether-specific wisdom, chamber archives, and colony state via `aether update`
- Deleting artifacts from the repo without updating the hub leaves downstream repos with stale files
- Deleting artifacts from the hub without updating the repo breaks the publish pipeline's integrity check
- Removing `.aether/agents/` could break repos that were installed from an older version

**Prevention:**
- Before deleting any tracked `.aether/` file, run `aether integrity` to understand the full publish chain
- Check the hub (`~/.aether/system/`) for copies of the artifact before deleting from the repo
- Delete in the correct order: (1) remove from git, (2) republish with `aether publish`, (3) verify downstream repos update cleanly
- Add comprehensive `.gitignore` rules for all runtime-generated paths: `CONTEXT.md`, `CROWNED-ANTHILL.md`, `chambers/`, `QUEEN.md` (or make QUEEN.md a template that gets populated per-repo)
- For `.aether/agents/` specifically: determine whether any downstream code references it. If not, delete it and update `.gitignore`. If yes, add migration logic to `aether update`.

**Detection:**
- `git ls-files .aether/ | grep -v agents-claude | grep -v agents-codex | grep -v skills | grep -v commands | grep -v docs | grep -v templates | grep -v utils | grep -v exchange | grep -v .gitignore | grep -v .npmignore` reveals all tracked non-distribution artifacts
- `aether integrity` after any deletion catches publish chain breaks

---

### Pitfall 3: Charter Ceremony Breaking Non-Interactive / CI / Scripted Workflows

**What goes wrong:**
The shell version of the charter ceremony (`scan.sh` -> `charter-write` -> approval flow) was interactive by design: it scanned the repo, generated a charter, and required user approval before proceeding. The current Go stub (`init_research.go`) generates charter data and pheromone suggestions (10 deterministic patterns, lines 247-357) but has **no approval flow** (verified: zero matches for "charter.*approv" or "confirm" in cmd/).

Adding a rich init ceremony with approval gates to the existing `/ant-init` command creates a trap: users who currently run `/ant-init "Build feature X"` in automated scripts, CI pipelines, or non-interactive environments will suddenly hit an approval prompt that blocks execution.

**Why it happens:**
The original shell version was designed for interactive terminal use. The Go migration simplified it to non-interactive output. Restoring the full ceremony means re-adding interactivity, but the command's existing users (and all 50 slash commands that reference init behavior) expect it to work without prompts.

**Consequences:**
- CI pipelines that run `/ant-init` as part of setup break
- Users running `/ant-run` (autopilot) hit an unexpected approval prompt on the first build
- Codex CLI (runtime-native, no wrapper) cannot display interactive prompts in the same way as Claude/OpenCode wrappers

**Prevention:**
- Add a `--yes` / `--non-interactive` / `--skip-approval` flag to the charter ceremony
- Make the approval step opt-in via a config option or pheromone signal, not the default
- The ceremony should produce the charter data regardless of interactivity; the approval gate is a separate concern that only applies in interactive contexts
- For Codex, the ceremony must work without terminal prompts -- output charter data to stdout and let the wrapper decide whether to present an approval step
- Default behavior should match current behavior (no approval required) to avoid breaking existing workflows

**Detection:**
- `/ant-run` hangs after init phase when run in a non-interactive shell
- Codex users report that init produces no output (because it's waiting for approval)
- CI test suite fails when init is called programmatically

---

### Pitfall 4: colony-prime_context.go Modifications Causing Cascade Test Failures

**What goes wrong:**
`colony_prime_context.go` is 800 lines and is the single most complex function in the codebase. It assembles 9+ sections within an 8,000 character budget. The v1.11 changes touch at least three areas that feed into colony-prime:

1. **Suggest-analyze restoration** adds pheromone suggestions that colony-prime injects into worker context
2. **Rich init-research** produces colony context (tech stack, governance, complexity) that colony-prime uses for worker priming
3. **Charter data** may need to be included in the colony-prime context for workers

Each modification risks changing the character budget allocation, causing different sections to be trimmed. Since 15 test files reference `colony_prime_context` (verified via grep), any change can cascade into dozens of test failures with confusing error messages about "trimmed" sections.

**Why it happens:**
The function uses a greedy ranking algorithm (`RankContextCandidates`) with composite scores. Adding new section types changes the ranking pool, which changes which sections get included and which get trimmed. The existing tests have hardcoded expectations about which sections appear in the output.

**Consequences:**
- A change to suggest-analyze causes pheromone signals to be trimmed (they drop from priority 9 to outside the budget)
- A change to init-research context causes queen wisdom to be trimmed
- Tests that assert on specific section content fail with "expected section X, got section Y" errors
- Debugging is painful because the failure is in the ranking algorithm, not in the feature code

**Prevention:**
- Add new sections with explicit budget caps (e.g., suggest-analyze: max 400 chars, charter: max 300 chars)
- Set new section priorities deliberately -- suggest-analyze should be priority 5-6 (below pheromones at 9, above rolling summary at 1)
- Before modifying colony-prime, add a "budget snapshot" test that records the current section allocation and fails only if the total exceeds 8000 chars
- Make new sections opt-in via a flag or config so the existing test suite passes unchanged
- Consider splitting `buildColonyPrimeOutput` into smaller composable functions before adding new sections

**Detection:**
- `go test ./cmd/ -run TestColonyPrime` fails after unrelated changes
- Workers report missing pheromone signals or queen wisdom after a colony-prime change
- Character budget utilization jumps from ~65% to >90% after adding new sections

---

## Moderate Pitfalls

### Pitfall 5: Mirror Divergence Between .aether/agents/ and .aether/agents-claude/

**What goes wrong:**
Two directories contain agent definitions that should be identical but are not:
- `.aether/agents/` (26 files) -- older format, missing structured metadata like `tools`, `color`, `model`
- `.aether/agents-claude/` (26 files) -- newer format with structured frontmatter

Every single file differs (verified via `diff -rq`). The `.aether/agents/` versions have simpler descriptions and lack the structured execution flow that `.aether/agents-claude/` has. During cleanup, if the wrong copy is kept, downstream repos get stale agent definitions.

**Why it happens:**
`.aether/agents/` appears to be a legacy directory from before the packaging mirror system was established. The CLAUDE.md documents that `.aether/agents-claude/` is the "packaging mirror" and `.claude/agents/ant/` is canonical. But `.aether/agents/` is also tracked in git and may be referenced by older versions of the install command.

**Prevention:**
- Determine which directory the install/publish system actually copies from (check `install_cmd.go` sync pairs)
- Delete the unused directory after confirming no code references it
- Add a CI check that verifies `.aether/agents-claude/` is byte-identical to `.claude/agents/ant/` (the canonical source)
- Add `.aether/agents/` to `.gitignore` after deletion

**Detection:**
- `diff -rq .aether/agents/ .aether/agents-claude/` shows all files differ
- `aether update` in a downstream repo installs agent definitions from the wrong directory

---

### Pitfall 6: Suggest-Analyze Pattern Drift Between Shell Reference and Go Implementation

**What goes wrong:**
The shell `suggest.sh` (618 lines) implemented suggest-analyze with 10+ codebase analysis patterns that produced pheromone suggestions. The current Go stub in `init_research.go` implements 10 deterministic patterns (lines 247-357) but the patterns are **simpler** than the shell version. The shell version used `grep` with regex patterns to detect code smells, anti-patterns, and project-specific signals. The Go version uses simple file-existence checks (`hasFile`, `fileContains`).

If the restoration targets the shell version's full pattern set, the Go implementation will need substantially more complex analysis logic (regex scanning, file content analysis, pattern matching). If it targets the current Go stub, the restoration is incomplete.

**Why it happens:**
The Go migration simplified suggest-analyze from regex-based code analysis to file-presence heuristics. This was a deliberate simplification during the migration, not a bug. But the PROJECT.md lists "Suggest-analyze restoration (automatic pheromone suggestions during builds)" as a v1.11 requirement, implying the full shell functionality should be restored.

**Prevention:**
- Before implementing, enumerate the exact patterns from the shell version and decide which ones to restore. Not all shell patterns are valuable -- some were bash-specific (e.g., checking for `source` statements).
- For each restored pattern, write a Go implementation that produces equivalent output, not equivalent code. The Go version should use `filepath.Walk` and `regexp` instead of `find` and `grep`.
- Add golden-file tests that compare Go output against shell reference output for the same input directory.
- Consider whether the current 10 file-presence patterns are sufficient for v1.11 and defer the full regex-based analysis to a later milestone.

**Detection:**
- Suggest-analyze produces fewer suggestions than the shell version for the same codebase
- Suggestions are less specific (file-existence based vs code-pattern based)
- Users who remember the shell version's suggestions notice missing patterns

---

### Pitfall 7: OpenCode Parity Gaps Hidden by Identical File Counts

**What goes wrong:**
Claude and OpenCode command directories have identical file counts (51 each) and identical file sizes for matching commands (verified via diff). But "same number of files" does not mean "same behavior." The OpenCode commands may have different runtime command calls, different flag formats, or different error handling that only manifests at runtime.

The v1.11 plan includes "Platform hardening -- fix OpenCode parity gaps." Without a systematic diff of the actual command content (not just file names), parity gaps will be missed.

**Why it happens:**
OpenCode uses a different command surface than Claude Code. The wrapper markdown files may call the same Go subcommands but with different flags, different output formatting, or different error handling. The Medic health check validates mirror counts but not behavioral parity.

**Prevention:**
- For each of the 51 commands, diff the actual content between `.claude/commands/ant/` and `.opencode/commands/ant/` and classify differences as: (a) platform-specific formatting (acceptable), (b) missing flags (bug), (c) different subcommand calls (bug), (d) different error handling (bug)
- Focus especially on commands that were added or modified in v1.10 (seal, init, status, entomb, resume, discuss, chaos, oracle, patrol) as these are most likely to have parity gaps
- Add a test that extracts Go CLI calls from both platforms' wrapper markdown and verifies they use the same flags
- For the 26 Codex TOML agents, verify each agent's `tools` list matches the corresponding Claude agent's tools

**Detection:**
- Running the same command on Claude and OpenCode produces different output
- OpenCode commands fail with "unknown flag" errors that work on Claude
- Codex agents lack tools that their Claude counterparts have

---

### Pitfall 8: 54,832 Lines of Chamber Archives Bloating the Repo

**What goes wrong:**
`.aether/chambers/` contains 241 tracked files totaling 54,832 lines of entombed colony data from Aether developing itself. These include COLONY_STATE.json, CROWNED-ANTHILL.md, HANDOFF.md, colony-archive.xml, dreams/, and constraints.json for 15+ past colonies. This data is historical and serves no functional purpose for downstream repos, but it ships with every `aether update`.

**Why it happens:**
The entomb command archives colony data to `.aether/chambers/` and commits it. When Aether was used to develop itself, every completed colony was entombed in the same repo. The `.gitignore` for `.aether/` does not exclude chambers.

**Consequences:**
- Repo size grows unnecessarily (54K+ lines of historical data)
- `aether update` transfers this data to every downstream repo
- Chamber archives from the Aether repo's own colonies are irrelevant to users building their projects
- New users see 15+ chamber directories and may be confused about what they are

**Prevention:**
- Move historical chambers to a separate branch or archive (e.g., `git mv .aether/chambers/ chambers-archive/` then commit and delete)
- Add `.aether/chambers/` to `.gitignore` (except `.gitkeep`) so future entombs are not tracked
- Keep the `.gitkeep` so the directory structure exists for future entombs
- Consider whether chambers should be stored in the hub (`~/.aether/chambers/`) rather than in the repo

**Detection:**
- `git ls-files .aether/chambers/ | wc -l` returns 241 (should be 1 for .gitkeep)
- `aether update` in a fresh repo creates `.aether/chambers/` with historical data

---

## Minor Pitfalls

### Pitfall 9: Bayesian Trust Scoring Restoration Requiring Dependency Chain

**What goes wrong:**
The shell `trust-scoring.sh` (354 lines) implemented a 40/35/25 weighted scoring system with 60-day half-life decay and 7 trust tiers. This was part of the "Structural Learning Stack" that also included `learning.sh` (2,007 lines), `event-bus.sh` (308 lines), and `instinct-store.sh` (408 lines). Restoring trust scoring without its dependency chain (event bus for observations, instinct store for persistence, learning pipeline for promotion) produces an orphaned feature that calculates scores but has nowhere to store or use them.

**Prevention:**
- If restoring trust scoring, restore the full chain: observation capture -> trust calculation -> instinct creation -> queen promotion -> hive promotion
- Alternatively, defer trust scoring and focus on the features that have clear standalone value (suggest-analyze, charter, init-research)

---

### Pitfall 10: Codex CLI Cannot Display Rich Init Ceremony Output

**What goes wrong:**
Codex is "runtime-native" with no markdown wrapper ceremony. The CLAUDE.md states: "Codex gets UX improvements through the runtime renderer only." If the rich init ceremony produces ANSI-formatted output, colored text, or multi-stage interactive prompts, these must be rendered by `cmd/codex_visuals.go` rather than by markdown wrappers. The current Codex visuals system (`casteColorMap`, `casteEmojiMap`, `stage markers`) supports banners and progress bars but may not support the full ceremony format.

**Prevention:**
- Design the ceremony output as structured JSON from the Go runtime, then render it differently per platform: markdown for Claude/OpenCode, ANSI for Codex
- Keep ceremony state in COLONY_STATE.json so all platforms can read it regardless of rendering capability

---

### Pitfall 11: suggest-analyze Deduplication Against Existing Pheromones

**What goes wrong:**
The current suggest-analyze in `init_research.go` generates suggestions without checking whether equivalent pheromone signals already exist. The shell version (`suggest.sh`) had deduplication logic that compared suggestions against `pheromones.json` and session suggestions. Without dedup, running suggest-analyze multiple times (e.g., during build step 4.2) produces duplicate suggestions that pollute the pheromone signal space.

**Prevention:**
- Before generating suggestions, load existing pheromones and skip patterns that match existing active signals
- Use the content hash deduplication from the pheromone system (SHA-256 of type + content)

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| Self-hosting cleanup | Deleting artifacts that break downstream repos | Run `aether integrity` before and after; test `aether update` in a fresh repo |
| Self-hosting cleanup | Removing `.aether/agents/` when older install code references it | Check `install_cmd.go` sync pairs; add migration to `aether update` |
| Charter ceremony | Adding interactivity that breaks non-interactive workflows | Add `--yes` flag; default to non-interactive |
| Init-research | Restoring shell patterns that don't map cleanly to Go | Extract specs from shell, implement idiomatically in Go |
| Suggest-analyze | Pattern drift between shell reference and Go stub | Write golden-file tests comparing outputs |
| Colony-prime modifications | Budget blowout from new sections | Add budget caps per section; snapshot test before changes |
| Platform hardening | Parity gaps hidden by identical file counts | Diff actual command content, not just file names |
| Chamber cleanup | Accidentally deleting .gitkeep | Remove files, keep .gitkeep; gitignore chambers except .gitkeep |

---

## "Looks Done But Isn't" Checklist

- [ ] **Shell reference fully audited:** Often only the main function is ported, not the error handling or edge cases -- verify all paths from the shell version are covered
- [ ] **All 4 agent mirrors updated:** Often Claude is updated but Codex TOML is forgotten -- verify all 26 agents x 4 locations
- [ ] **Downstream repos tested:** Often cleanup works in the Aether repo but breaks `aether update` in downstream repos -- test in a fresh clone
- [ ] **Non-interactive mode works:** Often ceremony is tested interactively but not in CI/pipe mode -- test with `echo | aether init`
- [ ] **Colony-prime budget still fits:** Often new sections push budget over 8000 chars -- run budget test after every colony-prime change
- [ ] **Chamber archives not shipped:** Often chambers are gitignored but already exist in the hub -- clear hub too
- [ ] **Codex renders ceremony output:** Often ceremony works on Claude/OpenCode but produces raw JSON on Codex -- test on all 3 platforms

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Shell-to-Go port deadlock | LOW | Add timeouts to all storage calls; no data migration needed |
| Self-hosting artifact breakage | MEDIUM | Restore from git; republish hub; notify downstream repos to `aether update --force` |
| Charter ceremony breaking CI | LOW | Add `--yes` flag; update CI scripts; no data migration |
| Colony-prime budget blowout | LOW | Adjust section caps; revert to previous ranking weights |
| Mirror sync drift | LOW | Re-sync from canonical source; run Medic check |
| Chamber bloat | LOW | Move to archive branch; add gitignore; republish |

---

## Sources

- Direct codebase analysis: git history of commits `92d6c8d6` and `0063be8b` (58 deleted shell scripts, 31,411 lines)
- Direct codebase analysis: `cmd/init_research.go` (597 lines, current Go stub)
- Direct codebase analysis: `cmd/colony_prime_context.go` (800 lines, 15 test files reference it)
- Direct codebase analysis: `cmd/install_cmd.go` (sync pairs, `.aether/agents/` handling)
- Direct codebase analysis: `.aether/agents/` vs `.aether/agents-claude/` (all 26 files differ)
- Direct codebase analysis: `git ls-files .aether/` (241 chamber files tracked, CONTEXT.md and CROWNED-ANTHILL.md tracked)
- Shell reference analysis: `suggest.sh` (618 lines), `scan.sh` (867 lines), `trust-scoring.sh` (354 lines), `council.sh` (432 lines), `consolidation.sh` (134 lines), `immune.sh` (515 lines)
- Platform parity analysis: `.claude/commands/ant/` (51 files) vs `.opencode/commands/ant/` (51 files) -- identical file counts and sizes
- [clig.dev](https://clig.dev/) -- CLI design guidelines for init ceremony and non-interactive modes
- [ThoughtWorks CLI guidelines](https://www.thoughtworks.com/en-us/insights/blog/engineering-effectiveness/elevate-developer-experiences-cli-design-guidelines) -- init command overriding existing config pitfall
- [GitHub copilot-cli #2699](https://github.com/github/copilot-cli/issues/2699) -- real-world example of cross-platform command divergence

---
*Pitfalls research for: Aether v1.11 Aether Unification*
*Researched: 2026-04-28*
