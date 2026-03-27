---
name: nextjs
description: Use when the project uses Next.js for server-side rendering or full-stack React
type: domain
domains: [frontend, ssr, fullstack]
agent_roles: [builder]
detect_files: ["next.config.*"]
detect_packages: ["next"]
priority: normal
version: "1.0"
---

# Next.js Best Practices

## App Router vs Pages Router

Next.js 13+ uses the App Router by default. Check which router the project uses before making changes. App Router uses `app/` directory with `layout.tsx`, `page.tsx`, and `loading.tsx` conventions. Pages Router uses `pages/` with `_app.tsx` and `_document.tsx`. Never mix patterns within the same route.

## Server vs Client Components

In the App Router, components are Server Components by default. Only add `"use client"` when you need browser APIs, event handlers, hooks, or browser-only libraries. Push `"use client"` as far down the tree as possible -- wrap only the interactive leaf, not the entire page.

Server Components can fetch data directly without `useEffect` or API routes. This is the preferred pattern for data loading.

## Data Fetching

Use `fetch` in Server Components with caching options (`cache: 'force-cache'` for static, `cache: 'no-store'` for dynamic). For mutations, use Server Actions -- functions marked with `"use server"` that run on the server but can be called from client forms.

Avoid calling your own API routes from Server Components. You have direct access to the database and services -- use them.

## Route Handlers

API routes live in `app/api/*/route.ts`. Export named functions matching HTTP methods: `GET`, `POST`, `PUT`, `DELETE`. Always return `NextResponse` objects with appropriate status codes.

## Common Gotchas

Middleware runs on the Edge Runtime -- no Node.js APIs like `fs` or `path`. Keep middleware lean: auth checks, redirects, header modifications only.

Dynamic imports with `next/dynamic` for heavy client components keep initial bundle sizes small. Always provide a loading fallback.

Environment variables prefixed with `NEXT_PUBLIC_` are exposed to the browser. Never put secrets in `NEXT_PUBLIC_` variables.

## Image and Font Optimization

Use `next/image` for all images -- it handles lazy loading, sizing, and format optimization automatically. Use `next/font` for fonts to eliminate layout shift.
