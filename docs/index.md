# Project Documentation

This directory contains the maintained project documentation for `go-clean-arch-poc`.

## Document Map

- Unified project status and todo list: `docs/status.md`
- Architecture overview: `docs/architecture.md`
- Diagram catalog: `docs/diagrams.md`
- Architecture Decision Records: `docs/adr/`

## Current Snapshot

The repository already delivers a working HTTP application with a clear layered structure around domain, application, transport, and infrastructure code. REST endpoints, JWT-based authentication, RBAC and ACL checks, realtime transports, persistence, and broad automated tests are already present.

The main delivery gaps are transport parity and feature completion around label management, file attachments, GraphQL exposure, and gRPC service registration.

The preferred operational view is `docs/status.md`, which combines completed work and planned next steps in one checklist.

## Primary Code References

- HTTP composition root: `cmd/server/main.go`
- gRPC composition root: `cmd/grpc/main.go`
- REST router: `internal/transport/rest/router.go`
- Core use cases: `internal/application/usecase/`
- Domain model: `internal/domain/`
- Infrastructure adapters: `internal/infrastructure/`
- Initial database schema: `migrations/000001_init.up.sql`

## Navigation

Start with `docs/status.md` if the goal is a single actionable checklist.

Start with `docs/architecture.md` if the goal is understanding code structure and runtime composition.

Start with `docs/diagrams.md` if the goal is visual reference.
