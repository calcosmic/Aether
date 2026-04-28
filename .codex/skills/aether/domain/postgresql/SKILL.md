---
name: postgresql
description: Use when the project uses PostgreSQL for relational data storage
type: domain
domains: [database, sql, data]
agent_roles: [builder]
detect_files: ["*.sql"]
detect_packages: ["pg", "psycopg2"]
priority: normal
version: "1.0"
---

# PostgreSQL Best Practices

## Schema Design

Design tables in third normal form unless you have a measured performance reason to denormalize. Use meaningful table and column names -- `user_email` over `ue`. Always define primary keys, preferably `BIGSERIAL` or `UUID` depending on scale requirements.

Add `created_at` and `updated_at` timestamps to every table. Use `TIMESTAMPTZ` (timestamp with time zone), never `TIMESTAMP` without timezone -- timezone-naive timestamps cause bugs when servers span time zones.

## Indexing

Create indexes to support your query patterns, not speculatively. Use `EXPLAIN ANALYZE` to verify that queries use indexes. A missing index on a frequently-queried column is a common source of slow pages.

Use partial indexes for queries that filter on a condition: `CREATE INDEX idx_active_users ON users(email) WHERE active = true`. Use composite indexes with columns ordered by selectivity (most selective first).

Do not over-index. Every index slows down writes. Remove unused indexes identified by `pg_stat_user_indexes` where `idx_scan = 0`.

## Queries

Always use parameterized queries -- never concatenate user input into SQL strings. This prevents SQL injection and lets PostgreSQL cache query plans.

Use `LIMIT` and `OFFSET` for simple pagination, but switch to cursor-based pagination (keyset pagination using `WHERE id > last_seen_id ORDER BY id LIMIT N`) for large datasets. `OFFSET` scans and discards rows, getting slower as pages increase.

## Migrations

Write migrations as idempotent scripts when possible. Use `IF NOT EXISTS` for `CREATE TABLE` and `CREATE INDEX`. Always test migrations against a copy of production data before deploying.

For large tables, avoid `ALTER TABLE ... ADD COLUMN ... DEFAULT` on PostgreSQL versions before 11 -- it rewrites the entire table. Add the column as nullable first, backfill data, then add the constraint.

## Transactions

Keep transactions short. Long-running transactions hold locks and block other operations. Never leave a transaction open while waiting for user input or external API calls.

Use `SERIALIZABLE` isolation only when you need it -- `READ COMMITTED` (the default) is sufficient for most workloads and has better concurrency.

## Connection Management

Use a connection pool (PgBouncer, or your driver's built-in pool). PostgreSQL creates a process per connection -- too many connections waste memory and degrade performance. Set pool size based on `max_connections` and your application's concurrency needs.

## Backups

Use `pg_dump` for logical backups and `pg_basebackup` for physical backups. Test restoring from backups regularly. An untested backup is not a backup.
