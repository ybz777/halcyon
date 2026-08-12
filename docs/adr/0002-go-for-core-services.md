# ADR-0002: Go for Core Backend Services

## Status

Accepted

## Context

Halcyon requires multiple backend services (auth, workflow, ingestion,
correlation, notifications). We need a primary language for security-critical
and high-throughput paths.

## Decision

Use **Go** for core microservices starting with `auth`.

Python (FastAPI) remains available for AI-heavy workloads in later services
per the SDD, but platform and domain services default to Go.

## Alternatives

| Alternative | Why rejected |
|---|---|
| TypeScript everywhere | Weaker single-binary deploy story; runtime overhead for hot paths |
| Python everywhere | GIL and packaging complexity for ingestion/correlation throughput |
| Rust | Excellent performance but slower iteration for portfolio timeline |
| Java/Kotlin | Heavy JVM operational footprint for early-stage solo development |

## Consequences

### Positive

- Static typing, fast compile, small containers
- Strong ecosystem for HTTP, Kafka, PostgreSQL
- Common choice at Stripe, Cloudflare, Datadog—recognizable to reviewers

### Negative

- AI/RAG ergonomics weaker than Python—AI service may use Python later
- Verbose error handling compared to Rust
