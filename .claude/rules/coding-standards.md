# Coding Standards

## Shell Scripts

- Use `#!/bin/bash` shebang
- Enable strict mode: `set -euo pipefail`
- Quote all variables: `"$var"` not `$var`
- Use `[[ ]]` for tests, not `[ ]`
- Check for required tools before using: `command -v foo >/dev/null || exit 1`

## JavaScript

- Use `const` by default, `let` when reassignment needed
- Prefer `async/await` over raw promises
- Use template literals for string interpolation
- Handle errors with structured error classes (see `bin/lib/errors.js`)

## File Naming

- Shell scripts: `kebab-case.sh`
- JavaScript modules: `camelCase.js`
- Markdown docs: `kebab-case.md`

## Code Organization

- One function per logical unit
- Keep functions under 50 lines
- Extract helpers for repeated patterns
- Document public APIs with JSDoc-style comments
