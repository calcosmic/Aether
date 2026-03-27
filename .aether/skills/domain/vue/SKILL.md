---
name: vue
description: Use when the project uses Vue.js for building reactive user interfaces
type: domain
domains: [frontend, components, ui]
agent_roles: [builder]
detect_files: ["*.vue", "vue.config.*"]
detect_packages: ["vue"]
priority: normal
version: "1.0"
---

# Vue Best Practices

## Composition API

Prefer the Composition API (`<script setup>`) over the Options API for new code. It provides better TypeScript support, more flexible logic reuse through composables, and cleaner component organization. Check which API the existing codebase uses before introducing the other.

## Reactivity System

Use `ref()` for primitives and `reactive()` for objects. Remember that `reactive()` loses reactivity when destructured -- use `toRefs()` if you need to destructure. Always access `ref` values with `.value` in script, but not in templates.

Avoid replacing an entire `reactive()` object -- mutate its properties instead. Replacing the reference breaks reactivity tracking.

## Composables

Extract reusable logic into composables (functions prefixed with `use`). A composable returns reactive state and methods. Keep composables in a `composables/` directory and each in its own file.

Composables must be called at the top level of `setup()` or `<script setup>`, never inside conditionals or loops.

## Component Communication

Props down, events up. Use `defineProps()` and `defineEmits()` in `<script setup>`. For deeply nested communication, use `provide/inject` rather than prop drilling.

Never mutate props directly. If a child needs to modify parent data, emit an event.

## Performance

Use `v-show` instead of `v-if` for elements that toggle frequently -- `v-show` keeps the DOM node and just toggles CSS display. Use `v-if` for conditional blocks that rarely change.

Add `key` attributes on `v-for` lists using stable unique identifiers, never indices for dynamic lists.

Lazy-load route components with dynamic imports: `() => import('./views/Heavy.vue')`.

## Template Discipline

Keep templates declarative. Complex logic belongs in computed properties, not inline expressions. If a template expression has more than one ternary or logical operator, extract it to a computed.

Use scoped styles (`<style scoped>`) to prevent CSS leaking between components. Use `:deep()` only when you genuinely need to style child component internals.
