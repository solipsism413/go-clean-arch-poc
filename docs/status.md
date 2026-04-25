# Project Status

This file is the single operational view for implementation progress, current priorities, and forward roadmap.

## Done

- [x] Establish layered structure across domain, application, transport, and infrastructure.
- [x] Implement core domain model for users, tasks, roles, permissions, labels, and ACL entries.
- [x] Implement main use cases for auth, task, and user management.
- [x] Expose REST endpoints for auth, tasks, and users.
- [x] Implement `POST /api/v1/auth/register` with default member-role assignment and immediate token issuance.
- [x] Wire JWT authentication into protected routes.
- [x] Integrate RBAC and ACL authorization checks.
- [x] Bootstrap PostgreSQL, Redis, Kafka, and S3 or MinIO in server startup.
- [x] Add realtime transports for WebSocket, SSE, and Socket.IO.
- [x] Add initial migrations, SQLC-generated access, repositories, and query builders.
- [x] Generate Swagger documentation for REST.
- [x] Add broad automated test coverage across domain, use cases, transport, and infrastructure.

## Next

- [ ] Add dedicated label CRUD use cases and REST endpoints.
- [ ] Add integration tests for the main auth and task flows against real dependencies.
- [ ] Normalize API behavior for validation, authorization, and pagination responses.
- [ ] Align gRPC startup with `cfg.GRPC.Port` instead of a fixed port.

## Later

- [ ] Expose GraphQL over HTTP using the existing schema.
- [ ] Register real gRPC services for task, user, and auth operations.
- [ ] Add background consumers or subscribers for published domain events.
- [ ] Expand Redis usage into token revocation, session invalidation, or read caching.
- [ ] Implement task attachment upload and download flows backed by S3 or MinIO.
- [ ] Add stronger observability around health, readiness, metrics, and tracing.
- [ ] Add CI, delivery automation, and release workflow support.
- [ ] Improve capability parity across REST, GraphQL, gRPC, and realtime transports.

## Notes

- GraphQL currently exists as schema only.
  Reference: `internal/transport/graphql/schema.graphqls`
- gRPC currently exists as server shell and proto contract only.
  Reference: `cmd/grpc/main.go`, `internal/transport/grpc/`
- File storage is initialized, but no user-facing attachment workflow exists yet.
  Reference: `internal/infrastructure/storage/`, `migrations/000001_init.up.sql`

## Scope Summary

- Working today: REST API, JWT auth, RBAC and ACL, realtime transports, PostgreSQL, Redis, Kafka, S3 or MinIO bootstrap, Swagger, and broad automated tests.
- Partial today: GraphQL schema, gRPC server shell, label infrastructure, and file storage adapters.
- Main gap today: feature parity beyond core REST flows, especially labels and non-REST transports.
