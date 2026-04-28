---
name: prisma
description: Use when the project uses Prisma ORM for database access
type: domain
domains: [database, orm, schema]
agent_roles: [builder]
detect_files: ["schema.prisma"]
detect_packages: ["@prisma/client"]
priority: normal
version: "1.0"
---

# Prisma Best Practices

## Schema Definition

Define your data model in `schema.prisma` using clear, singular model names (`User`, not `Users`). Prisma generates the table names. Use `@id` with `@default(autoincrement())` or `@default(uuid())` for primary keys.

Add `@map` and `@@map` annotations when table or column names must differ from model names (e.g., mapping to an existing database). Use `@relation` explicitly for clarity, even when Prisma could infer it.

Define enums in the schema for fields with a fixed set of values. This provides type safety across the application.

## Migrations

Run `prisma migrate dev` in development to create and apply migrations. Run `prisma migrate deploy` in production -- it never prompts or resets. Never run `migrate dev` in production.

Review generated migration SQL before applying. Prisma generates migrations in `prisma/migrations/` -- each is a timestamped directory with a SQL file. Edit the SQL if you need data transformations alongside schema changes, but annotate why.

## Client Usage

Generate the client with `prisma generate` after schema changes. Import `PrismaClient` and create a single instance -- reuse it across your application. In serverless environments, store the client in a global variable to avoid connection exhaustion.

```
// Correct: single instance
const prisma = new PrismaClient();
export default prisma;
```

## Querying

Use `select` to fetch only the fields you need -- avoid loading entire records when you need two columns. Use `include` for relations, but be deliberate about depth. Nested includes can generate expensive joins.

Use `findUnique` when querying by unique fields (id, email) and `findFirst`/`findMany` for non-unique queries. Prefer `findUniqueOrThrow` when the record must exist.

## Performance Gotchas

Avoid N+1 queries by using `include` in a single query rather than looping and querying per item. Use `createMany` for bulk inserts instead of looping `create` calls.

For large datasets, use cursor-based pagination: `findMany({ take: 20, skip: 1, cursor: { id: lastId } })`. Offset pagination degrades on large tables.

Raw queries via `$queryRaw` are available when Prisma's query builder is insufficient, but use parameterized templates (`Prisma.sql`) to prevent injection.

## Type Safety

Prisma generates TypeScript types from your schema. Use `Prisma.UserCreateInput`, `Prisma.UserWhereInput`, etc. for type-safe function parameters. Never bypass these types with `any` -- they catch schema mismatches at compile time.

## Seeding

Define seed scripts in `prisma/seed.ts` and register them in `package.json` under `prisma.seed`. Use `upsert` in seeds to make them idempotent -- running the seed twice should not create duplicates.
