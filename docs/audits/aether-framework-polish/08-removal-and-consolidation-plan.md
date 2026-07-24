# Removal And Consolidation Plan

This is not a request to erase ant identity. It removes duplicate authorities and turns prompt-only personas into skills where that is their actual implementation.

## Removal Rules

Remove or hide an item when at least one is true:

1. It can report success without real work.
2. It duplicates a canonical owner.
3. It has no unique input/output/permission/tool contract.
4. It exposes an internal storage primitive as a beginner command.
5. Its output cannot change a later decision or be traced to evidence.
6. Its ceremony is not derived from a real process or committed transition.

Every removal gets one deprecation release when a released user-facing command is affected. Internal success stubs are quarantined immediately.

## Runtime And Control Planes

Implementation checkpoint (2026-07-22): `control-ts` is now private and
fail-closed; its writable state is quarantined. Production TS-host sources no
longer import the legacy TypeScript provider launcher and delegate subprocess
execution to Go. The remaining work is deletion after schema salvage, native
wrapper run-owner binding, and permission enforcement.

| Proposed action | Evidence | Migration impact |
| --- | --- | --- |
| Delete retired `control-ts/src/adapters/{claude,codex,opencode,mcp}.ts` after schema salvage | Baseline returned empty success; current remediation reports unavailable and throws | Current package is private/fail-closed. Preserve only schemas that join the Go adapter contract; no user workflow may import these implementations |
| Delete or replace `control-ts/src/events` in-memory "Go bridge" | Comments explicitly say no file I/O and production would tail NDJSON | Salvage event schemas only; use canonical journal client |
| Keep `control-ts` integration tests isolated until package deletion | Live test deleted `COLONY_STATE.json` and raced on cleanup; remediation restored cached state and proved unchanged hashes | Tests now inject temporary state and default legacy writes under `control-ts/.retired-state`; retain an out-of-root guard |
| Bind one production launch owner to every run | Subprocess paths now use Go; platform-native wrappers remain an explicit second adapter mode | Record Go-adapter versus native-wrapper ownership in the run journal and reject duplicate dispatch for one manifest |
| Remove checked-in same-basename JS beside TS source/tests where generated | 40 source and 42 test TS/JS duplicate basenames in `.aether/ts-host` | Build JS into release `dist`; source tests execute TS or built package, not both |
| Remove direct canonical writes outside Go scoped core | Hive and control-plane tests bypass Store/locks; docs say Go is sole writer | Add core APIs and migrate callers; fail static/runtime write audit |

## Commands To Merge Or Hide

| Current commands/family | Target | Evidence | Migration impact |
| --- | --- | --- | --- |
| `continue` | Public `verify`; keep `continue` alias for one cycle | User job is verification/advancement, not generic continuation | Update docs/wrappers; scripts receive deprecation warning then alias remains compatibility-only |
| `plan --refresh`, `phase-insert`, roadmap mutation utilities | `plan revise` with suboperations | Current refresh rejects completed work; insertion is the only corrective mutation | Migrate phase IDs to stable node IDs; preserve old insert as translated transaction |
| `patrol`, `medic-*`, state health utilities | `doctor` / `doctor repair` | Overlapping health/recovery surface | Preserve JSON result fields through adapter layer; repair always previews/backups |
| `watch`, worker-process, spawn-tree read utilities | `workers` / `workers watch` / `workers explain` | Observability split across commands/files | Rebuild from run journal; aliases forward without owning state |
| `focus`, `redirect`, `feedback`, `pheromones`, housekeeping/display internals | `signals add/list/explain/resolve` | One signal domain with inconsistent effective expiry/conflict behavior | Keep familiar verbs as aliases; migrate records to scoped signal schema |
| Hive, instinct, memory, Queen promotion utilities | `knowledge` advanced namespace | Internal pipeline primitives are exposed as public commands | Block direct confidence manipulation; offer read/explain/revoke first, promotion only policy-mediated |
| `hive-init/store/read/abstract/promote` | Hidden/experimental until redesigned | Journey F contamination; direct unlocked writes | Migrate legacy entries to quarantine file; no automatic injection; export/review tool for users |
| `build-finalize`, `plan-finalize`, `continue-finalize`, colonize/seal finalizers | Hidden `internal finalize` API | Internal host boundary appears alongside public lifecycle | Keep machine contract for chosen host; remove top-level help/completion; require run token |
| `host plan/build/continue/...` | One internal scheduler entry or delete | Duplicates direct lifecycle names | Existing scripts receive an execution-owner migration guide; public users use normal lifecycle |
| `run`, `swarm` | Hide behind experimental until core passes | Autopilot inherits false completion; swarm is routing/watch compatibility | Re-enable `run` only after acceptance; `swarm` becomes alias to `run` or `workers watch` based explicit args |
| `lay-eggs` and setup/install bootstrap internals | Public `start` invokes setup as needed; keep `lay-eggs` advanced alias | Beginner should not understand hub/scaffold topology | Idempotent setup retains ant-flavored message but no extra mandatory step |
| 356 catalogued public utilities | `debug`, `knowledge`, `signals`, `workers`, `adapters`, `internal` namespaces or package APIs | Command catalog itself demonstrates cognitive load | Generate compatibility map; retain stable JSON schemas where external scripts exist; remove unclassified entries before release |

## Caste Consolidation

| Current castes | Target | Evidence | Migration impact |
| --- | --- | --- | --- |
| Surveyor Nest, Disciplines, Pathogens, Provisions | Scout in Colonizer mode with four typed lenses | Same platform/tools and one colonize job; fallback left four narrated as spawned | Keep four report sections and ant subtitles; one process or policy-selected parallel read workers only when proven |
| Tracker + Scout + Archaeologist | Scout/Investigator modes | All gather evidence; only prompts/lenses differ; write isolation absent | Preserve mode-specific prompt assets and outputs; enforce read-only adapter profile |
| Keeper + Sage | Optional Knowledge Curator | Both synthesize patterns/knowledge; current memory pipeline split | Disable by default until typed learning passes; migrate prompts into one skill set |
| Medic + Fixer | Deterministic `doctor` plus Builder repair mode | Health diagnosis is code-driven; repair is implementation work | Preserve approval modes as adapter permission policy, not persona prose |
| Probe + Watcher | Watcher verification plus Builder test-author mode | Probe writes tests/coverage while Watcher verifies; current build runs both by default | Keep separate permission profiles: test-author writes scoped tests, verifier read-only |
| Auditor + Gatekeeper + Includer + Measurer | Watcher review policies/plugins | No unique execution contract today; selected via keywords/depth | Preserve named ant identity in optional report line when policy runs; do not spawn separate model unless measured benefit |
| Architect | Route-Setter design lens or optional plan reviewer | Design work precedes implementation and shares proposal-only contract | Existing architect prompt becomes a review skill; no default pre-wave on simple phases |
| Weaver + Chronicler + Ambassador | Builder skills | Refactor, docs, and integration are implementation modes | Preserve routing keywords to skill selection; use scoped credentials for integration |
| Chaos | Isolated resilience plugin | Destructive/edge testing needs a real sandbox not shared write prompt | Remove from default light/full budgets; re-enable only with isolated adapter capability |
| Porter | Delivery plugin after seal | Publish/push/deploy is high-risk and not core implementation | Require explicit human approval and command allowlist; no automatic spawn |
| Oracle | Advanced research lifecycle, not build caste | Standalone loop is valuable; build inclusion adds cost and duplicates Scout | Keep name and command; feed typed research decisions into plans |
| Queen, Route-Setter, Scout, Builder, Watcher | Five core castes | These map to coordination, planning, investigation, implementation, verification | Add plain-English roles and enforced permission/result contracts; maintain ant visual identity |

## Skill Consolidation

Skill files should not be deleted solely by text similarity. First run behavioral matching/evaluation. These clusters are concrete merge candidates because they occupy the same user transition.

| Skills | Target | Evidence | Migration impact |
| --- | --- | --- | --- |
| `worker-priming`, `colony-interaction`, `colony-lifecycle` | `worker-runtime-contract` | All explain context, transition behavior, and interaction rules; observed prompt was extremely large | Preserve mandatory safety rules in adapter-generated brief; move presentation guidance out |
| `context-management`, `session-handoff`, `session-reporting`, `cross-session-knowledge-threads` | `context-continuity` | Same session/projection job split across four skills | One typed continuity skill; legacy names alias for one cycle |
| `acceptance-test-generation`, `acceptance-verification`, `validation-gap-filling`, `cross-phase-acceptance-scan`, `evaluation-coverage-audit` | `acceptance-evidence` with author/reviewer modes | Overlapping acceptance/test/coverage responsibilities | Maintain read/write modes and role eligibility; reduce simultaneous injection |
| `brownfield-codebase-analysis`, `comprehensive-codebase-map`, `focused-codebase-scan`, `phase-context-gathering`, `documentation-ingestion` | `repository-investigation` with lenses/depth | Same repository evidence acquisition job | Matcher selects depth/lenses rather than injecting multiple long skills |
| `verified-phase-planning`, `phase-dependency-analysis`, `planning-assumption-audit`, `spec-refinement`, `roadmap-management`, `milestone-gap-planning` | `plan-revision` modules | Planning graph creation/revision is fragmented | Keep sections as on-demand references, one front door and schema contract |
| `focused-technical-research`, `technical-feasibility-spike`, `hypothesis-debugging`, `workflow-failure-forensics` | `investigation-methods` with research/spike/debug/forensics modes | Evidence-first investigation patterns share stop/source/result rules | Preserve distinct outputs; avoid loading all modes for every Scout |
| `pheromone-protocol`, `pheromone-visibility` | `signals-contract` | Producer/consumer semantics currently conflict and expiry differs | Replace prose precedence with runtime-resolved signal block and a short skill |
| `milestone-audit`, `milestone-lifecycle`, `artifact-archive-cleanup`, `safe-rollback` | `closure-and-recovery` or core runtime docs | Much is deterministic state/archive behavior, not model expertise | Move safety operations to Go; retain reviewer checklist where judgment is required |
| `documentation-generation`, `pull-request-shipping`, `future-idea-capture`, `github-inbox-triage` | Optional delivery/product skills | Not required for core build loop | Ship in extended pack; no default match |
| Broad domain skills such as Supabase/API security | Keep, but require codebase evidence threshold | Journey C matched Supabase to a Go CLI platform bug | Add negative matching and observed dependency proof; report why each skill matched |

## State And Artifact Consolidation

| Current sources | Target | Evidence | Migration impact |
| --- | --- | --- | --- |
| `COLONY_STATE.json` plus `session.json` | Journal + generated state/session projections | Duplicate goal/phase/next/baseline | N-1 migration imports both, reports conflict, state wins unless later journal evidence |
| `CONTEXT.md` plus `HANDOFF.md` | Generated continuity projections | Handoff recovery lost plan and showed contradictory flags | Keep filenames for platform compatibility; add revision/hash and rebuild command |
| planning Markdown plus `phase-plan.json` plus canonical plan | Plan proposal/evidence records + accepted plan revision | Valid plan can remain uncommitted then be deleted | Archive by run ID; no destructive clear; projection indicates accepted/superseded |
| build manifest, result collection, claims, handoffs, spawn files | One run/evidence ledger with views | All describe one dispatch with different freshness/status | Preserve compatibility JSON views through projection layer |
| `queen-state-N`, `queen-audit-N`, recovery logs, gate reports | One verification decision record | Multiple files consolidate one advancement choice | Migration importer links legacy files and flags disagreements |
| State `Memory.Instincts`, `instincts.json`, event bus, `colony.db` | Typed knowledge journal; DB/index projections | Multiple learning paths and failing consolidation | Quarantine unmatched legacy records; preserve raw source for audit |
| Local/global Queen prose | Project coordination projection + typed user preference store | Scope/type mixing | Parser imports headings conservatively as unverified advisory entries |
| Hive JSON | Quarantined optional knowledge store | Cross-project leakage and no lock/evidence | Export existing entries for review; do not auto-inject or delete user data |
| Constraints legacy plus pheromones | Typed signals | Docs already call constraints legacy | One migration copies active constraints with explicit provenance/expiry |

## Platform And Prompt Consolidation

| Proposed action | Evidence | Migration impact |
| --- | --- | --- |
| Generate Claude/OpenCode/Codex role files from one schema plus platform overlays | 27 definitions in each of three formats | Preserve platform-specific fields; CI compares generated semantic contract |
| Generate thin command wrappers from command/adapter contract | Seven core Claude and OpenCode wrapper sets are line-for-line equal in size and duplicate orchestration prose | Wrappers retain native syntax and interview UX; state/verification behavior removed |
| Remove hard-coded prompt copies from Go after adapter asset loader is authoritative | Boundary doc lists planned prompt extraction; current Go and assets both contain behavior | Keep compiled fallback only for repair/diagnostic mode with visible version |
| Remove duplicate command YAML/Markdown claims that runtime cannot validate | Docs and generated companion files overstate local topology/parity | Add schema validation and executable quickstart tests before generation |

## Ceremony Removal

Remove or rewrite any line that cannot be generated from a real event:

- `build_completed` when only a packet/manifest exists.
- `spawned` for fallback surveyors that did not launch.
- `merged` without verified application of that worker's diff.
- `passed` when a required command was unresolved/skipped.
- `resumed:true` when state fields were discarded.
- Worker completion frames from synthetic/stub adapters unless visibly labeled simulation.

Retain only: goal, accepted plan revision, phase, real process/worker purpose, operation, accepted evidence, blockers, signals/conflicts, and next legal transition. Visual ant names/colors remain presentation over those events.

## Documentation-Only Or Aspirational Features To Relabel

- Self-organization beyond hard-coded scoring and model-written spawn claims.
- "Parallel" in shared-tree builder mode.
- Cross-platform parity beyond the published adapter capability matrix.
- Trusted cross-project learning.
- Worker permission isolation.
- Automatic research-driven replanning.
- Stable custom caste/plugin contract.
- Reliability percentages or claims that Aether never repeats mistakes.

Migration impact: change to `experimental`, `limited`, or `planned`; link each to its acceptance gate. Do not silently remove user-facing identity or stored data.

## Postpone

- Visual dashboard.
- Marketplace/community packages.
- Multi-user/network colony state.
- More platforms.
- More castes or domain skills.
- Richer ceremony/animation.
- Autonomous publish/deploy.
- Automatic Hive promotion.
- Multi-phase unattended run.

These resume only after the core release suite passes on two consecutive release candidates.
