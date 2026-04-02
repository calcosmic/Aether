#!/usr/bin/env node
/**
 * npx-entry.js — Entry point for `npx aether-colony`
 *
 * With no subcommand: launches interactive setup menu.
 * With a subcommand (e.g. `npx aether-colony install`): delegates to full CLI.
 */

const args = process.argv.slice(2);

// If a subcommand is provided (not a flag), delegate to the full CLI
if (args.length > 0 && !args[0].startsWith('-')) {
  const { run } = require('./cli.js');
  run();
} else {
  const { interactiveSetup } = require('./lib/interactive-setup');
  interactiveSetup().catch(err => {
    console.error('Setup failed:', err.message);
    process.exit(1);
  });
}
