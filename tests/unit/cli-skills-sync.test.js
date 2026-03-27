/**
 * Unit Tests for skills directory sync in setupHub
 *
 * Tests the syncSkillsToHub function which handles manifest-aware
 * skill synchronization, protecting user-created skills.
 *
 * @module tests/unit/cli-skills-sync.test
 */

const test = require('ava');
const proxyquire = require('proxyquire');
const crypto = require('crypto');

// Helper to compute SHA256 hash
function computeHash(content) {
  return 'sha256:' + crypto.createHash('sha256').update(content).digest('hex');
}

// Helper to create mock Dirent object
function createMockDirent(name, isDirectory = false) {
  return {
    name,
    isDirectory: () => isDirectory,
    isFile: () => !isDirectory,
    isBlockDevice: () => false,
    isCharacterDevice: () => false,
    isFIFO: () => false,
    isSocket: () => false,
    isSymbolicLink: () => false
  };
}

// Create a single shared mock state
const sharedMockState = {
  files: {},
  directories: new Set(),
  copyCalls: [],
  chmodCalls: [],
  unlinkCalls: [],
  copyErrorPath: null,
  logMessages: []
};

// Create mock implementations that use shared state
const mockFs = {
  existsSync: (p) => {
    return sharedMockState.files.hasOwnProperty(p) || sharedMockState.directories.has(p);
  },

  readFileSync: (p, options) => {
    if (!sharedMockState.files.hasOwnProperty(p)) {
      const error = new Error(`ENOENT: no such file or directory, open '${p}'`);
      error.code = 'ENOENT';
      throw error;
    }
    const content = sharedMockState.files[p];
    if (typeof options === 'string' && options === 'utf8') {
      return content.toString('utf8');
    }
    return content;
  },

  writeFileSync: (p, data) => {
    sharedMockState.files[p] = Buffer.isBuffer(data) ? data : Buffer.from(data);
  },

  mkdirSync: (p, options) => {
    sharedMockState.directories.add(p);
    if (options?.recursive) {
      let parent = p;
      while (parent !== '/' && parent !== '') {
        parent = parent.substring(0, parent.lastIndexOf('/')) || '/';
        if (parent !== '/' && parent !== '') {
          sharedMockState.directories.add(parent);
        }
      }
    }
  },

  readdirSync: (p, options) => {
    if (!sharedMockState.directories.has(p)) {
      const error = new Error(`ENOTDIR: not a directory, scandir '${p}'`);
      error.code = 'ENOTDIR';
      throw error;
    }

    const entries = [];

    // Find all files in this directory
    for (const filePath of Object.keys(sharedMockState.files)) {
      const dir = filePath.substring(0, filePath.lastIndexOf('/')) || '/';
      if (dir === p || (p === '/' && dir === '')) {
        const name = filePath.substring(filePath.lastIndexOf('/') + 1);
        if (options?.withFileTypes) {
          entries.push(createMockDirent(name, false));
        } else {
          entries.push(name);
        }
      }
    }

    // Find all subdirectories
    for (const dirPath of sharedMockState.directories) {
      if (dirPath !== p) {
        const parent = dirPath.substring(0, dirPath.lastIndexOf('/')) || '/';
        if (parent === p) {
          const name = dirPath.substring(dirPath.lastIndexOf('/') + 1);
          if (options?.withFileTypes) {
            entries.push(createMockDirent(name, true));
          } else {
            entries.push(name);
          }
        }
      }
    }

    return entries;
  },

  rmdirSync: (p) => {
    sharedMockState.directories.delete(p);
  },

  copyFileSync: (src, dest) => {
    sharedMockState.copyCalls.push({ src, dest });
    if (sharedMockState.copyErrorPath && src.includes(sharedMockState.copyErrorPath)) {
      const error = new Error('Permission denied');
      error.code = 'EACCES';
      throw error;
    }
    if (!sharedMockState.files.hasOwnProperty(src)) {
      const error = new Error(`ENOENT: no such file or directory, copyfile '${src}'`);
      error.code = 'ENOENT';
      throw error;
    }
    sharedMockState.files[dest] = Buffer.from(sharedMockState.files[src]);
  },

  unlinkSync: (p) => {
    sharedMockState.unlinkCalls.push(p);
    if (!sharedMockState.files.hasOwnProperty(p)) {
      const error = new Error(`ENOENT: no such file or directory, unlink '${p}'`);
      error.code = 'ENOENT';
      throw error;
    }
    delete sharedMockState.files[p];
  },

  chmodSync: (p, mode) => {
    sharedMockState.chmodCalls.push({ path: p, mode });
  },

  constants: {
    F_OK: 0,
    R_OK: 4,
    W_OK: 2,
    X_OK: 1
  }
};

const mockPath = {
  join: (...parts) => {
    const filtered = parts.filter(p => p !== '');
    if (filtered.length === 0) return '/';
    const joined = filtered.join('/').replace(/\/+/g, '/');
    return joined.startsWith('/') ? joined : '/' + joined;
  },

  dirname: (p) => {
    const parts = p.split('/').filter(Boolean);
    parts.pop();
    return parts.length === 0 ? '/' : '/' + parts.join('/');
  },

  relative: (base, full) => {
    const basePath = base.replace(/\/$/, '');
    const fullPath = full.replace(/\/$/, '');
    if (fullPath.startsWith(basePath + '/')) {
      return fullPath.substring(basePath.length + 1);
    }
    return fullPath.startsWith('/') ? fullPath.substring(1) : fullPath;
  },

  resolve: (...parts) => {
    const joined = parts.join('/').replace(/\/+/g, '/');
    return joined.startsWith('/') ? joined : '/' + joined;
  },

  sep: '/'
};

const mockChildProcess = {
  execSync: () => {
    throw new Error('Not a git repo');
  }
};

// Capture log messages
const origLog = console.log;

// Load CLI module once with mocks
const cli = proxyquire('../../bin/cli.js', {
  fs: mockFs,
  path: mockPath,
  child_process: mockChildProcess
});

// Reset function to clear state between tests
function resetMockState() {
  sharedMockState.files = {};
  sharedMockState.directories.clear();
  sharedMockState.copyCalls = [];
  sharedMockState.chmodCalls = [];
  sharedMockState.unlinkCalls = [];
  sharedMockState.copyErrorPath = null;
  sharedMockState.logMessages = [];
}

test.beforeEach(() => {
  resetMockState();
});

// --- TDD Cycle 1: syncSkillsToHub exists and is exported ---

test.serial('syncSkillsToHub is exported from cli module', t => {
  t.is(typeof cli.syncSkillsToHub, 'function');
});

// --- TDD Cycle 2: Does nothing when source skills dir does not exist ---

test.serial('syncSkillsToHub returns early when source skills dir does not exist', t => {
  const result = cli.syncSkillsToHub('/nonexistent/skills', '/hub/skills');
  t.deepEqual(result, { synced: [], skipped: [], notices: [] });
});

// --- TDD Cycle 3: Creates hub category directories ---

test.serial('syncSkillsToHub creates hub category directories', t => {
  // Setup: source has colony and domain categories
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/domain');

  // Empty manifests
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify({ skills: [] }));
  sharedMockState.files['/src/skills/domain/.manifest.json'] = Buffer.from(JSON.stringify({ skills: [] }));

  cli.syncSkillsToHub('/src/skills', '/hub/skills');

  t.true(sharedMockState.directories.has('/hub/skills/colony'));
  t.true(sharedMockState.directories.has('/hub/skills/domain'));
});

// --- TDD Cycle 4: Copies manifest files ---

test.serial('syncSkillsToHub copies .manifest.json to hub', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');

  const manifestData = { skills: ['tdd', 'error-handling'] };
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify(manifestData));

  cli.syncSkillsToHub('/src/skills', '/hub/skills');

  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/colony/.manifest.json'));
  const copiedManifest = JSON.parse(sharedMockState.files['/hub/skills/colony/.manifest.json'].toString('utf8'));
  t.deepEqual(copiedManifest.skills, ['tdd', 'error-handling']);
});

// --- TDD Cycle 5: Syncs managed skills from manifest ---

test.serial('syncSkillsToHub syncs managed skills listed in manifest', t => {
  // Setup source with a managed skill
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/colony/tdd');

  const manifestData = { skills: ['tdd'] };
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify(manifestData));
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('TDD skill content');

  const result = cli.syncSkillsToHub('/src/skills', '/hub/skills');

  // The skill should have been synced
  t.true(result.synced.includes('colony/tdd'));
  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/colony/tdd/SKILL.md'));
});

// --- TDD Cycle 6: Skips user-created skills ---

test.serial('syncSkillsToHub skips user-created skills not in manifest', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');

  // Manifest only lists 'tdd' as managed
  const manifestData = { skills: ['tdd'] };
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify(manifestData));

  // Source has 'tdd' skill
  sharedMockState.directories.add('/src/skills/colony/tdd');
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('TDD content');

  // Hub already has a user-created skill 'my-custom'
  sharedMockState.directories.add('/hub/skills/colony');
  sharedMockState.directories.add('/hub/skills/colony/my-custom');
  sharedMockState.files['/hub/skills/colony/my-custom/SKILL.md'] = Buffer.from('My custom skill');

  const result = cli.syncSkillsToHub('/src/skills', '/hub/skills');

  // User skill should not be in synced list
  t.false(result.synced.includes('colony/my-custom'));
  // User skill should still exist (untouched)
  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/colony/my-custom/SKILL.md'));
  t.is(sharedMockState.files['/hub/skills/colony/my-custom/SKILL.md'].toString('utf8'), 'My custom skill');
});

// --- TDD Cycle 7: Logs notice when user skill collides with new shipped skill ---

test.serial('syncSkillsToHub logs notice when user skill collides with shipped skill', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/colony/tdd');

  // Manifest includes 'tdd' as managed
  const manifestData = { skills: ['tdd'] };
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify(manifestData));
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('Shipped TDD');

  // Hub already has a user-created 'tdd' skill (same name as shipped)
  // But because it IS in the manifest, it's a managed skill and will be overwritten
  // This scenario is: user skill exists with same name before manifest ships it
  // Since tdd is in the manifest, it's treated as managed and will be synced (overwritten)
  // The collision case is when a SOURCE dir exists with same name as a hub dir but is NOT in manifest

  // Actually, let me re-read the spec. The collision is:
  // - Source has a skill dir NOT listed in the manifest (shouldn't normally happen)
  // - Hub has a user-created skill with the same name
  // But actually looking at the plan code more carefully:
  // The notice fires when fs.existsSync(hubCat/dir.name) for source dirs NOT in manifest
  // Let's test that scenario

  // Reset and set up a different scenario
  resetMockState();

  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/colony/tdd');

  // Manifest does NOT include 'tdd' — simulating a source dir that's not in manifest
  const manifestNoTdd = { skills: [] };
  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify(manifestNoTdd));
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('Source TDD');

  // Hub already has a user-created 'tdd' skill
  sharedMockState.directories.add('/hub/skills/colony');
  sharedMockState.directories.add('/hub/skills/colony/tdd');
  sharedMockState.files['/hub/skills/colony/tdd/SKILL.md'] = Buffer.from('User TDD');

  const result = cli.syncSkillsToHub('/src/skills', '/hub/skills');

  // Should have a notice about the collision
  t.true(result.notices.length > 0);
  t.true(result.notices[0].includes('tdd'));
  // User skill should be preserved
  t.is(sharedMockState.files['/hub/skills/colony/tdd/SKILL.md'].toString('utf8'), 'User TDD');
});

// --- TDD Cycle 8: Copies README.md from category source ---

test.serial('syncSkillsToHub copies README.md from category source to hub', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/domain');

  const manifestData = { skills: [] };
  sharedMockState.files['/src/skills/domain/.manifest.json'] = Buffer.from(JSON.stringify(manifestData));
  sharedMockState.files['/src/skills/domain/README.md'] = Buffer.from('# Domain Skills');

  cli.syncSkillsToHub('/src/skills', '/hub/skills');

  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/domain/README.md'));
  t.is(sharedMockState.files['/hub/skills/domain/README.md'].toString('utf8'), '# Domain Skills');
});

// --- TDD Cycle 9: Handles missing manifest gracefully ---

test.serial('syncSkillsToHub handles missing manifest gracefully (no managed skills)', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/colony/tdd');
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('TDD content');

  // No manifest file — should treat all as unmanaged
  const result = cli.syncSkillsToHub('/src/skills', '/hub/skills');

  // With no manifest, no skills should be synced as managed
  t.deepEqual(result.synced, []);
});

// --- TDD Cycle 10: Handles both colony and domain in one call ---

test.serial('syncSkillsToHub processes both colony and domain categories', t => {
  sharedMockState.directories.add('/src/skills');
  sharedMockState.directories.add('/src/skills/colony');
  sharedMockState.directories.add('/src/skills/colony/tdd');
  sharedMockState.directories.add('/src/skills/domain');
  sharedMockState.directories.add('/src/skills/domain/react');

  sharedMockState.files['/src/skills/colony/.manifest.json'] = Buffer.from(JSON.stringify({ skills: ['tdd'] }));
  sharedMockState.files['/src/skills/domain/.manifest.json'] = Buffer.from(JSON.stringify({ skills: ['react'] }));
  sharedMockState.files['/src/skills/colony/tdd/SKILL.md'] = Buffer.from('TDD content');
  sharedMockState.files['/src/skills/domain/react/SKILL.md'] = Buffer.from('React content');

  const result = cli.syncSkillsToHub('/src/skills', '/hub/skills');

  t.true(result.synced.includes('colony/tdd'));
  t.true(result.synced.includes('domain/react'));
  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/colony/tdd/SKILL.md'));
  t.true(sharedMockState.files.hasOwnProperty('/hub/skills/domain/react/SKILL.md'));
});
