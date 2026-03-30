# Research Synthesis

## Topic
Scientifically established and industry-standard software engineering metrics for measuring development tool efficiency and AI-assisted coding productivity — for Aether website showcase to sponsors and backers

## Aether Context
- 22K+ lines of bash orchestration code
- 580+ tests across 64 test files
- 24 specialized agents
- 44 slash commands
- 28 skills (10 colony + 18 domain)
- Pheromone signaling system (FOCUS/REDIRECT/FEEDBACK)
- Cross-colony wisdom sharing (Hive Brain)
- Colony state lifecycle with phases and milestones
- Midden failure tracking system
- Learning/instinct promotion pipeline

## Findings by Question

### Q1: DORA Metrics — Definitions, Formulas, and Benchmarks [partial, 35%]

**The Four Metrics (Forsgren et al. "Accelerate", 2018):**
The four DORA metrics originated from Dr. Nicole Forsgren, Jez Humble, and Gene Kim's research spanning 6+ years of State of DevOps surveys [S1][S3]:
1. **Deployment Frequency (DF)** — How often code is deployed to production
2. **Lead Time for Changes (LT)** — Time from code commit to production deployment
3. **Change Failure Rate (CFR)** — Percentage of deployments causing production failures requiring rollback or hotfix
4. **Mean Time to Recovery (MTTR)** — Time to restore service after a production failure

**Calculation Approaches:**
- DF uses the **average (mean)** of successful deployments per time period
- LT, CFR, and MTTR all use the **median** (less skewed by outliers) [S4]

**2024 Performance Benchmarks (from survey clustering, NOT fixed thresholds):**

| Level | Lead Time | Deploy Freq | Failure Rate | Recovery Time | % of Respondents |
|-------|-----------|-------------|--------------|---------------|-----------------|
| Elite | <1 day | On demand | ~5% | <1 hour | 19% |
| High | 1 day–1 week | Daily–weekly | ~20% | <1 day | 22% |
| Medium | 1 week–1 month | Weekly–monthly | ~10% | <1 day | 35% |
| Low | >1 month | <monthly | >20% | >1 day | 25% |

These levels shift year-to-year based on survey response clustering [S2][S6].

**2025 Evolution:**
The 2025 DORA report moved away from the low/medium/high/elite cluster model entirely, introducing "archetypes" instead. A fifth metric (reliability) has been added [S5].

**Gaps remaining:** Exact formulas from the original Accelerate book (2018 edition), details on the 2025 archetype model, reliability metric definition, and how the fifth metric changes the framework.

### Q2: AI Coding Benchmarks [open, 0%]
(No findings yet)

### Q3: Multi-Agent AI System Evaluation [open, 0%]
(No findings yet)

### Q4: Software Quality Metrics with Academic Backing [open, 0%]
(No findings yet)

### Q5: Reliability Measurement Metrics [open, 0%]
(No findings yet)

### Q6: Aether Instrumentation Strategy [open, 0%]
(No findings yet)

### Q7: Presentation Frameworks for Sponsors [open, 0%]
(No findings yet)

## Last Updated
Iteration 0 -- 2026-03-30T00:30:00Z
