# Phase 4: CLI Improvements - Research

**Researched:** 2026-02-13
**Domain:** Node.js CLI tooling (commander.js, picocolors)
**Confidence:** HIGH

## Summary

This research covers migrating the Aether CLI from manual argument parsing to commander.js with colored output via picocolors. The current CLI (`bin/cli.js`) uses a simple `switch` statement on `process.argv[2]` with manual flag parsing. The migration will provide:

1. **Structured command definitions** with declarative API instead of manual parsing
2. **Automatic help generation** with consistent formatting
3. **Type-safe argument parsing** with validation
4. **Colored output** via picocolors (14x smaller, 2x faster than chalk)
5. **Backward compatibility** through deprecation warnings for old syntax

**Primary recommendation:** Use commander.js's fluent API with `.command()` for each subcommand, implement a custom help formatter to show CLI/slash command mappings, and use picocolors with `--no-color` flag support via `NO_COLOR` environment variable.

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| commander | ^12.x | CLI framework | Most popular Node.js CLI framework, declarative API, auto-help |
| picocolors | ^1.x | Terminal colors | 7kB vs 101kB (chalk), NO_COLOR friendly, fastest loading |

### Installation
```bash
npm install commander picocolors
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| commander | yargs | yargs has more features but larger bundle; commander is simpler and sufficient |
| commander | minimist + manual | minimist only parses args, no help generation or command structure |
| picocolors | chalk | chalk has better nesting API (`chalk.red.bold()`) but 14x larger; picocolors nesting is acceptable |
| picocolors | ansi-colors | Similar size to picocolors, but picocolors is more actively maintained |

## Architecture Patterns

### Recommended Project Structure
```
bin/
├── cli.js              # Entry point - minimal, delegates to commands
├── commands/           # Command implementations (if splitting)
│   ├── install.js
│   ├── update.js
│   ├── version.js
│   └── uninstall.js
└── lib/
    ├── colors.js       # Centralized color palette wrapper
    ├── errors.js       # Already exists - keep using
    └── logger.js       # Already exists - integrate with colors
```

### Pattern 1: Flat Command Structure (Recommended)
**What:** All commands defined at top level with `.command()`
**When to use:** Current Aether CLI has flat structure (install, update, version, uninstall, help)
**Example:**
```javascript
// Source: Commander.js README + Aether requirements
const { program } = require('commander');
const pc = require('picocolors');

program
  .name('aether')
  .description('Aether Colony - Multi-agent system using ant colony intelligence')
  .version(VERSION);

// Flat commands (no subcommand grouping)
program
  .command('install')
  .description('Install slash-commands and set up distribution hub')
  .option('-q, --quiet', 'suppress output')
  .action(wrapCommand(async (options) => {
    // Implementation
  }));

program
  .command('update')
  .description('Update current repo from hub')
  .option('-f, --force', 'stash dirty files and force update')
  .option('-a, --all', 'update all registered repos')
  .option('-l, --list', 'show registered repos and versions')
  .option('-n, --dry-run', 'preview changes without modifying files')
  .action(wrapCommand(async (options) => {
    // Implementation
  }));

program
  .command('version')
  .description('Show installed version')
  .action(() => {
    console.log(`aether-colony v${VERSION}`);
  });

program
  .command('uninstall')
  .description('Remove slash-commands (preserves project state and hub)')
  .action(wrapCommand(async () => {
    // Implementation
  }));

program.parse();
```

### Pattern 2: Custom Help with CLI/Slash Command Mapping
**What:** Override help to show mapping between CLI commands and slash commands
**When to use:** Required by CLI-03 and context decisions
**Example:**
```javascript
// Source: Commander.js README help customization
const { Command } = require('commander');

class AetherHelp extends Command {
  helpInformation() {
    const pc = require('picocolors');

    return `
${pc.cyan('aether-colony')} v${VERSION}

${pc.bold('CLI Commands:')}
  install              Install slash-commands and set up distribution hub
  update [options]     Update current repo from hub
  version              Show installed version
  uninstall            Remove slash-commands

${pc.bold('Slash Commands (Claude Code / OpenCode):')}
  /ant:init            Initialize colony → maps to: aether init (deprecated)
  /ant:status          Show colony status
  /ant:plan            Generate project plan
  /ant:build <n>       Build phase N

${pc.dim('Use --no-color to disable colored output')}
`;
  }
}

const program = new AetherHelp();
```

### Pattern 3: Color Palette Wrapper
**What:** Centralized color definitions for consistent theming
**When to use:** Required by context decisions ("Define a palette")
**Example:**
```javascript
// bin/lib/colors.js
const pc = require('picocolors');

// Check for --no-color or NO_COLOR env
const colorEnabled = !process.argv.includes('--no-color') && !process.env.NO_COLOR;

// Create colors instance with explicit enabled state
const colors = colorEnabled ? pc : pc.createColors(false);

module.exports = {
  // Primary palette
  header: colors.cyan,
  success: colors.green,
  warning: colors.yellow,
  error: colors.red,
  info: colors.blue,
  dim: colors.dim,
  bold: colors.bold,

  // Semantic aliases
  primary: colors.cyan,
  secondary: colors.magenta,
  accent: colors.yellow,

  // Utility
  isEnabled: () => colorEnabled
};
```

### Pattern 4: Backward Compatibility with Deprecation Warnings
**What:** Support old syntax but warn users
**When to use:** Required by context decisions
**Example:**
```javascript
// Pattern for deprecated commands
program
  .command('init <goal>')
  .description('(deprecated) Use /ant:init instead')
  .action((goal) => {
    const pc = require('picocolors');
    console.warn(pc.yellow('Warning: "aether init" is deprecated.'));
    console.warn(pc.yellow('  Use /ant:init in Claude Code instead:'));
    console.warn(pc.yellow(`  /ant:init "${goal}"`));
    process.exit(1);
  });

// Pattern for deprecated flags
program
  .command('update')
  .option('--dryrun', '(deprecated) Use --dry-run instead')
  .action((options) => {
    if (options.dryrun) {
      console.warn(pc.yellow('Warning: --dryrun is deprecated. Use --dry-run instead.'));
      options.dryRun = true;
    }
    // Continue with normal logic
  });
```

### Pattern 5: Global Error Handling Integration
**What:** Preserve existing error handling while using commander
**When to use:** Aether has structured error classes in bin/lib/errors.js
**Example:**
```javascript
// Source: Current Aether CLI + Commander.js patterns
const { wrapCommand } = require('./lib/errors');

// Wrap all action handlers
program
  .command('install')
  .action(wrapCommand(async (options) => {
    // Errors thrown here are caught by wrapCommand
    // and output as structured JSON to stderr
  }));

// Override commander's default error behavior
program.configureOutput({
  writeErr: (str) => {
    // Route through Aether's error system
    const error = new ValidationError(str.trim());
    console.error(JSON.stringify(error.toJSON(), null, 2));
  }
});

program.exitOverride((err) => {
  // Handle commander exit codes consistently
  if (err.code === 'commander.help') {
    process.exit(0);
  }
  if (err.code === 'commander.version') {
    process.exit(0);
  }
  process.exit(1);
});
```

### Anti-Patterns to Avoid
- **Don't use `.addCommand()` for simple cases:** Use `.command()` for flat structure as it's more readable
- **Don't parse args manually alongside commander:** Let commander handle all argument parsing
- **Don't call `program.parse()` multiple times:** Call it once at the end
- **Don't use nested subcommands:** Keep flat structure per context decisions (`aether update --all` not `aether repo update --all`)

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Argument parsing | Manual `process.argv` slicing | commander.js | Handles `--flag value`, `--flag=value`, `-abc` combined short flags, type coercion |
| Help text generation | Manual template strings | commander.js `.help()` | Auto-generates from command definitions, handles alignment, supports custom sections |
| Version flag | Manual `if (args[0] === '--version')` | commander.js `.version()` | Handles `-V`, `-v`, `--version` automatically |
| Color detection | Manual `process.env.NO_COLOR` check | picocolors built-in | picocolors respects `NO_COLOR` automatically; use `createColors()` for explicit control |
| Option validation | Manual type checking | commander.js `.option('--port <number>', 'port', parseInt)` | Built-in coercion and validation |
| Required arguments | Manual `if (!arg)` checks | commander.js `.argument('<required>')` | Automatic error messages for missing args |

**Key insight:** Manual argument parsing seems simple but edge cases (quoted args, combined flags, `--` separator) are error-prone. Commander handles these correctly.

## Common Pitfalls

### Pitfall 1: Commander's Strict Option Parsing
**What goes wrong:** Commander throws error for unknown options by default
**Why it happens:** Commander is strict about unrecognized options
**How to avoid:** Use `.allowUnknownOption()` only if needed, or define all options explicitly
**Warning signs:** `error: unknown option '--unknown-flag'`

### Pitfall 2: Option Name Transformation
**What goes wrong:** `--no-color` becomes `options.color` (not `options.noColor`)
**Why it happens:** Commander strips `--no-` prefix for negatable booleans
**How to avoid:** Access as `options.color` (true by default, false when `--no-color` passed)
**Example:**
```javascript
program.option('--no-color', 'disable colors');
// In action handler:
console.log(options.color); // true normally, false with --no-color
```

### Pitfall 3: Async Action Handlers
**What goes wrong:** Unhandled promise rejections in async actions
**Why it happens:** Commander doesn't automatically catch async errors
**How to avoid:** Always wrap async actions with error handler:
```javascript
.action(async (options) => {
  try {
    await doSomething();
  } catch (err) {
    // Handle or re-throw for global handler
  }
})
// Or use wrapCommand utility
```

### Pitfall 4: Global Options with Subcommands
**What goes wrong:** Global options not accessible in subcommands
**Why it happens:** Options are scoped to command by default
**How to avoid:** Use `.optsWithGlobals()` to access parent options, or store in program:
```javascript
program.option('--verbose');
program.on('option:verbose', () => {
  program.verbose = true;
});
```

### Pitfall 5: Picocolors Nesting
**What goes wrong:** `pc.red.bold('text')` doesn't work (unlike chalk)
**Why it happens:** Picocolors uses nested calls, not chained methods
**How to avoid:** Use `pc.red(pc.bold('text'))` instead
**Migration from chalk:**
```javascript
// Chalk style (doesn't work with picocolors)
chalk.red.bold('text')

// Picocolors style
pc.red(pc.bold('text'))
```

### Pitfall 6: Help Override Timing
**What goes wrong:** Custom help not showing when using `.helpInformation()`
**Why it happens:** Must override before calling `.parse()`
**How to avoid:** Define custom help class or override methods before `.parse()`

### Pitfall 7: Exit Code Handling
**What goes wrong:** Commander calls `process.exit()` which bypasses cleanup
**Why it happens:** Default behavior for --help, --version, errors
**How to avoid:** Use `.exitOverride()` for testing or custom exit handling:
```javascript
program.exitOverride();
try {
  program.parse();
} catch (err) {
  if (err.code === 'commander.help') {
    // Custom help handling
  }
}
```

## Code Examples

### Complete CLI Setup Pattern
```javascript
#!/usr/bin/env node
// bin/cli.js - Complete migration example

const { program } = require('commander');
const pc = require('picocolors');
const VERSION = require('../package.json').version;

// Import existing Aether utilities
const { wrapCommand } = require('./lib/errors');

// Color palette (respects --no-color and NO_COLOR)
const c = {
  header: pc.cyan,
  success: pc.green,
  warning: pc.yellow,
  error: pc.red,
  dim: pc.dim,
  bold: pc.bold
};

program
  .name('aether')
  .description('Aether Colony - Multi-agent system using ant colony intelligence')
  .version(VERSION, '-v, --version', 'show version')
  .option('--no-color', 'disable colored output')
  .helpOption('-h, --help', 'show help');

// Global option handling for --no-color
program.on('option:no-color', () => {
  // Re-export colors with disabled state if needed
  process.env.NO_COLOR = '1';
});

// Commands
program
  .command('install')
  .description('Install slash-commands and set up distribution hub')
  .option('-q, --quiet', 'suppress output')
  .action(wrapCommand(async (options) => {
    console.log(c.header(`aether-colony v${VERSION}`) + ' — installing...');
    // ... install logic
    console.log(c.success('Install complete.'));
  }));

program
  .command('update')
  .description('Update current repo from hub')
  .option('-f, --force', 'stash dirty files and force update')
  .option('-a, --all', 'update all registered repos')
  .option('-l, --list', 'show registered repos and versions')
  .option('-n, --dry-run', 'preview changes without modifying files')
  .action(wrapCommand(async (options) => {
    if (options.list) {
      // List logic
    } else if (options.all) {
      // Update all logic
    } else {
      // Update current repo logic
    }
  }));

program
  .command('version')
  .description('Show installed version')
  .action(() => {
    console.log(`aether-colony v${VERSION}`);
  });

program
  .command('uninstall')
  .description('Remove slash-commands (preserves project state and hub)')
  .action(wrapCommand(async () => {
    console.log(c.header(`aether-colony v${VERSION}`) + ' — uninstalling...');
    // ... uninstall logic
    console.log(c.success('Uninstall complete.'));
  }));

// Custom help with CLI/Slash command mapping
program.on('--help', () => {
  console.log('');
  console.log(c.bold('Slash Commands (Claude Code):'));
  console.log('  /ant:init       Initialize colony');
  console.log('  /ant:status     Show colony status');
  console.log('  /ant:plan       Generate project plan');
  console.log('');
  console.log(c.dim('Locations:'));
  console.log(c.dim('  Commands: ~/.claude/commands/ant/'));
  console.log(c.dim('  Hub: ~/.aether/'));
});

program.parse();
```

### Deprecation Warning Pattern
```javascript
// For deprecated commands - show warning and redirect
function deprecatedCommand(oldName, newName, isSlashCommand = false) {
  return () => {
    console.error(pc.yellow(`Warning: "aether ${oldName}" is deprecated.`));
    if (isSlashCommand) {
      console.error(pc.yellow(`  Use ${newName} in Claude Code instead.`));
    } else {
      console.error(pc.yellow(`  Use "aether ${newName}" instead.`));
    }
    process.exit(1);
  };
}

program
  .command('init')
  .description('(deprecated) Use /ant:init instead')
  .action(deprecatedCommand('init', '/ant:init', true));
```

### Colored Output with Conditional Support
```javascript
// bin/lib/colors.js - Full implementation
const pc = require('picocolors');

// Detect color support
function isColorEnabled() {
  // Check explicit --no-color flag
  if (process.argv.includes('--no-color')) return false;
  // Check environment variable
  if (process.env.NO_COLOR) return false;
  // Check if stdout is TTY
  if (!process.stdout.isTTY) return false;
  return true;
}

const enabled = isColorEnabled();
const p = enabled ? pc : pc.createColors(false);

// Aether brand palette
module.exports = {
  // Primary
  queen: p.magenta,      // Queen ant - magenta
  colony: p.cyan,        // Colony - cyan
  worker: p.yellow,      // Workers - yellow

  // Semantic
  success: p.green,
  warning: p.yellow,
  error: p.red,
  info: p.blue,

  // Text styles
  bold: p.bold,
  dim: p.dim,
  italic: p.italic,
  underline: p.underline,

  // Headers
  header: (text) => p.bold(p.cyan(text)),
  subheader: (text) => p.bold(text),

  // Utility
  isEnabled: () => enabled,

  // For raw picocolors access
  raw: p
};
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Manual argv parsing | commander.js | 2025 (this phase) | Declarative, auto-help, validation |
| chalk for colors | picocolors | 2025 (this phase) | 14x smaller, 2x faster, NO_COLOR friendly |
| Custom help strings | commander.js help | 2025 (this phase) | Consistent formatting, less code |
| process.exit() everywhere | wrapCommand + structured errors | Already implemented | Better error handling, testable |

**Deprecated/outdated:**
- Manual `process.argv` slicing: Use commander.js instead
- Chalk for simple CLIs: Use picocolors for smaller footprint
- Custom argument validators: Use commander's built-in validation

## Open Questions

1. **Help Text Customization Depth**
   - What we know: Commander supports custom help via `.helpInformation()` override
   - What's unclear: Exact format for two-column CLI/slash command alignment
   - Recommendation: Implement custom help class extending Command

2. **Backward Compatibility Scope**
   - What we know: Context specifies deprecation warnings for "old syntax"
   - What's unclear: Which exact old syntax patterns need support
   - Recommendation: Review current CLI usage analytics if available; otherwise support common patterns like `--dryrun` → `--dry-run`

3. **Color Palette Specifics**
   - What we know: Context says "Define a palette" and "base on Aether brand"
   - What's unclear: Exact brand colors (magenta for Queen, cyan for Colony are suggested)
   - Recommendation: Use semantic naming (success, warning, error, queen, colony) rather than raw colors

4. **Quiet Mode Integration**
   - What we know: Current CLI has `--quiet` flag
   - What's unclear: Whether quiet mode should also disable colors
   - Recommendation: `--quiet` suppresses output; `--no-color` disables colors; they can be combined

## Sources

### Primary (HIGH confidence)
- Commander.js README: https://github.com/tj/commander.js/blob/master/Readme.md
  - Topics: Basic setup, subcommands, options, help customization, exit override
- Picocolors README: https://github.com/alexeyraspopov/picocolors
  - Topics: API, NO_COLOR support, createColors(), comparison with chalk

### Secondary (MEDIUM confidence)
- Current Aether CLI implementation: `/Users/callumcowie/repos/Aether/bin/cli.js`
  - Topics: Existing command structure, error handling patterns, flags
- Aether context decisions: `/Users/callumcowie/repos/Aether/.planning/phases/04-cli-improvements/04-CONTEXT.md`
  - Topics: Flat command structure, kebab-case, color requirements, backward compatibility

### Tertiary (LOW confidence)
- None - all findings verified with primary sources

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Verified with official READMEs
- Architecture: HIGH - Based on commander.js documented patterns
- Pitfalls: MEDIUM-HIGH - From README + common community knowledge

**Research date:** 2026-02-13
**Valid until:** 2026-05-13 (90 days for stable libraries)

**Migration complexity estimate:** LOW-MEDIUM
- Current CLI has only 5 commands (install, update, version, uninstall, help)
- Existing error handling (`wrapCommand`) integrates cleanly
- No breaking changes required (backward compatibility maintained)
