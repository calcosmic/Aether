# Research Plan

**Topic:** Post-worktree-merge recovery: fix Go compile errors (cmd/flags.go, cmd/history.go, cmd/phase.go unused imports), close the worktree merge-back gap in build playbooks, verify npm publish flow with Go code, plan push/publish sequence, assess other repos via registry, and chart Go transition roadmap from .planning/ phases.
**Status:** active | **Iteration:** 5 of 30
**Overall Confidence:** 40%

## Questions
| # | Question | Status | Confidence |
|---|----------|--------|------------|
| q1 | Fix the compile errors: read cmd/flags.go, cmd/history.go, cmd/phase.go and ALL other Go files. Identify every unused import and fix them so go build ./cmd/aether succeeds. Verify with go test ./... and go test -race ./pkg/... | answered | 95% |
| q2 | Fix the worktree merge-back gap: read build playbooks (.aether/docs/command-playbooks/build-*.md), continue playbooks (.aether/docs/command-playbooks/continue-*.md), and command orchestrators (.claude/commands/ant/build.md, continue.md). Identify WHERE in the wave lifecycle to add merge-back, design the step with safety checks and conflict handling, and decide if it belongs in build, continue, or a new dedicated playbook. | partial | 35% |
| q3 | Verify the publish flow: read package.json, bin/validate-package.sh, .npmignore, .gitignore. Determine if Go binary needs compilation in npm publish, whether go.mod/Go sources are included or excluded, check pre-publish hooks, and validate .gitignore doesn't exclude needed Go files. | partial | 40% |
| q4 | Plan the push and publish sequence: determine branch strategy (main?), whether go mod tidy should run, whether npm version needs bumping, and what repos need updating via aether update. Read ~/.aether/registry/ to find affected repos. | partial | 72% |
| q5 | Assess other repos that may be affected: check ~/.aether/registry/ for repos using Aether, check for stale colony state referencing old branch structure, and determine if aether update needs manual intervention. | open | 0% |
| q6 | Go transition roadmap: read .planning/ROADMAP.md and .planning/STATE.md, assess phase status, determine where we are in shell-to-Go conversion, identify next logical phase to implement, and recommend which high-frequency shell subcommands should migrate to Go next. | open | 0% |

## Next Steps
Next investigation: Assess other repos that may be affected: check ~/.aether/registry/ for repos using Aether, check for stale colony state referencing old branch structure, and determine if aether update needs manual intervention.

## Source Trust
| Total Findings | Multi-Source | Single-Source | Trust Ratio |
|----------------|-------------|---------------|-------------|
| 24 | 13 | 11 | 54% |

---
*Generated from plan.json -- do not edit directly*
