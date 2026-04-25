# ADR 0001: Use Hexagonal Architecture

## Status

Accepted

## Context

The project aims to serve as a clean architecture proof of concept while still supporting multiple transports and multiple infrastructure adapters. Business rules need to remain testable and not be tightly coupled to HTTP, PostgreSQL, Kafka, Redis, or S3.

## Decision

Use a ports-and-adapters structure with clear separation between:

- domain
- application
- transport adapters
- infrastructure adapters

Keep use cases in the application layer and define dependency boundaries through input and output ports.

## Consequences

- The codebase stays easier to test in isolation.
- Multiple transports can be added without rewriting core business logic.
- Infrastructure concerns can be swapped more safely.
- The project requires more upfront structure and more interface design discipline.
