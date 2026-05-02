---
phase: 89
slug: gate-self-healing-smart-planning
status: verified
threats_open: 0
asvs_level: standard
created: 2026-05-02
---

# Phase 89 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| gate-results JSON -> Fixer context | Gate failure data crosses into agent prompt | Low — sanitized before injection |
| Fixer output -> gate-results update | Agent response mutates colony state | Low — Go runtime validates gate names |
| Config -> attempt cap | User-controlled config value | Low — bounds checked in runtime |
| User input -> confidence target | CLI flag value for Oracle | Low — validated 1-100 range |
| Repo files -> brief synthesis | File content from repo displayed to user | Low — read-only, not executed |
| User editor -> brief content | User modifies launch brief | None — advisory only |
| Config file -> provider URLs | URLs for LLM provider connections | Low — scheme validation enforced |

---

## Threat Register

| Threat ID | Category | Component | Disposition | Mitigation | Status |
|-----------|----------|-----------|-------------|------------|--------|
| T-89-01 | Tampering | Gate-results -> Fixer prompt | mitigate | SanitizeSignalContent applied to Detail/FixHint in fixer_dispatch.go and unblock_cmd.go (commit 4e3840a7) | closed |
| T-89-02 | Tampering | Attempt cap bypass via file edit | accept | Developer CLI — same trust as COLONY_STATE.json | closed |
| T-89-03 | Spoofing | Fixer reports false resolution | mitigate | resolveFixedGates validates gate names exist in current results | closed |
| T-89-04 | DoS | Fixer infinite repair loops | mitigate | Default attempt cap (1) + circuit breaker (LOOP-03) | closed |
| T-89-05 | Spoofing | Confidence inflation | mitigate | Oracle rubric penalizes single-source claims; target requires multi-source evidence | closed |
| T-89-06 | DoS | --confidence-target 99 unbounded loops | mitigate | max_iterations cap (default 8) enforced; max_iterations_reached status returned | closed |
| T-89-07 | Tampering | Init brief injection via repo content | accept | Content displayed to user for review, not executed | closed |
| T-89-08 | Info Disclosure | Gate status reveals internal gate names | accept | Developer CLI — gate names are not sensitive | closed |
| T-89-09 | Tampering | Callback URL injection via config | mitigate | validateCallbackURL + validateCallbackURLScheme reject non-http(s) schemes | closed |
| T-89-10 | Spoofing | BaseURL manipulation | accept | Provider URL is user-controlled config — same trust as API keys | closed |
| T-89-05r | DoS | Tighter gate causes more iterations | mitigate | max_iterations cap still enforced; users can lower target if needed | closed |

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-89-01 | T-89-02 | Developer CLI tool — user has filesystem access, attempt tracking uses atomic JSON writes | gsd-security-auditor | 2026-05-02 |
| AR-89-02 | T-89-07 | Brief content read from repo, displayed to user for approval, never executed | gsd-security-auditor | 2026-05-02 |
| AR-89-03 | T-89-08 | Gate names in /ant-status are not sensitive for a developer CLI tool | gsd-security-auditor | 2026-05-02 |
| AR-89-04 | T-89-10 | Provider baseURL is user-controlled config with same trust model as API keys | gsd-security-auditor | 2026-05-02 |

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-05-02 | 11 | 11 | 0 | gsd-security-auditor + inline fix |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-05-02
