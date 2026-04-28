---
name: observability
description: Use when the project uses logging, metrics, tracing, or monitoring infrastructure for observability
type: domain
domains: [infrastructure, backend, devops, monitoring]
agent_roles: [builder]
detect_files: ["otel-collector-config.yaml", "prometheus.yml", "grafana.ini", "jaeger-config.yaml"]
detect_packages: ["opentelemetry", "prometheus", "grafana", "winston", "pino"]
priority: normal
version: "1.0"
---

# Observability Best Practices

## OpenTelemetry Integration

- Instrument with OpenTelemetry SDKs for automatic trace propagation and metric collection
- Use `OTEL_RESOURCE_ATTRIBUTES` to attach service name, version, and environment to all telemetry
- Configure the OTel Collector as a central pipeline: receive -> process -> export to backends
- Use auto-instrumentation agents for quick wins; add manual spans for business-critical code paths
- Propagate trace context across service boundaries with W3C TraceContext headers (`traceparent`)

## Logging Standards

- Use structured logging (JSON) in production: include `timestamp`, `level`, `message`, `trace_id`, `span_id`
- Attach trace IDs to log entries to correlate logs with traces during incident investigation
- Define log levels consistently: ERROR (action required), WARN (degraded), INFO (business events), DEBUG (dev only)
- Never log secrets, tokens, or PII -- use log sanitization middleware to redact sensitive fields
- Centralize logs in a single aggregation system (Loki, Elasticsearch, CloudWatch Logs)

## Metrics with Prometheus/Grafana

- Expose a `/metrics` endpoint in Prometheus format; use the standard client library for your language
- Follow naming conventions: `namespace_subsystem_unit` (e.g., `http_request_duration_seconds`)
- Use `Histogram` for latency distributions, `Counter` for cumulative events, `Gauge` for point-in-time values
- Build Grafana dashboards per service: golden signals (latency, traffic, errors, saturation) as the baseline
- Record dashboards as code (Grafana Terraform provider or JSON models) in version control

## Distributed Tracing

- Use Jaeger or Zipkin as the tracing backend; configure sampling strategies to balance cost and coverage
- Add spans at service boundaries and external call points (DB queries, HTTP calls, message publishes)
- Use baggage to propagate business context (user_id, tenant_id) across the trace
- Set up trace-based alerting: alert on p99 latency spikes correlated with specific endpoints

## SLO/SLI and Alerting

- Define SLIs that measure user experience: request latency (p99 < 500ms), error rate (< 0.1%), availability (> 99.9%)
- Set SLOs per service; track error budget burn rate to balance reliability with feature velocity
- Alert on symptoms (user-facing degradation) not causes (CPU high); use multi-burn-rate alerts
- Page only for conditions requiring human intervention; route everything else to ticket-based workflows
- Run chaos experiments periodically to validate that alerts fire before SLOs are breached
