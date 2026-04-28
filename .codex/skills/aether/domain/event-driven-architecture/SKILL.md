---
name: event-driven-architecture
description: Use when the project uses message queues, event sourcing, CQRS, or asynchronous messaging patterns
type: domain
domains: [backend, infrastructure, architecture, distributed-systems]
agent_roles: [builder]
detect_files: ["kafka/**/*.properties", "rabbitmq.conf", "serverless.yml", "eventbridge*.json"]
detect_packages: ["kafkajs", "amqplib", "@aws-sdk/client-sqs", "@aws-sdk/client-eventbridge"]
priority: normal
version: "1.0"
---

# Event-Driven Architecture Best Practices

## Message Queue Selection

- Use Kafka for high-throughput, durable event streaming with replay capability and ordered partitions
- Use RabbitMQ for task queues with complex routing (exchanges, bindings) and lower throughput needs
- Use SQS/SNS for AWS-native decoupling: SQS for point-to-point, SNS for fan-out, EventBridge for schema routing
- Use Redis Streams for lightweight, low-latency messaging when durability requirements are moderate
- Consider Pulsar for geo-replicated streaming with multi-tenancy requirements

## Event Sourcing

- Store every state change as an immutable event in an append-only log; reconstruct state by replaying events
- Define events as contracts with a schema registry (Avro, Protobuf, JSON Schema) for evolution compatibility
- Use snapshots to optimize replay: persist a materialized state snapshot every N events
- Keep events small and focused: one event per business action, include only what changed plus relevant context
- Version events from the start: include `event_version` field and support upcasting for backward compatibility

## CQRS Pattern

- Separate command (write) and query (read) models: optimize each independently for its access pattern
- Use eventual consistency between write and read models; communicate the trade-off clearly to stakeholders
- Project events into read models asynchronously: subscribe to the event stream and update denormalized views
- Keep the write model authoritative -- the event store is the single source of truth
- Rebuild read models from the event stream when schema changes; this is a feature, not a bug

## Saga Pattern

- Use sagas for distributed transactions across services: choreography (event-driven) or orchestration (central coordinator)
- Define compensating actions for every saga step: if step 3 fails, undo steps 2 and 1 in reverse order
- Persist saga state to survive process crashes; use idempotent handlers to handle duplicate saga events
- Set timeouts on every saga step; trigger compensation if a participant doesn't respond
- Monitor saga completion rates and duration: track sagas-in-progress and alert on stuck sagas

## Idempotency

- Design every event consumer to be idempotent: processing the same event twice must produce the same outcome
- Use idempotency keys (event ID, correlation ID) with deduplication stores (Redis, DB unique constraint)
- Return consistent responses for duplicate requests: store and replay the original response
- Handle out-of-order delivery: design consumers to handle events arriving in any order using versioning or timestamps
- Test idempotency by replaying events intentionally in staging environments
