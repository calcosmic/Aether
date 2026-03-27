---
name: docker
description: Use when the project uses Docker for containerization and deployment
type: domain
domains: [infrastructure, containers, deployment]
agent_roles: [builder]
detect_files: ["Dockerfile*", "docker-compose.*"]
priority: normal
version: "1.0"
---

# Docker Best Practices

## Dockerfile Construction

Use multi-stage builds to keep production images small. Build dependencies in one stage, copy only the compiled output to a minimal runtime stage. This can reduce image sizes by 10x or more.

Order instructions from least to most frequently changed. `COPY package.json` and `RUN npm install` before `COPY . .` so dependency installation is cached when only source code changes. Cache invalidation on early layers rebuilds everything after them.

Use specific base image tags (`node:20-alpine`), never `latest`. Pin versions for reproducible builds. Alpine-based images are smaller but may lack libraries -- use slim variants if Alpine causes compatibility issues.

## Image Security

Run containers as a non-root user. Add `RUN addgroup -S app && adduser -S app -G app` and `USER app` in your Dockerfile. Running as root inside a container is a security risk if the container is compromised.

Do not include secrets in images. Never use `COPY .env .` or `ARG SECRET_KEY`. Use runtime environment variables or Docker secrets instead.

Minimize the attack surface: install only what you need, remove package manager caches (`rm -rf /var/cache/apk/*`), use `.dockerignore` to exclude `node_modules`, `.git`, and test files from the build context.

## Docker Compose

Use `docker-compose.yml` for local development and testing environments. Define services, networks, and volumes clearly. Use named volumes for data persistence -- anonymous volumes are harder to manage and back up.

Set `depends_on` with health checks for service ordering. Without health checks, `depends_on` only waits for the container to start, not for the service inside it to be ready.

Use `.env` files with Compose for configuration, but never commit them to the repository. Document required environment variables in a `.env.example` file.

## Networking

Use Docker networks to isolate services. Only expose ports that need external access. Internal services (databases, caches) should communicate on internal Docker networks, not through published ports.

## Logging

Log to stdout/stderr, not to files inside the container. Docker captures stdout/stderr and makes it available via `docker logs`. File-based logging inside containers is difficult to access and fills up container storage.

## Health Checks

Add `HEALTHCHECK` instructions to Dockerfiles. A health check lets orchestrators (Compose, Kubernetes) detect when a container is running but the application inside has crashed or hung.

## Resource Limits

Set memory and CPU limits in Compose or your orchestrator. Without limits, a single misbehaving container can starve the host and crash other services.
