---
name: ant:verify-castes
description: "Verify model routing is working for all castes"
---

You are the **Queen**. Verify that model routing is active and working.

## Step 1: Check Proxy Health

Run using the Bash tool: `node bin/cli.js caste-models list`

Capture the proxy status line. If proxy is unhealthy:
- Display warning about proxy not running
- Show instructions to start LiteLLM proxy
- Continue with verification anyway (will show failures)

## Step 2: Verify Each Caste Assignment

For each caste in [prime, builder, watcher, oracle, scout, chaos, architect, archaeologist, colonizer, route_setter]:

1. Get assigned model using the Bash tool:
   ```
   node -e "const mp = require('./bin/lib/model-profiles'); const p = mp.loadModelProfiles('.'); console.log(mp.getEffectiveModel(p, 'CASTE').model)"
   ```
   Replace CASTE with the actual caste name.

2. Verify model is not "default" (should be specific model from profiles)
3. Log result with checkmark or X

Display results in a table:
```
Caste Verification:
─────────────────────────────────────────
✓ prime: glm-5 (z_ai)
✓ builder: kimi-k2.5 (kimi)
✓ watcher: kimi-k2.5 (kimi)
...
```

## Step 3: Test Spawn Verification (Optional but Recommended)

If proxy is healthy, spawn a test worker to verify model routing actually works:

1. Create a temporary test script that:
   - Reports which ANTHROPIC_MODEL it sees
   - Reports ANTHROPIC_BASE_URL
   - Exits successfully

2. Use the Write tool to create a temporary test script at `/tmp/aether-test-spawn.js`:
```javascript
console.log('ANTHROPIC_MODEL:', process.env.ANTHROPIC_MODEL || '(not set)');
console.log('ANTHROPIC_BASE_URL:', process.env.ANTHROPIC_BASE_URL || '(not set)');
process.exit(0);
```

3. Spawn test worker via Task tool with builder caste environment variables:
   - Set ANTHROPIC_MODEL to the builder's assigned model
   - Set ANTHROPIC_BASE_URL to the proxy endpoint

4. Capture output and verify:
   - Model environment variable is set correctly
   - Base URL points to proxy

5. Display result:
```
Test Spawn (builder):
─────────────────────────────────────────
Model: kimi-k2.5 ✓
Base URL: http://localhost:4000 ✓
```

## Step 4: Summary Report

Display final verification report:
```
═══════════════════════════════════════════
Verification Complete
═══════════════════════════════════════════
Proxy Health: ✓ Healthy
Caste Assignments: 10/10 verified
Test Spawn: ✓ Working

Status: All systems operational
```

Or if issues found:
```
Issues Detected:
- Proxy unhealthy (not running)
- 3 castes using default model

Recommendations:
1. Start LiteLLM proxy: litellm --config proxy.yaml
2. Run `aether caste-models set <caste>=<model>` for missing assignments
```
