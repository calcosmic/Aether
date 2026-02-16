# Aether Implementation Plan

## Executive Summary

Aether is a multi-agent CLI framework with significant technical debt and untapped potential. This plan provides a wave-based roadmap to transform Aether from its current state to production-ready status over 12 implementation waves.

**Current State Snapshot:**
- 3,592-line core utility file with known critical bugs
- 34 Claude Code commands + 33 OpenCode commands (13,573 lines duplicated)
- 22 worker castes with model routing configured but unverified
- 5 XSD schemas (sophisticated but dormant XML system)
- Session freshness detection recently completed (21/21 tests passing)

**Key Challenges:**
1. **Critical Bugs:** Lock deadlock (BUG-005/011), error code inconsistency (BUG-007), template path hardcoding (ISSUE-004)
2. **Code Duplication:** 13K lines manually mirrored between Claude and OpenCode
3. **Dormant Systems:** XML infrastructure exists but isn't integrated into production commands
4. **Documentation Debt:** 1,152+ markdown files with significant overlap and stale content

**Target State:**
- Zero critical bugs, consistent error handling
- Single-source-of-truth command generation (YAML-based)
- Active XML system for cross-colony memory
- Consolidated, current documentation
- Verified model routing and spawn discipline

---

## Wave Overview Table

| Wave | Theme | Tasks | Est. Effort | Dependencies | Status |
|------|-------|-------|-------------|--------------|--------|
| W1 | Foundation Fixes (Critical Bugs) | 4 | 2 days | None | Ready |
| W2 | Error Handling Standardization | 3 | 2 days | W1 | Ready |
| W3 | Template Path & queen-init Fix | 2 | 1 day | W1 | Ready |
| W4 | Command Consolidation Infrastructure | 4 | 5 days | W2 | Ready |
| W5 | XML System Activation (Phase 1) | 4 | 4 days | W4 | Ready |
| W6 | XML System Integration (Phase 2) | 3 | 4 days | W5 | Ready |
| W7 | Testing Expansion | 4 | 5 days | W1-W3 | Ready |
| W8 | Model Routing Verification | 2 | 2 days | W7 | Ready |
| W9 | Documentation Consolidation | 3 | 4 days | W4 | Ready |
| W10 | Colony Lifecycle Management | 3 | 4 days | W1, W5 | Ready |
| W11 | Performance & Hardening | 3 | 3 days | W7, W8 | Ready |
| W12 | Production Readiness | 3 | 3 days | All | Ready |

**Total Estimated Effort:** 39 days (approximately 8 weeks with parallel work)

---

## Detailed Wave Breakdown

---

### Wave 1: Foundation Fixes (Critical Bugs)

**Wave Goal:** Eliminate all critical bugs that could cause data loss or system deadlock.

---

#### W1-T1: Fix Lock Deadlock in flag-auto-resolve

**Task ID:** W1-T1

**Description:**
The flag-auto-resolve command has a critical lock leak. When jq fails during flag resolution, the lock acquired at line 1364 is never released because json_err exits without releasing it. This causes a deadlock where subsequent flag operations hang indefinitely.

The fix requires wrapping jq operations in error handlers that release the lock before calling json_err.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (lines 1367-1390)

**Files to Create:**
- None

**Dependencies:**
- None

**Effort:** Small (2-4 hours)

**Priority:** P0 (Critical)

**Success Criteria:**
1. jq failure during flag-auto-resolve releases lock before exiting
2. Subsequent flag operations succeed after jq failure
3. No regression in normal flag resolution path

**Verification Steps:**
```bash
# Test 1: Simulate jq failure and verify lock release
bash .aether/aether-utils.sh flag-auto-resolve "build_pass"
# Verify: No hanging, returns error JSON with lock released

# Test 2: Verify normal operation still works
bash .aether/aether-utils.sh flag-add "test" "Test flag" --auto-resolve-on="build_pass"
bash .aether/aether-utils.sh flag-auto-resolve "build_pass"
# Verify: Returns {"resolved":1,...}

# Test 3: Verify lock file is not left behind
ls .aether/data/locks/
# Verify: No stale lock files
```

**Risk Assessment:**
- **Risk:** Fix could introduce new error handling bugs
- **Mitigation:** Comprehensive test coverage before and after fix
- **Impact:** High - affects all flag operations

**Rollback Plan:**
```bash
# Revert to previous version
git checkout HEAD -- .aether/aether-utils.sh
```

---

#### W1-T2: Fix Error Code Inconsistency (BUG-007)

**Task ID:** W1-T2

**Description:**
17+ locations in aether-utils.sh use hardcoded error strings instead of the E_* constants defined in error-handler.sh. This inconsistency makes error handling fragile and prevents proper recovery suggestion mapping.

The fix requires auditing all json_err calls and replacing hardcoded strings with proper E_* constants.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (17+ locations)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-error-codes.sh` (regression test)

**Dependencies:**
- None

**Effort:** Medium (1 day)

**Priority:** P0 (Critical)

**Success Criteria:**
1. All json_err calls use E_* constants
2. No hardcoded error strings in error paths
3. Recovery suggestions work for all error types
4. Regression test prevents future inconsistency

**Verification Steps:**
```bash
# Test 1: Verify no hardcoded error strings
grep -n 'json_err "' .aether/aether-utils.sh | grep -v 'json_err "\$E_'
# Verify: Only legitimate non-error calls remain

# Test 2: Run regression test
bash tests/bash/test-error-codes.sh
# Verify: All tests pass

# Test 3: Verify recovery suggestions work
bash .aether/aether-utils.sh flag-add 2>&1 | jq '.error.recovery'
# Verify: Recovery suggestion is present
```

**Risk Assessment:**
- **Risk:** Mass find/replace could introduce typos
- **Mitigation:** Review each change individually, run full test suite
- **Impact:** Medium - affects error message consistency

**Rollback Plan:**
```bash
# Revert changes
git checkout HEAD -- .aether/aether-utils.sh
```

---

#### W1-T3: Fix Lock Deadlock in flag-add (BUG-002)

**Task ID:** W1-T3

**Description:**
Similar to W1-T1, the flag-add command has a lock leak in its error path. If jq fails during flag addition, the lock is not released before json_err exits.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (flag-add section around line 814)

**Files to Create:**
- None

**Dependencies:**
- W1-T1 (same fix pattern)

**Effort:** Small (1-2 hours)

**Priority:** P0 (Critical)

**Success Criteria:**
1. jq failure during flag-add releases lock before exiting
2. Lock file cleanup happens in all error paths

**Verification Steps:**
```bash
# Test: Verify lock release on error
bash .aether/aether-utils.sh flag-add "test" "Test" 2>&1
# Verify: Error returned, no lock file left
```

**Risk Assessment:**
- **Risk:** Low - same pattern as W1-T1
- **Mitigation:** Apply same fix pattern

**Rollback Plan:**
```bash
git checkout HEAD -- .aether/aether-utils.sh
```

---

#### W1-T4: Fix atomic-write Lock Leak (BUG-006)

**Task ID:** W1-T4

**Description:**
The atomic-write.sh utility has a lock leak on JSON validation failure at line 66. If the JSON is invalid, the lock is not released.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/utils/atomic-write.sh`

**Files to Create:**
- None

**Dependencies:**
- W1-T1, W1-T3

**Effort:** Small (1 hour)

**Priority:** P0 (Critical)

**Success Criteria:**
1. JSON validation failure releases lock
2. All error paths in atomic-write release locks

**Verification Steps:**
```bash
# Test: Write invalid JSON
bash -c 'source .aether/utils/atomic-write.sh; atomic_write "test.json" "invalid json"'
# Verify: Error returned, no lock file
```

**Rollback Plan:**
```bash
git checkout HEAD -- .aether/utils/atomic-write.sh
```

---

### Wave 2: Error Handling Standardization

**Wave Goal:** Establish consistent error handling patterns across all utilities.

---

#### W2-T1: Add Missing Error Code Constants

**Task ID:** W2-T1

**Description:**
Add error code constants for common error cases that currently use generic E_UNKNOWN:
- E_PERMISSION_DENIED (file permission issues)
- E_TIMEOUT (operation timeout)
- E_CONFLICT (concurrent modification)
- E_INVALID_STATE (colony state issues)

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/utils/error-handler.sh`

**Files to Create:**
- None

**Dependencies:**
- W1-T2

**Effort:** Small (2 hours)

**Priority:** P1 (High)

**Success Criteria:**
1. All new error codes have recovery suggestions
2. Error codes follow naming convention
3. Documentation updated

**Verification Steps:**
```bash
# Verify all constants are exported
grep -E '^E_' .aether/utils/error-handler.sh | wc -l
# Should show count >= 14
```

---

#### W2-T2: Standardize Error Handler Usage

**Task ID:** W2-T2

**Description:**
Ensure all utility scripts consistently use error-handler.sh. Some scripts may have fallback json_err that doesn't match the enhanced signature.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (fallback json_err at lines 66-73)
- `/Users/callumcowie/repos/Aether/.aether/utils/xml-utils.sh` (xml_json_err)

**Files to Create:**
- None

**Dependencies:**
- W2-T1

**Effort:** Medium (1 day)

**Priority:** P1 (High)

**Success Criteria:**
1. All json_err calls use 4-parameter signature
2. Fallback implementations removed
3. Consistent error format across all utilities

**Verification Steps:**
```bash
# Verify consistent error format
bash .aether/aether-utils.sh invalid-command 2>&1 | jq '.error | keys'
# Should show: ["code", "message", "details", "recovery", "timestamp"]
```

---

#### W2-T3: Add Error Context Enrichment

**Task ID:** W2-T3

**Description:**
Enhance error messages with context about what operation was being performed. Add operation name and relevant file paths to error details.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (major commands)
- `/Users/callumcowie/repos/Aether/.aether/utils/error-handler.sh`

**Files to Create:**
- None

**Dependencies:**
- W2-T2

**Effort:** Medium (1 day)

**Priority:** P1 (High)

**Success Criteria:**
1. Error details include operation context
2. File paths in errors are relative to project root
3. Stack trace available in debug mode

---

### Wave 3: Template Path & queen-init Fix

**Wave Goal:** Fix ISSUE-004 where queen-init fails when Aether is installed via npm.

---

#### W3-T1: Fix Template Path Resolution

**Task ID:** W3-T1

**Description:**
The queen-init command checks for templates in runtime/ first, which doesn't exist in npm installs. It should check .aether/ first (source of truth) and fall back to ~/.aether/system/.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (lines 2680-2705)

**Files to Create:**
- None

**Dependencies:**
- None

**Effort:** Small (2 hours)

**Priority:** P0 (High)

**Success Criteria:**
1. queen-init works with npm-installed Aether
2. Template resolution order: .aether/ > ~/.aether/system/ > runtime/
3. Clear error message if template not found

**Verification Steps:**
```bash
# Test npm install scenario
npm install -g .
mkdir /tmp/test-queen && cd /tmp/test-queen
bash ~/.aether/system/aether-utils.sh queen-init
# Verify: QUEEN.md created successfully
```

**Risk Assessment:**
- **Risk:** Could break git clone workflow
- **Mitigation:** Test both npm and git workflows

**Rollback Plan:**
```bash
git checkout HEAD -- .aether/aether-utils.sh
```

---

#### W3-T2: Add Template Validation

**Task ID:** W3-T2

**Description:**
Add validation that templates are complete and valid before using them. Check for required placeholders and valid markdown structure.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
- `/Users/callumcowie/repos/Aether/.aether/templates/QUEEN.md.template`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-template-validation.sh`

**Dependencies:**
- W3-T1

**Effort:** Small (1 day)

**Priority:** P1 (Medium)

**Success Criteria:**
1. Templates validated before use
2. Clear error if template is corrupted
3. Tests for template validation

---

### Wave 4: Command Consolidation Infrastructure

**Wave Goal:** Eliminate 13K lines of duplication between Claude and OpenCode commands.

---

#### W4-T1: Design YAML Command Schema

**Task ID:** W4-T1

**Description:**
Design a YAML schema for command definitions that can generate both Claude and OpenCode formats. The schema should capture:
- Command metadata (name, description, version)
- Parameters and arguments
- Tool mappings (Claude vs OpenCode tool names)
- Prompt template
- Execution steps

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/src/commands/_meta/template.yaml` (enhance existing)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/src/commands/_meta/schema.json` (YAML schema validation)
- `/Users/callumcowie/repos/Aether/docs/COMMAND-YAML-SCHEMA.md`

**Dependencies:**
- W2-T3 (error handling patterns established)

**Effort:** Large (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. YAML schema supports all 22 commands
2. Schema validation passes for all command definitions
3. Documentation complete

**Verification Steps:**
```bash
# Validate schema
node -e "const schema = require('./src/commands/_meta/schema.json'); console.log('Valid')"
```

---

#### W4-T2: Create Command Generator Script

**Task ID:** W4-T2

**Description:**
Build the generate-commands.sh script that reads YAML definitions and generates both Claude and OpenCode command files. Support:
- Full generation (all commands)
- Single command generation
- Dry-run mode
- Diff mode (show what would change)

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/bin/generate-commands.sh` (enhance existing)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/src/commands/definitions/` (YAML files for each command)
- `/Users/callumcowie/repos/Aether/tests/bash/test-command-generator.sh`

**Dependencies:**
- W4-T1

**Effort:** Large (3 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Generator produces identical output to current manual files
2. All 22 commands generate successfully
3. CI check passes
4. Generator handles tool mapping correctly

**Verification Steps:**
```bash
# Generate all commands
./bin/generate-commands.sh

# Verify no diff with current files
diff .claude/commands/ant/build.md <(./bin/generate-commands.sh --command build --platform claude)
# Should produce no output (identical)
```

**Risk Assessment:**
- **Risk:** Generator bugs could break commands
- **Mitigation:** Extensive testing, gradual rollout

---

#### W4-T3: Migrate Commands to YAML

**Task ID:** W4-T3

**Description:**
Convert all 22 command definitions from markdown to YAML. Start with simple commands (status, help) before complex ones (build, oracle).

**Files to Modify:**
- Create YAML definitions in `/Users/callumcowie/repos/Aether/src/commands/definitions/`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/src/commands/definitions/*.yaml` (22 files)

**Dependencies:**
- W4-T2

**Effort:** Large (3 days)

**Priority:** P1 (High)

**Success Criteria:**
1. All 22 commands have YAML definitions
2. Generated files match current manual files
3. Zero diff when comparing generated vs manual

**Verification Steps:**
```bash
# Generate and compare
./bin/generate-commands.sh --verify
# Should output: "All commands match"
```

---

#### W4-T4: Add CI Check for Command Sync

**Task ID:** W4-T4

**Description:**
Add a CI check that verifies generated commands match YAML source. Fail the build if they're out of sync.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.github/workflows/ci.yml`
- `/Users/callumcowie/repos/Aether/package.json` (lint:sync script)

**Files to Create:**
- None

**Dependencies:**
- W4-T3

**Effort:** Small (4 hours)

**Priority:** P1 (High)

**Success Criteria:**
1. CI fails if commands are out of sync
2. Clear error message showing how to fix
3. lint:sync script works locally

**Verification Steps:**
```bash
npm run lint:sync
# Should pass
```

---

### Wave 5: XML System Activation (Phase 1)

**Wave Goal:** Activate the dormant XML system for cross-colony memory.

---

#### W5-T1: Integrate xml-utils into aether-utils.sh

**Task ID:** W5-T1

**Description:**
The xml-utils.sh exists but isn't fully integrated. Add subcommands to aether-utils.sh for XML operations:
- xml-validate: Validate XML against XSD
- xml-query: XPath queries
- xml-export: Export colony data to XML
- xml-import: Import XML data

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (add xml-* subcommands)

**Files to Create:**
- None

**Dependencies:**
- W4-T4 (command infrastructure ready)

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. All xml-* commands available via aether-utils.sh
2. Commands return JSON like other utilities
3. Graceful degradation if XML tools not installed

**Verification Steps:**
```bash
bash .aether/aether-utils.sh xml-validate .aether/schemas/pheromone.xsd
# Should validate successfully
```

---

#### W5-T2: Create Pheromone XML Export

**Task ID:** W5-T2

**Description:**
Implement pheromone export from JSON to XML format. Export should include:
- All active pheromones
- Colony namespace attribution
- Timestamp and metadata
- Validation against pheromone.xsd

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-pheromone-xml.sh`

**Dependencies:**
- W5-T1

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. pheromones.json exports to valid XML
2. XML validates against pheromone.xsd
3. Namespace correctly identifies source colony
4. Export is idempotent

**Verification Steps:**
```bash
bash .aether/aether-utils.sh pheromone-export
# Creates: .aether/data/pheromones.xml

xmllint --schema .aether/schemas/pheromone.xsd .aether/data/pheromones.xml
# Should validate successfully
```

---

#### W5-T3: Implement Cross-Colony Pheromone Merge

**Task ID:** W5-T3

**Description:**
Implement merging of pheromone XML files from multiple colonies using XML namespaces to prevent collisions.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/utils/xml-utils.sh`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-xml-merge.sh`

**Dependencies:**
- W5-T2

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Can merge pheromones from multiple colonies
2. Namespaces prevent ID collisions
3. Original source colony tracked
4. Merge is associative and commutative

---

#### W5-T4: Add XML Documentation

**Task ID:** W5-T4

**Description:**
Document the XML system for users and developers. Include examples of pheromone XML, validation, and cross-colony sharing.

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.aether/docs/XML-SYSTEM.md`
- `/Users/callumcowie/repos/Aether/.aether/docs/examples/pheromone-example.xml`

**Dependencies:**
- W5-T3

**Effort:** Small (1 day)

**Priority:** P2 (Medium)

**Success Criteria:**
1. Documentation explains when to use XML vs JSON
2. Examples for all XML operations
3. Schema reference complete

---

### Wave 6: XML System Integration (Phase 2)

**Wave Goal:** Integrate XML system into production commands.

---

#### W6-T1: Add XML Export to seal Command

**Task ID:** W6-T1

**Description:**
When a colony is sealed, export pheromones to XML for eternal storage. Archive XML alongside other colony artifacts.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/seal.md`
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/seal.md`

**Files to Create:**
- None

**Dependencies:**
- W5-T2

**Effort:** Small (1 day)

**Priority:** P1 (High)

**Success Criteria:**
1. seal command exports pheromones.xml
2. XML archived in chamber
3. Export happens automatically

---

#### W6-T2: Add XML Import to init Command

**Task ID:** W6-T2

**Description:**
When initializing a colony, offer to import pheromones from sealed colonies. Use XML merge to combine signals.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/init.md`
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/init.md`

**Files to Create:**
- None

**Dependencies:**
- W6-T1

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. init command can import from sealed colonies
2. Imported pheromones merged correctly
3. User can select which colonies to import from

---

#### W6-T3: Implement QUEEN.md XML Backend

**Task ID:** W6-T3

**Description:**
Create XML backend for QUEEN.md with XSLT transformation to markdown. queen-read should query XML, queen-init should create XML structure.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (queen-* commands)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.aether/utils/queen-to-md.xsl`
- `/Users/callumcowie/repos/Aether/tests/bash/test-queen-xml.sh`

**Dependencies:**
- W5-T4

**Effort:** Large (3 days)

**Priority:** P2 (Medium)

**Success Criteria:**
1. queen-wisdom.xml stores structured wisdom
2. XSLT generates readable QUEEN.md
3. queen-read queries XML directly
4. Promotion thresholds enforced by schema

---

### Wave 7: Testing Expansion

**Wave Goal:** Fill test coverage gaps and fix failing tests.

---

#### W7-T1: Audit Current Test Coverage

**Task ID:** W7-T1

**Description:**
Audit all existing tests to understand what they test and identify gaps. Document which utilities have tests and which don't.

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/COVERAGE-AUDIT.md`

**Dependencies:**
- None

**Effort:** Medium (1 day)

**Priority:** P1 (High)

**Success Criteria:**
1. All existing tests catalogued
2. Coverage gaps identified
3. Priority order for new tests established

---

#### W7-T2: Add Unit Tests for Bug Fixes

**Task ID:** W7-T2

**Description:**
Add regression tests for all bugs fixed in Wave 1. Ensure bugs cannot reoccur.

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-w1-regressions.sh`

**Dependencies:**
- W1 (all bug fixes)

**Effort:** Medium (2 days)

**Priority:** P0 (Critical)

**Success Criteria:**
1. Tests for BUG-005/011 lock deadlock
2. Tests for BUG-007 error codes
3. Tests for ISSUE-004 template path
4. All tests pass

---

#### W7-T3: Add Integration Tests for Commands

**Task ID:** W7-T3

**Description:**
Add integration tests for major commands (init, plan, build, continue, seal). Test full colony lifecycle.

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/integration/colony-lifecycle.test.js`

**Dependencies:**
- W7-T2

**Effort:** Large (3 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Full colony lifecycle tested
2. Tests run in isolated temp directories
3. Tests clean up after themselves
4. CI integration

---

#### W7-T4: Fix Failing Tests

**Task ID:** W7-T4

**Description:**
Identify and fix any currently failing tests. Ensure 100% test pass rate before production.

**Files to Modify:**
- Various test files as needed

**Files to Create:**
- None

**Dependencies:**
- W7-T3

**Effort:** Medium (2 days)

**Priority:** P0 (Critical)

**Success Criteria:**
1. npm test passes 100%
2. All bash tests pass
3. No skipped or pending tests

**Verification Steps:**
```bash
npm test
# Should show: all tests passing

bash tests/bash/test-aether-utils.sh
# Should show: all tests passing
```

---

### Wave 8: Model Routing Verification

**Wave Goal:** Verify and fix model routing for caste-based worker assignment.

---

#### W8-T1: Fix Model Routing Implementation

**Task ID:** W8-T1

**Description:**
The model routing configuration exists but environment variable inheritance is unverified. Fix the spawn-with-model.sh script to ensure ANTHROPIC_MODEL is properly passed to spawned workers.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/utils/spawn-with-model.sh`
- `/Users/callumcowie/repos/Aether/bin/lib/model-profiles.js`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-model-routing.sh`

**Dependencies:**
- W7 (testing infrastructure)

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Builder caste uses kimi-k2.5
2. Oracle caste uses minimax-2.5
3. Prime caste uses glm-5
4. Verification test passes

**Verification Steps:**
```bash
# Run verification
/ant:verify-castes
# Step 3 should show: ANTHROPIC_MODEL=kimi-k2.5 for builder
```

---

#### W8-T2: Add Interactive Caste Configuration

**Task ID:** W8-T2

**Description:**
Implement the interactive caste model configuration command. Allow users to view and modify caste-to-model assignments within Claude Code.

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/verify-castes.md` (enhance)
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/verify-castes.md` (enhance)

**Files to Create:**
- None

**Dependencies:**
- W8-T1

**Effort:** Medium (2 days)

**Priority:** P2 (Medium)

**Success Criteria:**
1. Interactive prompts for caste selection
2. Model selection with multiple choice
3. Confirmation before applying
4. Verification after change

---

### Wave 9: Documentation Consolidation

**Wave Goal:** Consolidate 1,152 markdown files and archive stale docs.

---

#### W9-T1: Audit Documentation

**Task ID:** W9-T1

**Description:**
Audit all documentation files to identify:
- Duplicate content
- Stale/outdated information
- Files that should be archived
- Missing documentation

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/docs/DOCUMENTATION-AUDIT.md`

**Dependencies:**
- None

**Effort:** Medium (1 day)

**Priority:** P2 (Medium)

**Success Criteria:**
1. All docs catalogued by purpose
2. Duplicates identified
3. Stale docs flagged for archive
4. Gaps documented

---

#### W9-T2: Consolidate Core Documentation

**Task ID:** W9-T2

**Description:**
Consolidate core documentation into single source of truth:
- Merge duplicate README files
- Consolidate pheromone documentation
- Merge architecture docs
- Create documentation index

**Files to Modify:**
- Various docs in `.aether/docs/`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.aether/docs/INDEX.md`

**Dependencies:**
- W9-T1

**Effort:** Medium (2 days)

**Priority:** P2 (Medium)

**Success Criteria:**
1. No duplicate core documentation
2. INDEX.md provides navigation
3. All docs have clear purpose
4. Stale docs moved to archive

---

#### W9-T3: Archive Stale Documentation

**Task ID:** W9-T3

**Description:**
Move stale and outdated documentation to `.aether/docs/archive/`. Add README explaining archive status.

**Files to Modify:**
- None (moves only)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.aether/docs/archive/README.md`

**Dependencies:**
- W9-T2

**Effort:** Small (1 day)

**Priority:** P3 (Low)

**Success Criteria:**
1. Stale docs moved to archive
2. Archive README explains status
3. Main docs directory contains current docs only
4. No broken links

---

### Wave 10: Colony Lifecycle Management

**Wave Goal:** Implement colony lifecycle management (archive, seal, history).

---

#### W10-T1: Implement Archive Command

**Task ID:** W10-T1

**Description:**
Implement `/ant:archive` command that archives current colony state and resets for new work. Archive includes:
- Completion report
- Final pheromone export (XML)
- Colony state snapshot
- Activity log summary

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/seal.md` (enhance)
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/seal.md` (enhance)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/archive.md`
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/archive.md`

**Dependencies:**
- W5-T2 (pheromone XML export)
- W1 (bug fixes for reliable operation)

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Archive command creates complete colony snapshot
2. COLONY_STATE.json reset after archive
3. Can init new colony after archive
4. Archive browsable via history command

---

#### W10-T2: Implement History Command

**Task ID:** W10-T2

**Description:**
Implement `/ant:history` command to browse archived colonies. Show summary of each archived colony with goal, completion status, and key metrics.

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/history.md`
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/history.md`

**Dependencies:**
- W10-T1

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Lists all archived colonies
2. Shows goal, date, outcome for each
3. Can view details of specific archive
4. Can restore pheromones from archive

---

#### W10-T3: Implement Milestone Auto-Detection

**Task ID:** W10-T3

**Description:**
Implement automatic milestone detection based on colony state:
- First Mound: Phase 1 complete
- Brood Stable: All tests passing
- Ventilated Nest: Build + lint clean
- Sealed Chambers: All phases complete
- Crowned Anthill: User confirms release-ready

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (add milestone-detect)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/status.md`

**Files to Create:**
- None

**Dependencies:**
- W10-T2

**Effort:** Small (1 day)

**Priority:** P2 (Medium)

**Success Criteria:**
1. Milestone auto-detected from state
2. Status command shows current milestone
3. Milestone transitions logged

---

### Wave 11: Performance & Hardening

**Wave Goal:** Optimize performance and harden against edge cases.

---

#### W11-T1: Optimize aether-utils.sh Loading

**Task ID:** W11-T1

**Description:**
The 3,592-line aether-utils.sh loads entirely for every command. Optimize by:
- Lazy-loading heavy functions
- Caching parsed JSON
- Reducing subshell usage

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/performance/benchmark-utils.sh`

**Dependencies:**
- None

**Effort:** Medium (2 days)

**Priority:** P2 (Medium)

**Success Criteria:**
1. Command execution time reduced by 30%
2. No functional changes
3. Benchmarks track performance

---

#### W11-T2: Add Spawn Limits Enforcement

**Task ID:** W11-T2

**Description:**
Enforce spawn discipline limits programmatically:
- Max spawn depth: 3
- Max spawns at depth 1: 4
- Max spawns at depth 2: 2
- Global workers per phase: 10

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` (spawn tracking)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/build.md` (enforce limits)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-spawn-limits.sh`

**Dependencies:**
- W7 (testing infrastructure)

**Effort:** Medium (2 days)

**Priority:** P1 (High)

**Success Criteria:**
1. Spawn limits enforced automatically
2. Clear error when limits exceeded
3. Tests verify enforcement

---

#### W11-T3: Add Graceful Degradation

**Task ID:** W11-T3

**Description:**
Enhance graceful degradation for missing dependencies:
- jq not installed: use fallback JSON parsing
- git not available: skip git integration
- XML tools missing: disable XML features

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh`
- `/Users/callumcowie/repos/Aether/.aether/utils/feature-detection.sh` (create)

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/bash/test-graceful-degradation.sh`

**Dependencies:**
- W11-T1

**Effort:** Small (1 day)

**Priority:** P2 (Medium)

**Success Criteria:**
1. System works with minimal dependencies
2. Clear warnings about disabled features
3. Core functionality always available

---

### Wave 12: Production Readiness

**Wave Goal:** Final validation and production deployment preparation.

---

#### W12-T1: End-to-End Testing

**Task ID:** W12-T1

**Description:**
Complete end-to-end testing of all workflows:
- Fresh install workflow
- Colony lifecycle (init -> plan -> build -> seal)
- Multi-repo update workflow
- Error recovery workflows

**Files to Modify:**
- None

**Files to Create:**
- `/Users/callumcowie/repos/Aether/tests/e2e/complete-workflow.sh`

**Dependencies:**
- All previous waves

**Effort:** Large (2 days)

**Priority:** P0 (Critical)

**Success Criteria:**
1. All workflows complete successfully
2. No manual intervention required
3. Error paths handled gracefully
4. Data integrity maintained

---

#### W12-T2: Security Audit

**Task ID:** W12-T2

**Description:**
Security audit of:
- File permissions
- Path traversal prevention
- Command injection prevention
- Secret handling

**Files to Modify:**
- Any files with security issues found

**Files to Create:**
- `/Users/callumcowie/repos/Aether/SECURITY-AUDIT.md`

**Dependencies:**
- All previous waves

**Effort:** Medium (1 day)

**Priority:** P0 (Critical)

**Success Criteria:**
1. No path traversal vulnerabilities
2. No command injection vectors
3. Secrets not logged
4. Audit report complete

---

#### W12-T3: Release Preparation

**Task ID:** W12-T3

**Description:**
Prepare for production release:
- Version bump
- Changelog update
- Release notes
- npm publish dry-run

**Files to Modify:**
- `/Users/callumcowie/repos/Aether/package.json`
- `/Users/callumcowie/repos/Aether/CHANGELOG.md`

**Files to Create:**
- `/Users/callumcowie/repos/Aether/RELEASE-NOTES.md`

**Dependencies:**
- W12-T1, W12-T2

**Effort:** Small (1 day)

**Priority:** P0 (Critical)

**Success Criteria:**
1. Version bumped to 1.1.0
2. Changelog complete
3. npm pack works
4. Release notes published

---

## Dependency Graph

```
W1 (Foundation Fixes)
├── W2 (Error Handling)
│   └── W4 (Command Consolidation)
│       ├── W5 (XML Activation)
│       │   ├── W6 (XML Integration)
│       │   └── W10 (Lifecycle)
│       └── W9 (Documentation)
├── W3 (Template Path)
├── W7 (Testing)
│   ├── W8 (Model Routing)
│   └── W11 (Performance)
└── W12 (Production)

W5 (XML Activation) ──> W10 (Lifecycle)
```

---

## Critical Path

The minimum sequence to production-ready status:

1. **W1-T1, W1-T2, W1-T3, W1-T4** - Fix critical bugs (4 days)
2. **W7-T2, W7-T4** - Regression tests for bugs (2 days)
3. **W12-T1** - End-to-end testing (2 days)
4. **W12-T2** - Security audit (1 day)
5. **W12-T3** - Release preparation (1 day)

**Minimum critical path: 10 days**

---

## Risk Analysis

### High-Risk Tasks

| Task | Risk | Mitigation |
|------|------|------------|
| W1-T1 (Lock Deadlock) | Could introduce new bugs | Extensive testing, small scope |
| W4-T2 (Command Generator) | Could break all commands | Parallel operation, gradual rollout |
| W8-T1 (Model Routing) | May require upstream changes | Fallback to default model |
| W12-T1 (E2E Testing) | May reveal major issues | Buffer time for fixes |

### Risk Mitigation Strategies

1. **Comprehensive Testing:** Every wave includes verification steps
2. **Rollback Plans:** Every task has rollback instructions
3. **Incremental Changes:** Large changes broken into smaller tasks
4. **Parallel Operation:** New systems run alongside old during transition

---

## Resource Requirements

### Skills Needed

| Skill | Waves | Level |
|-------|-------|-------|
| Bash/Shell | All | Expert |
| Node.js | W4, W7, W8 | Intermediate |
| XML/XSD | W5, W6 | Intermediate |
| YAML | W4 | Intermediate |
| Testing | W7, W12 | Expert |
| Security | W12 | Expert |

### Total Effort Estimate

- **Developer Days:** 39 days
- **Calendar Time:** 8 weeks (with parallel work)
- **Testing Time:** 8 days (included in waves)
- **Documentation Time:** 5 days (included in waves)

---

## Definition of Done

Aether is "operating perfectly" when:

### Functional Requirements
1. All 22 commands work identically in Claude and OpenCode
2. Zero critical bugs (no deadlocks, no data loss)
3. Model routing verified and working
4. XML system active for cross-colony memory
5. Colony lifecycle management complete

### Quality Requirements
1. 100% test pass rate
2. No known security vulnerabilities
3. Documentation current and complete
4. Performance within benchmarks

### Operational Requirements
1. Single-source-of-truth for commands (YAML)
2. CI/CD passing
3. Graceful degradation for missing dependencies
4. Clear error messages with recovery suggestions

### User Experience Requirements
1. Commands work out of the box
2. Clear progress indicators
3. Helpful error messages
4. Consistent behavior across platforms

---

## Appendix A: Task Summary Table

| Task ID | Title | Effort | Priority | Wave |
|---------|-------|--------|----------|------|
| W1-T1 | Fix Lock Deadlock in flag-auto-resolve | Small | P0 | W1 |
| W1-T2 | Fix Error Code Inconsistency | Medium | P0 | W1 |
| W1-T3 | Fix Lock Deadlock in flag-add | Small | P0 | W1 |
| W1-T4 | Fix atomic-write Lock Leak | Small | P0 | W1 |
| W2-T1 | Add Missing Error Code Constants | Small | P1 | W2 |
| W2-T2 | Standardize Error Handler Usage | Medium | P1 | W2 |
| W2-T3 | Add Error Context Enrichment | Medium | P1 | W2 |
| W3-T1 | Fix Template Path Resolution | Small | P0 | W3 |
| W3-T2 | Add Template Validation | Small | P1 | W3 |
| W4-T1 | Design YAML Command Schema | Large | P1 | W4 |
| W4-T2 | Create Command Generator Script | Large | P1 | W4 |
| W4-T3 | Migrate Commands to YAML | Large | P1 | W4 |
| W4-T4 | Add CI Check for Command Sync | Small | P1 | W4 |
| W5-T1 | Integrate xml-utils into aether-utils.sh | Medium | P1 | W5 |
| W5-T2 | Create Pheromone XML Export | Medium | P1 | W5 |
| W5-T3 | Implement Cross-Colony Pheromone Merge | Medium | P1 | W5 |
| W5-T4 | Add XML Documentation | Small | P2 | W5 |
| W6-T1 | Add XML Export to seal Command | Small | P1 | W6 |
| W6-T2 | Add XML Import to init Command | Medium | P1 | W6 |
| W6-T3 | Implement QUEEN.md XML Backend | Large | P2 | W6 |
| W7-T1 | Audit Current Test Coverage | Medium | P1 | W7 |
| W7-T2 | Add Unit Tests for Bug Fixes | Medium | P0 | W7 |
| W7-T3 | Add Integration Tests for Commands | Large | P1 | W7 |
| W7-T4 | Fix Failing Tests | Medium | P0 | W7 |
| W8-T1 | Fix Model Routing Implementation | Medium | P1 | W8 |
| W8-T2 | Add Interactive Caste Configuration | Medium | P2 | W8 |
| W9-T1 | Audit Documentation | Medium | P2 | W9 |
| W9-T2 | Consolidate Core Documentation | Medium | P2 | W9 |
| W9-T3 | Archive Stale Documentation | Small | P3 | W9 |
| W10-T1 | Implement Archive Command | Medium | P1 | W10 |
| W10-T2 | Implement History Command | Medium | P1 | W10 |
| W10-T3 | Implement Milestone Auto-Detection | Small | P2 | W10 |
| W11-T1 | Optimize aether-utils.sh Loading | Medium | P2 | W11 |
| W11-T2 | Add Spawn Limits Enforcement | Medium | P1 | W11 |
| W11-T3 | Add Graceful Degradation | Small | P2 | W11 |
| W12-T1 | End-to-End Testing | Large | P0 | W12 |
| W12-T2 | Security Audit | Medium | P0 | W12 |
| W12-T3 | Release Preparation | Small | P0 | W12 |

---

## Appendix B: File Paths Reference

### Core System Files
- `/Users/callumcowie/repos/Aether/.aether/aether-utils.sh` - Main utility layer (3,592 lines)
- `/Users/callumcowie/repos/Aether/.aether/utils/error-handler.sh` - Error handling
- `/Users/callumcowie/repos/Aether/.aether/utils/file-lock.sh` - File locking
- `/Users/callumcowie/repos/Aether/.aether/utils/atomic-write.sh` - Atomic writes
- `/Users/callumcowie/repos/Aether/.aether/utils/xml-utils.sh` - XML operations

### Command Files (34 Claude + 33 OpenCode)
- `/Users/callumcowie/repos/Aether/.claude/commands/ant/*.md`
- `/Users/callumcowie/repos/Aether/.opencode/commands/ant/*.md`

### Schema Files
- `/Users/callumcowie/repos/Aether/.aether/schemas/pheromone.xsd`
- `/Users/callumcowie/repos/Aether/.aether/schemas/queen-wisdom.xsd`
- `/Users/callumcowie/repos/Aether/.aether/schemas/colony-registry.xsd`
- `/Users/callumcowie/repos/Aether/.aether/schemas/worker-priming.xsd`
- `/Users/callumcowie/repos/Aether/.aether/schemas/prompt.xsd`

### Test Files
- `/Users/callumcowie/repos/Aether/tests/unit/*.test.js`
- `/Users/callumcowie/repos/Aether/tests/integration/*.test.js`
- `/Users/callumcowie/repos/Aether/tests/e2e/*.test.js`
- `/Users/callumcowie/repos/Aether/tests/bash/*.sh`

---

*Generated: 2026-02-16*
*Version: 1.0*
*Status: Ready for Implementation*
