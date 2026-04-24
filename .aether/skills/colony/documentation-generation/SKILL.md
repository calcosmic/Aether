---
name: documentation-generation
description: Use when project documentation must be generated or updated from verified codebase facts
type: colony
domains: [documentation, codebase, generation]
agent_roles: [builder, chronicler]
workflow_triggers: [build, continue]
task_keywords: [docs, documentation, readme, api docs, generate]
priority: normal
version: "1.0"
---

# Documentation Generation

## Purpose

Generates and updates project documentation by reading the actual codebase and producing docs that reflect reality -- not aspirational descriptions. The colony verifies every claim against the code, so documentation stays accurate even as the code evolves.

## When to Use

- User says "generate docs" or "update the README"
- Phase completion requires documentation updates
- API documentation is out of date
- Onboarding documentation needed for new team members
- User asks "is our documentation accurate?"

## Instructions

### 1. Document Types

```
README:        Project overview, setup, usage
API_DOCS:      Endpoint documentation with request/response examples
ARCHITECTURE:  System design and module descriptions
CHANGELOG:     Version history from git commits
CONTRIBUTING:  Development setup and contribution guidelines
INLINE:        Code comments and JSDoc/docstring generation
```

### 2. Generation Protocol

```
SCAN:
  1. Read existing documentation (if any)
  2. Scan relevant source code sections
  3. Compare docs to code -- identify gaps and inaccuracies
  4. Check codebase intelligence index for context

GENERATE:
  1. Draft documentation based on code analysis
  2. Include code examples extracted from actual usage
  3. Generate API documentation from route definitions
  4. Create diagrams from module dependency data

VERIFY:
  1. Cross-reference every claim against source code
  2. Verify all code examples compile/run
  3. Check all file paths and module names are current
  4. Ensure configuration examples match actual config schema

WRITE:
  1. Write documentation files
  2. Update existing docs where inaccurate
  3. Mark deprecated sections if code has moved on
  4. Generate verification report
```

### 3. Accuracy Verification

```
Every doc must pass these checks:
   All mentioned files exist
   All mentioned functions exist with matching signatures
   All code examples are syntactically valid
   All configuration values match actual defaults
   All API endpoints match route definitions
   No claims contradict the actual code behavior
```

### 4. Update Detection

```
When updating existing docs:
  1. Diff current code against doc last-updated timestamp
  2. Identify changed modules and APIs
  3. Update only affected documentation sections
  4. Preserve manual additions (sections marked with <!-- manual -->)
  5. Flag sections that need human review
```

### 5. Style Configuration

```yaml
# .aether/config.yml
docs:
  style: casual          # casual | professional | technical
  include_examples: true
  example_language: typescript
  verify_against_code: true
  preserve_manual: true
  output_dir: docs/
```

## Key Patterns

- **Code is truth**: Documentation must match the code, not the other way around.
- **Verify, don't trust**: Every factual claim in docs gets verified against source.
- **Incremental updates**: Update only what changed, don't regenerate everything.
- **Preserve human touches**: Manual sections are sacred -- never overwrite them.

## Output Format

```
 DOCS -- {document_type}
   Files: {list of generated/updated docs}
   Verified: {count}/{total} claims verified against code
   Gaps: {count} sections need manual review
   Changes: {new} new, {updated} updated, {deprecated} deprecated
```

## Examples

**Generate API docs:**
> "Generated API documentation for 47 endpoints. All request/response examples verified against route handlers. 3 endpoints had mismatched docs -- corrected. 2 new endpoints documented."

**Update README:**
> "README updated: setup instructions now match current package.json scripts. Added environment variable table (verified against config.ts). Removed outdated Docker instructions."
