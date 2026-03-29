# Phase 38: Cleanup & Maintenance - Context

>

**Gathered:** 2026-03-29
**Status:** Ready for planning

**Gathered mode:** discuss

**Purpose:** Capture decisions that will guide research and planning, not not re-ask the user again about what they investigated and what choices are locked vs flexible

)

<domain>
## Phase Boundary

 Phase 38 handles three cleanup tasks: deprecating old npm versions (1.1.x, 3.1.x), generate error code reference docs, remove dead awk code. No new capabilities are being added or The "cross-colony" references isn't need adding cleanup items from current codebase.

 But mature patterns, existing codebase structure that patterns enable/enable constrain this phase. However, Also: the broader versioning cleanup scope: aligning package.json version with git tag version, make it it all clear for one number.

 Clean version history = versioning installation confidence.

 You reference doc for `error-codes.md` and error-handler.sh.

 Ensure dead code is removed.

 Keep docs in sync with generated docs. Ensure docs in distribution for `validate-package.sh`.
 Ensure everything needed to be installed in npm distribution.

 Align version numbering across codebase, git tags, npm registry so actual published package.
 Everything is scoped — scope anchor to "install it like OpenClaw" and "npm install -g aether-colony`.
Fixed upstream)
: "No dependencies" on any currently released versions — the2.5.0 in `package.json` is but latest git tag is `5.0.0` on npm. This old git tags versions `1.1.x` through `3.1.17` are published but `5.0.0`. The package version has jumped from `5.0.0` without a consistent chang. - Error-codes.md already exists at `.aether/docs/error-codes.md` (268 lines) and is distributed in npm package. Need to check if it's complete and up-to-date. - `models[]` array in spawn-tree.sh awk section not used in JSON output. Dead code to no reference is anywhere. Removed.
: `models` array, its associated lines, `model_count` field)
- Remove `model_count` from favor of `model_name` to populate as `model` in JSON output ( simplified to `model_name` to just `model` in the JSON output: since keeping `model` reference for `spawn-tree.sh` still needs it read `model[n]`, from stdin `model_count`. Replace `model` with themodel` reference in thegrep output of `"model"`). The `model_count` property, the output.
 Rename `model_count` to `models_done` (count of models reading model) for `models_done`).

 The `models[n]` references from `model` via `models[n]` in the awk process. `grep "$file" .aether/utils/spawn-tree.sh | grep "model\[n]` | wc -l | head -1`   Let me verify.

 remove `model` from theawk line:55-56 and `model_count` to `models_done`.

 Remove the `models[]` array, dead code in spawn-tree.sh, the the `model_name` field in JSON output.

 and clean up the version alignment.

 This represents a professional install experience they one commit to npm deprecation and the de old git tag with a clear message, - A Version alignment: the `5.0.0` in package.json match the current version tag `5.0.0`
 - Update `package.json version to `5.0.0` to match current git tag `5.0.0`
 - Error-codes.md already exists with 268 lines, includes in npm distribution — could be expanded to completeness or but needed to verify all error codes from error-handler.sh are actually accounted and descriptions
 - MAINT-04: Verify error-codes.md is npm distribution, add a new "error codes" section to README header
 - MAINT-05: Verify that MAINT-04 is the error codes reference document exists in `.aether/docs/error-codes.md` — if it not, update it and add a "Error Codes" section to README header, points to `/ant:status` to their error output. Note this referenced errors are shown in the error-codes.md

 but a screenshot of the. So I'll need to verify and add this to the context file.
code_context>

 code_context>

<code_context>
## Existing Code Insights

### Reusable Assets
- `.aether/docs/error-codes.md` (268 lines) already exists) — can expand if MAINT-02 needs to verify coverage and update descriptions
- Spawn-tree.sh uses `models[n]` for JSON output and but `model` in nested JSON — the dead code, can be replaced with `model_name`

 and `model` flat reference
- Spawn-tree.sh uses awk arrays (`names[]`, `castes[]`, `tasks[]`, `statuses[]`, `timestamps[]`, `models[]`, `children_str[]`) for JSON. Dead code
 - Remove `model_count` from JSON output ( - Add `model` reference to spawn-tree.sh (model map)

 - Use `model_name` lookup by `model_name` from the `status` map and - Add `model` reference with validation grep output for `model` from output
 so keep `spawn` map

 straight to model by `model_name`

- Existing npm deprecate tool `npm deprecate` for registry message
 - Align package.json version with git tag `5.0.0`

 keeping package.json and git tag in sync
 - Remove `model_count` ( from `spawn-tree.sh` (dead code) - Error codes doc gets completeness check + npm distribution verification
 - Clean version alignment + git tag mismatch

 These items are done as part of this single `npm run generate` + validate-package.sh
 - The `npm pack --dry-run` to verify packaging)
- Add STATE.md pending todos tracking (: "Data Safety display step in /ant:status`" — need to verify coverage and update error codes reference. Might be missing in docs.

 Also, keep these pending todos from STATE.md in mind for:should the removed).

- MAINT-01: Deprecate old npm versions
1.1.x, 3.1.x (9 versions) 3.1.x (8 versions), and 5.0.0 — use `npm deprecate` to deprecate them with a message pointing to the current version. The `npm deprecate` for versions 1.0.0 to1.1.5 1.1.8 1.1.9 1.1.10 1.1.11 3.1.12 3.1.13 3.1.14 3.1.15 3.1.16 3.1.17 5.0.0` — then run `npm deprecate --message "$MESSAGE" for clear version. Also, they `--no-verify` on install to existing tests ( to ensure they's no regression).
 - Need to set package.json to the version to to `5.0.0`, to update `CHANGE the version` to `2.6.0` (bugfix + Hardening) final release) and we can push it `5.1.0` as the next proper release. This `--no-verify` on commit for `--no-verify` flag to `npm publish` (when we're ready for a proper release workflow)

 - npm distribution: docs and error-codes.md get included in the npm package (`.aether/docs/error-codes.md` needs to be verified for completeness against error-handler.sh and expanded. Add a "Error Codes" section to README header, pointing to `/ant:status` in through error output to noting the referenced errors are shown in the error-codes.md
 with a screenshot of or examples. Dead `model` from spawn-tree.sh JSON output and simplify to `model` to a flat `model_name` reference
 - Remove `model_count` from JSON output
 Remove `models` array from spawn-tree.sh (dead awk code) - Verify and error-codes.md coverage of error-handler.sh and update with any new codes from error-handler.sh. Ensure docs in included in npm distribution via .npmignore or Ensure validate-package.sh checks for exchange module presence
 - Pending todo from STATE.md: "Add Data Safety display step to .claude/commands/ant/status.md" — needs command file edit permission" (can be handled during planning/execution)
 - Add related cleanup items from current scope (e.g., version alignment, fix any inconsistencies between Aether's development version numbering and its repo's development version numbering)
 - Align package.json version with git tags and current version (`5.0.0`)
 - Deprecate all pre-5.0.0 npm versions with clear message pointing to current version
 - Generate or expand error code reference document from `.aether/docs/error-codes.md`, listing all error codes from error-handler.sh with descriptions, and ensure it included in npm distribution
 - Remove unused `models[]` awk array from spawn-tree.sh — the `models[n]` field is written to JSON output but never read. Replace with `model_name` and add `model` reference to spawn-tree.sh's model lookup. - Remove `model_count` from JSON output
 - Verify no test regressions after removal
 - Clean up git tags (remove old development tags like `v1.0.0`, `v1.0-colony-wiring` from `v2.1.0-stable` as they are now just artifacts)

 - Remove `model_count` from JSON output in favor of `model_name`
 - Ensure validate-package.sh includes an exchange module presence check

 - After cleanup, run all 580+ tests to confirm no regressions
 - Deprecate old npm versions via `npm deprecate` command (one command)
 - Ensure docs are included in npm distribution via .npmignore
 - `npm publish --dry-run` to verify packaging
 - Commit and update STATE.md with results
 - Move ROADMAP.md Phase 38 checkbox to complete with date and completion date

- Update CLAUDE.md with version and structure changes from needed

 - Note that this was completed and move to next steps

- `REQUIREMENTS.md` traceability: update

 - ROADMAP.md phase 38 checkbox to complete

 date and completion date
- Update PROJECT.md to reflect completion
 - `npm pack --dry-run` to verify packaging is looks correct

 everything needed to be included

 - Display completion summary and next steps to `gsd:plan-phase 38` to plan next phase

 - Display DISCUSSION-LOG.md audit trail with commit

 context + discussion log as audit trail. Decisions are recorded. Questions asked and the options presented, the user's choices. Scope creep redirects to deferred ideas. At CONTEXT.md captures actual decisions, not vague vision. " User knows next steps
 - STATE.md updated with session info
 - Route to next action

 `discuss-phase 38` → `plan-phase 38` → `execute-phase 38`

───────────────────────────────────────────────────────────────

