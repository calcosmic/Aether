# Phase 10: Colony Maturity - Research

**Researched:** 2026-02-02
**Domain:** Multi-agent system testing, end-to-end validation, production readiness
**Confidence:** MEDIUM

## Summary

Phase 10 requires comprehensive end-to-end testing of the Aether multi-agent colony system. The research covered testing strategies for autonomous agent spawning, memory compression validation, voting system verification, event-driven scalability testing, concurrency safety, circuit breaker validation, and performance optimization for Claude-native systems.

The standard approach for multi-agent testing in 2025 emphasizes modular test suites with top-down orchestration - testing critical paths first (full workflow) then breaking down into components. For Aether's bash-based architecture, this means validating that Queen can provide intention and the colony self-organizes through all components without regressions.

**Primary recommendation:** Use a modular test suite with node-tap for TAP-style output, test critical path top-down, isolate failures with clean slate between tests, and use comprehensive metrics (timing, tokens, file I/O, subprocess spawns) to identify bottlenecks without arbitrary pass/fail thresholds.

## Standard Stack

The established libraries/tools for multi-agent system testing and validation:

### Core Testing Framework
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **node-tap** | Latest | TAP-style test framework | Official TAP implementation, outstanding TypeScript support, batteries included (CLI, assertions, spies, mocking, coverage), color-accessible reporters |
| **TAP Protocol** | Specification | Language-agnostic test communication | Machine-parseable and human-intelligible output, extensible reporting, standard across ecosystems |

### Performance & Stress Testing
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **AutoCannon** | Latest | HTTP/1.1 benchmarking | Quick load testing, CLI utility, 200k+ weekly downloads, supports HTTP pipelining |
| **Artillery** | Latest | Cloud-native distributed load testing | Production-scale tests, serverless on AWS/Azure, YAML test scripts, Playwright integration |
| **k6** | Latest | Developer-friendly load testing | Grafana integration, JS scripting API, modular logic, CI/CD integration |

### Concurrency & Safety Testing
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **Fray** | Latest | Deterministic concurrent testing | Detects race conditions through thread interleaving exploration |
| **proper-lockfile** | Latest | Cross-process file locking | Prevents concurrent access corruption, atomic operations |

### Performance Profiling
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **Node.js built-in profiler** | Native | CPU/memory profiling | `--prof` flag for tick sampling, `--prof-process` for analysis |
| **clinic.js** | Latest | Enhanced profiling | Better visualization than built-in, health checks |
| **0x** | Latest | Zero-config profiling | Flame graphs automatically, easy bottleneck identification |

### Installation
```bash
# Core testing
npm install --save-dev tap

# Performance testing
npm install --global autocannon
npm install --global artillery

# Concurrency testing
npm install --save-dev proper-lockfile

# Profiling
npm install --global clinic
npm install --global 0x
```

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| node-tap | Tape, Jest | Tape is simpler but fewer features; Jest is heavier, not TAP-compatible |
| AutoCannon | wrk, wrk2 | wrk2 is more established but not Node-native, requires separate installation |
| k6 | Artillery, Loader.io | k6 has better Grafana integration; Artillery has better cloud-native support |

## Architecture Patterns

### Recommended Project Structure
```
.planning/phases/10-colony-maturity/
├── tests/
│   ├── integration/
│   │   ├── full-workflow.test.ts     # End-to-end colony workflow
│   │   ├── autonomous-spawn.test.ts  # Spawning validation
│   │   ├── memory-compress.test.ts   # Memory compression
│   │   └── voting-verify.test.ts     # Voting verification
│   ├── stress/
│   │   ├── concurrent-access.test.ts # File locking stress
│   │   ├── spawn-limits.test.ts      # Circuit breaker validation
│   │   └── event-scalability.test.ts # Event bus load testing
│   ├── performance/
│   │   ├── timing-baseline.test.ts   # Baseline metrics
│   │   └── profiling.test.ts         # Profiling hooks
│   └── helpers/
│       ├── colony-setup.ts           # Test colony initialization
│       ├── cleanup.ts                # State reset between tests
│       └── assertions.ts             # Custom assertions for colony
├── docs/
│   └── COLONY_GUIDE.md               # Updated comprehensive guide
└── 10-RESEARCH.md                    # This file
```

### Pattern 1: Top-Down Critical Path Testing
**What:** Test full workflow first, then break down into components if it fails
**When to use:** End-to-end validation, production readiness checks
**Example:**
```typescript
// Source: Based on "Testing Multi-Agent Systems in the LLM Age" (2025)
// https://realworlddatascience.net/applied-insights/tutorials/posts/2025/12/12/MAS-guide.html

import test from 'tap';

test('Full Colony Workflow', async (t) => {
  // 1. Initialize colony
  const colony = await setupTestColony('Build REST API');

  // 2. Verify Queen emits intention
  t.ok(colony.pheromones.INIT, 'Queen emits INIT pheromone');

  // 3. Verify Workers spawn autonomously
  t.ok(colony.workers.length > 0, 'Workers spawn autonomously');

  // 4. Verify phases execute
  t.ok(colony.currentPhase > 0, 'Colony progresses through phases');

  // 5. Verify completion
  t.ok(colony.status === 'complete', 'Colony completes goal');

  await cleanup(colony);
});
```

### Pattern 2: Clean Slate State Isolation
**What:** Reset all state between tests to prevent cross-contamination
**When to use:** Integration tests, stateful system testing
**Example:**
```typescript
// Source: Node-tap best practices, official documentation
// https://node-tap.org/

test.beforeEach(async () => {
  // Ensure clean slate for each test
  await exec('git clean -fd .aether/data');
  await exec('rm -rf .aether/backups/*');
  await exec('rm -f .aether/data/*.json');
});

test.afterEach(async () => {
  // Cleanup after test
  await cleanupTestColony();
});
```

### Pattern 3: Autonomous Spawning Validation
**What:** Verify Workers spawn Workers without Queen intervention
**When to use:** Testing emergence, capability gap detection
**Example:**
```typescript
// Source: Based on "Autonomous AI Agents" research (2025)
// https://kodexolabs.com/what-are-autonomous-ai-agents/

test('Autonomous Spawning - Capability Gap Detection', async (t) => {
  const colony = await setupTestColony();

  // Simulate capability gap in Colonizer
  const colonizer = colony.getWorker('colonizer');
  colonizer.simulateGap('security_analysis');

  // Verify Scout spawns autonomously
  await colony.waitForWorkerSpawn('scout', 5000);
  t.ok(colony.hasWorker('scout'), 'Security Scout spawned autonomously');

  // Verify Queen did NOT intervene
  t.equal(colony.queenInterventions, 0, 'Queen did not intervene');
});
```

### Pattern 4: Memory Compression Validation
**What:** Verify no context rot after extended session
**When to use:** Testing triple-layer memory, DAST compression
**Example:**
```typescript
// Source: Based on "Context Window Management" research (2025)
// https://www.getmaximai.com/articles/context-window-management-strategies/

test('Memory Compression - No Context Rot', async (t) => {
  const colony = await setupTestColony();

  // Fill working memory to 180k tokens (near 200k limit)
  await fillWorkingMemory(colony, 180000);

  // Trigger compression
  await colony.compressMemory();

  // Verify compression ratio
  const originalSize = colony.memory.working.length;
  const compressedSize = colony.memory.shortTerm.length;
  const ratio = originalSize / compressedSize;

  t.ok(ratio >= 2.5, `Compression ratio ${ratio.toFixed(2)}x meets 2.5x target`);

  // Verify key information retained
  const searchResults = await colony.memory.search('database schema');
  t.ok(searchResults.length > 0, 'Key information retained after compression');

  // Verify working memory cleared
  t.equal(colony.memory.working.length, 0, 'Working memory cleared after compression');
});
```

### Pattern 5: Voting Verification Testing
**What:** Validate supermajority logic and issue aggregation
**When to use:** Testing Colony Verification phase
**Example:**
```typescript
// Source: Based on "Consensus Protocols for Voting" research (2025)
// https://www.researchgate.net/publication/396885322_Consensus_Protocols_Tailored_for_Electronic_Voting

test('Voting Verification - Supermajority Logic', async (t) => {
  const colony = await setupTestColony();

  // Create output needing verification
  const output = { code: 'function example() {}' };

  // Simulate multiple perspectives
  const votes = [
    { perspective: 'security', vote: 'approve', issues: [] },
    { perspective: 'performance', vote: 'approve', issues: [] },
    { perspective: 'maintainability', vote: 'reject', issues: ['no error handling'] },
  ];

  // Verify supermajority calculation
  const result = colony.verifyOutput(output, votes);

  t.equal(result.approved, false, 'Output rejected with dissenting vote');
  t.ok(result.issues.includes('no error handling'), 'Issue aggregated correctly');

  // Test with all approve
  const unanimousVotes = votes.map(v => ({ ...v, vote: 'approve', issues: [] }));
  const unanimousResult = colony.verifyOutput(output, unanimousVotes);

  t.ok(unanimousResult.approved, 'Output approved with unanimous vote');
});
```

### Pattern 6: Meta-Learning Validation
**What:** Verify confidence updates and recommendation improvements
**When to use:** Testing Colony Learning phase
**Example:**
```typescript
// Source: Based on "Meta-Learned Confidence" research (2025)
// https://www.researchgate.net/publication/399022152_Meta-Learned_Confidence_for_Graph-Based_Semi

test('Meta-Learning - Confidence Updates', async (t) => {
  const colony = await setupTestColony();

  // Initial recommendation
  const initialRec = colony.recommend('use PostgreSQL for database');
  t.ok(initialRec.confidence > 0, 'Initial recommendation has confidence');

  // Simulate positive feedback
  await colony.recordFeedback(initialRec.id, 'positive');

  // Verify confidence increased
  const updatedRec = colony.recommend('use PostgreSQL for database');
  t.ok(updatedRec.confidence > initialRec.confidence, 'Confidence increased after positive feedback');

  // Simulate negative feedback
  await colony.recordFeedback(updatedRec.id, 'negative');

  // Verify confidence decreased
  const finalRec = colony.recommend('use PostgreSQL for database');
  t.ok(finalRec.confidence < updatedRec.confidence, 'Confidence decreased after negative feedback');
});
```

### Anti-Patterns to Avoid
- **Testing implementation details:** Test observable behaviors, not internal functions. Tests should validate colony emerges correctly, not how specific functions work.
- **Fragile test ordering:** Each test must be independent. Use clean slate between tests.
- **Ignoring concurrency bugs:** Single-threaded tests miss race conditions. Use Fray or similar for concurrent testing.
- **Arbitrary performance thresholds:** Measure and report, don't pass/fail on arbitrary values. Bottlenecks vary by hardware.
- **Testing Queen in isolation:** Queen only makes sense with colony. Test end-to-end workflows.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| **Test runner** | Custom bash scripts | node-tap | TAP output, assertions, coverage, reporters, TypeScript support |
| **File locking** | Manual lock files | proper-lockfile | Cross-process locking, automatic cleanup, deadlock prevention |
| **Atomic writes** | Manual temp+rename | write-file-atomic | Handles crashes, permissions, edge cases automatically |
| **Load testing** | Custom concurrent loops | AutoCannon/k6 | Proper HTTP pipelining, statistics, reporting, established tools |
| **Profiling** | Manual timestamps | clinic.js/0x | Flame graphs, memory allocation, better visualization |
| **Concurrent testing** | Manual thread spawning | Fray | Deterministic interleaving, race condition detection |

**Key insight:** Testing infrastructure is complex and error-prone. Existing solutions handle edge cases you won't anticipate (crashes during writes, lock starvation, profiling overhead, etc.).

## Common Pitfalls

### Pitfall 1: Testing Without State Isolation
**What goes wrong:** Tests pass individually but fail in suite due to state leakage
**Why it happens:** Previous tests leave files, memory, or pheromones that affect subsequent tests
**How to avoid:**
- Always run `git clean` and `rm -rf .aether/data` between tests
- Use `test.beforeEach()` and `test.afterEach()` hooks
- Verify clean slate in test assertions
**Warning signs:** Flaky tests, order-dependent failures, tests pass alone but fail in suite

### Pitfall 2: Ignoring Concurrency Bugs
**What goes wrong:** Tests pass in single-threaded execution but fail in production under concurrent access
**Why it happens:** File corruption, race conditions, and deadlocks only manifest with specific thread interleavings
**How to avoid:**
- Use Fray for deterministic concurrent testing
- Test with multiple processes accessing colony state simultaneously
- Verify file locking prevents corruption
**Warning signs:** "Works on my machine," production errors not reproducible in tests

### Pitfall 3: Performance Testing on Wrong Hardware
**What goes wrong:** Performance baselines don't match production, bottlenecks misidentified
**Why it happens:** Development machines differ from production (CPU, RAM, disk speed)
**How to avoid:**
- Report metrics without pass/fail thresholds
- Track relative improvements (before/after optimization)
- Document test hardware specs
- Focus on bottleneck identification, not absolute numbers
**Warning signs:** Performance regressions in production despite passing tests

### Pitfall 4: Testing Queen Without Colony
**What goes wrong:** Tests validate Queen emits signals but don't verify colony responds
**Why it happens:** Treating Queen as controller rather than signal source
**How to avoid:**
- Always test end-to-end: Queen → Signals → Colony → Result
- Verify Workers spawn autonomously
- Check emergence, not orchestration
**Warning signs:** Tests pass but colony doesn't work in practice

### Pitfall 5: Missing Memory Compression Validation
**What goes wrong:** Context rot after extended sessions, memory leaks
**Why it happens:** Tests run quickly, don't fill memory enough to trigger compression
**How to avoid:**
- Artificially fill working memory to near 200k limit
- Verify compression ratio meets 2.5x target
- Search for key information after compression
- Check working memory cleared
**Warning signs:** Colony slows down after long sessions, memory grows unbounded

### Pitfall 6: Not Testing Circuit Breakers
**What goes wrong:** Infinite loops, unbounded spawning, resource exhaustion
**Why it happens:** Tests assume happy path, don't trigger failure modes
**How to avoid:**
- Simulate capability gaps that require spawning
- Verify spawn limits enforced
- Test circuit breaker triggers
- Verify depth limits prevent infinite loops
**Warning signs:** Production hangs, runaway processes, resource exhaustion

### Pitfall 7: Inadequate Stress Testing
**What goes wrong:** System works under normal load but fails under traffic spikes
**Why it happens:** Only testing expected scenarios, not edge conditions
**How to avoid:**
- Use Artillery or k6 for load testing
- Test with 10x normal concurrent operations
- Verify graceful degradation, not crashes
- Monitor circuit breakers under stress
**Warning signs:** Production outages under load, cascading failures

### Pitfall 8: Performance Bottlenecks in Bash
**What goes wrong:** Colony operations slow, poor user experience
**Why it happens:** Bash subprocess overhead, file I/O, synchronous operations
**How to avoid:**
- Profile with Node.js `--prof` flag or clinic.js
- Measure per-task timing for optimization targets
- Track file I/O counts, subprocess spawns
- Optimize hot paths, not everything
**Warning signs:** Long delays between phases, unresponsive commands

## Code Examples

Verified patterns from official sources:

### TAP-Style Test Structure
```typescript
// Source: node-tap official documentation
// https://node-tap.org/

import test from 'tap';

test('Colony End-to-End Workflow', async (t) => {
  t.plan(5); // Expect 5 assertions

  const colony = await setupTestColony('Build REST API');

  t.ok(colony, 'Colony initialized');
  t.ok(colony.pheromones.INIT, 'INIT pheromone emitted');
  t.ok(colony.workers.length > 0, 'Workers spawned');
  t.ok(colony.currentPhase > 0, 'Phase progressed');
  t.ok(colony.status === 'complete', 'Goal completed');

  await cleanup(colony);
});

// Nested tests for organization
test('Memory System', async (t) => {
  t.test('Working Memory Limit', async (t) => {
    const colony = await setupTestColony();
    await fillWorkingMemory(colony, 200000);

    const size = colony.memory.working.length;
    t.ok(size <= 200000, `Working memory ${size} within 200k token limit`);

    await cleanup(colony);
  });

  t.test('Compression Ratio', async (t) => {
    const colony = await setupTestColony();
    await fillWorkingMemory(colony, 180000);
    await colony.compressMemory();

    const ratio = colony.memory.compressionRatio;
    t.ok(ratio >= 2.5, `Compression ${ratio.toFixed(2)}x meets 2.5x target`);

    await cleanup(colony);
  });
});
```

### Stress Testing with AutoCannon
```typescript
// Source: AppSignal Blog - "Performance and Stress Testing in Node.js" (2025)
// https://blog.appsignal.com/2025/06/04/performance-and-stress-testing-in-nodejs.html

import autocannon from 'autocannon';

test('Event Bus Scalability', async (t) => {
  const colony = await setupTestColony();

  // Start colony server
  const server = await colony.startServer();

  // Run load test
  const result = await autocannon({
    url: 'http://localhost:3000/api/events',
    connections: 50,        // 50 concurrent connections
    duration: 30,           // 30 seconds
    pipelining: 1,
    requests: [
      {
        method: 'POST',
        body: JSON.stringify({ type: 'FOCUS', target: 'database' })
      }
    ]
  });

  // Verify no errors
  t.equal(result.errors, 0, 'No errors under load');

  // Verify throughput
  t.ok(result.requests.mean > 100, `${result.requests.mean.toFixed(2)} req/s meets threshold`);

  // Verify latency acceptable
  t.ok(result.latency.mean < 100, `${result.latency.mean.toFixed(2)}ms mean latency acceptable`);

  await colony.stopServer();
  await cleanup(colony);
});
```

### Concurrency Testing with Fray
```typescript
// Source: "Deterministic Concurrent Testing Using Fray" (2025)
// https://softwaremill.com/deterministic-concurrent-testing-using-fray/

import { fray } from 'fray';

test('Concurrent State Access', async (t) => {
  const colony = await setupTestColony();

  // Run concurrent operations
  await fray(10, async (i) => {
    // 10 concurrent workers accessing colony state
    await colony.emitPheromone(`WORKER_${i}`, 'FOCUS', `target-${i}`);
    await colony.updateState(`worker-${i}`, { status: 'active' });
  });

  // Verify no corruption
  const state = await colony.loadState();
  t.ok(state, 'State loaded without corruption');
  t.equal(Object.keys(state.workers).length, 10, 'All 10 workers recorded');

  await cleanup(colony);
});
```

### Performance Profiling
```bash
# Source: Node.js official documentation - "Profiling Node.js Applications"
# https://nodejs.org/en/learn/getting-started/profiling

# Generate profile
NODE_ENV=production node --prof colony.js

# Process profile
node --prof-process isolate-0xnnnnnnnnnnnn-v8.log > processed.txt

# Use clinic.js for better visualization
clinic doctor -- node colony.js

# Use 0x for zero-config profiling
0x colony.js
```

### Circuit Breaker Testing
```typescript
// Source: "The Therac-25 Lesson: Why AI Agents Need Circuit Breakers" (2025)
// https://medium.com/@kudelin.dev/the-therac-25-lesson-why-ai-agents-need-a-circuit-breaker-architecture

test('Circuit Breaker - Spawn Limits', async (t) => {
  const colony = await setupTestColony();
  colony.config.maxWorkers = 5; // Set low limit for testing

  // Attempt to spawn 10 workers
  for (let i = 0; i < 10; i++) {
    colony.attemptSpawn(`worker-${i}`);
  }

  // Verify circuit breaker triggered
  t.equal(colony.workers.length, 5, 'Spawn limit enforced');
  t.ok(colony.circuitBreakerTripped, 'Circuit breaker triggered');

  // Verify no infinite spawn attempts
  t.equal(colony.spawnAttempts, 10, 'Exactly 10 spawn attempts logged');

  await cleanup(colony);
});

test('Circuit Breaker - Depth Limits', async (t) => {
  const colony = await setupTestColony();
  colony.config.maxDepth = 3;

  // Create deep spawn chain
  await colony.spawnChain('worker-1', 10); // Attempt depth 10

  // Verify depth limit enforced
  const maxDepth = colony.getMaxDepth();
  t.ok(maxDepth <= 3, `Max depth ${maxDepth} within limit`);

  await cleanup(colony);
});
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| **Jest/Mocha** | **node-tap (TAP)** | Ongoing | Better CI integration, language-agnostic, extensible |
| **Manual load testing** | **AutoCannon/k6** | 2023-2024 | Proper HTTP pipelining, established tools, better reporting |
| **Ignore concurrency** | **Fray deterministic testing** | 2024-2025 | Race condition detection, reproducible concurrent failures |
| **Guess bottlenecks** | **clinic.js/0x profiling** | 2024-2025 | Flame graphs, automatic bottleneck identification |
| **Arbitrary thresholds** | **Measure and report** | 2024-2025 | Focus on improvements, not pass/fail on specific hardware |

**Deprecated/outdated:**
- **Custom bash test runners**: Use node-tap for TAP output and proper test structure
- **wrk/wrk2 for Node.js**: Use AutoCannon for native Node.js support
- **Manual file locking**: Use proper-lockfile for cross-process safety
- **Testing in isolation**: Test end-to-end workflows for multi-agent systems

## Open Questions

Things that couldn't be fully resolved:

1. **Aether-Specific Testing Patterns**
   - What we know: General multi-agent testing patterns exist
   - What's unclear: How to test bash-based colony with Claude Code commands specifically
   - Recommendation: Start with general patterns, adapt as we learn Aether's specifics

2. **Memory Compression Verification**
   - What we know: Should test 2.5x compression ratio and retention
   - What's unclear: How to programmatically verify "key information retained"
   - Recommendation: Use search queries for known critical info, verify retrieval

3. **Circuit Breaker Thresholds**
   - What we know: Need spawn limits and depth limits
   - What's unclear: What specific values prevent problems without blocking legitimate work
   - Recommendation: Measure during testing, set conservatively, adjust based on real usage

4. **Performance Baselines**
   - What we know: Should measure timing, tokens, file I/O, subprocess spawns
   - What's unclear: What values constitute "good" vs "needs optimization"
   - Recommendation: Track relative improvements, document hardware, avoid absolute thresholds

5. **Event Bus Scalability**
   - What we know: Should test with AutoCannon under concurrent load
   - What's unclear: How many concurrent events is realistic for Aether's use case
   - Recommendation: Test 10x expected normal load, verify graceful degradation

## Sources

### Primary (HIGH confidence)
- [node-tap official documentation](https://node-tap.org/) - Features, TypeScript support, TAP protocol
- [Node.js Profiling Guide](https://nodejs.org/en/learn/getting-started/profiling) - Built-in profiler, tick processing, flame graphs
- [AppSignal Blog - Performance Testing](https://blog.appsignal.com/2025/06/04/performance-and-stress-testing-in-nodejs.html) - AutoCannon, Artillery, k6 comparison

### Secondary (MEDIUM confidence)
- [Real World Data Science - Testing Multi-Agent Systems](https://realworlddatascience.net/applied-insights/tutorials/posts/2025/12/12/MAS-guide.html) - MAS testing strategies
- [Medium - The Therac-25 Lesson: Circuit Breakers](https://medium.com/@kudelin.dev/the-therac-25-lesson-why-ai-agents-need-a-circuit-breaker-architecture-789fca88272a) - Circuit breaker patterns
- [SoftwareMill - Deterministic Concurrent Testing](https://softwaremill.com/deterministic-concurrent-testing-using-fray/) - Fray usage
- [ResearchGate - Consensus Protocols for Voting](https://www.researchgate.net/publication/396885322_Consensus_Protocols_Tailored_for_Electoral_Voting) - Voting system validation

### Tertiary (LOW confidence)
- [Medium - Context Window Management](https://www.getmaximai.com/articles/context-window-management-strategies-for-long-context-ai-agents-and-chatbots/) - Memory compression strategies
- [ResearchGate - Meta-Learned Confidence](https://www.researchgate.net/publication/399022152_Meta-Learned_Confidence_for_Graph-Based_Semi) - Confidence scoring
- [Galileo AI - Stability in Multi-Agent Systems](https://galileo.ai/blog/stability-strategies-dynamic-multi-agents) - Multi-agent stability
- [UiPath - AI Agent Best Practices](https://www.uipath.com/blog/ai/agent-builder-best-practices) - Agent reliability

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - Official documentation verified (node-tap, Node.js profiler, AppSignal)
- Architecture: MEDIUM - General patterns verified, Aether-specific needs validation
- Pitfalls: MEDIUM - Common testing pitfalls known, Aether-specific needs empirical validation

**Research date:** 2026-02-02
**Valid until:** 2026-03-02 (30 days - testing patterns stable, but tooling evolves)

**Key gaps requiring validation:**
- Aether's bash-based architecture may need custom test helpers
- Claude Code command integration testing needs exploration
- Memory compression "key information" verification needs concrete definition
- Performance baselines require empirical measurement on actual hardware
