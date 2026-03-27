---
name: react
description: Use when the project uses React for building user interfaces
type: domain
domains: [frontend, components, ui]
agent_roles: [builder]
detect_files: ["*.jsx", "*.tsx"]
detect_packages: ["react"]
priority: normal
version: "1.0"
---

# React Best Practices

## Component Design

Prefer function components with hooks over class components. Keep components small and focused on a single responsibility. If a component file exceeds 200 lines, it likely needs splitting.

Extract custom hooks when you find yourself reusing stateful logic across components. Name hooks with the `use` prefix and keep them in a dedicated `hooks/` directory.

## State Management

Use `useState` for local UI state, `useReducer` for complex state transitions, and context for cross-cutting concerns like themes or auth. Avoid putting everything in global state -- most state is local.

Lift state up only when two sibling components need the same data. If prop drilling goes deeper than 2-3 levels, consider context or composition patterns instead.

## Performance Gotchas

Wrap expensive computations in `useMemo` and callback references in `useCallback`, but only when you measure an actual performance problem. Premature memoization adds complexity without benefit.

Never create new objects or arrays inline as props -- this triggers unnecessary re-renders. Define defaults outside the component body.

Avoid anonymous functions in `useEffect` dependency arrays. Extract effect logic into named functions for clarity and stable references.

## Keys and Lists

Always use stable, unique identifiers as keys in lists -- never array indices (unless the list is static and will never reorder). Bad keys cause subtle bugs with component state getting mixed up between items.

## Error Boundaries

Wrap major UI sections in error boundaries. A crash in a sidebar widget should not take down the entire page. Use `react-error-boundary` or build your own class component for this.

## File Organization

Group by feature, not by type. Colocate a component with its tests, styles, and hooks rather than having separate `components/`, `styles/`, and `tests/` trees. This makes features self-contained and easy to move or delete.
