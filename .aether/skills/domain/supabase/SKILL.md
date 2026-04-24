---
name: supabase
description: Use when the project uses Supabase as backend-as-a-service for auth, database, storage, or realtime
type: domain
domains: [backend, database, auth, realtime, storage]
agent_roles: [builder]
detect_files: [".env.local", "supabase/config.toml", "supabase/migrations/*.sql"]
detect_packages: ["@supabase/supabase-js", "@supabase/auth-helpers", "supabase-py"]
priority: normal
version: "1.0"
---

# Supabase Best Practices

## Client Initialization

- Create a single Supabase client instance and share it via dependency injection or a singleton module
- Use `createClient()` with the project URL and anon key from environment variables -- never hardcode keys
- For server-side: use the service role key only in secure backend contexts; validate JWTs on every request
- Apply `@supabase/auth-helpers-nextjs` (or platform equivalent) for automatic session management in SSR

## Row Level Security (RLS)

- Enable RLS on every public-facing table -- this is the primary security boundary
- Write RLS policies using `auth.uid()` to scope data to the authenticated user
- Use `USING` clause for read policies and `WITH CHECK` for write policies
- Create helper functions for common RLS patterns: `auth.uid() = user_id` as a reusable policy template
- Test RLS policies by executing queries as different authenticated roles in the SQL editor

## Authentication Flows

- Use Supabase Auth for sign-up, sign-in, password reset, and magic links -- do not build custom auth
- Configure OAuth providers (Google, GitHub, Apple) in the dashboard; handle callbacks in the app
- Store the session token securely: use HttpOnly cookies in SSR apps, SecureStorage in mobile
- Implement PKCE flow for server-side auth; implicit flow for client-only SPAs
- Use `supabase.auth.onAuthStateChange()` to react to session changes across the app

## Edge Functions

- Use Deno-based Edge Functions for server-side logic: webhooks, data transforms, scheduled jobs
- Keep functions small and focused; one responsibility per function
- Use `serve()` from `deno_std/http` and validate inputs with zod schemas at the boundary
- Access environment secrets via `Deno.env.get()`; never commit secrets to function code
- Deploy with `supabase functions deploy`; set secrets with `supabase secrets set`

## Realtime Subscriptions

- Subscribe to INSERT, UPDATE, DELETE events on tables using `.on('postgres_changes', ...)`
- Enable Realtime on specific tables via the dashboard or `supabase.realtime.enable()`
- Use presence channels for collaborative features: track who is online with `channel.presence`
- Unsubscribe on component unmount to prevent memory leaks: `supabase.removeChannel(channel)`
- Apply broadcast channels for ephemeral messages that don't need persistence

## Storage Buckets

- Create separate buckets for public assets vs. private user uploads
- Use storage policies to restrict upload size, file types, and folder scoping per user
- Generate signed URLs for time-limited access to private files: `createSignedUrl(path, expiresIn)`
- Upload with `supabase.storage.from(bucket).upload(path, file)` and transform images with built-in CDN
