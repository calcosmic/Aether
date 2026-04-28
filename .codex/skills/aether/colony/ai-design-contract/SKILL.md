---
name: ai-design-contract
description: Use when a phase involves LLMs, AI agents, RAG, ML inference, or prompt/tool integration design
type: colony
domains: [ai, machine-learning, llm, data-pipeline]
agent_roles: [architect, builder, scout]
workflow_triggers: [plan, build]
task_keywords: [ai, llm, model, prompt, rag, embedding, agent]
priority: normal
version: "1.0"
---

# AI Design Contract

## Purpose

Generates a structured AI design contract (AI-SPEC.md) for phases that involve building AI-powered features. Ensures model selection, prompt engineering, evaluation strategy, safety guardrails, and integration architecture are specified before code is written. Prevents common AI integration failures by forcing explicit decisions about cost, latency, fallbacks, and data privacy upfront.

## When to Use

- The phase integrates LLM APIs (OpenAI, Anthropic, local models)
- The phase builds ML inference pipelines, RAG systems, or embedding search
- The phase creates AI agents, tool-use systems, or autonomous workflows
- The phase processes user input through AI models for generation, classification, or extraction
- User says "design the AI contract" or "spec the AI integration"

## Instructions

Generate an `AI-SPEC.md` file in the phase directory with these sections:

### 1. Problem Definition

- What the AI feature does in user-facing terms
- Input: what data flows into the model (format, size, language)
- Output: what the model produces (text, structured data, scores, actions)
- Success criteria: measurable outcomes the AI must achieve
- Failure modes: what happens when the AI cannot produce a valid result

### 2. Model Selection

- Primary model: provider, model ID, version (e.g., `gpt-4o-2024-08-06`, `claude-sonnet-4-20250514`)
- Fallback model: alternative if primary is unavailable or rate-limited
- Justification: why this model fits the task (context window, latency, cost, capability)
- Local vs. API: decision criteria and fallback strategy
- Cost estimation: per-request cost, projected monthly cost at expected volume

### 3. Prompt Engineering

- System prompt: full text of the system instruction
- User prompt template: parameterized template with variable placeholders
- Few-shot examples: at least 3 input/output pairs demonstrating expected behavior
- Output format: JSON schema, structured output constraints, or parsing instructions
- Token budget: max input tokens, max output tokens, and the reasoning for each limit

### 4. Evaluation Strategy

- Offline evaluation: benchmark dataset, metrics (accuracy, F1, BLEU, ROUGE, or task-specific)
- Online evaluation: A/B test structure, guardrail metrics, human review sampling rate
- Regression testing: golden test cases that must always pass (stored in version control)
- Evaluation frequency: run evals on every prompt change, weekly on production traffic
- Failure threshold: specific metric values that trigger a rollback or alert

### 5. Safety and Guardrails

- Input validation: reject inputs that are too long, contain harmful content, or fall outside scope
- Output validation: schema validation, content filtering, PII detection and redaction
- Rate limiting: per-user and global limits to prevent abuse and control costs
- Fallback behavior: graceful degradation when the model returns invalid or harmful output
- Content policy: alignment with the project's acceptable use policy

### 6. RAG and Context Strategy (if applicable)

- Data sources: what documents, databases, or APIs provide grounding context
- Chunking strategy: chunk size, overlap, and metadata attached to each chunk
- Embedding model: provider and model ID for vectorization
- Retrieval parameters: top-k, similarity threshold, reranking approach
- Context window management: how to prioritize and truncate context to fit within token limits

### 7. Integration Architecture

- Synchronous vs. asynchronous: when the user waits vs. when results are delivered later
- Caching strategy: cache identical requests, TTL, cache invalidation triggers
- Observability: logging prompt/response pairs (with PII redacted), latency tracking, error rates
- Cost monitoring: track token usage per feature, alert on cost anomalies
- Versioning: how prompt versions are tracked, deployed, and rolled back

### 8. Data and Privacy

- Data residency: where prompts and responses are processed and stored
- Retention policy: how long prompts and responses are kept, deletion schedule
- User consent: how users are informed that AI processes their data
- PII handling: what personally identifiable information is sent to models and how it's protected
- Compliance: applicable regulations (GDPR, CCPA, HIPAA) and how they're addressed

## Generation Process

1. **Define the problem:** Clarify what the AI must do and what success looks like
2. **Select the model:** Choose based on capability, latency, cost, and data sensitivity
3. **Design prompts:** Write system instructions, templates, and examples
4. **Plan evaluation:** Define test cases, metrics, and monitoring before building
5. **Set guardrails:** Specify input/output validation and fallback behaviors
6. **Address privacy:** Document data flows, retention, and compliance requirements

## Key Patterns

- **Cost-aware from the start**: Every AI contract includes per-request and monthly cost estimates so there are no billing surprises.
- **Fallback-first design**: Every primary model has a fallback model specified before implementation begins.
- **Evaluation before deployment**: Define metrics and golden test cases before writing prompts, not after.
- **Privacy by default**: Data residency, retention, and PII handling are explicit contract sections, not afterthoughts.
- **Version-controlled prompts**: Prompts are treated as code with versioning, rollback, and A/B testing capability.

## Output Format

```
 AI-SPEC | Phase {N}: {name}
    Model: {primary} (fallback: {secondary})
    Cost: ${per_request}/req | ~${monthly}/mo
    Evaluation: {metric_count} metrics defined
    Safety: {guardrail_count} guardrails configured
    Privacy: {compliance_standards} addressed
    Contract: .aether/phases/{N}/AI-SPEC.md
```

## Examples

**LLM integration phase:**
> "Generated AI-SPEC.md for phase 4 (Content Generation). Primary: GPT-4o, fallback: Claude Sonnet. Estimated cost: $0.003/req, $450/mo at projected volume. 12 evaluation metrics defined including accuracy, latency, and safety. PII redaction configured for user inputs. Contract locked and ready for implementation."

**RAG system phase:**
> "Generated AI-SPEC.md for phase 6 (Knowledge Search). Embedding model: text-embedding-3-small. Retrieval: top-5 with reranking. Context window: 8k tokens with priority truncation. 3 golden test cases for retrieval accuracy. No PII sent to embedding API."
