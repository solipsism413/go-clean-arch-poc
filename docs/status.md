# Project Status

This file is the single operational view for implementation progress, current priorities, and forward roadmap.

## Done

- [x] Establish layered structure across domain, application, transport, and infrastructure.
- [x] Implement core domain model for users, tasks, roles, permissions, labels, and ACL entries.
- [x] Implement main use cases for auth, task, and user management.
- [x] Expose REST endpoints for auth, tasks, and users.
- [x] Implement `POST /api/v1/auth/register` with default member-role assignment and immediate token issuance.
- [x] Add dedicated label CRUD use cases and REST endpoints with case-insensitive unique names.
- [x] Wire JWT authentication into protected routes.
- [x] Integrate RBAC and ACL authorization checks.
- [x] Bootstrap PostgreSQL, Redis, Kafka, and S3 or MinIO in server startup.
- [x] Add realtime transports for WebSocket, SSE, and Socket.IO.
- [x] Add initial migrations, SQLC-generated access, repositories, and query builders.
- [x] Generate Swagger documentation for REST.
- [x] Add broad automated test coverage across domain, use cases, transport, and infrastructure.
- [x] Add integration tests for the main auth and task flows against real dependencies.
- [x] Normalize API behavior for validation, authorization, and pagination responses.
- [x] Align gRPC startup with `cfg.GRPC.Port` instead of a fixed port.
- [x] Register real gRPC services for task, user, auth, and label operations.
- [x] Add CI, delivery automation, and release workflow support.
- [x] Add background consumers or subscribers for published domain events.
- [x] Expose GraphQL over HTTP using the existing schema.
- [x] Expand Redis usage into token revocation, session invalidation, and read caching.
- [x] Add stronger observability around attachment cleanup retries and storage failures.
- [x] Improve GraphQL parity with REST and gRPC for task lifecycle, overdue task queries, and label lookup.

## Next

- [x] Close remaining transport parity gaps such as task attachments and GraphQL or realtime subscription coverage.

## Later

- [x] Add richer operational metrics and alerting beyond structured logging for background workflows.

## Notes

- GraphQL HTTP endpoint is now available at `/graphql` with a Playground at `/graphql/playground`.
  Reference: `internal/transport/graphql/`, `cmd/server/main.go`
- GraphQL now covers task completion, task archiving, overdue task queries, task attachments, and task subscriptions in addition to its existing CRUD surface.
  Reference: `internal/transport/graphql/schema.graphqls`, `internal/transport/graphql/schema.resolvers.go`
- gRPC services are fully implemented and registered for task, user, auth, and label operations.
  Reference: `cmd/grpc/main.go`, `internal/transport/grpc/services.go`
- Task attachment workflows are available over REST and GraphQL and store blobs in S3 or MinIO.
  Reference: `internal/transport/rest/handler/task_handler.go`, `internal/transport/graphql/schema.graphqls`, `internal/application/usecase/task/task_usecase.go`
- Attachment uploads enforce a 32 MiB request limit, and failed blob deletions publish retry events for background cleanup.
  Reference: `internal/transport/rest/handler/task_handler.go`, `internal/domain/event/task_events.go`, `cmd/server/main.go`, `cmd/grpc/main.go`
- Attachment cleanup retries and upload rollback failures now emit structured logs with task, attachment, object-key, and retry context in both HTTP and gRPC startup paths.
  Reference: `internal/application/usecase/task/task_usecase.go`, `internal/application/worker/task_attachment_cleanup_handler.go`, `cmd/server/main.go`, `cmd/grpc/main.go`
- Background workflows now export expvar metrics and active alert state at `/debug/vars`, and local in-process event fanout keeps realtime subscriptions active even when Kafka-backed subscribers are unavailable.
  Reference: `internal/infrastructure/observability/background/monitor.go`, `internal/infrastructure/messaging/fanout/publisher.go`, `cmd/server/main.go`, `cmd/grpc/main.go`

## Scope Summary

- Working today: REST API, GraphQL HTTP endpoint, gRPC services, JWT auth, RBAC and ACL, realtime transports, PostgreSQL, Redis, Kafka, S3 or MinIO bootstrap, Swagger, CI, and broad automated tests.
- Working today: attachment cleanup retries can also fall back to the local in-process event bus when Kafka subscribers are unavailable.
- Remaining gap today: external alert delivery is still not wired to a paging or incident-management system.
