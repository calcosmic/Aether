---
name: redis
description: Use when the project uses Redis for caching, pub/sub, session storage, or message streaming
type: domain
domains: [backend, infrastructure, caching, messaging]
agent_roles: [builder]
detect_files: ["redis.conf", "docker-compose.yml", "*.lua"]
detect_packages: ["redis", "ioredis", "redis-py", "go-redis", "StackExchange.Redis"]
priority: normal
version: "1.0"
---

# Redis Best Practices

## Data Structure Selection

- Use `STRING` for simple key-value: session tokens, counters, cached HTML
- Use `HASH` for object-like data: user profiles, configuration maps -- `HGETALL`, `HMSET`
- Use `LIST` for ordered queues and stacks: `LPUSH`/`RPOP` for FIFO, `LPUSH`/`LPOP` for LIFO
- Use `SET` for unique membership: tags, followers -- `SADD`, `SISMEMBER`, `SINTER` for intersections
- Use `SORTED SET` for leaderboards and ranked data: `ZADD` with scores, `ZRANGEBYSCORE` for ranges
- Use `STREAM` for append-only event logs: `XADD`, `XREADGROUP` with consumer groups for processing

## Caching Strategies

- Implement cache-aside as the default: app checks Redis first, on miss fetches from DB and writes back
- Set TTL on every cached key: `SETEX key 3600 value` -- never store without expiration unless intentional
- Use key naming conventions: `{entity}:{id}:{field}` for hash fields, `{entity}:list:{query}` for result sets
- Handle cache stampede with lock-and-fetch: use `SETNX` as a mutex to prevent thundering herd
- Invalidate with explicit `DEL` or pattern-based `SCAN`+`DEL`; avoid `KEYS *` in production

## Pub/Sub and Messaging

- Use `PUBLISH`/`SUBSCRIBE` for fire-and-forget messaging: notifications, real-time updates
- Use `STREAM` with consumer groups for durable, replayable message processing
- Design messages as JSON or MessagePack; include `id`, `timestamp`, `type`, and `payload` fields
- Handle backpressure: use `XREADGROUP` with `BLOCK` and `COUNT` to control consumption rate
- Claim pending messages with `XAUTOCLAIM` or `XPENDING`+`XCLAIM` for crash recovery

## Redis Cluster and Scaling

- Use Redis Cluster for horizontal scaling when data exceeds a single node's memory
- Hash tags `{key}` ensure related keys land on the same slot for multi-key operations
- Use `CLUSTER NODES` and `CLUSTER INFO` for monitoring cluster health in automation
- Avoid `KEYS`, `SORT`, and multi-key operations across slots in cluster mode
- Configure `maxmemory-policy` -- use `allkeys-lru` for caching, `noeviction` for queues

## Modules and Extensions

- Use RediSearch for full-text search: `FT.CREATE` indexes, `FT.SEARCH` with filtering and ranking
- Use RedisJSON for native JSON document storage: `JSON.SET`, `JSON.GET` with path queries
- Use RedisTimeSeries for IoT metrics: `TS.CREATE`, `TS.ADD` with compaction rules
- Evaluate module cost vs. benefit -- modules increase memory usage and operational complexity
