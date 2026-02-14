#!/usr/bin/env node
/**
 * Caste Color Definitions Module
 *
 * Centralized caste styling with colors and emojis for consistent
 * theming across the Aether colony system.
 *
 * Caste Colors (ANSI + picocolors):
 * - builder:  blue   (🔨) - Construction, implementation
 * - watcher:  green  (👁️) - Observation, monitoring
 * - scout:    yellow (🔍) - Exploration, investigation
 * - chaos:    red    (🎲) - Testing, disruption
 * - prime:    magenta (👑) - Coordination, leadership
 */

const pc = require('picocolors');

// Caste definitions with colors and emojis (per CONTEXT.md decisions)
const CASTE_STYLES = {
  builder:  { color: 'blue',   emoji: '🔨', ansi: '\033[34m', pc: pc.blue },
  watcher:  { color: 'green',  emoji: '👁️',  ansi: '\033[32m', pc: pc.green },
  scout:    { color: 'yellow', emoji: '🔍', ansi: '\033[33m', pc: pc.yellow },
  chaos:    { color: 'red',    emoji: '🎲', ansi: '\033[31m', pc: pc.red },
  prime:    { color: 'magenta',emoji: '👑', ansi: '\033[35m', pc: pc.magenta }
};

// Get style for a caste (case-insensitive)
function getCasteStyle(caste) {
  const key = caste.toLowerCase();
  return CASTE_STYLES[key] || { color: 'reset', emoji: '🐜', ansi: '\033[0m', pc: (s) => s };
}

// Format ant name with color and emoji: "🔨 Builder" (both colored)
function formatAnt(name, caste) {
  const style = getCasteStyle(caste);
  return `${style.emoji} ${style.pc(name)}`;
}

// Format with ANSI codes (for bash scripts)
function formatAntAnsi(name, caste) {
  const style = getCasteStyle(caste);
  const reset = '\x1b[0m';
  return `${style.ansi}${style.emoji} ${name}${reset}`;
}

// Get all castes for iteration
function getCastes() {
  return Object.keys(CASTE_STYLES);
}

module.exports = {
  CASTE_STYLES,
  getCasteStyle,
  formatAnt,
  formatAntAnsi,
  getCastes
};
