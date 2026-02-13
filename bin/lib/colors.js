#!/usr/bin/env node
/**
 * Aether Color Palette Module
 *
 * Centralized color definitions for consistent CLI theming.
 * Uses picocolors for lightweight, NO_COLOR-friendly terminal colors.
 *
 * Aether Brand Colors:
 * - queen: magenta (Queen ant)
 * - colony: cyan (Colony/nest)
 * - worker: yellow (Workers)
 * - success: green
 * - warning: yellow
 * - error: red
 * - info: blue
 */

const pc = require('picocolors');

/**
 * Detect if color output is enabled
 * Checks --no-color flag and NO_COLOR environment variable
 * @returns {boolean} True if colors should be enabled
 */
function isColorEnabled() {
  // Check explicit --no-color flag
  if (process.argv.includes('--no-color')) {
    return false;
  }
  // Check environment variable (NO_COLOR set to any non-empty value disables colors)
  if (process.env.NO_COLOR && process.env.NO_COLOR !== '') {
    return false;
  }
  // Check if stdout is TTY (disable colors when piped)
  if (!process.stdout.isTTY) {
    return false;
  }
  return true;
}

const enabled = isColorEnabled();
const p = enabled ? pc : pc.createColors(false);

/**
 * Aether brand color palette
 * Semantic naming based on ant colony hierarchy and message types
 */
module.exports = {
  // Aether brand - ant colony hierarchy
  queen: p.magenta,      // Queen ant - magenta
  colony: p.cyan,        // Colony/nest - cyan
  worker: p.yellow,      // Workers - yellow

  // Semantic message colors
  success: p.green,
  warning: p.yellow,
  error: p.red,
  info: p.blue,

  // Text styles
  bold: p.bold,
  dim: p.dim,
  italic: p.italic,
  underline: p.underline,
  strikethrough: p.strikethrough,

  // Headers (combined styles)
  header: (text) => p.bold(p.cyan(text)),
  subheader: (text) => p.bold(text),

  // Utility
  isEnabled: () => enabled,

  // Raw picocolors access for advanced usage
  raw: p
};
