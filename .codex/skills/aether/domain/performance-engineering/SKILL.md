---
name: performance-engineering
description: Use when the project uses performance-critical features requiring optimization, load testing, profiling, or Core Web Vitals improvement
type: domain
domains: [frontend, backend, infrastructure, database]
agent_roles: [builder]
detect_files: ["k6/**/*.js", "artillery.yml", "lighthouse.config.js", "wrk.lua"]
detect_packages: ["k6", "artillery", "lighthouse", "clinic", "py-spy", "pprof"]
priority: normal
version: "1.0"
---

# Performance Engineering Best Practices

## Core Web Vitals

- Target LCP < 2.5s: preload hero images, use `fetchpriority="high"`, optimize server response time (TTFB < 800ms)
- Target INP < 200ms: break long tasks into <50ms chunks with `scheduler.yield()` or `requestIdleCallback`
- Target CLS < 0.1: set explicit `width`/`height` or `aspect-ratio` on images and embeds; avoid late-loading layout shifts
- Use `loading="lazy"` on below-fold images; use `loading="eager"` only for above-fold critical resources
- Measure in the field with `web-vitals` library and Real User Monitoring (RUM); don't rely solely on Lighthouse

## Backend Profiling

- Profile CPU and memory in staging and production with continuous profilers (Pyroscope, Parca, Datadog)
- Use language-specific profilers: `pprof` (Go), `py-spy` (Python), `clinic` (Node.js), `instruments` (Swift)
- Identify hot paths: focus optimization effort on the top 5 functions by cumulative CPU time
- Measure allocation rate in GC languages -- high allocation pressure causes pause spikes
- Profile before optimizing; never optimize based on assumptions -- always measure before and after

## Database Query Optimization

- Use `EXPLAIN ANALYZE` on slow queries; look for sequential scans, nested loops, and missing indexes
- Create composite indexes matching query predicates: column order matters (equality before range)
- Implement pagination with cursor-based approach (`WHERE id > cursor LIMIT n`) instead of `OFFSET`
- Use connection pooling (PgBouncer, Prisma connection pool) to reduce connection overhead
- Batch queries and use `INSERT ... ON CONFLICT` for upserts instead of read-then-write patterns

## Caching Strategies

- Implement multi-layer caching: CDN -> application cache (Redis) -> database query cache
- Use stale-while-revalidate: serve cached content while asynchronously refreshing in the background
- Cache at the right granularity: per-user for personalized data, global for shared/reference data
- Implement cache warming for critical paths: pre-populate caches on deploy or during off-peak hours
- Monitor cache hit ratios: >90% for hot paths, investigate misses below 70%

## Load Testing

- Define performance budgets per endpoint: p50, p95, p99 latency targets under expected load
- Use k6 for developer-friendly load testing: script in JS, run locally or in CI, export to Grafana
- Use Artillery for scenario-based testing: multi-step user flows with think time and data variability
- Start with smoke tests (1-5 VUs), then ramp to average load, then stress/spike tests for resilience
- Run load tests in a staging environment that mirrors production topology and data volume
