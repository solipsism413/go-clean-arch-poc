# ADR 0002: Use REST as the Primary Transport

## Status

Accepted

## Context

The repository contains early groundwork for REST, GraphQL, gRPC, WebSocket, SSE, and Socket.IO. Only one transport needs to be production-shaped first so that the application contract, error model, and auth flow are stable before transport parity work expands.

## Decision

Use REST as the primary delivery transport for the current phase of the project. Treat GraphQL and gRPC as follow-on adapters that should reuse the same application services once the REST contract is mature enough.

## Consequences

- Delivery can focus on one transport surface first.
- Swagger and HTTP handlers remain the main integration point for early consumers.
- GraphQL and gRPC may lag behind until the REST contract is stable.
- Future transport parity work will need explicit planning.
