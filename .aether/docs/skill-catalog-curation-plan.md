# Skill Catalog Curation Plan

Saved: 2026-04-25

Status: implemented in the 2026-04-25 skill catalog curation work. The final shipped catalog is 83 skills: 52 colony skills and 31 domain skills.

## Summary

Import the useful skills from `/Users/callumcowie/- MASTER - Aether /Aether Skills:Agents:Prompts/skills` into Aether's real skill system as a curated migration, not a blind copy.

The implementation preserves useful body content, renames over-themed skills to functional names, repairs frontmatter, maps skills to real Aether castes/workflows, mirrors them into `skills-codex`, and updates the matcher so a larger colony skill catalog does not flood every worker prompt.

## Key Changes

- Add optional skill metadata fields:
  - `workflow_triggers`: Aether commands/workflows where the skill is eligible, for example `colonize`, `plan`, `build`, `continue`, `medic`, `seal`.
  - `task_keywords`: plain task terms that strengthen matching, for example `survey`, `dependency`, `failure`, `rollback`, `acceptance`, `frontend`.
- Update skill matching:
  - Domain skills keep current behavior: repo/task evidence required.
  - Existing broad colony skills without new metadata keep current role-based behavior.
  - Newly curated colony skills with `workflow_triggers` or `task_keywords` require workflow/task evidence, not just role.
  - Runtime callsites pass workflow context into skill resolution for `colonize`, `plan`, `build`, and `continue`.
- Keep `.aether/skills/` as the source of truth and mirror every imported/changed skill into `.aether/skills-codex/`.
- Update expected packaged skill counts after migration and keep hub/integrity/medic checks consistent.

## Migration Map

### Domain Skills To Import

`accessibility`, `api-security`, `event-driven-architecture`, `flutter`, `kotlin`, `kubernetes`, `observability`, `performance-engineering`, `redis`, `rust`, `supabase`, `swift`, `terraform`.

### Colony Skills To Import Or Rename

| Current | Final name | Workflows/castes |
|---|---|---|
| `colonize-analyzer` | `brownfield-codebase-analysis` | `colonize`; surveyors, scout, architect |
| `codebase-scanner` | `focused-codebase-scan` | `colonize`, `plan`; scout, surveyors |
| `codebase-mapper` | `comprehensive-codebase-map` | `colonize`; surveyors, architect |
| `doc-ingester` | `documentation-ingestion` | `colonize`, `plan`; scout, chronicler, keeper |
| `context-gatherer` | `phase-context-gathering` | `plan`, `build`; scout, architect, watcher |
| `assumption-surfacers` | `planning-assumption-audit` | `discuss`, `plan`; oracle, architect, scout, route_setter |
| `spec-refiner` | `spec-refinement` | `discuss`, `plan`; architect, route_setter, watcher |
| `phase-planner` | `verified-phase-planning` | `plan`; route_setter, architect |
| `dependency-analyzer` | `phase-dependency-analysis` | `plan`; route_setter, architect, scout |
| `research-isolator` | `focused-technical-research` | `plan`, `build`; oracle, scout, architect |
| `feasibility-spiker` | `technical-feasibility-spike` | `discuss`, `plan`; scout, oracle, architect |
| `idea-explorer` | `idea-shaping` | `discuss`; scout, oracle, architect |
| `ai-design-contract` | `ai-design-contract` | `plan`, `build`; architect, builder, scout |
| `ui-design-contract` | `frontend-design-contract` | `plan`, `build`; architect, builder, scout |
| `ui-reviewer` | `frontend-ui-audit` | `continue`; watcher, auditor, includer |
| `code-reviewer` | `code-review` | `continue`; watcher, auditor, probe |
| `security-auditor` | `security-audit` | `continue`; gatekeeper, auditor, watcher |
| `test-generator` | `acceptance-test-generation` | `build`, `continue`; watcher, probe, builder |
| `uat-verifier` | `acceptance-verification` | `continue`; watcher, auditor |
| `uat-cross-scanner` | `cross-phase-acceptance-scan` | `seal`, `continue`; auditor, scout, watcher |
| `eval-coverage-reviewer` | `evaluation-coverage-audit` | `continue`; auditor, probe, watcher |
| `validation-gap-filler` | `validation-gap-filling` | `build`, `continue`; builder, probe |
| `phase-forensics` | `workflow-failure-forensics` | `medic`, `continue`; medic, tracker, scout |
| `scientific-debugger` | `hypothesis-debugging` | `build`, `medic`; tracker, probe, scout |
| `safe-rollback` | `safe-rollback` | `medic`, `build`; keeper, watcher, medic |
| `colony-cleanup` | `artifact-archive-cleanup` | `medic`, `seal`; medic, keeper, chronicler |
| `learning-extractor` | `phase-learning-extraction` | `continue`, `seal`; keeper, chronicler |
| `session-handoff` | `session-handoff` | `resume`, `pause`; keeper, chronicler, queen |
| `session-reporter` | `session-reporting` | `pause`, `seal`; chronicler, keeper |
| `knowledge-threads` | `cross-session-knowledge-threads` | `resume`, `plan`; keeper, chronicler |
| `seed-planter` | `future-idea-capture` | `discuss`, `plan`; keeper, chronicler, queen |
| `milestone-auditor` | `milestone-audit` | `seal`; auditor, architect, queen |
| `milestone-gap-planner` | `milestone-gap-planning` | `seal`, `plan`; architect, route_setter |
| `milestone-lifecycle` | `milestone-lifecycle` | `seal`, `init`; keeper, chronicler, queen |
| `plan-importer` | `external-plan-import` | `plan`; architect, route_setter |
| `roadmap-manager` | `roadmap-management` | `plan`; route_setter, architect, queen |
| `pr-shipper` | `pull-request-shipping` | `seal`, `ship`; builder, queen |
| `github-inbox-triage` | `github-inbox-triage` | `plan`; scout, queen |
| `docs-generator` | `documentation-generation` | `build`, `continue`; builder, chronicler |
| `cross-model-reviewer` | `cross-model-plan-review` | `plan`, `continue`; architect, auditor |
| `design-sketcher` | `design-prototyping` | `discuss`, `plan`; architect, scout, oracle |

### Merge Or Defer Instead Of Importing Standalone

Merge into existing skills: `wave-executor`, `context-budget-manager`, `parallel-workstreams`, `colony-autopilot`, `quick-task`, `colony-navigator`, `intent-router`, `colony-progress`, `colony-dashboard`, `colony-stats`, `colony-settings`, `colony-init`, `colony-todo`, `audit-fix-pipeline`, `pheromone-evolution`.

Defer until there is explicit runtime/index design: `knowledge-graph`, `codebase-intelligence`, `review-auto-fixer`, `user-profiler`.

## Verification

- New metadata parses and indexes.
- Domain skills still require repo/task evidence.
- Curated colony skills require workflow/task evidence when they declare `workflow_triggers` or `task_keywords`.
- Colonize, plan, build, and continue pass workflow context into skill resolution.
- `.aether/skills` and `.aether/skills-codex` stay mirrored.
- `go test ./cmd -run 'Test(ParseSkillFrontmatter|IndexSkillDir|SkillMatch|SkillInject|ScanWrapperParityHealthy|ScanHubPublishIntegrityHealthy|CheckStalePublish)' -count=1`
- `go test ./... -count=1`
- `go build ./cmd/aether`
- `go vet ./...`
- `git diff --check`
