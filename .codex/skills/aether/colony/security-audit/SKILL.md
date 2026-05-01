---
name: security-audit
description: Use when completed work or security-sensitive code needs a structured vulnerability and threat mitigation audit
type: colony
domains: [security, compliance, threat-analysis, vulnerability-assessment]
agent_roles: [gatekeeper, auditor, watcher]
workflow_triggers: [continue]
task_keywords: [security, auth, authorization, vulnerability, threat, secret]
priority: normal
version: "1.0"
---

# Security Audit

## Purpose

Retroactive security audit that examines completed phase implementations to verify threat mitigations are effective and identify vulnerabilities. Maps findings to OWASP Top 10 and MITRE ATT&CK frameworks so results are standardized and actionable. Produces a SECURITY.md artifact with prioritized remediation guidance.

## When to Use

- After completing a phase that handles authentication, authorization, data storage, or external integrations
- Before deploying to production or a public-facing environment
- When the colony or a human requests a security review of specific code
- After a security-sensitive dependency is updated
- Periodically as part of a security maintenance cadence

## Instructions

### 1. Define Audit Scope

Determine what to audit:

- **Phase scope**: All files changed in a specific phase
- **Feature scope**: All files related to a named feature or module
- **Path scope**: Specific directories or file patterns
- **Full scope**: All source files in the project

Record the scope clearly. Include the commit range if git-based scoping is used.

### 2. Threat Modeling

Before line-by-line review, build a threat model for the scope:

**A. Identify trust boundaries**
- Where does the system accept external input? (APIs, forms, file uploads, websockets)
- Where does it cross privilege levels? (anonymous -> authenticated -> admin)
- Where does it interact with external systems? (databases, third-party APIs, file system)

**B. Identify assets**
- What data is stored or processed? (PII, credentials, financial data, business logic)
- What capabilities are exposed? (admin functions, data export, configuration changes)
- What infrastructure is involved? (servers, containers, cloud services)

**C. Identify threat actors**
- Unauthenticated external users
- Authenticated users (standard and elevated privilege)
- Internal threat actors (developers, admins)
- Automated attackers (bots, scanners)

**D. Enumerate threats using STRIDE**

| Category | Question |
|----------|----------|
| Spoofing | Can an attacker impersonate a legitimate user or service? |
| Tampering | Can an attacker modify data in transit or at rest? |
| Repudiation | Can actions be denied due to missing audit trails? |
| Information Disclosure | Can sensitive data be exposed to unauthorized parties? |
| Denial of Service | Can an attacker overwhelm the system? |
| Elevation of Privilege | Can a user gain higher access than intended? |

### 3. Code-Level Audit

Systematically scan the codebase for vulnerability classes:

**A. Injection (OWASP A03:2021)**
- SQL injection: String concatenation or template literals in queries
- NoSQL injection: Unsanitized object keys passed to database operations
- Command injection: User input in shell commands or eval statements
- LDAP injection: Unescaped input in LDAP queries
- XSS: Unescaped user input rendered in HTML (reflected, stored, DOM-based)

**B. Broken Authentication (OWASP A07:2021)**
- Weak password policies or missing rate limiting
- Session fixation or predictable session tokens
- Missing multi-factor authentication for sensitive operations
- Credentials stored in plaintext or with weak hashing

**C. Broken Access Control (OWASP A01:2021)**
- Missing authorization checks on endpoints
- Insecure direct object references (IDOR)
- Privilege escalation through parameter manipulation
- CORS misconfiguration allowing unauthorized origins

**D. Security Misconfiguration (OWASP A05:2021)**
- Default credentials or keys left in place
- Unnecessary features enabled (directory listing, debug mode)
- Missing security headers (CSP, HSTS, X-Frame-Options)
- Verbose error messages exposing internal details

**E. Sensitive Data Exposure (OWASP A02:2021)**
- Data transmitted without encryption
- Sensitive data logged or stored in plaintext
- Weak cryptographic algorithms (MD5, SHA1 for passwords)
- Missing data classification and handling rules

**F. Vulnerable Dependencies (OWASP A06:2021)**
- Known CVEs in direct or transitive dependencies
- Outdated packages with security patches available
- Dependencies pulled from untrusted registries

**G. Security Logging & Monitoring (OWASP A09:2021)**
- Missing audit logs for security events (login, access denied, data changes)
- Logs that contain sensitive data
- No alerting on anomalous patterns

**H. Server-Side Request Forgery (OWASP A10:2021)**
- User-controlled URLs fetched server-side
- Missing allowlist for outbound requests
- Internal services exposed through SSRF

### 4. Map Findings to Frameworks

For each vulnerability found, create a structured entry:

**OWASP Mapping:**
- Reference the specific OWASP Top 10 category and year
- Note the CWE (Common Weakness Enumeration) ID if applicable

**MITRE ATT&CK Mapping:**
- Identify the tactic (Initial Access, Execution, Persistence, etc.)
- Identify the technique and sub-technique
- This helps defenders understand the attack chain

| Finding Example | OWASP | CWE | MITRE Tactic | MITRE Technique |
|----------------|-------|-----|-------------|-----------------|
| SQL injection in search | A03:2021 | CWE-89 | Execution | T1190 Exploit Public-Facing App |
| Hardcoded API key | A07:2021 | CWE-798 | Credential Access | T1078 Valid Accounts |
| Missing auth check | A01:2021 | CWE-862 | Privilege Escalation | T1548 Abuse Elevation Control |

### 5. Classify Severity

Rate each finding using CVSS-like criteria adapted for code review:

| Severity | Criteria | Response Time |
|----------|----------|---------------|
| Critical | Remote exploitation, no auth required, full system compromise | Immediate |
| High | Remote exploitation with auth, significant data exposure | Within 24 hours |
| Medium | Local exploitation, limited data exposure, requires specific conditions | Within 1 week |
| Low | Informational, best practice violations, defense-in-depth improvements | Next release cycle |

### 6. Produce SECURITY.md

```markdown
# Security Audit -- {Scope}
**Date:** {ISO date}
**Auditor:** {agent name}
**Scope:** {description of what was audited}
**Commit Range:** {from..to if applicable}

## Executive Summary
{3-5 sentences: overall security posture, critical findings count, top risks}

## Threat Model
### Trust Boundaries
{list of boundaries identified}

### Assets at Risk
{list of assets and their sensitivity classification}

### Threat Actors
{list of relevant threat actors}

## Findings

### Critical ({count})
{Each finding with: title, OWASP reference, CWE, MITRE mapping, description, affected code location, proof of concept, remediation}

### High ({count})
{Same structure}

### Medium ({count})
{Same structure}

### Low ({count})
{Same structure}

## Dependency Audit
| Package | Version | Known CVEs | Severity | Recommendation |
|---------|---------|-----------|----------|----------------|
| ... | ... | ... | ... | ... |

## Positive Findings
{Security controls that are correctly implemented -- acknowledge good practices}

## Remediation Priority
| Priority | Finding | Effort | Impact |
|----------|---------|--------|--------|
| 1 | ... | ... | ... |

## Compliance Notes
{Any regulatory or compliance implications (GDPR, SOC2, HIPAA, etc.)}
```

## Key Patterns

### Credential Detection Regex Patterns

Scan for these patterns in source code:
- `/sk-[a-zA-Z0-9]{32,}/` -- OpenAI-style API keys
- `/AKIA[0-9A-Z]{16}/` -- AWS access keys
- `/-----BEGIN (RSA |EC )?PRIVATE KEY-----/` -- Private keys
- `/password\s*[:=]\s*['"][^'"]{8,}/i` -- Hardcoded passwords
- `/api[_-]?key\s*[:=]\s*['"][^'"]+/i` -- API key assignments
- `/secret\s*[:=]\s*['"][^'"]+/i` -- Secret assignments

### Security Headers Checklist

Verify these headers are set for web applications:
- `Content-Security-Policy` -- Prevents XSS and data injection
- `Strict-Transport-Security` -- Forces HTTPS
- `X-Content-Type-Options: nosniff` -- Prevents MIME type sniffing
- `X-Frame-Options: DENY` or `SAMEORIGIN` -- Prevents clickjacking
- `Referrer-Policy` -- Controls referrer information leakage
- `Permissions-Policy` -- Restricts browser features

### Input Validation Checklist

For every point where external input is accepted:
- Is the input type validated (string, number, array)?
- Is the length bounded?
- Are special characters handled or escaped?
- Is the input sanitized before use in queries, commands, or HTML?
- Is the validated input used consistently (not re-reading raw input)?

## Output Format

Produces `SECURITY.md` in the project root or specified output path.

## Examples

### Example 1: Audit a phase

```
Security audit phase 5 -- payment processing
```

Threat models the payment flow, audits all payment-related code for PCI-relevant vulnerabilities, maps findings to OWASP, produces SECURITY.md.

### Example 2: Audit specific paths

```
Security audit src/auth/ src/api/ src/middleware/
```

Focused audit on authentication and API layers with full threat model for the auth boundary.

### Example 3: SECURITY.md excerpt

```markdown
### Critical (1)

**CRIT-01: SQL Injection in User Search**
- **OWASP:** A03:2021 -- Injection
- **CWE:** CWE-89 (SQL Injection)
- **MITRE:** T1190 -- Exploit Public-Facing Application
- **Location:** `src/api/search.ts:34`
- **Description:** User-supplied `query` parameter is concatenated directly
  into a SQL WHERE clause without parameterization.
- **Proof of Concept:** `GET /api/search?query=' OR 1=1 --` returns all records.
- **Remediation:** Use parameterized queries:
  ```typescript
  db.query('SELECT * FROM users WHERE name LIKE $1', [`%${query}%`]);
  ```

### Positive Findings

- Password hashing uses bcrypt with cost factor 12 (strong)
- JWT tokens have appropriate expiration (1 hour access, 7 day refresh)
- Rate limiting is applied to all authentication endpoints
- CORS is restricted to known origins only
```
