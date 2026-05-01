---
name: svelte
description: Use when the project uses Svelte or SvelteKit for building reactive UIs
type: domain
domains: [frontend, components, ui]
agent_roles: [builder]
detect_files: ["*.svelte", "svelte.config.*"]
detect_packages: ["svelte"]
priority: normal
version: "1.0"
---

# Svelte Best Practices

## Reactivity Model

Svelte uses compile-time reactivity. Variables declared at the top level of a component are reactive automatically -- assignments trigger re-renders. Use `$:` for derived values and reactive statements. Remember that only assignments trigger reactivity: `array.push(item)` does NOT trigger updates -- use `array = [...array, item]` instead.

## Component Structure

Keep components focused on one concern. A Svelte component file should ideally contain its own script, markup, and styles. Use `<script>` at the top, markup in the middle, and `<style>` at the bottom.

## Stores

Use writable stores for shared state across components. Subscribe with the `$store` syntax in components for automatic subscription cleanup. For derived data, use `derived()` stores rather than computing values in each component.

Keep stores in a `stores/` directory. Export store creation functions rather than store instances when stores need initialization parameters.

## Props and Events

Declare props with `export let propName`. Provide default values for optional props. Use `createEventDispatcher()` for component events, and forward DOM events with `on:click` without a handler.

Use two-way binding (`bind:value`) sparingly -- it makes data flow harder to trace. Prefer explicit prop + event patterns for complex components.

## SvelteKit Specifics

SvelteKit uses file-based routing in `src/routes/`. Page components are `+page.svelte`, layouts are `+layout.svelte`, and server-side logic lives in `+page.server.ts` or `+server.ts`.

Load data with `load` functions in `+page.ts` (universal) or `+page.server.ts` (server-only). Use form actions for mutations rather than custom API endpoints.

## Styling

Svelte styles are scoped by default -- styles in `<style>` only affect the current component. Use `:global()` modifier cautiously when you need to reach outside component boundaries.

## Performance

Svelte compiles away the framework, so bundles are naturally small. Use `{#key}` blocks to force re-creation of components when data changes identity. Use `{#each}` with keyed items for efficient list updates. Lazy-load heavy components with dynamic imports.
