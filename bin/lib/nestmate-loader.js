const fs = require('fs');
const path = require('path');

/**
 * Find nestmate colonies (sibling directories with .aether/)
 * @param {string} currentRepoPath - Current repository path
 * @returns {Array<{name: string, path: string, goal: string|null}>} Nestmate info
 */
function findNestmates(currentRepoPath) {
  const parentDir = path.dirname(currentRepoPath);
  const currentName = path.basename(currentRepoPath);

  try {
    const entries = fs.readdirSync(parentDir, { withFileTypes: true });
    const nestmates = [];

    for (const entry of entries) {
      if (!entry.isDirectory()) continue;
      if (entry.name === currentName) continue;
      if (entry.name.startsWith('.')) continue;

      const siblingPath = path.join(parentDir, entry.name);
      const aetherPath = path.join(siblingPath, '.aether');

      if (fs.existsSync(aetherPath)) {
        // Try to read colony goal from state
        let goal = null;
        try {
          const statePath = path.join(aetherPath, 'data', 'COLONY_STATE.json');
          if (fs.existsSync(statePath)) {
            const state = JSON.parse(fs.readFileSync(statePath, 'utf8'));
            goal = state.goal || null;
          }
        } catch (e) {
          // Ignore read errors
        }

        nestmates.push({
          name: entry.name,
          path: siblingPath,
          goal: goal
        });
      }
    }

    return nestmates;
  } catch (error) {
    return [];
  }
}

/**
 * Load TO-DOs from a nestmate
 * @param {string} nestmatePath - Path to nestmate repository
 * @returns {Array<{file: string, todos: string[]}>} TO-DO items
 */
function loadNestmateTodos(nestmatePath) {
  const todos = [];
  const planningPath = path.join(nestmatePath, '.planning');

  try {
    // Look for TODO files in .planning/
    if (fs.existsSync(planningPath)) {
      const entries = fs.readdirSync(planningPath);
      for (const entry of entries) {
        if (entry.toLowerCase().includes('todo')) {
          const todoPath = path.join(planningPath, entry);
          const content = fs.readFileSync(todoPath, 'utf8');
          // Extract TODO items (lines starting with - [ ] or TODO:)
          const items = content.split('\n')
            .filter(line => line.match(/^\s*-\s*\[\s*\]|TODO:/i))
            .map(line => line.trim());

          if (items.length > 0) {
            todos.push({ file: entry, items });
          }
        }
      }
    }
  } catch (error) {
    // Ignore errors
  }

  return todos;
}

/**
 * Get colony state summary from a nestmate
 * @param {string} nestmatePath - Path to nestmate repository
 * @returns {Object|null} State summary
 */
function getNestmateState(nestmatePath) {
  try {
    const statePath = path.join(nestmatePath, '.aether', 'data', 'COLONY_STATE.json');
    if (!fs.existsSync(statePath)) return null;

    const state = JSON.parse(fs.readFileSync(statePath, 'utf8'));
    return {
      goal: state.goal,
      state: state.state,
      currentPhase: state.current_phase,
      milestone: state.milestone
    };
  } catch (error) {
    return null;
  }
}

/**
 * Format nestmate info for display
 * @param {Array} nestmates - Nestmate array from findNestmates
 * @returns {string} Formatted string
 */
function formatNestmates(nestmates) {
  if (nestmates.length === 0) {
    return 'No nestmates found.';
  }

  return nestmates.map(n => {
    const goal = n.goal ? ` - ${n.goal.substring(0, 40)}${n.goal.length > 40 ? '...' : ''}` : '';
    return `  • ${n.name}${goal}`;
  }).join('\n');
}

module.exports = {
  findNestmates,
  loadNestmateTodos,
  getNestmateState,
  formatNestmates
};
