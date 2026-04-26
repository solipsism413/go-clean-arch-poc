# Architecture

This project follows a ports-and-adapters style that is described in the repository as Hexagonal Architecture.

## Intent

The codebase is structured to keep domain and use case logic independent from HTTP, databases, brokers, and storage providers.

The architectural center of gravity is:

- domain entities and value objects in `internal/domain/`
- application services and ports in `internal/application/`
- adapters in `internal/transport/` and `internal/infrastructure/`

## Layers

### Domain

The domain layer contains business entities, value objects, domain events, and domain-level errors.

Primary references:

- `internal/domain/entity/`
- `internal/domain/valueobject/`
- `internal/domain/event/`
- `internal/domain/error/domain_error.go`

### Application

The application layer defines use cases, DTOs, validation rules, and the input and output ports used by the adapters.

Primary references:

- `internal/application/dto/`
- `internal/application/validation/`
- `internal/application/port/input/`
- `internal/application/port/output/`
- `internal/application/usecase/`

### Transport

The transport layer adapts external protocols into application service calls. REST is the only transport that is fully active in the running server today.

Primary references:

- REST: `internal/transport/rest/`
- WebSocket: `internal/transport/websocket/`
- SSE: `internal/transport/sse/`
- Socket.IO: `internal/transport/socketio/`
- GraphQL schema: `internal/transport/graphql/schema.graphqls`
- gRPC primitives: `internal/transport/grpc/`

### Infrastructure

The infrastructure layer provides concrete adapters for persistence, cache, messaging, storage, JWT handling, and observability.

Primary references:

- Database: `internal/infrastructure/database/`
- Cache: `internal/infrastructure/cache/`
- Messaging: `internal/infrastructure/messaging/`
- Storage: `internal/infrastructure/storage/`
- Auth: `internal/infrastructure/auth/jwt/`
- Observability: `internal/infrastructure/observability/`
- Factory support: `internal/infrastructure/factory/`

## Runtime Composition

### HTTP Server

`cmd/server/main.go` acts as the main composition root for the running application.

It is responsible for:

- loading configuration from `pkg/config/config.go`
- bootstrapping PostgreSQL, Redis, Kafka, and S3 adapters
- constructing repositories and use cases
- constructing JWT auth, RBAC, and ACL middleware dependencies
- registering REST, WebSocket, SSE, and Socket.IO transports
- registering background consumers for task-event retries such as attachment cleanup
- starting the HTTP server and handling graceful shutdown

### gRPC Server

`cmd/grpc/main.go` mirrors the same dependency bootstrap for gRPC service hosting.

Current state:

- concrete task, user, auth, and label services are registered
- Kafka-backed background consumers are also started for task-event retries such as attachment cleanup
- configured gRPC port support is active through `cfg.GRPC.Port`

## Security Model

The application currently combines three mechanisms:

- JWT for authentication
- RBAC for role-based permission checks
- ACL for resource-specific access decisions

Primary references:

- `internal/infrastructure/auth/jwt/token_service.go`
- `internal/auth/middleware.go`
- `internal/auth/rbac/authorizer.go`
- `internal/auth/acl/checker.go`
- `internal/transport/rest/router.go`

## Persistence Model

PostgreSQL is the primary system of record. The schema is defined in `migrations/000001_init.up.sql` and includes:

- users and roles
- permissions and role mappings
- tasks and labels
- ACL entries
- task attachment metadata

Attachment blob data itself lives in S3 or MinIO. The database stores attachment metadata and object keys, while cleanup retries are published as task events when immediate blob deletion fails.

Persistence implementation combines:

- SQLC-generated database access under `internal/infrastructure/database/sqlc/`
- repositories under `internal/infrastructure/database/repository/`
- query builders under `internal/infrastructure/database/querybuilder/`

## Architectural Observations

- The repository is structurally consistent with a hexagonal approach.
- REST is the current primary delivery transport.
- GraphQL and gRPC currently represent intended extension points rather than finished runtime interfaces.
- The infrastructure factory is useful as an abstraction point, but it is not yet the main bootstrap path.
- `AuthUseCase` currently depends on the concrete JWT token service type, which slightly weakens the inward dependency rule.

## Diagram References

- System architecture: `docs/diagrams/01-system-architecture.mmd`
- Create task sequence: `docs/diagrams/02-create-task-sequence.mmd`
- Database ERD: `docs/diagrams/03-database-erd.mmd`
- Authentication and authorization flow: `docs/diagrams/04-authn-authz-flow.mmd`
