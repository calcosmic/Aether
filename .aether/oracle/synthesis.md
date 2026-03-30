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

### Q2: AI Coding Benchmarks — Definitions, Scoring, and SOTA [partial, 30%]

**Core Benchmarks:**

| Benchmark | Size | Language(s) | Scoring | Best SOTA | Key Source |
|-----------|------|-------------|---------|-----------|------------|
| HumanEval | 164 problems | Python | pass@k | 96.2% (o1-mini) | [S7][S8] |
| MBPP | 974 problems | Python | pass@k | ~89.5% | [S7] |
| SWE-bench Verified | 484 issues | Python | Resolved rate | 77.2% (Claude 4 Sonnet) | [S9][S12] |
| SWE-bench Full | ~2,300 issues | Python | Resolved rate | — | [S11] |
| BigCodeBench | 1,140 tasks | Multi (139 libs) | Calibrated pass@1 + Elo | 61% Complete, 51% Instruct | [S10] |
| MultiPL-E | HumanEval/MBPP translated | 18 languages | Test-case correctness | — | [S13] |
| CodeContests | Competitive problems | Multi | Execution-based | — | [S11] |

**The pass@k Formula (Chen et al. 2021):**
```
pass@k = 1 - C(n-c, k) / C(n, k)
```
Where n = total samples generated, c = correct samples (pass all unit tests), k = number of draws. This is an unbiased estimator of the probability that at least one of k solutions is correct. Standard evaluation: generate n≥k samples (typically n=200), count correct c, compute formula. [S8][S15]

**SWE-bench Methodology:**
Models receive a codebase and a GitHub issue description, then generate a patch. The patch is applied and the repository's original unit tests are run — the model cannot modify tests. "Resolved rate" = percentage of issues where the patch passes. Three variants exist: Full (~2,300 tasks), Lite (300 curated), Verified (484 human-validated from 12 Python repos). [S9][S11]

**BigCodeBench — Next-Gen Benchmark:**
Positioned as successor to HumanEval. Tasks require composing multiple function calls across 139 libraries in 7 domains. Uses calibrated Pass@1 (missing imports added during eval) and an Elo rating system (chess-style, 500 bootstrap iterations). Human performance: 97%. Best models reach only ~61% — far harder than HumanEval. Two variants: Complete (docstring-based) and Instruct (natural language). [S10]

**SWE-bench Leaderboard (March 2026):**

| Rank | Model | SWE-bench Verified |
|------|-------|--------------------|
| 1 | Claude 4 Sonnet | 77.2% |
| 2 | GPT-5 | 74.9% |
| 3 | Gemini 2.5 Pro | 73.1% |
| 4 | Claude Opus 4 | 71.8% |
| 5 | GPT-4o | 70.3% |
| 6 | o3-mini | 69.5% |
| 7 | DeepSeek V3 | 68.4% |

Note: Scores reflect agent scaffolding quality as much as raw model capability. [S12]

**Emerging Benchmarks (2025-2026):**
- **SWE-EVO** — Long-horizon software evolution across multiple commits [S14]
- **SWE-bench Pro** — Enterprise-level complexity
- **FeatureBench** — Complex feature development
- **DPAI Arena** (JetBrains, Oct 2025) — Multi-workflow, multi-language, full engineering lifecycle [S14]
- **DevQualityEval** — Evolving framework with expanding language support [S11]

**Multi-Agent Relevance:**
Existing benchmarks primarily evaluate isolated issue resolution (single patches). No established benchmark directly measures task delegation efficiency, inter-agent communication overhead, or collective problem-solving. SWE-EVO and DPAI Arena begin to address multi-step and full-lifecycle evaluation. Aether's 24-agent phase-based architecture aligns more with SWE-EVO's "long-horizon" framing than traditional single-issue benchmarks. [S14][S11]

**Gaps remaining:** Detailed CodeContests methodology, exact EvalPlus extended test suite numbers, Aider polyglot benchmark details, deeper analysis of scaffolding impact on SWE-bench scores, and how multi-agent frameworks specifically perform vs single-agent on these benchmarks.

### Q3: Multi-Agent AI System Evaluation [partial, 35%]

**Established Metrics from ChatDev (ACL 2024):**
ChatDev introduced four metrics for evaluating multi-agent software development systems [S16]:
1. **Completeness** — Percentage of generated software without any placeholder code snippets
2. **Executability** — Percentage of software that compiles successfully and runs directly
3. **Consistency** — Cosine distance between semantic embeddings of textual requirements and generated code
4. **Quality** — Product of completeness × executability × consistency (composite score)

Comparative results [S16][S20]:

| Metric | ChatDev | MetaGPT | GPT-Engineer |
|--------|---------|---------|--------------|
| Completeness | 0.5600 | 0.4834 | 0.5022 |
| Executability | 0.8800 | 0.4145 | 0.3583 |
| Consistency | 0.8021 | 0.7601 | 0.7887 |
| Quality | 0.3953 | 0.1523 | 0.1419 |

ChatDev won 88% of human pairwise evaluations against MetaGPT and 90.16% against GPT-Engineer.

**Communication Overhead — Concrete Data:**
Multi-agent systems have quantifiable communication costs [S16][S20]:

| Metric | ChatDev | MetaGPT | GPT-Engineer (single-agent) |
|--------|---------|---------|----------------------------|
| Duration | 148.2s | 154.0s | 15.6s |
| Token usage | 22,949 | 29,279 | 7,183 |
| Code files generated | 4.39 | 4.42 | 3.95 |
| Lines of code | 144.3 | 153.3 | 70.2 |

Multi-agent systems cost ~3-4x more tokens but produce ~2x more code with significantly higher quality. Large multi-agent groups can exceed $10 per HumanEval task due to serial message billing.

**Agent Interaction Protocols:**
- ChatDev limits communication to 10 rounds per subtask or 2 unchanged code modifications (whichever first) [S16]
- MetaGPT uses Standard Operating Procedures (SOPs) with role-based delegation: Product Manager → Architect → Project Manager → Engineer → QA [S21]
- MetaGPT achieves 100% task completion and 3.9/4.0 executability but lower quality score than ChatDev due to less cooperative communication [S20]

**Graph-Based Evaluation — GEMMAS (EMNLP 2025 Industry):**
GEMMAS introduces structural metrics for multi-agent collaboration beyond task accuracy [S18]:
- **Information Diversity Score (IDS)** — Measures diversity of information exchanged among agents; whether agents leverage distinct knowledge sources
- **Unnecessary Path Ratio (UPR)** — UPR = (Actual Steps - Optimal Steps) / Actual Steps; identifies redundant or circular interactions
- Together these assess information richness (IDS) and communication overhead (UPR) as complementary dimensions

**Benchmark Frameworks:**
- **MultiAgentBench (2025)** — Evaluates using milestone-based KPIs across collaboration and competition scenarios. Tests coordination topologies (star, chain, tree, graph, group discussion, cognitive planning). Graph topology performs best for research tasks; cognitive planning improves milestones by 3% [S19]
- **COMMA Benchmark** — Multimodal agent benchmark using puzzles to test language communication between agents

**Critical Research Gap (ACM TOSEM, He et al. 2025):**
Major survey identifies that no standardized evaluation metrics exist for multi-agent SE collaboration. Proposes five dimensions needing development [S17]:
1. Collaborative Design — agents' capacity to converge on unified architecture
2. Task Coordination — effective task division based on expertise
3. Conflict Resolution — handling conflicts constructively in real time
4. Integration Quality — seamless code integration and peer review
5. Proactive Communication — agents preemptively requesting needed information

These remain proposed frameworks, not implemented metrics. Gartner reported a 1,445% surge in multi-agent system inquiries from Q1 2024 to Q2 2025, reflecting growing interest in systematizing evaluation.

**Gaps remaining:** Park et al. generative agents specific evaluation methodology, AutoGen's evaluation approach, deeper GEMMAS mathematical formulation (IDS calculation), MultiAgentBench detailed scenario descriptions, and how Aether's specific architecture (pheromone signals, hive brain, midden tracking) maps to these emerging metrics.

### Q4: Software Quality Metrics with Academic Backing [partial, 30%]

**McCabe Cyclomatic Complexity (1976):**
Developed by Thomas J. McCabe Sr. Measures the number of independent paths through a program's control flow graph. Formula: M = E - N + 2P (edges minus nodes plus 2 times connected components). McCabe recommended an upper bound of 10, calling it "a reasonable, but not magical, upper limit." NIST SP 500-235 validated this threshold with substantial corroborating evidence. [S22][S23]

Published risk thresholds:

| Complexity | Risk Level | Recommendation |
|------------|-----------|----------------|
| 1-10 | Simple, low risk | Standard |
| 11-20 | Moderate complexity | Acceptable with experienced staff, formal design |
| 21-50 | Complex, high risk | Should be decomposed |
| 50+ | Untestable | Must be refactored |

Limits above 10 are only appropriate for projects with experienced staff, formal design, modern language, structured programming, code walkthroughs, and comprehensive test plans. [S23]

**Halstead Complexity Measures (1977):**
Introduced by Maurice Howard Halstead as part of establishing an empirical science of software development. Based on interpreting source code as a sequence of tokens classified as operators or operands. [S24][S25]

Four base counts: n1 (distinct operators), n2 (distinct operands), N1 (total operators), N2 (total operands).

Derived metrics:
- Vocabulary: n = n1 + n2
- Program Length: N = N1 + N2
- Calculated Length: N' = n1*log2(n1) + n2*log2(n2)
- Volume: V = N * log2(n)
- Difficulty: D = (n1/2) * (N2/n2)
- Effort: E = D * V
- Time to program: T = E/18 seconds
- Predicted bugs: B = V/3000

Published thresholds: function Volume should be 20-1000; Volume > 1000 indicates the function is doing too many things. [S25]

**ISO/IEC 25010:2023 — Product Quality Model:**
Updated November 2023 from 2011 version. Defines nine quality characteristics (was eight), each with subcharacteristics [S26][S27]:

1. Functional Suitability
2. Performance Efficiency
3. Compatibility
4. Interaction Capability (renamed from Usability)
5. Reliability
6. Security
7. Maintainability
8. Flexibility (renamed from Portability)
9. Safety (NEW in 2023)

Key changes from 2011: Safety added as a characteristic. New subcharacteristics: inclusivity, self-descriptiveness, resistance, scalability. User interface aesthetics replaced by user engagement; maturity replaced by faultlessness. Part of the SQuaRE (Systems and software Quality Requirements and Evaluation) standards family.

**Maintainability Index (Oman & Hagemeister, 1992):**
Composite metric combining Halstead Volume, Cyclomatic Complexity, and Lines of Code. [S28]

Original formula: MI = 171 - 5.2 * ln(HV) - 0.23 * CC - 16.2 * ln(LOC)

Visual Studio normalized to 0-100: MI = MAX(0, (171 - 5.2*ln(HV) - 0.23*CC - 16.2*ln(LOC)) * 100/171)

| Score | Rating | Meaning |
|-------|--------|---------|
| 20-100 | Green | Good maintainability |
| 10-19 | Yellow | Moderate maintainability |
| 0-9 | Red | Low maintainability (high confidence of issue) |

Microsoft chose conservative thresholds to minimize false positives — red means high confidence of a real problem.

**SQALE (Letouzey, 2012):**
Software Quality Assessment based on Lifecycle Expectations. Evaluates technical debt based on ISO 9126/ISO 25010 quality model. Measures "distance to conformity" as remediation cost — estimated time to fix each non-conformance found by static analysis. [S29][S30]

Two cost models:
1. **Remediation cost** — time to fix the non-conformance
2. **Non-remediation cost** — ongoing business impact of leaving debt unfixed

Quality characteristics mapped directly to ISO 25010 hierarchy. Enables cost-benefit analysis for technical debt prioritization. Used by SonarQube as the underlying technical debt measurement model.

**Gaps remaining:** Cognitive Complexity (SonarSource's alternative to cyclomatic complexity), exact Halstead threshold tables beyond Volume, ISO 25010 subcharacteristic definitions and how they map to measurable metrics, SQALE rating system (A-E), deeper comparison of Maintainability Index criticisms and alternatives.

### Q5: Reliability Measurement Metrics [open, 0%]
(No findings yet)

### Q6: Aether Instrumentation Strategy [open, 0%]
(No findings yet)

### Q7: Presentation Frameworks for Sponsors [open, 0%]
(No findings yet)

## Last Updated
Iteration 3 -- 2026-03-30T08:45:00Z
