---
name: nodejs
description: Use when the project uses Node.js for server-side JavaScript
type: domain
domains: [backend, server, api]
agent_roles: [builder]
detect_files: ["*.js", "*.mjs"]
detect_packages: ["express", "fastify"]
priority: normal
version: "1.0"
---

# Node.js Best Practices

## Error Handling

Never ignore errors. Every callback gets an `err` first argument -- check it. With async/await, wrap operations in try/catch and handle failures explicitly. Unhandled promise rejections crash Node.js in modern versions.

Use custom error classes that extend `Error` for different failure types (ValidationError, NotFoundError, AuthorizationError). This makes error handling in middleware clean and predictable.

## Async Patterns

Use `async/await` over raw callbacks or `.then()` chains. For parallel operations, use `Promise.all()`. For parallel with partial failure tolerance, use `Promise.allSettled()`.

Never use `async` functions as `EventEmitter` listeners without wrapping them -- thrown errors will be silently swallowed. Wrap in a try/catch and emit an error event.

## Project Structure

Organize by feature, not by technical role. Group routes, controllers, services, and models for a feature together rather than having separate `routes/`, `controllers/`, `services/` directories.

Keep `index.js` or `app.js` thin -- it should only wire up middleware and routes, not contain business logic.

## Environment and Configuration

Load configuration from environment variables using a dedicated config module. Never hardcode ports, database URLs, or API keys. Validate required env vars at startup and fail fast if any are missing.

Use `.env` files only for local development. Never commit `.env` files. Add `.env` to `.gitignore` immediately.

## Security

Validate and sanitize all input. Use parameterized queries for database access -- never interpolate user input into SQL strings. Set HTTP security headers with `helmet` middleware.

Rate-limit API endpoints to prevent abuse. Use `express-rate-limit` or equivalent. Enable CORS explicitly for known origins only, never `*` in production.

## Process Management

Handle `SIGTERM` and `SIGINT` for graceful shutdown. Close database connections, finish in-flight requests, then exit. This is critical for container deployments where processes receive termination signals during scaling.

Never use `process.exit()` in library code. Reserve it for the top-level application entry point.

## Logging

Use a structured logger (pino, winston) -- not `console.log`. Log with levels (error, warn, info, debug). Include request IDs for tracing across async operations. Never log sensitive data like passwords, tokens, or full credit card numbers.
