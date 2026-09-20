# Codex public `$ant-*` skill surface — naming, installation and coverage research

Date: 2026-09-16. Source inspected: `/Users/callumcowie/repos/Aether`, HEAD `404731ccffda7bbca64ce801b74b0752a7161315`.

Research only. No repository files, installed skills, shared home configuration, project state, the owner's prepared project, or worktrees were changed. No install/publish/update/lifecycle commands or test suites were run. The only output is this scratch report. The untracked `.gsd/` directory was left untouched.

## Plain-English conclusion

The menus are named differently because Aether generates Codex's own skill files with `aether-` names. Changing those names is straightforward. Making every command appear and work takes more: Codex currently exposes only nine of the 64 commands that Claude Code and OpenCode expose, and its normal update path does not refresh those skill files today.

Recommend **canonical `$ant-*` public command skills, without a second public `$aether-*` command menu**. Keep the actual `aether` program, `.aether` state folders, worker IDs and other internal names unchanged. Treat support instructions separately from user commands.

## 1. Verified inventory and the three different counts

Read-only enumeration of `*.yaml` and `*.md` found identical basename sets:

| Surface | Count | Meaning |
|---|---:|---|
| `.aether/commands/*.yaml` | 64 | Public command source definitions |
| `.claude/commands/ant/*.md` | 64 | Claude command instructions |
| `.opencode/commands/ant/*.md` | 64 | OpenCode command instructions |
| Generated Codex command skills | 9 | Named `aether-init`, `aether-discuss`, `aether-oracle`, `aether-colonize`, `aether-plan`, `aether-build`, `aether-continue`, `aether-swarm`, `aether-seal` |
| Generated Codex helper skills | 5 | `aether-command-guide`, `aether-skill-loader`, `aether-colony-creation`, `aether-colony-research`, `aether-colony-build-cycle` |
| Installed Aether Codex skill directories on this machine | 14 | Nine command entries + five helpers; same shape as generator |

Evidence: `cmd/platform_sync.go:647` starts five helper definitions; `cmd/platform_sync.go:685` hard-codes the nine commands; `cmd/platform_sync.go:705` sets both directory and frontmatter name to `aether-<command>`. `cmd/wrapper_command_names.go:15` enumerates all 64 public wrapper names. `CLAUDE.md:30` independently reports 64 per primary platform. Installed example: `/Users/callumcowie/.codex/skills/aether/aether-plan/SKILL.md:2` declares `name: aether-plan`.

The 86 reusable worker skills mentioned elsewhere are a separate collection. They are not 86 public Codex commands. Renaming nine public command skills yields **9/64 coverage**, with **55 missing**. Keeping the five helpers exposed while adding all public entries yields 69 discoverable skills, of which 64 are user commands.

### Missing 55 entrypoints

`archaeology`, `ask`, `assumptions`, `bump-version`, `chaos`, `council`, `data-clean`, `dream`, `entomb`, `export-signals`, `feedback`, `flag`, `flags`, `focus`, `help`, `history`, `import-signals`, `improve`, `insert-phase`, `interpret`, `lay-eggs`, `maintenance`, `maturity`, `medic`, `memory-details`, `migrate-state`, `organize`, `patrol`, `pause`, `phase`, `pheromones`, `porter`, `preferences`, `profile`, `queen-compose`, `quick`, `redirect`, `reference-index`, `reference-list`, `reference-match`, `resume`, `run`, `shelf`, `shelf-add`, `shelf-dismiss`, `shelf-list`, `shelf-promote`, `skill-create`, `spec`, `status`, `tunnels`, `unblock`, `update`, `verify-castes`, `watch`.

## 2. Exact command categories and mappings

Do not widen `codexCommandSkillShims()` to all catalog entries unchanged. It currently skips `def.Literal` (`cmd/platform_sync.go:690`). Worse, `commandGuideCatalog()` supplies `aether <public-name>` for every default literal entry (`cmd/command_guide.go:135`). That is not a valid semantic mapping for several public commands.

### A. Nine existing orchestrated commands

`init`, `discuss`, `oracle`, `colonize`, `plan`, `build`, `continue`, `swarm`, `seal` already receive detailed skills generated from `command-guide`. Rename their public identity to `ant-*`, retaining their existing underlying runtime operations, manifests and finalizers. Their actual runtime command is often a host or finalizer step, not simply `aether <name>`; the body renderer already uses `def.RunCommand` (`cmd/platform_sync.go:716`, `cmd/platform_sync.go:732`). Workflow parity is the other researcher's responsibility.

### B. Six public names that must map explicitly

| Public skill | Current canonical wrapper route | Evidence / caveat |
|---|---|---|
| `$ant-ask` | `AETHER_OUTPUT_MODE=json aether colony-prime --question "<question>"`, then answer from that briefing | `.aether/commands/ask.yaml:5`; conversational answer and no worker dispatch at `:9` and `:12` |
| `$ant-assumptions` | `AETHER_OUTPUT_MODE=visual aether assumptions-analyze <arguments>` | `.aether/commands/assumptions.yaml:5`; follow-ups are `assumption-list` / `assumption-validate` at `:11` |
| `$ant-council` | `AETHER_OUTPUT_MODE=visual aether council-deliberate --topic "<topic>"` | `.aether/commands/council.yaml:5`; a `council` parent exists at `cmd/council.go:279` but has no RunE and is not this action |
| `$ant-patrol` | `AETHER_OUTPUT_MODE=visual aether patrol-check <arguments>` | `.aether/commands/patrol.yaml:5`; raw `aether patrol` aliases `colony-vital-signs` (`cmd/memory_details.go:166`), a different operation from health checks (`cmd/patrol_check.go:53`) |
| `$ant-profile` | `AETHER_OUTPUT_MODE=visual aether profile-read <arguments>` | `.aether/commands/profile.yaml:5`; optionally `profile-update` before reading at `:8` |
| `$ant-shelf` | `AETHER_OUTPUT_MODE=visual aether shelf-list <arguments>` | `.aether/commands/shelf.yaml:5`; runtime alias `shelf` also exists at `cmd/shelf_cmd.go:24` |

`council.yaml:8` says there is no single council command; source confirms a parent container exists, so interpret that line as no single executable council action. Do not repeat the stronger claim that no command named council exists.

### C. Five prompt-only commands

`archaeology`, `chaos`, `dream`, `interpret`, `organize` have **no direct same-name CLI action**. Both platforms' corresponding instruction files are byte-identical, and each states at line 7 that it is a pure prompt command and must not call `aether <name>`.

- `.claude/commands/ant/archaeology.md:7`: read-only history investigation; 333 lines.
- `.claude/commands/ant/chaos.md:7`: investigate resilience and report; 366 lines.
- `.claude/commands/ant/dream.md:7`: explore and write a dream journal, not code/state; 267 lines, journal path at `:81`.
- `.claude/commands/ant/interpret.md:7`: read-only dream interpretation; 15 lines.
- `.claude/commands/ant/organize.md:7`: report-only hygiene work, including Keeper dispatch at `:58`; 235 lines. Its sanctioned report exception is `.aether/commands/organize.yaml:8`.

Their YAML files are only 6–8 lines and do not carry these full instructions. A generator that reads only YAML metadata cannot reproduce them. Reuse/adapt the proven shared instructions with Codex tool names and argument handling, or move the common body to a platform-neutral support source consumed by all platforms. The latter is useful but is not a prerequisite for the basic nine-name rename.

### D. Remaining 44 names with a same-name runtime route

`bump-version`, `data-clean`, `entomb`, `export-signals`, `feedback`, `flag`, `flags`, `focus`, `help`, `history`, `import-signals`, `improve`, `insert-phase`, `lay-eggs`, `maintenance`, `maturity`, `medic`, `memory-details`, `migrate-state`, `pause`, `phase`, `pheromones`, `porter`, `preferences`, `queen-compose`, `quick`, `redirect`, `reference-index`, `reference-list`, `reference-match`, `resume`, `run`, `shelf-add`, `shelf-dismiss`, `shelf-list`, `shelf-promote`, `skill-create`, `spec`, `status`, `tunnels`, `unblock`, `update`, `verify-castes`, `watch`.

These can share a small command-skill template **after carrying over command-specific guardrails and follow-ups**. Same spelling does not mean trivial behavior: release automation, delivery, a guided skill-creation wizard, or answering a question may include important work beyond one subprocess call. For example `.aether/commands/bump-version.yaml:12` includes a release sequence beyond the version-write command. `spec` is runtime-native and deliberately excluded from the current nine (`cmd/command_guide.go:197`). The catalog's current literal label must not be mistaken for proof of full behavioral parity.

## 3. Naming changes required

1. In `cmd/platform_sync.go:685`, separate **public skill name**, **public command ID**, and **underlying execution route**. Emit `Dir: ant-<id>` and `Name: ant-<id>`; update the description, title, body opening and explicit trigger examples to `$ant-<id>`. Keep `aether command-guide <id> --platform codex` and the actual shell program unchanged. Current keyword list includes `/ant-` and bare `ant-`, but does not name `$ant-` (`:694`). Merely adding another keyword does not rename a skill.
2. Replace the hard-coded nine-command inventory for the complete release with a single tested public command inventory matching the 64 YAML/Claude/OpenCode names. Add a route type for runtime, orchestrated and prompt-only actions. Do not derive routes just by stripping `ant-`.
3. Preserve runtime-owned machine fields such as `runtime_command`, IDs, event sources and worker caste IDs. `$ant-*` is a user invocation convention, not a shell executable. `aether-builder` agent IDs, `aether-plan` event source labels, the npm package and storage paths are legitimate existing names.
4. Update user-facing Next Up/help formatting to show `$ant-*` on Codex for existing public skills. Today Codex returns raw CLI unchanged in `cmd/codex_visuals.go:777`, uses `aether <verb>` in `:800`, and returns raw CLI in `cmd/lifecycle_projection.go:255`. Keep literal environment-prefixed shell invocations and commands with no public skill (publish/install/host/finalizers) unchanged. For the nine-entry intermediate release, display only those nine as skills; for complete coverage use the full public inventory.
5. Update the descriptive capability contract once discovery is proved: `pkg/codex/platform_contract.go:112` currently labels the native command surface unavailable because it is defined as slash commands. Skills should be represented honestly as a separate invocation mechanism, without implying Codex has slash commands or stronger worker isolation.

## 4. Install, update, migration and mechanisms that can undo the change

### Current path

- Stable install/publish generates skills directly into `~/.codex/skills/aether/`: `cmd/install_cmd.go:466`, `:513`; publish calls this at `cmd/publish_cmd.go:120`.
- `syncCodexSkillShims` uses the generated directory names as its allowed set (`cmd/platform_sync.go:761`). It removes every other skill directory except one declaring `source: custom` (`:776`), then writes generated content (`:787`). Renaming just local installed files will be undone on the next install/publish.
- Unknown custom directories survive, but an existing destination with the same allowed name is written unconditionally when content differs (`:788`). Thus the existing custom-preservation check does **not** protect a user-authored colliding `ant-plan` directory. A safe migration must identify ownership and preserve/report collisions.
- Current update is a transaction that plans exact targets. `appendMaintenanceUpdatePlatformTargets` (`cmd/update_cmd.go:467`) iterates `platformHomeHubSyncPairs` (`cmd/platform_sync.go:610`), which has agents and Claude/OpenCode commands, **no Codex skill pair**. It never calls the skill generator. The older `syncPlatformHomeAssetsFromHub` likewise ends without skills (`cmd/install_cmd.go:531`). Documentation explicitly says generated skills are not in the hub (`RUNTIME UPDATE ARCHITECTURE.md:69`). Therefore update alone currently cannot install this rename.

### Recommended migration

- Keep the `aether` grouping-folder name; changing that name does not improve the public invocation. Select the discovery root through fresh-client qualification: this session sees `~/.codex/skills/aether`, while current official guidance documents user/repository `.agents/skills`. Keeping the legacy root is the smallest change only if the supported target clients prove discovery there; otherwise include an owned migration to `.agents/skills/aether`.
- Generate canonical `ant-*` entries and remove only ownership-proven legacy shipped command entries `aether-{init,discuss,oracle,colonize,plan,build,continue,swarm,seal}`. Preserve modified/user-created content, unknown custom skills and symlinks; report a collision plainly instead of overwriting it.
- Use an explicit previous-shipped-name/content manifest or equivalent ownership proof. Avoid a broad `aether-*` deletion, which would also hit helpers and possibly user content. No compatibility duplicate is necessary for the requested primary path. A temporary compatibility action should only be added if an actual external caller needs it, and should not create a second default public menu.
- Add generation/removal as planned Codex-home targets in the existing update transaction, preserving preview, rollback and ownership checks. Both install/publish and update must use the same desired inventory and migration policy. The coordinator already has a Codex-home root (`cmd/lifecycle_transaction.go:279`); no new storage engine is necessary.
- Decide where desired generated content lives for upgrades. Minimum route: current binary generates it and new runtime is required. More durable route: publish a versioned generated Codex-skill payload in the hub and let update consume it. Do not let an older binary prune new entries solely because its old inventory lacks them. This issue already has a protective comment for Claude/OpenCode cleanup (`cmd/platform_sync.go:1248`) but not the Codex shim pruning loop.
- Preserve development-channel isolation. Install skips global homes by default on dev (`cmd/install_cmd.go:470`); transactional update refuses writing stable homes from dev (`cmd/platform_sync.go:149`). Test with isolated homes, not the owner's live installation.
- Include renamed/removed skill entries in the update receipt and restart guidance. Current restart target detection is keyed on `Skills (codex shims)` and only `copied > 0` (`cmd/codex_project_docs.go:131`, `:156`); pure removals also require a refresh.

### Repository mirrors and instructions

Do not solve installation by writing generated public skills inside consumer projects. Legacy cleanup removes the repo-local `.codex/skills/aether` tree during setup (`cmd/platform_sync.go:1186`, `:1229`) and the separate forced prune removes non-custom mirrors (`:1466`). Production transactional update currently has a different legacy cleanup planner (`cmd/update_cmd.go:413`) which does not reproduce every older prune helper; cover the actual production path rather than assume helper tests prove it.

Project instructions are generated from templates. `cmd/update_cmd.go:357` rewrites managed `AGENTS.md` and `.codex/CODEX.md`, while custom documents are preserved at `:372`; setup uses `cmd/codex_project_docs.go:17`. Editing only the source root's instructions will not fix consumer projects.

## 5. Documentation and support instructions

Required current-source updates, scoped to public command guidance:

- `AGENTS.md:33`, `:107`: currently direct CLI-first and old helper references.
- `.codex/CODEX.md:115`, `:196`: currently no public skill entrypoint guide.
- `.aether/templates/agents-md-template.md:14`, `:43`, `:73` and `.aether/templates/codex-md-template.md:5`, `:23`: generated instructions currently teach only CLI commands.
- `README.md:178`, `:222`, `:230`, `:315`, `:559`: platform onboarding, command counts and primary public workflow. Several sections still claim 60 wrappers, although the actual current corpus is 64.
- `CLAUDE.md:30`, `:150`, `:183`: cross-platform overview currently says Codex is runtime-only.
- `RUNTIME UPDATE ARCHITECTURE.md:52`, `:69`, `:207` and `.aether/docs/publish-update-runbook.md:321`, `:444`: storage, distribution, and stale count of five Codex shims.
- `.aether/docs/wrapper-runtime-ux-contract.md:168`: document skill entrypoints without changing state ownership.
- `cmd/command_guide.go:203`, `:234` and `.aether/skills/colony/aether-colony-creation/SKILL.md:108` forbid translating closeout into the previously deferred Codex-native surface; revise for the newly authorized public vocabulary.
- `.aether/skills/colony/aether-colony-build-cycle/SKILL.md:45` records that `$ant-*` was deliberately deferred. Replace that historical scope fence in current guidance.

Source checking currently covers only Claude/OpenCode wrapper directories (`cmd/source_check.go:502`). Add generated Codex inventory/name/route checks so a future command addition cannot silently leave Codex behind.

### Five internal helper names: separate decision, not a blocker

The nine existing command names can be changed without renaming or refactoring five helpers. Keeping internal names and source paths `aether-colony-*` is safe and avoids unrelated churn. Label them as supporting instructions rather than advertising them as alternative public actions.

If the requirement is **no visible `$aether-*` entries of any kind**, stop installing these five as separately discoverable skills and reference private support files from public `ant-*` skills. Three detailed helpers already live in shipped `.aether/skills/colony/aether-colony-*/SKILL.md`; the two short command-guide/loader helpers can be folded into generated shared instructions. Update the generator's load instructions (`cmd/platform_sync.go:720`) and guide references, while retaining private IDs. This is a small packaging choice, not a reason to delay the simple naming migration. Do not rename all helpers to `ant-*` and imply they are five new public commands.

## 6. Focused engineering effort estimates

Judgments, **not observed timings**. Hours mean focused implementation/debugging time for someone familiar with the repository; not wall-clock promises. No provider runs or full suites were performed for this estimate.

| Work package | Low | Likely | High |
|---|---:|---:|---:|
| Nine public identities, triggers and name-aware presentation | 1 h | 2 h | 4 h |
| Safe existing-install migration, ownership/collision handling, update transaction wiring | 4 h | 8 h | 14 h |
| Generated templates, current docs and restart notices | 2 h | 4 h | 6 h |
| Focused install/update/name/rollback checks and corrections | 3 h | 4 h | 6 h |
| **Rename existing nine + safe migration total** | **10 h** | **18 h** | **30 h** |
| Add complete 64-name inventory, renderer categories and six explicit route mappings | 6 h | 10 h | 16 h |
| Package/adapt the five existing prompt-command bodies to Codex | 6 h | 12 h | 20 h |
| Complete inventory/display/source checks, docs and missing-entry smoke coverage | 4 h | 8 h | 14 h |
| **Additional surface work for all 64** | **16 h** | **30 h** | **50 h** |
| **Full 64-entry naming/install/documentation surface, including migration** | **26 h** | **48 h** | **80 h** |
| Optional: remove all five helpers from the visible skill list while retaining private shared instructions | +3 h | +6 h | +12 h |

A throwaway rename of just `Dir`/`Name`, with no upgrade path or current documentation work, is roughly 1–4 hours. That is not the requested complete change.

These figures include local focused checks of this surface; **they do not include proving/fixing complete workflow semantics, long provider-backed end-to-end runs, cross-platform regression/release proof, or release/deployment work**. The separate workflow and evidence researchers own those estimates. In a combined plan, avoid counting their shared prompt-adaptation and installation-test work twice.

If fresh-client qualification requires moving from the legacy `.codex/skills` location to the documented `.agents/skills` location, allow roughly **+4 / +8 / +16 focused hours** (low/likely/high) for destination/root authorization, safe old-path cleanup, collision tests and repeated-update proof. This is conditional, not evidence that the legacy path has stopped working. It may overlap the main migration high case and should not be charged twice.

Main uncertainty drivers: whether the new generated payload becomes hub-versioned; pre-existing custom-name collisions; five prompt bodies' adaptation details; stale installed binaries restoring old inventories; and whether hidden helpers are part of acceptance. Most of the 55 additions can use one renderer, but a renderer cannot manufacture workflow behavior absent from its source.

## 7. Discovery qualification and acceptance facts

The parent researcher checked current official [Build skills documentation](https://learn.chatgpt.com/docs/build-skills), which documents user/repository `.agents/skills` discovery and symlink support. The current conversation demonstrably includes the installed nested `~/.codex/skills/aether` skills. Those observations are compatible: legacy discovery working here is not proof that every supported fresh client will find the chosen destination. Also, a context-limited initial skill list can shorten or omit entries; absence from that list alone is not proof that installation failed.

Qualify a fresh client against the intended supported destination and explicit `$ant-*` invocation/selector. If switching to `.agents/skills`, change installer targets, update transaction roots, owned stale-path cleanup and restart guidance together. The present Codex-home transaction root is `.codex`, so it cannot simply be pointed at a sibling path without a deliberate destination model. Symlink support is a possible installation mechanism, not a reason to bypass existing symlink/ownership validation. No plugin bundle is required for ordinary skills.

Acceptance facts for the later implementation:

- Fresh Codex session discovers exactly the chosen 64 canonical public `ant-*` entries and no default duplicate legacy command entries.
- The nine-entry intermediate milestone must say nine, never claim 64-command parity.
- Every emitted name maps either to a verified runtime route or to explicit prompt behavior; no fabricated `aether archaeology`, `aether ask`, `aether assumptions`, etc.
- Existing installs migrate through the supported update route; repeated update is stable; custom collisions and rollback are exercised in temporary homes.
- A fresh managed consumer project teaches `$ant-*`; preserved custom instructions are reported, not silently overwritten.
- User-facing hints name supported skills; actual shell commands, runtime IDs and machine-readable authority stay unchanged.
- Fresh-session discovery/invocation and behavior need separate executable evidence, not merely file-count assertions. Existing tests also deliberately reject some deferred `$ant-*` surfaces (for example `cmd/maintenance_wrapper_contract_199_test.go:104`); update that old contract narrowly rather than disable checks wholesale.
