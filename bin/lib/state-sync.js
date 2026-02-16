#!/usr/bin/env node
/**
 * State Synchronization Module
 *
 * Fixes the "split brain" between .planning/STATE.md and COLONY_STATE.json.
 * Synchronizes planning state with runtime state to ensure consistency.
 *
 * @module bin/lib/state-sync
 */

const fs = require('fs');
const path = require('path');
const { FileLock } = require('./file-lock');

/**
 * Parse STATE.md markdown content to extract current state
 * @param {string} content - STATE.md file content
 * @returns {object} Parsed state: { phase, milestone, status, lastAction }
 */
function parseStateMd(content) {
  const result = {
    phase: null,
    milestone: null,
    status: null,
    lastAction: null
  };

  if (!content || typeof content !== 'string') {
    return result;
  }

  // Extract phase from "Phase X" or "Phase: X"
  const phaseMatch = content.match(/Phase\s*(\d+)/i);
  if (phaseMatch) {
    result.phase = parseInt(phaseMatch[1], 10);
  }

  // Extract milestone from "Current Milestone:** ..." or "Milestone:** ..."
  const milestoneMatch = content.match(/(?:Current\s*)?Milestone:\*\*?\s*([^\n]+)/i);
  if (milestoneMatch) {
    result.milestone = milestoneMatch[1].trim();
  }

  // Extract status from "Status:** ..." or similar
  const statusMatch = content.match(/Status:\*\*?\s*([^\n]+)/i);
  if (statusMatch) {
    result.status = statusMatch[1].trim();
  }

  // Extract last action from "Last Action:** ..." or similar
  const lastActionMatch = content.match(/Last\s*Action:\*\*?\s*([^\n]+)/i);
  if (lastActionMatch) {
    result.lastAction = lastActionMatch[1].trim();
  }

  return result;
}

/**
 * Parse ROADMAP.md to extract phases
 * @param {string} content - ROADMAP.md file content
 * @returns {Array} Array of phase objects: { number, name, status }
 */
function parseRoadmapMd(content) {
  const phases = [];

  if (!content || typeof content !== 'string') {
    return phases;
  }

  // Match phase headers like "## Phase 1: Name" or "### Phase 1: Name"
  const phaseRegex = /#{2,3}\s*Phase\s*(\d+)[:\s]+([^\n]+)/gi;
  let match;

  while ((match = phaseRegex.exec(content)) !== null) {
    const phaseNum = parseInt(match[1], 10);
    const phaseName = match[2].trim();

    // Warn on unreasonable phase numbers (PLAN-006 fix #8)
    if (phaseNum > 100 || phaseNum < 0) {
      console.warn(`Warning: Unusual phase number ${phaseNum} in ROADMAP.md`);
    }

    // Look for status indicators near this phase
    const sectionStart = match.index;
    const nextPhaseMatch = phaseRegex.exec(content);
    phaseRegex.lastIndex = sectionStart + 1; // Reset for next iteration

    const sectionEnd = nextPhaseMatch ? nextPhaseMatch.index : content.length;
    const section = content.substring(sectionStart, sectionEnd);

    // Determine status from section content
    let status = 'planned';
    if (section.includes('Status: complete') || section.includes('COMPLETE')) {
      status = 'completed';
    } else if (section.includes('Status: in progress') || section.includes('IN PROGRESS')) {
      status = 'in_progress';
    } else if (section.includes('Status: ready') || section.includes('READY')) {
      status = 'ready';
    }

    phases.push({
      number: phaseNum,
      name: phaseName,
      status
    });
  }

  return phases;
}

/**
 * Determine colony state based on planning status
 * @param {string} status - Status from STATE.md
 * @param {number} phase - Current phase number
 * @returns {string} Colony state: INITIALIZING|PLANNING|BUILDING|COMPLETED
 */
function determineColonyState(status, phase) {
  // No status - determine by phase (PLAN-006 fix #7)
  if (!status) {
    if (phase === null || phase === undefined) {
      return 'INITIALIZING';
    }
    // Phase 0 can be valid - don't force INITIALIZING
    return phase === 0 ? 'PLANNING' : 'BUILDING';
  }

  const statusLower = status.toLowerCase();

  if (statusLower.includes('complete') && !statusLower.includes('ready')) {
    return 'COMPLETED';
  }

  if (statusLower.includes('plan') || statusLower.includes('ready')) {
    return 'PLANNING';
  }

  if (statusLower.includes('build') || statusLower.includes('progress') || statusLower.includes('in progress')) {
    return 'BUILDING';
  }

  // Default based on phase - Phase 0 is PLANNING, not INITIALIZING
  return phase === 0 ? 'PLANNING' : 'BUILDING';
}

/**
 * Synchronize COLONY_STATE.json with .planning/STATE.md
 * @param {string} repoPath - Path to repository root
 * @returns {object} Sync result: { synced: boolean, updates: string[], error?: string, recovery?: string }
 */
function syncStateFromPlanning(repoPath) {
  const updates = [];
  const lockDir = path.join(repoPath, '.aether', 'locks');
  const colonyStatePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');

  // Create lock instance
  const lock = new FileLock({
    lockDir,
    timeout: 5000,
    maxRetries: 50
  });

  // Attempt to acquire lock
  if (!lock.acquire(colonyStatePath)) {
    return {
      synced: false,
      updates: [],
      error: 'Could not acquire state lock - another sync in progress'
    };
  }

  try {
    // Read STATE.md with improved error handling (PLAN-006 fix #9)
    const stateMdPath = path.join(repoPath, '.planning', 'STATE.md');
    try {
      if (!fs.existsSync(stateMdPath)) {
        return { synced: false, updates: [], error: '.planning/STATE.md not found' };
      }
    } catch (accessError) {
      if (accessError.code === 'EACCES') {
        return { synced: false, updates: [], error: '.planning/STATE.md not accessible (permission denied)' };
      }
      return { synced: false, updates: [], error: `Failed to check STATE.md: ${accessError.message}` };
    }

    let stateMdContent;
    try {
      stateMdContent = fs.readFileSync(stateMdPath, 'utf8');
    } catch (readError) {
      if (readError.code === 'EACCES') {
        return { synced: false, updates: [], error: '.planning/STATE.md not readable (permission denied)' };
      }
      return { synced: false, updates: [], error: `Failed to read STATE.md: ${readError.message}` };
    }
    const planningState = parseStateMd(stateMdContent);

    // Read ROADMAP.md for phases
    const roadmapPath = path.join(repoPath, '.planning', 'ROADMAP.md');
    let phases = [];
    if (fs.existsSync(roadmapPath)) {
      try {
        const roadmapContent = fs.readFileSync(roadmapPath, 'utf8');
        phases = parseRoadmapMd(roadmapContent);
      } catch (roadmapError) {
        // Non-fatal - phases are optional
        console.warn(`Warning: Could not read ROADMAP.md: ${roadmapError.message}`);
      }
    }

    // Read current COLONY_STATE.json with improved error handling
    try {
      if (!fs.existsSync(colonyStatePath)) {
        return { synced: false, updates: [], error: '.aether/data/COLONY_STATE.json not found' };
      }
    } catch (accessError) {
      if (accessError.code === 'EACCES') {
        return { synced: false, updates: [], error: 'COLONY_STATE.json not accessible (permission denied)' };
      }
      return { synced: false, updates: [], error: `Failed to check COLONY_STATE.json: ${accessError.message}` };
    }

    // Parse JSON with error handling (PLAN-002)
    let colonyState;
    try {
      colonyState = JSON.parse(fs.readFileSync(colonyStatePath, 'utf8'));
    } catch (parseError) {
      return {
        synced: false,
        updates: [],
        error: `COLONY_STATE.json contains invalid JSON: ${parseError.message}`,
        recovery: 'Manually fix or delete .aether/data/COLONY_STATE.json and reinitialize'
      };
    }

    // Track if any changes were made
    let changed = false;

    // Update goal from milestone
    if (planningState.milestone && planningState.milestone !== colonyState.goal) {
      colonyState.goal = planningState.milestone;
      updates.push(`goal: ${colonyState.goal}`);
      changed = true;
    }

    // Update current_phase
    if (planningState.phase !== null && planningState.phase !== colonyState.current_phase) {
      colonyState.current_phase = planningState.phase;
      updates.push(`current_phase: ${colonyState.current_phase}`);
      changed = true;
    }

    // Update state based on status
    const newState = determineColonyState(planningState.status, planningState.phase);
    if (newState !== colonyState.state) {
      colonyState.state = newState;
      updates.push(`state: ${colonyState.state}`);
      changed = true;
    }

    // Update plan.phases from ROADMAP
    if (phases.length > 0) {
      const existingPhases = colonyState.plan?.phases || [];
      if (JSON.stringify(phases) !== JSON.stringify(existingPhases)) {
        if (!colonyState.plan) {
          colonyState.plan = {};
        }
        colonyState.plan.phases = phases;
        updates.push(`plan.phases: ${phases.length} phases`);
        changed = true;
      }
    }

    // Add sync event if changed
    if (changed) {
      if (!colonyState.events) {
        colonyState.events = [];
      }

      colonyState.events.push({
        timestamp: new Date().toISOString(),
        type: 'state_synced_from_planning',
        worker: 'state-sync',
        details: {
          updates: updates,
          source: '.planning/STATE.md'
        }
      });

      // Update last_updated
      colonyState.last_updated = new Date().toISOString();

      // Atomic write: write to temp file, then rename (PLAN-002)
      const tempPath = `${colonyStatePath}.tmp`;
      fs.writeFileSync(tempPath, JSON.stringify(colonyState, null, 2) + '\n');
      fs.renameSync(tempPath, colonyStatePath);
    }

    return {
      synced: true,
      updates,
      changed
    };

  } catch (error) {
    return {
      synced: false,
      updates: [],
      error: error.message
    };
  } finally {
    // Always release lock
    lock.release();
  }
}

/**
 * Reconcile STATE.md and COLONY_STATE.json to detect mismatches
 * @param {string} repoPath - Path to repository root
 * @returns {object} Reconciliation result: { consistent: boolean, mismatches: string[], resolution: string }
 */
function reconcileStates(repoPath) {
  const mismatches = [];

  try {
    // Read STATE.md
    const stateMdPath = path.join(repoPath, '.planning', 'STATE.md');
    if (!fs.existsSync(stateMdPath)) {
      return {
        consistent: false,
        mismatches: ['STATE.md not found'],
        resolution: 'Create .planning/STATE.md'
      };
    }

    const stateMdContent = fs.readFileSync(stateMdPath, 'utf8');
    const planningState = parseStateMd(stateMdContent);

    // Read COLONY_STATE.json
    const colonyStatePath = path.join(repoPath, '.aether', 'data', 'COLONY_STATE.json');
    if (!fs.existsSync(colonyStatePath)) {
      return {
        consistent: false,
        mismatches: ['COLONY_STATE.json not found'],
        resolution: 'Run: aether init'
      };
    }

    // Parse JSON with error handling
    let colonyState;
    try {
      colonyState = JSON.parse(fs.readFileSync(colonyStatePath, 'utf8'));
    } catch (parseError) {
      return {
        consistent: false,
        mismatches: [`COLONY_STATE.json contains invalid JSON: ${parseError.message}`],
        resolution: 'Manually fix or delete .aether/data/COLONY_STATE.json and reinitialize'
      };
    }

    // Check phase mismatch
    if (planningState.phase !== null && planningState.phase !== colonyState.current_phase) {
      mismatches.push(`Phase mismatch: STATE.md says ${planningState.phase}, COLONY_STATE.json says ${colonyState.current_phase}`);
    }

    // Check goal/milestone mismatch
    if (planningState.milestone && planningState.milestone !== colonyState.goal) {
      mismatches.push(`Goal mismatch: STATE.md milestone differs from COLONY_STATE.json goal`);
    }

    // Check status contradiction
    const expectedState = determineColonyState(planningState.status, planningState.phase);
    if (expectedState !== colonyState.state) {
      mismatches.push(`Status contradiction: STATE.md implies ${expectedState}, COLONY_STATE.json is ${colonyState.state}`);
    }

    return {
      consistent: mismatches.length === 0,
      mismatches,
      resolution: mismatches.length > 0
        ? 'Run: aether sync-state'
        : 'No action needed'
    };

  } catch (error) {
    return {
      consistent: false,
      mismatches: [`Error during reconciliation: ${error.message}`],
      resolution: 'Check file permissions and try again'
    };
  }
}

/**
 * Full sync: read STATE.md and update COLONY_STATE.json
 * This is a convenience wrapper around syncStateFromPlanning
 * @param {string} repoPath - Path to repository root
 * @returns {object} Sync result
 */
function updateColonyStateFromPlanning(repoPath) {
  return syncStateFromPlanning(repoPath);
}

module.exports = {
  parseStateMd,
  parseRoadmapMd,
  determineColonyState,
  syncStateFromPlanning,
  reconcileStates,
  updateColonyStateFromPlanning
};
