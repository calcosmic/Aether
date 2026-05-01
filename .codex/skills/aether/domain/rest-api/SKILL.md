---
name: rest-api
description: Use when the project designs or consumes RESTful HTTP APIs
type: domain
domains: [api, http, design]
agent_roles: [builder]
detect_files: ["openapi.*", "swagger.*"]
priority: normal
version: "1.0"
---

# REST API Best Practices

## Resource Design

Use nouns for endpoints, not verbs: `/users` not `/getUsers`. Let HTTP methods convey the action: `GET` (read), `POST` (create), `PUT` (full replace), `PATCH` (partial update), `DELETE` (remove).

Use plural nouns for collections (`/users`), singular identifiers for items (`/users/123`). Nest resources for clear ownership: `/users/123/orders` for orders belonging to user 123. Avoid nesting deeper than two levels -- flatten with query parameters instead.

## HTTP Status Codes

Return the correct status code -- clients depend on it for error handling:
- `200` for successful GET/PUT/PATCH
- `201` for successful POST (resource created)
- `204` for successful DELETE (no content)
- `400` for bad request (validation errors)
- `401` for unauthenticated
- `403` for unauthorized (authenticated but not allowed)
- `404` for not found
- `409` for conflict (duplicate, version mismatch)
- `422` for unprocessable entity (valid JSON, invalid semantics)
- `500` for server errors (never expose internals)

## Request and Response Format

Use JSON for request and response bodies. Use `camelCase` for JSON field names (JavaScript convention) or `snake_case` (Python convention) -- be consistent within the API, never mix.

Return consistent error responses: `{"error": "message", "code": "VALIDATION_FAILED", "details": [...]}`. Clients should be able to parse errors programmatically, not just read human messages.

## Pagination

Always paginate collection endpoints. Use cursor-based pagination (`?cursor=abc123&limit=20`) for large or frequently changing datasets. Include pagination metadata in the response: `total`, `next_cursor`, `has_more`.

## Versioning

Version your API from day one: `/api/v1/users`. Use URL-based versioning for simplicity. Header-based versioning (`Accept: application/vnd.api.v1+json`) is cleaner but harder to test and debug.

## Authentication

Use bearer tokens (JWT or opaque) in the `Authorization` header. Never pass tokens in URL query parameters -- they appear in logs and browser history. Use short-lived access tokens with refresh tokens for longer sessions.

## Documentation

Maintain an OpenAPI specification (`openapi.yaml`). Generate it from code annotations or write it manually, but keep it in sync with the implementation. Stale API docs are worse than no docs -- they create false confidence.

## Rate Limiting

Implement rate limiting on all public endpoints. Return `429 Too Many Requests` with `Retry-After` header. Use sliding window counters, not fixed windows, to prevent burst abuse at window boundaries.
