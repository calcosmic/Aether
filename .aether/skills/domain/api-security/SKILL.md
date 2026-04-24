---
name: api-security
description: Use when the project uses APIs that require authentication, authorization, rate limiting, or threat protection
type: domain
domains: [backend, security, api]
agent_roles: [builder]
detect_files: ["*.env", "auth.config.*", "security.*", "middleware/*.ts"]
detect_packages: ["helmet", "cors", "express-rate-limit", "jose", "passport"]
priority: normal
version: "1.0"
---

# API Security Best Practices

## OAuth2 and OpenID Connect

- Use OAuth2 Authorization Code Flow with PKCE for public clients (SPAs, mobile); never use Implicit Flow
- Validate ID tokens server-side: verify `iss`, `aud`, `exp`, and signature before trusting claims
- Use the OIDC discovery endpoint (`/.well-known/openid-configuration`) for dynamic configuration
- Store tokens securely: HttpOnly+Secure+SameSite cookies for web, Keychain/Keystore for mobile
- Implement token rotation: short-lived access tokens (15 min) with refresh token rotation on use

## JWT Best Practices

- Keep JWT payloads minimal: include `sub`, `exp`, `iat`, and `scope` -- never embed sensitive data
- Use asymmetric signing (RS256, ES256) when multiple services verify tokens; symmetric (HS256) only for single-service
- Set a reasonable `exp` claim; validate expiration on every request -- never accept tokens without it
- Implement a token revocation mechanism: maintain a denylist or use short TTLs with refresh rotation
- Validate the `aud` (audience) and `iss` (issuer) claims to prevent token confusion attacks

## Rate Limiting

- Apply rate limiting at the API gateway or reverse proxy layer: per-user and per-IP strategies
- Use sliding window counters over fixed windows to prevent burst behavior at window boundaries
- Return `429 Too Many Requests` with `Retry-After` header; do not silently drop requests
- Differentiate limits by endpoint sensitivity: stricter for auth, relaxed for read-heavy public endpoints
- Implement backoff strategies client-side: exponential jitter backoff on 429 responses

## OWASP API Security Top 10

- **API1 -- Broken Object-Level Authorization:** Validate that the authenticated user owns the requested resource
- **API2 -- Broken Authentication:** Use proven auth libraries; never roll your own crypto or session management
- **API3 -- Broken Object Property-Level Authorization:** Return only fields the user is authorized to see; filter responses
- **API4 -- Unrestricted Resource Consumption:** Enforce pagination, field selection, and query complexity limits
- **API5 -- Broken Function-Level Authorization:** Enforce RBAC at the route/handler level; deny by default
- **API8 -- Security Misconfiguration:** Disable verbose errors in production, enforce HTTPS, remove default credentials

## Security Headers

- Set `Content-Type: application/json` on all API responses; reject unexpected content types on input
- Apply `Strict-Transport-Security: max-age=31536000; includeSubDomains` for HSTS
- Set `X-Content-Type-Options: nosniff` and `X-Frame-Options: DENY` as defense-in-depth
- Use CORS with explicit `Access-Control-Allow-Origin` -- never use `*` for authenticated endpoints
- Implement `Content-Security-Policy` for any API that serves HTML documentation or UI
