# Knowledge Gaps

## Open Questions
- Q1 (35%): Missing exact Accelerate book formulas, 2025 archetype model details, reliability metric definition
- Q2 (30%): Have core benchmarks (HumanEval, MBPP, SWE-bench, BigCodeBench, MultiPL-E) with scoring and SOTA. Missing: CodeContests details, EvalPlus numbers, Aider polyglot specifics, scaffolding impact analysis, multi-agent vs single-agent comparative data
- Q3 (35%): Have ChatDev/MetaGPT metrics (completeness, executability, consistency, quality), GEMMAS graph metrics (IDS, UPR), MultiAgentBench milestone-KPIs, and TOSEM proposed dimensions. Missing: Park et al. generative agents methodology, AutoGen evaluation, deeper GEMMAS math, detailed MultiAgentBench scenarios, mapping to Aether's architecture
- Q4 (30%): Have McCabe (1976, NIST-validated thresholds), Halstead (1977, formulas + Volume threshold), ISO 25010:2023 (nine characteristics), Maintainability Index (formula + Visual Studio thresholds), SQALE (technical debt measurement). Missing: Cognitive Complexity (SonarSource alternative), detailed Halstead thresholds beyond Volume, ISO 25010 subcharacteristic measurability, SQALE A-E rating system, Maintainability Index criticisms
- Q5 (0%): Reliability measurement metrics — defect density, mutation testing, IEEE/ISO benchmarks untouched
- Q6 (0%): Aether instrumentation strategy — dependent on q1-q5 findings
- Q7 (0%): Presentation frameworks for sponsors — CMMI, TRL, OKR dashboards unexplored

## Contradictions
- Q1: The 2024 DORA benchmarks show Medium performers with ~10% failure rate vs High performers at ~20% — this seems counterintuitive (higher tier = higher failure rate). May reflect that high-performing teams deploy more frequently, increasing absolute failure count while maintaining fast recovery.
- Q2: SWE-bench scores are presented as model rankings but actually reflect agent scaffolding quality as much as raw model capability — the same model scores very differently with different scaffolds. This complicates using SWE-bench as a pure model metric.
- Q3: ChatDev scores higher on Quality (0.3953) but lower on Executability (0.88 binary vs MetaGPT's 3.9/4.0 scale) — different evaluation scales make direct comparison tricky. MetaGPT achieves 100% task completion but lower quality; ChatDev has higher quality but lower executability on the 4-point scale. The "best" multi-agent approach depends on which metric you prioritize.
- Q4: Maintainability Index has known criticisms — it may oversimplify quality into a single number and the formula weights are somewhat arbitrary. Need to investigate alternatives and criticisms in deeper phases.

## Discovered Unknowns
- How do HumanEval Pro and MBPP Pro (self-invoking code generation variants from ACL 2025) change the benchmark landscape?
- What is the relationship between pass@1 and pass@k scores — how much does k matter in practice?
- METR (March 2026) found that "many SWE-bench-passing PRs would not be merged into main" — what does this mean for benchmark validity?
- How does the Aider polyglot benchmark (225 exercises across C++, Go, Java, JS, Python, Rust) compare methodologically to MultiPL-E?
- What is the token-to-quality tradeoff curve for multi-agent systems? (ChatDev uses 3x tokens for ~2.6x quality vs single-agent — is this ratio consistent?)
- How does GEMMAS IDS mathematically relate to information entropy / Shannon diversity?
- Does MultiAgentBench's graph topology advantage translate to real software engineering tasks or only research scenarios?
- What are the precise costs of Aether's 24-agent architecture compared to ChatDev's ~5-agent and MetaGPT's ~5-agent setups?
- How does SonarSource's Cognitive Complexity differ from McCabe's Cyclomatic Complexity in practice? (SonarSource argues CC penalizes common patterns unfairly)
- What is the empirical evidence for Halstead's bug prediction formula (B = V/3000) — has it held up in modern codebases?
- How do ISO 25010 quality characteristics translate to measurable, automatable metrics?

## Last Updated
Iteration 3 -- 2026-03-30T08:45:00Z
