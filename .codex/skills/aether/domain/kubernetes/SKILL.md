---
name: kubernetes
description: Use when the project uses Kubernetes for container orchestration, deployment, and service management
type: domain
domains: [infrastructure, devops, cloud, containers]
agent_roles: [builder]
detect_files: ["*.yaml", "Chart.yaml", "values.yaml", "kustomization.yaml", "Dockerfile"]
detect_packages: ["kubernetes", "helm", "kustomize", "kubectl"]
priority: normal
version: "1.0"
---

# Kubernetes Best Practices

## Deployment Strategies

- Use `Deployment` with `RollingUpdate` as default: set `maxSurge` and `maxUnavailable` for controlled rollouts
- Implement `readinessProbe` and `livenessProbe` on every container -- HTTP or exec-based
- Use `startupProbe` for slow-starting applications to prevent premature kill by the liveness probe
- Set resource `requests` and `limits` on every pod: CPU/Memory requests for scheduling, limits for QoS
- Apply `PodDisruptionBudget` to maintain availability during voluntary disruptions (node drains, upgrades)

## Service Mesh

- Use a service mesh (Istio, Linkerd) for mTLS, traffic shaping, and observability when microservices exceed ~10
- Define `DestinationRule` for circuit breaking and connection pooling
- Use `VirtualService` for canary deployments: route percentages between service versions
- Apply `PeerAuthentication` for strict mTLS mode across the mesh
- Keep mesh configuration in the same repository as the application manifests

## Helm Charts

- Structure charts with `templates/`, `values.yaml`, `Chart.yaml`, and `README.md`
- Use `values.yaml` for environment-agnostic defaults; override with `-f staging.yaml` per environment
- Template with `{{ .Values.* }}` and `{{ .Release.* }}`; use `{{-` to trim whitespace
- Use named templates (`_helpers.tpl`) for reusable label blocks and common metadata
- Run `helm lint` and `helm template` in CI; use `helm test` for release validation hooks

## RBAC and Security

- Apply least-privilege RBAC: create `Role`/`RoleBinding` per namespace, avoid `ClusterRole` unless necessary
- Use `NetworkPolicy` to restrict pod-to-pod traffic: default deny, explicitly allow required flows
- Run containers as non-root: set `runAsNonRoot: true`, `readOnlyRootFilesystem: true` in security contexts
- Use `SealedSecrets` or external secret managers (Vault, AWS Secrets Manager) -- never plaintext secrets
- Scan images with Trivy or Grype in CI; enforce admission controllers to block vulnerable images

## Autoscaling and GitOps

- Configure `HorizontalPodAutoscaler` with CPU/memory and custom metrics via Prometheus adapter
- Use `VerticalPodAutoscaler` for right-sizing requests in steady-state workloads
- Implement GitOps with ArgoCD or Flux: declarative repo -> automated cluster reconciliation
- Structure repos as `clusters/`, `apps/`, `infra/` for GitOps clarity; use app-of-apps pattern in ArgoCD
- Use `Kustomize` overlays for environment-specific patches without duplicating base manifests
