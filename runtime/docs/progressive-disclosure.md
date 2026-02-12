# Progressive Disclosure Output Format Standard

This specification defines the standard output format for all Aether CLI commands, enabling compact summaries with optional detailed expansion.

## Core Principles

1. **Default to compact**: Single-line summaries by default
2. **Show counts**: Use bracketed counts to indicate expandable detail
3. **Consistent triggers**: `--verbose` or `-v` expands all sections

## Format Specification

### Summary Line Format

```
<command>: <primary-metric> | <secondary-metrics...> [<count> <type>]
```

**Examples:**
```
status: clean | 3 staged, 1 modified [4 changes]
build: success | 2.3s [12 warnings]
plan: ready | 5 tasks [3 blocked]
```

### Count Format

Counts indicate expandable sections. Format:

```
[N <plural-noun>]
```

**Rules:**
- Always use brackets `[]`
- Number first, then descriptive noun
- Use plural form even for count of 1 (consistency)
- Noun should describe what expands, not the data type

**Examples:**
```
[3 TODOs]        # Expands to show 3 TODO items
[1 errors]       # Expands to show error details
[5 files]        # Expands to show file list
[12 warnings]    # Expands to show warning messages
```

### Expansion Trigger

| Flag | Effect |
|------|--------|
| (none) | Compact single-line output |
| `-v` / `--verbose` | Expand all bracketed counts |
| `--verbose=<section>` | Expand specific section only |

### Multi-Section Output

When multiple expandable sections exist, list them inline:

```
status: clean | [3 TODOs] [2 FIXMEs] [1 errors]
```

With `--verbose`, each section expands under a header:

```
status: clean | [3 TODOs] [2 FIXMEs] [1 errors]

TODOs:
  - src/main.rs:42: implement caching
  - src/lib.rs:15: add error handling
  - tests/mod.rs:8: add integration test

FIXMEs:
  - src/parser.rs:23: handle edge case
  - src/parser.rs:67: optimize loop

Errors:
  - build failed in module 'auth'
```

## Command-Specific Formats

### status

```
# Compact
status: clean | 3 staged, 1 modified [4 changes]

# Verbose
status: clean | 3 staged, 1 modified [4 changes]

Changes:
  M  src/main.rs
  A  src/new_file.rs
  A  src/another.rs
  M  README.md
```

### build

```
# Compact
build: success | 2.3s [12 warnings]

# Verbose
build: success | 2.3s [12 warnings]

Warnings:
  src/lib.rs:45: unused variable 'x'
  src/lib.rs:67: deprecated function call
  ...
```

### plan

```
# Compact
plan: ready | 5 tasks, 2 blocked [3 pending]

# Verbose
plan: ready | 5 tasks, 2 blocked [3 pending]

Pending:
  - [ ] Task 1.1: Setup infrastructure
  - [ ] Task 1.2: Configure dependencies
  - [ ] Task 2.1: Implement core logic
```

### todos

```
# Compact
todos: [3 TODOs] [2 FIXMEs] [1 HACK]

# Verbose
todos: [3 TODOs] [2 FIXMEs] [1 HACK]

TODOs:
  src/main.rs:42: implement caching
  src/lib.rs:15: add error handling
  tests/mod.rs:8: add integration test

FIXMEs:
  src/parser.rs:23: handle edge case
  src/parser.rs:67: optimize loop

HACK:
  src/temp.rs:5: temporary workaround for API bug
```

## Implementation Notes

### For Command Authors

1. Always output the summary line first
2. Detect `--verbose` flag before formatting
3. Use consistent indentation (2 spaces) for expanded content
4. Include file:line references where applicable
5. Truncate long lists with `...` and count (e.g., `... and 15 more`)

### Integration

Reference this spec in command implementations:

```rust
// See .aether/docs/progressive-disclosure.md for output format
```

```python
# Output format: .aether/docs/progressive-disclosure.md
```

## Color Guidelines (Terminal)

When outputting to a TTY:
- Counts in brackets: dim/gray
- Success states: green
- Warning counts: yellow
- Error counts: red
- File paths: cyan
- Line numbers: dim

When piped or redirected: no color codes.
