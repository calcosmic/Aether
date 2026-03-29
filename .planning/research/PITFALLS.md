# Domain Pitfalls: v2.6 Bugfix & Hardening

**Domain:** Fixing bugs in existing Aether colony orchestration system (bash + jq)
**Researched:** 2026-03-29
**Confidence:** HIGH -- grounded in direct codebase inspection of 5,200+ line dispatcher, 10 domain modules, 24 agent definitions, 616+ tests, validate-package.sh, file-lock.sh, spawn.sh, hive.sh, learning.sh, queen.sh

---

## Critical Pitfalls

### Pitfall 1: The Double-Escape Trap When Adding grep Escape Functions

**What goes wrong:**
The codebase has ~30 grep calls that interpolate user-derived variables (ant_name, content, stat_key, colony_name) directly into regex patterns. The instinct is to add an escape function like `escape_grep_regex()` and apply it everywhere. But the existing code has a critical inconsistency: some calls already use `sed` escaping (learning.sh line 1248: `sed 's/[\\/&]/\\&/g'`), some use `grep -F` (queen.sh lines 730, 891, 1039), and some use bare variable interpolation (spawn-tree.sh line 98: `grep "|$ant_name|"`). Adding a new escape function without auditing each call site produces double-escaped patterns.

**Why it happens:**
The codebase evolved across 2+ years and multiple contributors (human and AI). Different modules were written at different times with different escaping conventions. The learning.sh module uses sed-style escaping (for sed calls), queen.sh uses `grep -F` (fixed strings), and spawn-tree.sh uses raw regex. There is no centralized escaping utility -- each module does its own thing.

**Concrete failure modes:**

1. **Double-escape on learning.sh line 1253**: The content is already escaped with `sed 's/[\\/&]/\\&/g'` (line 1248) then passed to `grep -q "${escaped_content}"`. If a new `escape_grep_regex()` function also escapes backslashes, a content string containing `\n` would become `\\n` after sed escaping, then `\\\\n` after grep escaping. The grep would search for literal `\\\\n` and never match.

2. **Wrong escape flavor on spawn-tree.sh line 98**: `grep "|$ant_name|"` uses BRE regex. An escape function designed for ERE (grep -E) would escape different characters than needed for BRE. In BRE, `(` and `)` are literal, but `\(` is a group. In ERE, `(` is a group and `\(` is literal.

3. **The `--` separator missing**: `grep "|$current_ant|"` (spawn.sh line 152) does not use `--` before the pattern. If ant_name starts with `-`, grep interprets it as a flag. An escape function that only escapes regex special chars would miss this bash-level injection.

4. **macOS grep vs GNU grep differences**: macOS ships BSD grep, which has subtle differences from GNU grep in how it handles `[[:space:]]`, `\+`, and `\{n\}`. An escape function tested on Linux may behave differently on macOS.

**How to avoid:**

1. **Audit every grep call site first** -- categorize each into: (a) fixed-string (should use `grep -F`), (b) BRE regex (needs BRE escaping), (c) ERE regex (needs ERE escaping). Document the audit before writing any escape function.

2. **Prefer `grep -F` over escaping** -- most of these calls are searching for literal strings (ant names, content hashes, section headers). `grep -F` treats the pattern as a plain string, eliminating all regex escaping concerns. The queen.sh module already does this correctly (lines 730, 891, 1039).

3. **If regex is needed, use `grep -F` with the dynamic part and concatenate with fixed anchors** -- e.g., instead of `grep "|$ant_name|"`, use `grep -F -- "|$ant_name|"` (since `|` is literal in fixed-string mode). But note: `grep -F -- "$separator$ant_name$separator"` is cleaner.

4. **Never add escaping to calls that already have it** -- the learning.sh line 1248 escaping is for sed, not grep. If the grep on line 1253 needs escaping too, the sed-escaped string must NOT be re-escaped for grep. These are two different escaping contexts applied to the same string for two different tools.

5. **Write the escape function as three separate functions**: `escape_bre()`, `escape_ere()`, `escape_sed_replacement()`. Never use one function for all contexts.

6. **Test on macOS bash 3.2 specifically** -- the escape function must work with `echo "$var" | sed ...` (bash 3.2 compatible, no process substitution tricks). Avoid `${var//pattern/replacement}` (bash 3.2 supports it but edge cases differ from bash 4).

**Warning signs:**
- Tests pass on Linux but fail on macOS (BSD grep difference)
- grep returns no matches for strings that visibly exist in the file
- Content with backslashes (`\n`, `\t`, `\\`) fails to match
- Ant names containing `[`, `]`, `*`, `.` cause grep errors or wrong matches
- `grep: invalid option` errors from ant names starting with `-`

**Phase to address:** Phase 1 (grep escaping audit) -- must be done before any escape functions are written

---

### Pitfall 2: JSON Construction With printf -- The Unquoted Variable Injection

**What goes wrong:**
The core `json_ok` function (aether-utils.sh line 85) uses `printf '{"ok":true,"result":%s}\n' "$1"`. This is safe WHEN the caller passes already-valid JSON. But at least 6 call sites construct JSON inline with bash string interpolation:

- Line 796: `json_ok '{"appended":true,"date":"'"$date_str"'","phase":"'"$phase"'","plan":"'"$plan"'"}'`
- Line 134: `json_ok "{\"ant\":\"$ant_name\",\"depth\":1,\"found\":false}"`
- Line 2655: `json_ok '{"model":"glm-5-turbo","source":"default","caste":"'$caste'"}'`

If any of these variables contain double quotes, backslashes, or newlines, the JSON becomes invalid. The downstream consumer (jq or the Claude Code agent that reads the output) fails silently or crashes.

**Why it happens:**
The codebase was written incrementally. Some JSON was constructed with proper `jq -n --arg` patterns (hive.sh line 28-41 is exemplary), while other parts used printf or echo with string interpolation. The inconsistency was not caught because the variables in question (date_str, phase, plan) are typically controlled strings that rarely contain special characters. But ant_name can contain user-influenced content, and colony_name can contain arbitrary strings.

**Concrete failure modes:**

1. **Ant name with quotes**: If ant_name is `Builder "Alpha"`, line 134 produces `{"ant":"Builder "Alpha"","depth":1,"found":false}` -- invalid JSON with unescaped quotes. jq parsing this produces null or an error.

2. **Colony name in queen.sh line 641**: `grep -q "\"${colony_name}\"" "$tmp_file"` -- if colony_name contains `\`, the grep pattern becomes malformed. If colony_name contains `"`, the grep pattern matches wrong lines.

3. **The `json_err` fallback (line 96)**: `printf '{"ok":false,"error":{"code":"%s","message":"%s"}}\n' "$code" "$message"` -- if `$message` contains `"`, `%`, or `\`, the JSON is malformed. Error messages often include file paths, which on macOS can contain spaces but rarely quotes. However, if an error message includes user input (like a colony name with special chars), this breaks.

4. **The `stat_key` in learning.sh line 1279**: `grep "\"${stat_key}\":" "$tmp_file"` -- if stat_key contains regex-special characters, the grep fails. Currently stat_key values are hardcoded (`total_stack_entries`, `total_philosophies`), but a future change to allow custom stat keys would break here.

**How to avoid:**

1. **Migrate all JSON construction to `jq -n --arg`** -- this is the single most impactful hardening change. Every `json_ok` call that interpolates variables should be rewritten to use jq's `--arg` for strings and `--argjson` for numbers/booleans/arrays.

2. **Do NOT change the `json_ok` function signature** -- it correctly expects pre-validated JSON. The problem is at the call sites, not the function itself.

3. **Migration order matters** -- start with the highest-risk call sites: any that use user-derived variables (ant_name, colony_name, content). Then migrate the lower-risk ones (date_str, phase, plan).

4. **Add a `json_output` helper** that wraps `jq -n --arg` for the common case of simple key-value JSON:
```bash
# Instead of: json_ok "{\"ant\":\"$name\",\"depth\":$depth}"
# Use:       json_output '{"ant":$ant,"depth":$depth}' --arg ant "$name" --argjson depth "$depth"
```
   This makes safe JSON construction the path of least resistance.

5. **Add a lint rule or test** that greps for `json_ok.*\".*\$` and `json_err.*\".*\$` patterns (string-interpolated JSON), flagging them for migration.

6. **The `printf '%s'` pattern in json_ok is actually correct** -- it passes `$1` as a format argument, which handles literal `%` correctly. The problem is ONLY when callers construct JSON with string interpolation before passing it. Do not "fix" json_ok itself.

**Warning signs:**
- `jq: parse error` in downstream consumers
- Agent prompts contain malformed JSON fragments
- `null` values where strings should be in worker context
- Tests that mock `json_ok` output fail when variable content changes

**Phase to address:** Phase 1 (JSON escaping audit) -- run the grep for vulnerable patterns, categorize by risk, then migrate

---

### Pitfall 3: LOCK_DIR Mutation -- The Save/Restore Pattern Is Fragile

**What goes wrong:**
The hive.sh module temporarily mutates `LOCK_DIR` for cross-repo locking (lines 46-57, 133-152, 327-353). It saves the original value, sets `LOCK_DIR="$HOME/.aether/hive"`, acquires the lock, does work, then restores the original value. This pattern has three failure modes:

1. **Early return without restore**: If any code path between save and restore calls `return 1` or `exit 1` (via `json_err`), LOCK_DIR is left pointing to the hub directory. Subsequent lock acquisitions in the same shell session use the wrong directory.

2. **Concurrent hive operations in the same shell**: If two hive functions are called in sequence (e.g., `hive-store` then `hive-read`), and the first fails to restore LOCK_DIR, the second uses the hub LOCK_DIR even though it should use the repo LOCK_DIR.

3. **Subshell boundary confusion**: `acquire_lock` is called in the main shell but `release_lock` is called in a subshell (via `$()`). The `export LOCK_ACQUIRED=true` and `export CURRENT_LOCK=...` in acquire_lock (file-lock.sh lines 122-123) are visible in the parent shell, but if release_lock runs in a subshell, it cannot modify the parent's LOCK_ACQUIRED. The cleanup trap on EXIT (line 157) mitigates this, but only if the shell exits normally.

**Evidence from codebase:**
- hive.sh has 6 separate save/restore blocks (lines 46-57, 133-152, 327-353, and 3 more for error paths)
- Each block follows the pattern: `hv_saved_lock_dir="$LOCK_DIR"; LOCK_DIR="$HOME/.aether/hive"; ... LOCK_DIR="${hv_saved_lock_dir:-$LOCK_DIR}"`
- The `:-` fallback (`${hv_saved_lock_dir:-$LOCK_DIR}`) on restore means that if hv_saved_lock_dir is empty, LOCK_DIR stays as the hub value -- silently wrong
- The QUEEN.md instinct from a previous fix confirms this was a known issue: "Hub-level shared files need hub-level locks -- per-repo LOCK_DIR cannot protect cross-repo resources"

**How to avoid:**

1. **Do NOT use the save/restore pattern** -- instead, pass the lock directory as a parameter to a wrapper function. Create `acquire_hub_lock()` and `release_hub_lock()` that take an explicit directory parameter instead of mutating a global.

2. **If save/restore must be kept** (backward compatibility), wrap every call site in a trap:
```bash
(
  saved="$LOCK_DIR"
  LOCK_DIR="$HOME/.aether/hive"
  trap 'LOCK_DIR="${saved:-$LOCK_DIR}"' RETURN
  acquire_lock "$file" || { LOCK_DIR="$saved"; json_err ...; }
  # ... work ...
  release_lock 2>/dev/null || true
  LOCK_DIR="$saved"
)
```
   Using a subshell `( )` ensures LOCK_DIR is automatically restored on exit, regardless of how the subshell exits.

3. **Audit all `json_err` calls within LOCK_DIR mutation blocks** -- any early return via `json_err` must be preceded by LOCK_DIR restoration. Currently, hive.sh line 48 does this correctly (`{ LOCK_DIR="$hv_saved_lock_dir"; json_err ...; }`), but not all paths are covered.

4. **Add a test that calls hive-store followed by a non-hive function that acquires a lock** -- if the non-hive function uses the hub LOCK_DIR, the save/restore leaked.

5. **Consider using a dedicated lock file path for hub operations** instead of mutating LOCK_DIR: `acquire_lock "$HOME/.aether/hive/wisdom.json.lock"` (using the full path as the lock target). The lock file path is computed from `LOCK_DIR/basename(file).lock`, so if you pass a full path to a different directory, basename still works. Wait -- this does not work because `acquire_lock` uses `LOCK_DIR` to construct the path. The function would need to be modified to accept an optional lock directory parameter.

**Warning signs:**
- Lock files appearing in `~/.aether/hive/` for non-hive operations (or vice versa)
- Lock acquisition fails with "lock already held" when no other process is running
- After a hive-store error, subsequent non-hive lock acquisitions fail
- Test flakiness in parallel test execution (one test's LOCK_DIR leak affects another)

**Phase to address:** Phase 2 (cross-colony isolation fix) -- must be done before adding any new hub-level operations

---

### Pitfall 4: Adding Depth Selector Breaks Existing Spawn Assumptions

**What goes wrong:**
The v2.6 scope includes adding a colony depth selector (light/standard/deep/full) that gates Oracle and Scout spawns. The current spawn system has hardcoded depth limits in `_spawn_can_spawn` (spawn.sh lines 89-96): depth 1 allows 4 spawns, depth 2 allows 2 spawns, depth 3+ allows 0 spawns. Adding a depth selector changes these limits based on configuration.

The risk is not in the depth selector itself, but in **what depends on the current hardcoded limits**:

1. **Build playbooks assume depth 1 workers can always spawn 4**: The build-wave playbook spawns builders, watchers, and scouts in parallel. If the depth selector reduces depth 1 to 2 spawns, the build wave fails to spawn all required workers.

2. **The swarm system has its own cap** (`_spawn_can_spawn_swarm` in spawn.sh line 170-192, cap of 6). If the depth selector applies to swarm spawns, existing `/ant:swarm` calls break. If it does NOT apply, swarm spawns bypass the depth limit entirely, which may be the desired behavior but needs explicit design.

3. **Oracle RALF loop spawns sub-agents**: The Oracle research loop spawns research workers at depth 2. If the depth selector limits depth 2 to 0 spawns (in "light" mode), Oracle cannot spawn research sub-agents, breaking the entire Oracle workflow.

4. **The global cap of 10 (spawn.sh line 105) interacts with depth limits**: Even if depth limits allow more spawns, the global cap of 10 per phase may be hit first. Changing depth limits without adjusting the global cap can create confusing behavior where the error message says "depth exceeded" when the real problem is the global cap.

**How to avoid:**

1. **Make depth selector additive, not restrictive, for existing modes**: The "standard" mode should match the current hardcoded limits exactly (depth 1: 4, depth 2: 2, depth 3+: 0). "Light" reduces these. "Deep" and "full" increase them. This ensures backward compatibility.

2. **Exempt swarm spawns from depth limits**: The swarm system already has its own cap (line 170-192). The depth selector should only affect phase build spawns, not swarm or oracle spawns. Add a `--scope phase` flag to `spawn-can-spawn` that swarm calls use to bypass depth limits.

3. **Oracle spawns need special handling**: Oracle at depth 2 needs at least 1-2 spawn slots for research sub-agents. "Light" mode should still allow Oracle to spawn 1 sub-agent at depth 2. Alternatively, Oracle spawns should use a separate cap from build spawns.

4. **Update the build playbooks to check spawn capacity before spawning**: Currently the playbooks spawn workers optimistically. Add a `spawn-can-spawn` check before each worker spawn in the playbook, with a clear error if capacity is exhausted.

5. **Test with all four modes** (light/standard/deep/full) against the full build lifecycle. The "standard" mode must produce identical behavior to the current system.

**Warning signs:**
- Build wave fails to spawn all workers in "light" mode
- Oracle research returns no results (sub-agents blocked)
- `/ant:swarm` breaks after depth selector is added
- "standard" mode produces different results than pre-selector behavior

**Phase to address:** Phase 3 (depth selector) -- after escaping and isolation fixes, because the depth selector modifies spawn behavior that other fixes depend on

---

### Pitfall 5: YAML Command Generator Creates Synchronization Debt

**What goes wrong:**
The v2.6 scope includes a YAML command generator to eliminate duplication between Claude Code commands (`.claude/commands/ant/`) and OpenCode commands (`.opencode/commands/ant/`). Currently there are 44 commands in each directory, maintained separately. A YAML generator would read a single source of truth and produce both sets.

The pitfall is not in the generator itself, but in the **transition period** and **ongoing synchronization**:

1. **During transition, some commands are YAML-generated and others are hand-maintained**: If the generator only covers a subset of commands, developers must remember which commands to edit in YAML and which to edit directly. A developer edits a hand-maintained command, but the generator overwrites it on next run.

2. **Template divergence**: The YAML generator produces markdown from templates. If the markdown format changes (new Claude Code features, different frontmatter format), the templates must be updated. If the templates and hand-maintained commands drift apart, the generated output looks different from the hand-maintained output.

3. **validate-package.sh content checks**: The current validate-package.sh checks for specific file existence (REQUIRED_FILES array). If the YAML generator changes the file naming convention or adds new files, validate-package.sh must be updated. If validate-package.sh runs before the generator, it fails on missing files. If it runs after, it catches generator bugs but adds build time.

4. **The npm sync problem**: Commands are in `.claude/commands/ant/` (not in `.aether/`) and are NOT distributed via npm. They are local to the repo. OpenCode commands are in `.opencode/commands/ant/`. The YAML generator must run locally, not as part of the npm package. This means the generator is a dev tool, not a runtime component. If a user updates Aether via npm, they do NOT get updated commands -- they must regenerate them.

5. **Generated files in git**: If generated command files are committed to git, PRs show large diffs when the generator is re-run. If they are gitignored, users must run the generator after cloning. Either way, there is friction.

**How to avoid:**

1. **Generate ALL commands or NONE** -- partial generation is worse than no generation. If not all 44 commands can be generated in one phase, defer the generator entirely.

2. **Use a "source of truth" YAML file in .aether/**: Place the YAML definitions in `.aether/commands/` so they are distributed via npm. The generator is a dev-only script that reads from `.aether/commands/` and writes to `.claude/commands/ant/` and `.opencode/commands/ant/`.

3. **Add a header to generated files**: Each generated file should start with `<!-- GENERATED: do not edit manually. Source: .aether/commands/<name>.yaml -->`. This prevents accidental hand-editing of generated files.

4. **Add a CI check or npm script**: `npm run generate:commands` should be run as part of the build process. If generated files are out of date, the build fails.

5. **Do NOT commit generated files to git in the initial implementation** -- instead, generate them as part of the npm postinstall script or a setup step. This avoids the "large diff on re-generation" problem. But this means new clones require a setup step, which conflicts with the "works immediately after npm install" goal.

6. **The safest approach**: Generate commands at npm install time (postinstall hook), gitignore the generated directories, and add a README explaining the generator. But this is a bigger change than v2.6 scope suggests. Consider deferring to v2.7.

**Warning signs:**
- Commands in `.claude/` and `.opencode/` have different content after a manual edit
- validate-package.sh fails after adding new commands
- Users report "command not found" after npm update (generated files not regenerated)
- Git diffs show wholesale command rewrites on every PR

**Phase to address:** Phase 4 (YAML command generator) -- highest risk, lowest urgency. Consider deferring to v2.7 if time-constrained.

---

## Moderate Pitfalls

### Pitfall 6: The `awk -v content="$content"` Regex Injection

**What goes wrong:**
In learning.sh line 1260, `awk -v section="$section_header" -v content="$content"` passes user content as an awk variable, then uses it in a regex match: `$0 ~ content`. If `$content` contains regex special characters, awk interprets them as a pattern, not a literal string. For example, content containing `file.sh` would match `fileXsh`.

This is the same class of bug as the grep escaping issue, but in awk. The fix is different: awk has `index($0, content)` for literal string matching (no regex interpretation).

**Evidence:** learning.sh line 1260: `in_section && $0 ~ content { skip = 1; next }`

**How to avoid:**
Replace `$0 ~ content` with `index($0, content) > 0` for literal matching. Audit all awk `-v` variable usages where the variable is used in regex context (`~`, `!~`, `gsub`, `sub`).

**Phase to address:** Phase 1 (escaping audit) -- same audit pass as grep escaping

---

### Pitfall 7: The timing.log grep Pattern Matches Wrong Lines

**What goes wrong:**
In swarm.sh lines 915 and 960, `grep -q "^$ant_name|" "$timing_file"` matches lines starting with the ant name followed by `|`. If two ants have names where one is a prefix of the other (e.g., "Builder" and "Builder-Alpha"), the grep matches both. This causes wrong timing data to be returned.

**How to avoid:**
Use `grep -q "^${ant_name}|" "$timing_file"` with `grep -F` (fixed string mode) to ensure literal matching: `grep -Fq -- "${ant_name}|" "$timing_file"` with an anchor. But `grep -F` does not support `^`. Alternative: use `grep "^$(printf '%s' "$ant_name" | sed 's/[[\.*^$()+?{|]/\\&/g')|" "$timing_file"` or switch to awk: `awk -v name="$ant_name" 'BEGIN{FS="|"} $1==name{found=1;exit} END{exit !found}'`.

**Phase to address:** Phase 1 (grep escaping audit)

---

### Pitfall 8: `((depth++))` Fails Silently on bash 3.2

**What goes wrong:**
In spawn-tree.sh line 117, `((depth++))` increments depth. On bash 3.2, if `depth` is empty or non-numeric (due to a bug in upstream code), `((depth++))` returns exit code 1 but does NOT produce an error message. In a pipeline or `set -e` context, this can cause silent failure.

The same pattern appears in spawn.sh line 158: `depth=$((depth + 1))`. The `$(( ))` form is safer than `(( ))` because it always evaluates to 0 for empty strings (bash 3.2 behavior differs from bash 4).

**How to avoid:**
Prefer `$((depth + 1))` over `((depth++))` for portability. Always initialize numeric variables before incrementing: `local depth=0` before any `(( ))` or `$(( ))` usage.

**Phase to address:** Phase 1 (bash 3.2 compatibility audit)

---

### Pitfall 9: validate-package.sh Does Not Check Content Integrity

**What goes wrong:**
validate-package.sh checks that required files exist (REQUIRED_FILES array) and private directories are excluded (.npmignore check). But it does NOT check that the files contain the expected content. If a file exists but is empty, corrupted, or has the wrong version, validate-package.sh passes.

This is relevant to v2.6 because the JSON escaping fixes change the content of files like spawn.sh, learning.sh, and queen.sh. A typo in the escaping fix could produce a syntactically valid but semantically wrong file.

**How to avoid:**
Add content checks for critical files: verify that key functions exist in shell files (e.g., `grep -q '_spawn_log()' .aether/utils/spawn.sh`), verify that JSON templates parse correctly (`jq empty < template.json`), verify that shell files have no syntax errors (`bash -n file.sh`). These checks are fast and catch corruption early.

**Phase to address:** Phase 2 (validate-package.sh hardening)

---

### Pitfall 10: The `--` Separator Is Missing From Many grep Calls

**What goes wrong:**
Most grep calls in the codebase do not use `--` to separate options from patterns. If a variable used as a grep pattern starts with `-`, grep interprets it as an option flag. For example, if ant_name is `-verbose`, `grep "|-verbose|"` works (because `|` is before `-`), but `grep "$pattern" "$file"` where pattern is `-e` would fail.

**Evidence:** Of ~30 grep calls with variable interpolation, fewer than 5 use `--`.

**How to avoid:**
Add `--` to all grep calls that use variable patterns: `grep -q -- "$pattern" "$file"`. This is a mechanical change that can be done in a single pass.

**Phase to address:** Phase 1 (grep escaping audit) -- mechanical, low-risk, high-value

---

## Minor Pitfalls

### Pitfall 11: The changelog-append JSON Uses Single-Quote Concatenation

**What goes wrong:**
Line 796: `json_ok '{"appended":true,"date":"'"$date_str"'","phase":"'"$phase"'","plan":"'"$plan"'"}'` uses bash single-quote concatenation to embed variables in JSON. This works but is fragile: any variable containing a single quote (`'`) breaks the concatenation. Date strings are typically safe, but phase names could theoretically contain apostrophes.

**How to avoid:**
Migrate to `jq -n --arg date "$date_str" --arg phase "$phase" --arg plan "$plan" '{"appended":true,"date":$date,"phase":$phase,"plan":$plan}'`.

**Phase to address:** Phase 1 (JSON escaping audit)

---

### Pitfall 12: spawn-tree.sh and spawn.sh Have Duplicated Depth Logic

**What goes wrong:**
Both `spawn-tree.sh` (lines 82-123) and `spawn.sh` (lines 121-168) implement `get_spawn_depth` / `_spawn_get_depth` with slightly different implementations. The spawn-tree.sh version uses `((depth++))` (bash 4+), while spawn.sh uses `depth=$((depth + 1))` (bash 3.2 compatible). Both traverse the parent chain by grepping for `|$ant_name|` in spawn-tree.txt.

If the escaping fix changes the spawn-tree.txt format (e.g., escaping ant names), both implementations must be updated consistently. If only one is updated, depth calculation diverges.

**How to avoid:**
When fixing escaping, update both files in the same commit. Add a test that verifies both implementations return the same depth for the same ant name.

**Phase to address:** Phase 1 (grep escaping audit)

---

### Pitfall 13: The `echo "$var" | jq -Rs '.'` Pattern Drops Trailing Newlines

**What goes wrong:**
Line 1610: `json_ok "$(echo "$content" | jq -Rs '.')"`. The `echo` command adds a trailing newline, and `jq -Rs` slurps it as part of the string. If the original content already had a trailing newline, the result has a double newline. If the original content had no trailing newline, the result has an extra one.

This is a minor issue but can cause test mismatches when comparing round-tripped content.

**How to avoid:**
Use `printf '%s' "$content" | jq -Rs '.'` to avoid the extra newline from echo.

**Phase to address:** Phase 1 (JSON escaping audit)

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| printf JSON instead of jq -n | Faster to write, no jq dependency in simple cases | Silent corruption on special characters | Never for user-derived variables; acceptable for hardcoded strings |
| Save/restore LOCK_DIR | No function signature changes needed | Fragile, leaks on early returns | Only as temporary measure; migrate to parameter-passing within same phase |
| Bare grep without -F | Supports both regex and literal matching in same call | Requires escaping for literal strings | When the pattern is a controlled constant (e.g., `"^## "`) |
| Duplicated depth logic in spawn.sh and spawn-tree.sh | Each module is self-contained | Divergent behavior when one is fixed | Never -- should be consolidated |
| Hand-maintained command files | No generator complexity, immediate editing | 44 files x 2 directories = 88 files to keep in sync | Only until YAML generator is complete |

---

## Phase-Specific Warnings

| Phase Topic | Likely Pitfall | Mitigation |
|-------------|---------------|------------|
| grep escaping audit | Pitfall 1: double-escape trap | Categorize all grep calls first; prefer grep -F for literal matching |
| grep escaping audit | Pitfall 6: awk regex injection | Use index() instead of ~ for literal matching |
| grep escaping audit | Pitfall 7: timing.log prefix matching | Use awk with exact field match instead of grep |
| grep escaping audit | Pitfall 10: missing -- separator | Mechanical add -- to all grep calls |
| JSON escaping audit | Pitfall 2: printf JSON injection | Migrate to jq -n --arg; highest-risk call sites first |
| JSON escaping audit | Pitfall 11: changelog-append concatenation | Migrate to jq -n --arg |
| JSON escaping audit | Pitfall 13: trailing newline in jq -Rs | Use printf instead of echo |
| Cross-colony isolation | Pitfall 3: LOCK_DIR mutation leak | Use subshell isolation or parameter-passing |
| Cross-colony isolation | Pitfall 9: validate-package.sh content checks | Add function existence and syntax checks |
| Depth selector | Pitfall 4: build wave spawn assumptions | "standard" mode must match current behavior exactly |
| Depth selector | Pitfall 8: ((depth++)) on bash 3.2 | Use $((depth + 1)) consistently |
| YAML generator | Pitfall 5: synchronization debt | Generate ALL commands or NONE; add generated-file headers |
| YAML generator | Pitfall 12: duplicated spawn logic | Consolidate before changing format |

---

## "Looks Done But Isn't" Checklist

- [ ] **grep escaping**: All 30+ grep calls with variable interpolation audited and categorized -- verify with `grep -rn 'grep.*\${' .aether/`
- [ ] **grep -F migration**: All literal-string grep calls converted to `grep -F` -- verify no unescaped regex patterns remain for literal searches
- [ ] **JSON construction**: All `json_ok` calls with string interpolation migrated to `jq -n --arg` -- verify with `grep -rn 'json_ok.*\"\$' .aether/`
- [ ] **LOCK_DIR save/restore**: All hive.sh code paths that mutate LOCK_DIR verified to restore on every exit path -- verify with tracing test
- [ ] **Depth selector backward compatibility**: "standard" mode produces identical spawn behavior to pre-selector system -- verify with before/after test
- [ ] **YAML generator**: If implemented, all 44 commands generated from YAML -- verify file count matches
- [ ] **macOS bash 3.2**: All changes tested on macOS system bash (not Homebrew bash) -- verify with `bash --version`
- [ ] **validate-package.sh**: Content checks added for function existence and shell syntax -- verify with intentionally broken file

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| Double-escape in grep patterns | MEDIUM | Revert the escape function, re-audit each call site, apply correct escaping per context |
| JSON corruption from printf | LOW | Revert to jq -n --arg; downstream consumers already handle occasional null gracefully |
| LOCK_DIR leak | HIGH | Identify which shell session has the leak (check env), restart the session; add subshell isolation |
| Depth selector breaks builds | LOW | Revert to hardcoded limits; "standard" mode is defined to match current behavior |
| YAML generator overwrites hand-edits | MEDIUM | Restore from git; add generated-file header to prevent future hand-edits |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| 1: grep double-escape | Phase 1 (escaping audit) | Test with strings containing `.\*[]$^{}\|()\\` |
| 2: printf JSON injection | Phase 1 (JSON audit) | Test with strings containing `"`, `\`, `'`, newlines |
| 3: LOCK_DIR mutation leak | Phase 2 (isolation fix) | Test hive-store error followed by non-hive lock acquisition |
| 4: depth selector breaks builds | Phase 3 (depth selector) | Test full build lifecycle in all 4 modes |
| 5: YAML generator sync debt | Phase 4 (YAML generator) | Test adding a new command and regenerating |
| 6: awk regex injection | Phase 1 (escaping audit) | Test with content containing `.*[]` |
| 7: timing.log prefix match | Phase 1 (escaping audit) | Test with ant names that are prefixes of each other |
| 8: bash 3.2 ((depth++)) | Phase 1 (compat audit) | Run on macOS system bash 3.2 |
| 9: validate-package.sh content | Phase 2 (package hardening) | Test with empty/corrupted files in REQUIRED_FILES |
| 10: missing -- separator | Phase 1 (escaping audit) | Test with patterns starting with `-` |
| 11: changelog JSON concatenation | Phase 1 (JSON audit) | Test with phase names containing `'` |
| 12: duplicated depth logic | Phase 1 (escaping audit) | Test both implementations return same result |
| 13: jq -Rs trailing newline | Phase 1 (JSON audit) | Test round-trip of content with/without trailing newline |

---

## macOS bash 3.2 Specific Warnings

The PROJECT.md constraint says "bash 4+" but the CLAUDE.md and multiple code comments reference macOS bash 3.2 compatibility. The emoji-audit.sh (line 12) explicitly states "Compatible with bash 3.x (macOS system bash)." The actual constraint should be verified, but assume bash 3.2 compatibility is required.

| Feature | bash 3.2 | bash 4+ | Impact on v2.6 |
|---------|----------|---------|----------------|
| Associative arrays | Not supported | Supported | Not used in v2.6 scope |
| `${var^^}` uppercase | Not supported | Supported | Not used; use `tr` instead |
| `mapfile`/`readarray` | Not supported | Supported | Not used; use `while read` loop |
| `((var++))` empty var | Returns 1, empty var | Returns 1, empty var | Both fail; always initialize |
| `$((var + 1))` empty var | Evaluates to 0 | Evaluates to 0 | Safe, prefer this form |
| `=~` regex in `[[ ]]` | Supported (with quirks) | Supported | Works for simple patterns |
| `printf -v` | Supported | Supported | Safe to use |
| Process substitution `<()` | Supported | Supported | Used extensively, works fine |
| `local -a` (local arrays) | Supported | Supported | Safe to use |
| Array `+=` append | Not supported | Supported | Not used in v2.6; use `arr+=("${new}")` works in 3.2 for single elements |

---

## Sources

- HIGH confidence: Direct codebase inspection of `.aether/aether-utils.sh` (5,200+ lines), `.aether/utils/spawn.sh`, `.aether/utils/spawn-tree.sh`, `.aether/utils/hive.sh`, `.aether/utils/learning.sh`, `.aether/utils/queen.sh`, `.aether/utils/file-lock.sh`, `bin/validate-package.sh`
- HIGH confidence: `.planning/PROJECT.md` v2.6 scope definition (lines 43-55)
- HIGH confidence: Existing PITFALLS.md for per-caste model routing (cross-reference for LOCK_DIR patterns)
- HIGH confidence: QUEEN.md instinct "Hub-level shared files need hub-level locks" (verified in multiple chamber archives)
- MEDIUM confidence: Web search results on grep BRE/ERE escaping patterns (training data, no live sources found)
- MEDIUM confidence: Web search results on jq --arg vs string interpolation (training data, no live sources found)
- MEDIUM confidence: macOS bash 3.2 vs bash 4 compatibility (training data, well-established topic)
- LOW confidence: No live web sources found for current-year updates on these topics (rate limiting). Findings are based on well-established shell scripting knowledge that has not fundamentally changed.

---
*Pitfalls research for: Aether v2.6 Bugfix & Hardening*
*Researched: 2026-03-29*
