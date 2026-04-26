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

## Next

- [ ] Add stronger observability around attachment cleanup retries and storage failures.

## Later

- [ ] Improve capability parity across REST, GraphQL, gRPC, and realtime transports.

## Notes

- GraphQL HTTP endpoint is now available at `/graphql` with a Playground at `/graphql/playground`.
  Reference: `internal/transport/graphql/`, `cmd/server/main.go`
- gRPC services are fully implemented and registered for task, user, auth, and label operations.
  Reference: `cmd/grpc/main.go`, `internal/transport/grpc/services.go`
- Task attachment workflows are available over REST and store blobs in S3 or MinIO.
  Reference: `internal/transport/rest/handler/task_handler.go`, `internal/application/usecase/task/task_usecase.go`
- Attachment uploads enforce a 32 MiB request limit, and failed blob deletions publish retry events for background cleanup.
  Reference: `internal/transport/rest/handler/task_handler.go`, `internal/domain/event/task_events.go`, `cmd/server/main.go`, `cmd/grpc/main.go`

## Scope Summary

- Working today: REST API, GraphQL HTTP endpoint, gRPC services, JWT auth, RBAC and ACL, realtime transports, PostgreSQL, Redis, Kafka, S3 or MinIO bootstrap, Swagger, CI, and broad automated tests.
- Partial today: attachment cleanup retries depend on Kafka-backed background consumers.
- Main gap today: expanded observability and transport capability parity beyond the core REST, GraphQL, and gRPC surfaces.
