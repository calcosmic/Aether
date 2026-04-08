---
title: Aether Queen.md Research
domain: code
type: research
status: done
created: 2026-02-22
updated: 2026-02-22T03:18:33+01:00
id: 20260222031824
tags:
  - type/research
  - status/done
  - inbox/code
---

  # Problem Statement

Today, new projects often start with vague goals communicated in plain language. Without a structured dialogue, team members rely on incomplete notes or code comments, leading to drifting interpretations. Half of real requirements are missed or left ambiguous in practice, and single-agent AI workflows tend to “forget” earlier intents as they generate outputs. In Aether, this breaks alignment: workers lack a single source of truth, so features slip outside scope or remain undefined. The **Queen Discourse** solves this by guiding the non-technical “Queen” (stakeholder) through a stepped questionnaire, capturing intent explicitly and producing a canonical QUEEN.md specification. That document will serve as an authoritative, testable foundation (a “constitution”) for the colony’s agents, preventing drift and ensuring all requirements, constraints and acceptance criteria are clear from the start.

# Design Principles

- **Single Source of Truth:** Store all high-level goals and constraints in one version-controlled spec (QUEEN.md) that agents and humans reference.
- **Testable Outputs:** Every requirement must be measurable via explicit acceptance criteria or tests. (“VERIFIABLE” and “UNAMBIGUOUS” in requirement terms.)
- **Defer Decisions Safely:** Allow answers like “I’m not sure” or “decide later”, marking such items TBD rather than forcing guesses. Track confidence and priority to revisit ambiguities.
- **Plain-Language UX:** Use simple, non-technical questions with common words. (Avoid jargon – the Queen is a layperson.)
- **Minimize Free Text:** Prefer multiple-choice for speed and consistency. Only allow short text when needed for clarity.
- **Adaptive Branching:** Ask only relevant questions via conditional logic. Use branching (“skip logic”) so users aren’t shown irrelevant prompts.
- **Progressive Disclosure:** Start with broad questions and reveal detail gradually. (E.g. ask about major goals first, then drill into specifics.)
- **Confidence & Priority Capture:** For each answer, record how confident the user is and how important the issue is (must-have vs optional).
- **Schema-Driven Validation:** Use structured answer schemas (JSON) so answers can be validated automatically. Avoid open-ended formats that LLMs might misinterpret.
- **Versioned and Updateable:** QUEEN.md must be diff-friendly and evolve. Use source control for diffs. Add new items via updates, deprecate old ones without breaking agents.
- **Constitution First (Governance):** Treat the QUEEN.md like an “agent constitution” of rules. All downstream prompts and tasks must remain compatible with it. Changes to it are major governance events, versioned and tested.

# Question Graph Design

**Domains:** Questions are grouped by theme. Key domains include:

- _Product Vision:_ objectives, success metrics.
- _Users/Personas:_ Who uses it and why. (Jobs-to-be-done, scenarios.)
- _User Environment:_ Where and how it’s used (device, web/mobile, offline/online).
- _Scope (In-Scope vs Out-of-Scope):_ Core features to include or explicitly exclude.
- _Business Goals & Metrics:_ Outcomes the business cares about (revenue, engagement, speed).
- _UX/Look & Feel:_ Desired style/tone (casual vs formal, colour schemes, branding).
- _Functional Flows:_ Core user flows or tasks (e.g. “User logs in”, “fills form”, etc.).
- _Non-functional (Quality) Attributes:_ Performance, security, accessibility, privacy requirements.
- _Data & Integrations:_ What data is needed and where it comes from (APIs, databases).
- _Risks & Constraints:_ Known risks, regulatory/legal issues (GDPR, etc.) and constraints (budget, timeline).
- _Administration & Maintenance:_ Roles, content management, update frequency.

**Branching Model:** Questions adapt based on prior answers. Below is a simplified Mermaid diagram showing branching between topics. Start broad (goals/users) then drill into specifics (flows, tech, constraints).

Show code

**Sample Questions (MVP ~25):** (Each is multiple-choice with an “Other/Unsure” option.) Examples:

- **Vision:** “What is the primary goal for this product?” (e.g. “Increase sales”, “Engagement”, “Information portal”…)
- **Metrics:** “How will we measure success?” (Revenue %, user signups, error rate, etc.)
- **Users:** “Who are the main users?” (Options like “Customers buying [X]”, “Admins managing system”, “Employees using tool”.)
- **Jobs:** “What problem does the user solve with this?” (E.g. “Find product info quickly”, “Book appointments”, etc.)
- **Environment:** “Where will they use it?” (Browser on desktop, mobile app, offline mode…)
- **Scope (In/Out):** “Which features should the product include?” (List checkboxes like “User login”, “Dashboard”, “Notifications”, “Report export”.)
- **Exclude Scope:** “Which features should _NOT_ be in this scope?” (List examples or “None”).
- **UX Style:** “Choose UI style preference” (Modern/minimal, Corporate/formal, Playful, etc.)
- **Core Flow:** “Select the main workflow steps.” (E.g. 1) User sign-up, 2) Profile setup, 3) Perform X task.)
- **NFR – Performance:** “Do we need fast response times?” (Yes/No/Unsure; if Yes: “Target load time <X sec?”)
- **NFR – Security:** “Any special security or privacy needs?” (GDPR, auth levels, data encryption, etc.)
- **Data:** “What data must the system manage?” (User data, transactions, logs, none)
- **Integrations:** “Does it connect to other systems?” (CRM, email, payment gateway, etc.)
- **Milestones:** “What milestones or features come first?” (Release a MVP by date, Phase 1, 2…)
- **Risks:** “What could go wrong?” (Technical complexity, legal issues, high cost, etc.)
- **Content (if any):** “Is content provided or must we create it?” (We have content / Content to create).
- **Other:** “Anything else to note?” (Short free text).

**Extended Questions (80–120):** For each above category, drill deeper. E.g. if user is mobile, ask “iOS, Android or both?”; if high security, ask specifics (2FA, data retention). Ask confidence: “How certain are you about this answer?”; add prioritization: “Is this a Must-Have or Nice-to-Have?” for each major feature. Ensure alternative flows (errors, edge cases). Ask about non-functional subtleties (localization, concurrency, uptime). Each branch yields more targeted queries, up to ~100 total.

# Answer Model

Answers are captured in a structured JSON schema. For example, each question record might include:

json

Copy

```json
{
  "questionId": "user_environment",
  "answer": "Mobile App (iOS & Android)",
  "priority": "Must-Have",
  "confidence": 0.8,
  "notes": "Key for our on-the-go users"
}
```

**Schema Definition (simplified):**

json

Copy

```json
{
  "type": "object",
  "properties": {
    "questions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "questionId": { "type": "string" },
          "answer": { "type": "string" },
          "priority": { "type": "string", "enum": ["Must-Have","Nice-to-Have","TBD"] },
          "confidence": { "type": "number", "minimum": 0.0, "maximum": 1.0 },
          "notes": { "type": "string" }
        },
        "required": ["questionId","answer"]
      }
    },
    "version": { "type": "string" }
  },
  "required": ["questions","version"]
}
```

- The **`questionId`** links to the predefined question.
- **`answer`** holds the chosen option or short text; special value “Unsure/TBD” if undecided.
- **`priority`** denotes importance (Must/Nice/TBD).
- **`confidence`** is a numeric [0-1] guess of certainty.
- **`notes`** allow brief context if needed.

This structured format allows validation (against schema) and diffing of answers. Unknowns are explicit (“TBD” or null) rather than blank.

# Synthesis Compiler

Answers flow through a deterministic pipeline to build QUEEN.md:

1. **Normalize intents:** Map answers to internal representations. E.g. “Mobile App” → platforms = [iOS, Android]; metrics = ["user_growth"].
    
2. **Generate sections:**
    
    - **Objective/North Star:** From Vision answers, summarize the mission and success metrics.
    - **Users/Personas:** Draft persona descriptions and target users from user/job answers.
    - **Scope:** List “In Scope” features (from Must-Have answers) and “Out of Scope” items (explicit excludes).
    - **Flows:** Turn core workflow answers into bullet steps under “Core Flows.”
    - **UX Principles:** Based on style questions, list UI guidelines (e.g. “mobile-first”, “clean/minimal design”).
    - **Non-functional Reqs:** Collate performance, security, accessibility, etc., from answers.
    - **Data & Integrations:** Note data sources and external systems.
    - **Milestones & DoD:** From milestones answers, create timeline bullets with “Definition of Done” criteria (often the acceptance criteria).
    - **Acceptance Tests:** For each major feature or user story, generate explicit tests (“Given…, when…, then…”) based on flows and objectives. (Optionally use the “Intent → Tests” mechanic: first produce tests, then fill spec.)
    - **Risk Register:** Convert identified risks into a table (Risk, Impact, Mitigation).
    - **Glossary:** Add definitions for any specialized terms or abbreviations mentioned.
3. **Consistency Checks:** After initial synthesis, run automated checks. For example:
    
    - **Conflict Detection:** If answers contradict (e.g. user said “web only” but also “mobile interface”), flag an inconsistency. The compiler can either select the higher-priority answer or insert a “TBD” and mark for review.
    - **Completeness:** Ensure all key headings (Objective, Flows, Tests, etc.) have content. If missing, either omit or insert placeholders.
    - **Ambiguity Scan:** Lint the text for vague terms (“fast”, “user-friendly”) and mark them for follow-up.
    - **Schema Validation:** The JSON answer data is validated before use (ensuring required fields present).

If conflicts or missing critical info are found, the system either:

- **Asks a follow-up question:** e.g. “You selected both Web and Mobile. Do you want to support both?”
- **Uses defaults:** For minor conflicts, pick the “safe” choice (e.g. assume Must-Have is priority).
- **Marks TBD:** Leaves the point to decide in the spec, noting uncertainty (and possibly low confidence).

All text in QUEEN.md is generated to be **clear, declarative, and “agent-readable”**. Lists and bullet points structure requirements (not just prose). Each acceptance criterion is explicit.

# QUEEN.md Template

The final QUEEN.md uses fixed Markdown headings and sections. For example:

pgsql

Copy

```
# North Star (Objectives & Success Metrics)
**Objective:** [Concise product goal.]  
**Success Metrics:** 
- [Metric 1] (target: X)  
- [Metric 2] (target: Y)  

# Personas / Target Users
- **Persona A:** [Description of user, e.g. “Busy professional needing quick access to…”].  
- **Persona B:** [Another key user group].

# Scope & Core Flows
**In Scope:** 
- Feature 1 (brief description)  
- Feature 2 (…)
**Out of Scope:** 
- Explicitly excluded features (if any).

**Core User Flows:**  
1. *[Step 1]*: [E.g. “User logs in with username and password.”]  
2. *[Step 2]*: [“User searches for products using search bar.”]  
3. *[Step 3]*: [Etc.]
*(Include acceptance test checks for key steps, e.g. “Users must be able to see search results within 2s.”)*

# UI Principles & Tone
- Use [style choices, e.g. “clean, minimal design with brand colors (blue/white)”; “friendly, conversational tone”].  
- Accessibility: [E.g. “WCAG AA compliance required” if answered].  
- Mobile-first: [Yes/No depending on answers].

# Non-Functional Requirements (Quality Attributes)
- **Performance:** [e.g. “Page loads <2s for user dashboards.”]  
- **Security:** [“GDPR-compliant user data handling; HTTPS required.”]  
- **Reliability:** [“99.5% uptime, error-rate <0.1%.”]  
- **Privacy:** [“User data must be encrypted at rest. No tracking beyond features.”]  
- (Add others: scalability, localizability, etc., as identified【5†L142-L150】【7†L139-L147】.)

# Data & Integrations
- Data sources: [“Connect to CRM via API to fetch customer data.”]  
- Third-party services: [“Send emails via MailService API.”]  
- Data model hints: [Based on answers – e.g. “Users have profiles, purchases have timestamps.”]

# Milestones & Definition of Done
- **Milestone 1 – [Name or date]:** [Description of deliverable, e.g. “MVP with core search and login”].  
  - *Done when*: [Criteria, e.g. “User can register and search with 95% accuracy of results.”]  
- **Milestone 2 – [Name]:** [Next features].  
  - *Done when*: [“Admin dashboard implemented; user can export report CSV.”], etc.

# Acceptance Tests (Manual & Automated)
- *Manual:* [List key manual test procedures (e.g. “Test login with valid/invalid credentials.”)].  
- *Automated:* [Describe test coverage, e.g. “Unit tests cover business logic; end-to-end tests for critical flows.”].
*(These derive from the flows and requirements above.)*

# Risk Register & Mitigations
| Risk                         | Impact     | Mitigation                        |
|------------------------------|------------|-----------------------------------|
| [Risk 1, e.g. “Scope creep”] | [High/Med] | [Mitigation, e.g. “Strict change control; feature gating.”] |
| [Risk 2]                     | [ ]        | [ ]                               |
*(List items identified in questionnaire with planned mitigations.)*

# Glossary
- **Term1:** [Definition]  
- **Term2:** [Definition]  
*(Define any domain-specific terms or acronyms to avoid confusion.)*
```

Each section explicitly converts a question/answer domain into actionable content. The headings and bullet formatting ensure readability and scannability, fulfilling the “first-read, authoritative” role.

# Architectures

**Option A – CLI Wizard + Markdown Generator:**  
A simple command-line interface walks the Queen through questions (like a shell wizard) and fills out a template to produce QUEEN.md.

- _Pros:_ Easy to implement; straightforward UI; minimal dependencies.
- _Cons:_ Rigid flow; harder to handle complex branching or updates; prompt drift risk if questions lack context.
- _Failure Modes:_ User gives contradictory answers (no built-in fix); requires manual re-run to update; no schema enforcement (answers could be invalid or inconsistent).
- _Drift Prevention:_ Limited, since no learning; responses just fill blanks. Relies on careful prompt design.

**Option B – Question Graph Engine + Schema Validator + Compiler:**  
A dynamic questionnaire engine (like a state machine) with defined schema for each answer, integrated with an LLM-based compiler.

- _Pros:_ Highly structured; adaptive branching graph allows complex logic; JSON schema validation catches format errors; easier to update flow.
- _Cons:_ More complex to build (needs graph engine or DSL); potential performance overhead.
- _Failure Modes:_ Mis-specified graph rules could dead-end; requires rigorous testing of every branch (as [12†L61-L69] notes).
- _Drift Prevention:_ Schema and guided prompts enforce structure. Version-controlled question graph ensures any change is tracked.

**Option C – Constraint Solver / Decision-Record Pipeline:**  
Use constraint-solving to ensure answers don’t violate consistency rules, and auto-generate architecture decision records (ADRs). Possibly generate tests first (Intent→Tests).

- _Pros:_ Formal consistency guarantees; can highlight trade-offs automatically; strong foundations in verification. “Intent-to-tests” can yield very clear specs and catch contradictions early.
- _Cons:_ Very complex; requires mapping user answers to formal constraints; slow/overkill for small projects.
- _Failure Modes:_ Solver might find no solution to conflicting answers and stall; ADR generation needs careful templates.
- _Drift Prevention:_ By encoding requirements as constraints with provenance, any change must satisfy them or explicitly be relaxed (documented). The decision ledger ensures every choice is recorded and can be reviewed.

Each architecture balances complexity vs robustness. A is quick but brittle; B is a practical middle ground; C is the most rigorous (mitigating prompt drift by design) but also most intricate to build.

# Paradigm-Shifting Mechanisms

1. **Intent→Tests Compiler:** Instead of writing the spec first, immediately generate acceptance tests from user answers. For example, from the core flow answers produce Gherkin-style scenarios. Then use those tests as a scaffold to write the rest of QUEEN.md. This ensures requirements are **test-first** and exposes missing details (if you can’t write a test, ask more questions). Concretely, the system prompts: “From these desired features, list test cases that confirm each one.” Then the spec text is generated around those tests.
2. **Constraint Ledger (requirements provenance):** Treat each user answer as a “constraint” object with a source and confidence score. Store them in a ledger. During generation, solve or validate this constraint set (using light constraint logic) to find conflicts (e.g. requiring mutually exclusive features). This ledger can track where each requirement came from (“Queen said in Q5”) and how certain it is. Implementation: Each JSON answer becomes a formal predicate; a simple rule engine or SAT solver checks consistency and highlights conflicts.
3. **Disagreement/ADR Handling:** When user inputs conflict (or two stakeholders disagree), automatically create an Architecture Decision Record entry. For example, if one answer says “high security” and another says “public access”, the system creates an ADR summarizing the trade-off and options (e.g. “Authentication needed (pros/cons)”). Implementation: embed an ADR template generator that triggers when “unsure” or conflicting flags arise. This turns spec disagreements into documented design decisions.
4. **Spec Linting (Static Analysis):** After generating QUEEN.md, run automated analysis to catch issues: ambiguous adjectives (“fast, user-friendly”), missing subjects (“must support” without object), inconsistent naming. Use regex or NLP rules (based on [38], check for “weak terms” like “etc”). An output is a list of “spec warnings” for manual review.
5. **Ontology-backed Question Graph:** Leverage a domain ontology (or taxonomy) to ensure question consistency. For instance, if the user mentions a “widget” in one answer, later questions can ask, “How should widget behave?” (ontology links concepts). Implementation: Build a small knowledge graph of domain terms gleaned from early answers; use it to drive subsequent questions or validate terminology consistency.

Each mechanism shifts away from a flat form-filling approach and adds intelligence or structure (e.g. tests-first, constraint solving, decision records) to catch problems early and enforce rigor.

# Integration Plan with Aether

At colony (milestone) start (“egg laying”), QUEEN.md generation runs as a locked-down subprocess:

1. **Command:** Use an Aether CLI command, e.g. `ant:queen:init`. This triggers the interactive discourse.
2. **Location:** Save the output to `.aether/QUEEN.md` (preferred) or at repo root. `.aether/QUEEN.md` is loaded before `CLAUDE.md` or others.
3. **Workflow:**
    - On `ant:queen:init`, the queen answers the questions.
    - After completion, QUEEN.md is auto-committed (or flagged for commit) in version control.
    - CI checks can validate QUEEN.md format (schema + lint).
4. **Worker Integration:** All workers and tasks read `.aether/QUEEN.md` on startup. It is the first-read document (“precedence rules”). If a domain-specific spec (CLAUDE.md) also exists, QUEEN.md overrides for global constraints.
5. **Updates:** To revise, use `ant:queen:update`. This reloads the saved answers, allows edits (perhaps re-running questions or editing values), then re-compiles a new QUEEN.md diff. Old answers/versions remain in history for auditing.
6. **Rollback:** The spec is under source control (and versioned via Aether). To rollback, simply revert the commit in git or use a built-in `ant:queen:rollback [version]`. QUEEN.md should have a header with version number/date for easy rollback. Agents, when loading, can check the version of QUEEN.md (like a constitution version).
7. **Workers Read Rules:** Agents treat the _latest_ QUEEN.md as authoritative. If `.aether/QUEEN.md` exists, it supersedes `root/QUEEN.md`. Workers do not proceed if QUEEN.md validation fails (to avoid drift).

This ties QUEEN.md creation into the Aether lifecycle: it runs once per colony kickoff, can be manually invoked for changes, and has clear upgrade/rollback semantics.

# Evaluation & Verification

**Rubric:** Define measurable criteria for QUEEN.md quality. For example:

- _Completeness (20%):_ All major sections are present and populated. (Check: Are there any empty sections?)
- _Clarity (20%):_ No ambiguous terms; acceptance criteria are specific (no vague words).
- _Testability (20%):_ Each requirement has an associated acceptance check/test. (Check via keywords like “must”, “shall”, Gherkin triggers.)
- _Consistency (15%):_ No conflicting requirements (cross-checked via constraint solver).
- _Stakeholder Alignment (15%):_ Matches user answers (spot-check by asking the Queen to confirm).
- _Compliance/Quality Coverage (10%):_ All relevant NFRs (security, privacy, performance) addressed if applicable.

Assign numeric scores or pass/fail for each. Weighting can be tuned to project criticality.

**Automated Checks:**

- **Schema Validation:** Ensure answer JSON and generated QUEEN.md meet expected structure (using JSON schema, required headers).
- **Spell/Grammar:** Run a linter on QUEEN.md for typos or grammar.
- **Spec Lint:** Use rules (as above) to detect ambiguous terms or missing definition. Flag them.
- **Completeness:** Script verifies each section (like NFR, Risk) has content if related question was answered.
- **Acceptance Criteria Presence:** Scan for “GIVEN/WHEN/THEN” or bullet “- *”, ensure every In-Scope feature has at least one test statement.

**Human Review:**

- **Peer Review:** At least one developer or product manager reads QUEEN.md against the original answers. They watch for red flags:
    - Missing flows or uncovered features.
    - “TODO” or “TBD” marks left unresolved.
    - Untranslated jargon or assumptions.
- **Red Flags:** Instances like “some consideration needed”, “validate with legal” are flagged for clarification.
- **Final Approval:** The Queen (product owner) should sign off on the document as capturing intent correctly.

This multi-layer QA (automated + human) ensures QUEEN.md is actionable and reliable before agents proceed.

# Next Steps

**1-Day MVP Plan:**

- **Q1 (Morning):** Draft initial question list (10–15 core Q’s). Sketch simple CLI flow.
- **Q2:** Prototype CLI (script or notebook) to ask questions and fill a basic QUEEN.md template. Test end-to-end with sample answers.
- **Q3 (Afternoon):** Define JSON schema for answers. Validate with simple cases.
- **Q4:** Generate a rough QUEEN.md from sample data. Check against rubric: are core sections present?
- **Deploy:** Integrate into Aether as a prototype command. Demonstrate versioning and read by a dummy worker.

**1-Week Build Plan:**

- **Day 1:** (As above MVP) — Set up framework, draft schema, CLI scaffold, simple template.
- **Day 2:** Expand questions to ~25 MVP set, implement branching logic. Define label keys (question IDs). Build a small question-graph engine (or state machine) to navigate conditional flows.
- **Day 3:** Integrate LLM or template engine to synthesize QUEEN.md sections from answers (starting with Objective, Flows, etc.).
- **Day 4:** Add structured prompting: embed JSON schema into prompts for consistent output. Test with variations in answers.
- **Day 5:** Implement consistency checks (conflict detection, linting). Create automated tests for question branches and generation.
- **Day 6:** Develop acceptance-tests generation (“Intent→Tests”), risk table auto-fill, and glossary generation.
- **Day 7:** QA and iteration: refine question phrasing, improve UX, finalize integration with Aether CLI, and documentation. Get stakeholder feedback on the process.

Each day produces tangible artifacts (questionnaire prototype, schema, initial QUEEN.md, etc.). By week’s end, the Queen Discourse should run reliably, produce a versioned QUEEN.md, and meet the key requirements.

**Follow-up Questions:**

1. Which Q&A interface should we use (CLI vs GUI) for the Queen Discourse?
2. How should we prioritize features vs constraints if they conflict?
3. Should AI assistants participate in reviewing QUEEN.md as part of the workflow?

Want this in markdown?