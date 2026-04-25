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
- Realtime transports available through WebSocket, SSE, and Socket.IO.
- PostgreSQL, Redis, Kafka, and S3 or MinIO bootstrap are already wired.
- GraphQL and gRPC exist as partial scaffolding and are not yet feature-complete.

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

Optional gRPC server shell:

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

## Development Commands

```bash
make run-api
make run-grpc
make seed-db
make test
make generate
make docker-watch
```

Seeded development users are created with password `password123`.

## Notes

- GraphQL schema exists in `internal/transport/graphql/schema.graphqls`, but no HTTP endpoint is active yet.
- gRPC server bootstraps successfully, but application services are not registered yet.

## License

MIT License
