# Phase 38: Cleanup & Maintenance - Discussion Log



> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions captured in 38-CONTEXT.md — this log preserves the analysis.

>
**Date:** 2026-03-29
**Phase:** 38-cleanup-maintenance
**Areas discussed:** Deprecation scope, Error code docs, Dead awk code, Version alignment

 Npm packaging, Test verification

---

## Deprecation Scope

**User's choice:** You decide
 ✓
 **Notes:** Non-technical user wants professional npm install experience. Deprecate all pre-5.0.0 versions. Clean up development git tags.

 Align package.json version with current tag. Keep Aether versioning (v2.0.0) separate from npm version (5.0.0).

 MAINT-02 and MAINT-03 can be verified/executed during the execution phase.

 The Pending STATE.md todo for Data Safety display step to .claude/commands/ant/status.md will be folded into scope.

 MAINT-04, error-codes.md coverage check, MAINT-05, git tag cleanup, MAINT-06, npm distribution verification also included.

 MAINT-07, verify no test regressions after changes.

 MAINT-08, display version in output. MAINT-09, release workflow notes.

 Clean install experience like other CLI tools. A Aether is a development tool; GSD is a build tool for users install both as `aether-colony`.

 Both share the same repo.

 Document this connection clearly. Version numbering structure: Aether has its versioning (v2.0.0 in CLAUDE.md), npm publishes as 5.0.0, The two are different because the development versioning (GSD) and the release versioning. This `.claude/get-shit-done/` — the this is normal and will remain separate, GSD version in the agent context only.

 A slight future scope for GSD cleanup (version in tools like `gsd-tools.cjs`).

 the GSD docs reference release version.

 | Option | Description | Selected |
|--------|-------------|----------|
| You decide | ✓ | | |

## Dead awk code
**User's choice:** You decide | ✓
 **Notes:** Replace `models[n]` with `model_name`, keep `model` reference for model lookup. Remove `model_count` from JSON output in favor of just `model_name`. Verify all tests pass after removal. |

| Option | Description | Selected |
|--------|-------------|----------|
| You decide | ✓ | |

## Claude's Discretion
All areas decided by the user or deferred to Claude.
Clean up git tags (deprecate old development tags).
 Align package.json version with git tag `5.0.0`.
 Update CHANGELOG.md. Add release workflow notes.

 Expand error-codes.md if needed.
 Verify all test regressions.

## Deferred Ideas
None — discussion stayed within phase scope.
