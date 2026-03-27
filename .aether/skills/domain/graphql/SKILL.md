---
name: graphql
description: Use when the project uses GraphQL for API queries and mutations
type: domain
domains: [api, query, schema]
agent_roles: [builder]
detect_files: ["*.graphql"]
detect_packages: ["apollo", "graphql"]
priority: normal
version: "1.0"
---

# GraphQL Best Practices

## Schema Design

Design your schema around the client's needs, not your database tables. Use descriptive type names and field names. Every field should have a clear purpose -- do not expose internal implementation details.

Use `ID` type for identifiers, `String` for text, `Int`/`Float` for numbers, `Boolean` for flags. Define custom scalars (DateTime, URL, Email) for domain-specific values that need validation.

Make fields non-nullable by default (`String!`). Only use nullable fields when null carries meaning (e.g., "no value set"). Lists should be non-nullable with non-nullable items: `[User!]!`.

## Queries and Mutations

Separate reads (queries) from writes (mutations). Mutations should return the affected object so clients can update their cache without a refetch. Name mutations as verbs: `createUser`, `updatePost`, `deleteComment`.

Use input types for mutation arguments: `input CreateUserInput { name: String!, email: String! }`. This keeps the schema clean and makes validation straightforward.

## Pagination

Use Relay-style cursor pagination for all list fields: `Connection` type with `edges`, `node`, `pageInfo`, and `cursor`. This handles infinite scroll, bidirectional loading, and real-time updates cleanly. Avoid offset-based pagination -- it breaks when data changes between pages.

## N+1 Problem

GraphQL's nested resolution model naturally creates N+1 queries. Use DataLoader to batch and cache database lookups within a single request. Every resolver that touches the database should go through a DataLoader.

Without DataLoader, a query fetching 50 users with their posts makes 51 database calls (1 for users + 50 for each user's posts). With DataLoader, it makes 2.

## Error Handling

Return errors in the `errors` array, not in the data. Use error `extensions` for machine-readable codes: `{ "extensions": { "code": "UNAUTHORIZED" } }`. Partial failures are valid in GraphQL -- a query can return data for some fields and errors for others.

## Security

Limit query depth (typically 7-10 levels) to prevent deeply nested queries that overwhelm the server. Limit query complexity by assigning cost values to fields and rejecting queries that exceed a budget.

Disable introspection in production unless your API is intentionally public. Introspection exposes your entire schema to attackers.

## Performance

Persisted queries reduce bandwidth and prevent arbitrary query execution. Clients send a hash instead of the full query string. Cache responses at the field level, not the query level, since GraphQL queries are highly variable.
