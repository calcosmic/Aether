# Playbook: Oracle

## Overview

Deep research using the RALF loop (Research, Analyze, Learn, Finalize). The Oracle ant conducts thorough investigation on a topic and produces actionable recommendations with confidence evaluation.

## Stage 1: Research

### Step 1: Initialize Oracle Session

Run `aether load-state` to get colony context. Extract goal and active pheromones.

Generate Oracle name: `aether generate-ant-name "oracle"`
Log spawn: `aether spawn-log --parent "Queen" --caste "oracle" --name "{name}" --task "Deep research: {topic}" --depth 0`

### Step 2: Research Loop (RALF)

The Oracle performs iterative research passes:

**Pass 1 — Broad Scan:**
- Search codebase for relevant files
- Read documentation and READMEs
- Identify key stakeholders and patterns

**Pass 2 — Deep Dive:**
- Read critical files in detail
- Run relevant commands (tests, builds) to understand behavior
- Interview code via grep and targeted reads

**Pass 3 — Cross-Reference:**
- Compare findings against colony wisdom (QUEEN.md)
- Check for conflicting patterns
- Validate assumptions with evidence

After each pass, the Oracle evaluates confidence:
- If confidence >= 90%: proceed to finalize
- If confidence < 90% and max passes not reached: continue research
- If max passes reached: finalize with current confidence

## Stage 2: Analyze

### Step 3: Synthesize Findings

Compile research into structured findings:
- **Facts** — verified observations with evidence
- **Assumptions** — unverified but likely true
- **Gaps** — unknowns that need clarification
- **Risks** — potential problems

### Step 4: Confidence Evaluation

Score each finding:
- **High (0.8-1.0)** — verified by tests, documentation, or multiple sources
- **Medium (0.5-0.79)** — supported by evidence but not fully verified
- **Low (0.0-0.49)** — speculative or based on limited evidence

Overall confidence is the weighted average of all findings.

## Stage 3: Learn

### Step 5: Extract Patterns

Identify reusable patterns from research:
- Success patterns — approaches that worked well
- Failure patterns — approaches to avoid
- Decision patterns — how similar choices were made

Create instincts via `aether instinct-create` for high-confidence patterns.

### Step 6: Memory Capture

Record observations via `aether memory-capture`:
- Type: "research"
- Content: key finding with evidence
- Source: "oracle-{topic}"

## Stage 4: Finalize

### Step 7: Produce Research Plan

Write findings to `.aether/data/research/oracle-{topic}.md`:
- Executive summary
- Detailed findings
- Recommendations with priority
- Risks and mitigations
- Next steps

### Step 8: Return Results

Return JSON:
```json
{
  "ant_name": "{Oracle-Name}",
  "caste": "oracle",
  "status": "completed",
  "summary": "Brief summary",
  "findings": ["finding1", "finding2"],
  "recommendations": ["rec1", "rec2"],
  "risks": ["risk1"],
  "confidence": 0.85,
  "research_file": ".aether/data/research/oracle-{topic}.md",
  "blockers": []
}
```

Log completion: `aether spawn-complete --name "{name}" --status "completed" --summary "Research complete"`

### Step 9: Update Session

Run `aether session-update --command "/ant-oracle" --suggested-next "/ant-plan" --summary "Oracle research completed"`
