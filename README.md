# Go Clean Architecture PoC

Task management application built in Go with a ports-and-adapters style architecture.

## Documentation

- Project docs index: `docs/index.md`
- Project status and todo list: `docs/status.md`
- Architecture: `docs/architecture.md`
- Diagram catalog: `docs/diagrams.md`
- ADRs: `docs/adr/`
- Generated Swagger artifacts: `docs/api/swagger/`

## Current Snapshot

- Working HTTP server with REST routes, JWT auth, RBAC and ACL checks, and Swagger support.
- Self-registration is available at `POST /api/v1/auth/register` and returns JWT tokens.
- Label CRUD is available over REST with case-insensitive unique names.
- Task attachment upload, list, download, and delete flows are available over REST and use S3 or MinIO for blob storage.
- Realtime transports available through WebSocket, SSE, and Socket.IO.
- PostgreSQL, Redis, Kafka, and S3 or MinIO bootstrap are already wired.
- GraphQL exists as schema-only and is not yet exposed over HTTP.
- gRPC services for task, user, auth, and label operations are fully implemented and registered.

## Quick Start

### Prerequisites

- Go 1.26.2+
- Docker and Docker Compose

### Run with Docker Compose

```bash
docker compose --profile prod up -d
```

### Run for Local Development

```bash
make setup-infra
make seed-db
go run ./cmd/server
```

Optional gRPC server:

```bash
go run ./cmd/grpc
```

## Main Endpoints

- REST base URL: `http://localhost:8080/api/v1`
- Health: `GET http://localhost:8080/health`
- Swagger UI: `http://localhost:8080/swagger/`
- WebSocket: `GET /ws`
- SSE: `GET /events`
- Socket.IO: `GET /socket.io/`

## Attachment Endpoints

- `POST /api/v1/tasks/{id}/attachments` uploads a task attachment using `multipart/form-data`.
- `GET /api/v1/tasks/{id}/attachments` lists task attachments.
- `GET /api/v1/tasks/{id}/attachments/{attachmentId}` downloads an attachment.
- `DELETE /api/v1/tasks/{id}/attachments/{attachmentId}` removes attachment metadata and deletes the blob from storage.

Notes:

- Attachment uploads enforce a maximum request size of 32 MiB.
- Attachment cleanup retries are published as task events when immediate blob deletion fails.

## Development Commands

```bash
make run-api
make run-grpc
make migrate-create name=add_descriptive_name
make migrate-up
make migrate-down
make seed-db
make test
make generate
make docker-watch
```

Seeded development users are created with password `password123`.

## Notes

- Invalid pagination query params now return `400 Bad Request` instead of being silently coerced.
- Task attachment blobs are stored in S3 or MinIO under unique per-upload object keys while preserving the original filename in API responses.
- GraphQL schema exists in `internal/transport/graphql/schema.graphqls`, but no HTTP endpoint is active yet.
- gRPC server bootstraps successfully on `cfg.GRPC.Port` with fully registered task, user, auth, and label services.

## License

MIT License
