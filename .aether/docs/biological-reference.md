# Biological Reference: Ant Colony Research & Naming Taxonomy

This document collects 40+ biologically grounded resources and a complete ant-biologically-accurate command/milestone naming system for Aether.

---

## Part 1: Research Sources

### Foundational / Synthesis

1. Holldobler & Wilson, *The Ants* (book overview/reference).
2. Czaczkes et al. (2015) review: trail pheromones as regulation/feedback in colony organization (PDF).
3. Muscedere & Traniello (2012) division of labor in *Pheidole* (open access).
4. Lillico-Ouachour & Abouheif (2017) caste ratios / regulation in *Pheidole* (review).
5. de Pletincx et al. (2021) worker polymorphism evolution + social traits (PDF).

### Trail Pheromones / Foraging Decisions

6. Czaczkes (2012) complexity of foraging trails (open access).
7. Czaczkes et al. (2015) pheromone deposition adjusts with conditions (Royal Society).
8. Oberhauser et al. (2020) pheromone trails vs subjective reward (Frontiers).
9. Pheromone trail following in *Lasius niger* (2016, Wiley).
10. Sakiyama et al. (2025) naive ants moving faster when encountering returning flow / trail context (Sci Rep).

### Nest Architecture / Excavation / Ventilation

11. Tschinkel (2003) subterranean ant nests; plaster casting; architecture units (AntWiki PDF).
12. Tschinkel (2015) architecture of subterranean nests (AntWiki PDF).
13. Sankovitz et al. (2021) nest architecture shaped by adaptation/environment (Sci Rep).
14. Yang et al. (2022) review of ant nests (MDPI Buildings).
15. Yang et al. (2022) ventilation simulation in underground nests (MDPI Sustainability).
16. Bollazzi et al. (2021) CO2 levels + ventilation in *Acromyrmex* nests (Royal Society Open Science).
17. "Nest Entrance Architecture and regulation of foraging activity" (2025, Wiley).
18. "Colony demographics shape nest construction" (2024, eLife reviewed preprint).

### Brood Effects / Caste Aggregation / Development

19. Sempo et al. (2006) brood influences aggregation patterns in dimorphic ants (Springer).
20. Ravary et al. (2007) experience can generate lasting division of labor (Current Biology).
21. Enzmann et al. (2021) age-related division of labor appears early (PDF).
22. Khajehnejad et al. (2025) age polyethism can emerge from social learning (open access).
23. Trible et al. (2023) caste differentiation mutant; queen-like traits in workers (open access).
24. Hughes et al. (2003) genetic basis of worker caste polymorphism (PNAS).

### Nestmate Recognition / Cuticular Hydrocarbons (CHCs) / Communication Interference

25. Yusuf et al. (2010) CHCs + nestmate discrimination (PubMed).
26. Walsh et al. (2020) CHCs heritable; aggression/nestmate recognition (Royal Society).
27. "Learning and perceptual similarity among cuticular hydrocarbons" (2011, J. Insect Physiology).
28. "A Review of Ant Cuticular Hydrocarbons" (review listing CHC diversity) (ResearchGate).
29. Wittke et al. (2022) acclimation / communication interference; CHC functions intertwined (Functional Ecology).
30. Jiang et al. (2026) oxidizing pollutants disrupt recognition; CHCs in recognition/division of labor (PNAS).

### Trophallaxis (Social Fluid Exchange)

31. LeBoeuf et al. (2016) trophallaxis transfers growth/hormone regulators; communal control (open access).
32. Meurville & LeBoeuf (2021) trophallaxis forms/functions/evolution (AntWiki PDF).
33. Lenoir (1982) antennal communication around trophallaxis information transfer (ScienceDirect).
34. eLife press explainer on trophallaxis communication (background + pointers).
35. Myrmecological News blog summary pointing to review literature on trophallaxis (secondary).
36. Wired coverage (secondary) referencing Lausanne work on trophallaxis molecules (use as pointer only).

### Additional Biology/Behavior Pointers

37. Scientific American overview of pheromones (historical, general definition).
38. "The Guests of Ants" sample (division of labor framing in ants; background).
39. "Ants Sense, and Follow, Trail Pheromones of Ant Community Members" (pointer/landing).
40. "How brood influences caste aggregation patterns..." duplicate access route (landing).
41. "Nest-Building Behaviour in Ants..." (recent landing; verify underlying primary refs if used).

> **Note:** For stricter "primary-only," drop the obvious secondary/landing items (#35-36, #41, and any ResearchGate-only landings) and replace them with the underlying journal PDFs/DOIs.

---

## Part 2: Ant-Biologically-Accurate Command Naming System

### Naming Rules

- **Prefix** = caste/role (real in ant biology): queen, alate, worker, forager, scout, nurse, soldier, mason, undertaker, sentinel
- **Verb** = behavior (documented): trail, recruit, troph, brood, excavate, ventilate, recognize, quarantine
- **Object** = dev artifact: repo, deps, api, patch, build, test, release, docs, bench, log

### Recommended Shape

- CLI style: `colony <role>:<verb> [target]`
- Slash style: `/<role>:<verb> <target>`

---

### 1) Scout Commands (recon, unknowns, discovery)

| Command | Purpose |
|---|---|
| `scout:scan` | Repo surface scan |
| `scout:probe` | Single uncertainty check |
| `scout:trace` | Follow call chain |
| `scout:map` | Dependency map |
| `scout:diff` | Compare branches/builds |
| `scout:grep` | Targeted search |
| `scout:catalog` | Inventory files/assets |
| `scout:triage` | Classify issues |
| `scout:repro` | Reproduction script |
| `scout:scope` | What's in/out |
| `scout:verify` | Fact-check a claim |
| `scout:baseline` | Capture current metrics |

### 2) Forager Commands (fetch inputs, examples, references, fixtures)

| Command | Purpose |
|---|---|
| `forager:fetch` | Pull examples/specs |
| `forager:harvest` | Collect logs/traces |
| `forager:gather` | Bundle test assets |
| `forager:mirror` | Cache deps |
| `forager:pin` | Lock versions |
| `forager:seed` | Starter config/templates |
| `forager:hydrate` | Install deps |
| `forager:pack` | Vendor assets |
| `forager:stash` | Store artifacts |

### 3) Worker Commands (implementation)

| Command | Purpose |
|---|---|
| `worker:build` | Build |
| `worker:patch` | Patch |
| `worker:refactor` | Refactor |
| `worker:wire` | Integrate modules |
| `worker:glue` | Adapters |
| `worker:generate` | Scaffold |
| `worker:migrate` | Migrate |
| `worker:upgrade` | Upgrade |
| `worker:optimize` | Optimize |
| `worker:document` | Document |
| `worker:instrument` | Logs/metrics |
| `worker:benchmark` | Benchmark |

### 4) Nurse Commands (brood care = nurturing early-stage code/PRs)

| Command | Purpose |
|---|---|
| `nurse:incubate` | Feature branch incubation |
| `nurse:feed` | Add missing inputs/tests |
| `nurse:prune` | Remove dead code paths |
| `nurse:stabilize` | Reduce flakiness |
| `nurse:coach` | Lint/style fixes |
| `nurse:handoff` | Clean PR handover |
| `nurse:guardrails` | Add invariants |
| `nurse:heal` | Fix failing tests quickly |

### 5) Mason Commands (nest building = architecture, structure, infra)

| Command | Purpose |
|---|---|
| `mason:excavate` | New module/dir layout |
| `mason:chamber` | Subsystem boundary |
| `mason:tunnel` | Pipeline/CI routing |
| `mason:ventilate` | Performance/throughput work |
| `mason:reinforce` | Hardening |
| `mason:seal` | Interface freeze |
| `mason:cast` | Reproducible build "cast" |
| `mason:foundation` | Bootstrap |
| `mason:relocate` | Move components safely |

### 6) Pheromone/Trail Commands (shared state, routing, prioritization signals)

| Command | Purpose |
|---|---|
| `trail:mark` | Set priority |
| `trail:boost` | Raise confidence/urgency |
| `trail:dampen` | De-prioritize |
| `trail:follow` | Execute critical path |
| `trail:branch` | Explore alternative path |
| `trail:merge` | Consolidate findings |
| `trail:cleanup` | Remove stale signals |
| `trail:route` | Choose pipeline path |

### 7) Recognition Commands (CHC-inspired = identity/compat checks)

| Command | Purpose |
|---|---|
| `recognize:nestmate` | Same-environment check |
| `recognize:drift` | Config drift detection |
| `recognize:compat` | ABI/API compatibility |
| `recognize:fingerprint` | Artifact hash |
| `recognize:provenance` | Where built/from what |
| `recognize:attest` | SBOM/attestation step |
| `recognize:policy` | Rules compliance |

### 8) Sentinel Commands (tests, gates, watch)

| Command | Purpose |
|---|---|
| `sentinel:test` | Test |
| `sentinel:unit` | Unit tests |
| `sentinel:integration` | Integration tests |
| `sentinel:e2e` | End-to-end tests |
| `sentinel:soak` | Soak tests |
| `sentinel:load` | Load tests |
| `sentinel:gate` | Release gate |
| `sentinel:watch` | Tail logs/metrics |
| `sentinel:alarm` | Notify on regressions |
| `sentinel:assert` | Invariant checks |
| `sentinel:canary` | Canary deploy |
| `sentinel:rollback-check` | Rollback check |

### 9) Undertaker Commands (dead code, cleanup, postmortems)

| Command | Purpose |
|---|---|
| `undertaker:collect` | Dead code inventory |
| `undertaker:bury` | Delete safely |
| `undertaker:quarantine` | Isolate |
| `undertaker:autopsy` | Root cause |
| `undertaker:purge` | Cache/artifact cleanup |
| `undertaker:deprecate` | Soft removal |
| `undertaker:archive` | Close issue/milestone |

### 10) Soldier Commands (defense = security, reliability under attack)

| Command | Purpose |
|---|---|
| `soldier:shield` | Hardening pass |
| `soldier:scan` | Security scan |
| `soldier:patch` | CVE response |
| `soldier:lockdown` | Freeze deps |
| `soldier:rate-limit` | Rate limiting |
| `soldier:sandbox` | Sandboxing |
| `soldier:contain` | Incident containment |
| `soldier:drill` | Game-day |

### 11) Queen Commands (decisions, scope, release authority)

| Command | Purpose |
|---|---|
| `queen:decree` | Set direction |
| `queen:prioritize` | Prioritize |
| `queen:approve` | Merge/release approval |
| `queen:scope` | Scope |
| `queen:freeze` | Feature freeze |
| `queen:unfreeze` | Unfreeze |
| `queen:allocate` | Assign "castes" |
| `queen:standardize` | Naming/structure rules |

### 12) Alate Commands (nuptial flight = release/launch branching)

| Command | Purpose |
|---|---|
| `alate:flight` | Launch |
| `alate:tag` | Tag |
| `alate:ship` | Ship |
| `alate:announce` | Announce |
| `alate:handover` | Ops handoff |
| `alate:seed-new-nest` | Start next major |
| `alate:hotfix-flight` | Hotfix |
| `alate:rollback-flight` | Rollback |

---

## Part 3: Anthill Milestone Names

Biologically plausible metaphors for project milestones:

| Milestone | Meaning | Biological Basis |
|---|---|---|
| **First Mound** | First runnable | Initial nest construction |
| **Open Chambers** | Feature work underway | Excavation of functional chambers |
| **Brood Stable** | Tests consistently green | Brood care = healthy larvae development |
| **Ventilated Nest** | Perf/latency acceptable | Ants actively manage nest airflow |
| **Sealed Chambers** | Interfaces frozen | Completed chambers sealed from tunnels |
| **Crowned Anthill** | Release | Mature colony with established queen |
| **New Nest Founded** | Next major version line | Nuptial flight → new colony founding |
