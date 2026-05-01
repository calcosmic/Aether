---
name: typescript
description: Use when the project uses TypeScript for type-safe JavaScript development
type: domain
domains: [typing, language, safety]
agent_roles: [builder]
detect_files: ["tsconfig.json", "*.ts"]
priority: normal
version: "1.0"
---

# TypeScript Best Practices

## Configuration

Enable strict mode in `tsconfig.json` (`"strict": true`). This turns on `strictNullChecks`, `noImplicitAny`, `strictFunctionTypes`, and other checks that catch real bugs. Loosening strict mode after the fact is easy; adding it to an existing loose codebase is painful.

Set `"noUncheckedIndexedAccess": true` to make array and object index access return `T | undefined` instead of `T`. This catches a class of runtime errors where you assume an index exists.

## Type Design

Prefer `interface` for object shapes that may be extended. Use `type` for unions, intersections, and computed types. Both work for most cases, but `interface` gives better error messages and supports declaration merging.

Make impossible states unrepresentable. Use discriminated unions instead of optional fields:

```typescript
// Avoid: what does it mean when both are undefined?
type Result = { data?: Data; error?: Error };

// Prefer: exactly one state at a time
type Result = { status: "ok"; data: Data } | { status: "error"; error: Error };
```

## Avoid These Patterns

Never use `any` -- use `unknown` when the type is truly unknown, then narrow it with type guards. Every `any` disables type checking for everything it touches.

Avoid type assertions (`as Type`) unless you genuinely know more than the compiler. Prefer type guards (`if ('kind' in obj)`) that the compiler can verify.

Do not use `!` (non-null assertion) to silence nullable warnings. Fix the logic so the compiler can see the value is defined, or handle the null case.

## Generics

Use generics to make functions and types reusable without losing type information. Name generic parameters descriptively when the context is not obvious: `TItem` over `T` when there are multiple type parameters.

Constrain generics with `extends` to ensure minimum capability: `<T extends { id: string }>` guarantees `T` has an `id` field.

## Utility Types

Use built-in utility types: `Partial<T>` for optional fields, `Required<T>` for mandatory, `Pick<T, K>` to select fields, `Omit<T, K>` to exclude, `Record<K, V>` for dictionaries. These are clearer than manual type construction.

## Enums

Prefer `as const` objects or string literal unions over `enum`. Enums generate runtime code and have subtle behavior differences between string and numeric variants. `as const` objects are pure types with zero runtime cost.

## Error Handling

Type your error shapes. Define error types and use them in `Result` patterns. Catch blocks receive `unknown` in strict mode -- narrow before accessing properties.
