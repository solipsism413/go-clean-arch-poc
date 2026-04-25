# Go Clean Architecture PoC

A task management application built with Hexagonal (Ports & Adapters) Architecture in Go.

## Progress

### Completed

- [x] Hexagonal architecture structure is in place across `domain`, `application`, `transport`, and `infrastructure` layers.
- [x] Core domain model is available for users, tasks, roles, labels, permissions, and ACL entries.
- [x] Main use cases are implemented for task management, user management, and authentication.
- [x] REST API routes are wired for auth, tasks, and users.
- [x] JWT-based authentication and authorization middleware are connected to the REST layer.
- [x] RBAC and ACL checks are already integrated into protected endpoints.
- [x] Realtime transports are available through WebSocket, SSE, and Socket.IO endpoints.
- [x] PostgreSQL, Redis, Kafka, and S3/MinIO bootstrap code is already integrated into server startup.
- [x] SQLC queries and initial database migration are present.
- [x] Swagger documentation is generated for the REST API.
- [x] Automated tests already cover domain entities, use cases, repositories, query builders, cache, messaging, tracing, and REST handlers.

### Known Gaps

- [ ] `POST /api/v1/auth/register` still returns `501 Not Implemented`.
- [ ] GraphQL schema is present, but the GraphQL transport is not exposed by the running server yet.
- [ ] gRPC server infrastructure exists, but application services are not registered yet.
- [ ] Label management endpoints and dedicated label use cases are not exposed yet even though label entities and repositories already exist.
- [ ] File storage integration is initialized at startup, but no user-facing upload/download flow is implemented yet.

## Roadmap

### Near Term

- [ ] Implement user self-registration in `AuthHandler` and connect it to the user/auth use case flow.
- [ ] Complete REST coverage for label management: create, list, update, delete, and task-label workflows.
- [ ] Add integration tests for the main auth and task flows against real infrastructure dependencies.
- [ ] Improve API consistency for validation errors, auth failures, and pagination responses.

### Mid Term

- [ ] Expose GraphQL over HTTP using the existing schema as the starting contract.
- [ ] Register real gRPC services for task, user, and auth operations.
- [ ] Add Kafka consumers or background workers so published domain events trigger useful downstream behavior.
- [ ] Expand Redis usage beyond cache invalidation into token/session revocation or read-model caching.

### Later

- [ ] Implement file attachment flows backed by S3/MinIO for task-related assets.
- [ ] Add richer observability with tracing dashboards, metrics, and health/readiness checks for dependencies.
- [ ] Add delivery support for production deployment, CI pipeline, and release automation.
- [ ] Revisit multi-transport parity so REST, GraphQL, gRPC, and realtime channels expose a consistent capability set.

## Architecture

This project follows **Hexagonal Architecture** (Ports & Adapters) to keep business logic isolated from transport and infrastructure concerns.

### Core Principles

- **Domain centric**: business rules live in the domain and application layers.
- **Dependency rule**: dependencies point inward.
- **Framework independence**: core logic does not depend on HTTP, databases, or brokers.
- **Testability**: domain and use case logic can be tested in isolation.

### Layers

1. **Domain Layer**: entities, value objects, domain events, and domain errors.
2. **Application Layer**: use cases, DTOs, validation, and ports.
3. **Transport Layer**: REST, WebSocket, SSE, and Socket.IO adapters.
4. **Infrastructure Layer**: PostgreSQL, Redis, Kafka, S3/MinIO, and observability support.

```mermaid
graph TD
    subgraph Transport_Layer [Transport Layer]
        REST[REST API]
        WS[WebSocket]
        SSE[SSE]
        SIO[Socket.IO]
    end

    subgraph Application_Layer [Application Layer]
        UC[Use Cases / Input Ports]
        Ports[Output Ports / Interfaces]
    end

    subgraph Domain_Layer [Domain Layer]
        Entities[Entities]
        VO[Value Objects]
        Events[Domain Events]
    end

    subgraph Infrastructure_Layer [Infrastructure Layer]
        DB[(PostgreSQL)]
        Cache[Redis]
        Messaging[Kafka]
        Storage[S3/MinIO]
    end

    REST & WS & SSE & SIO --> UC
    UC --> Entities & VO & Events
    UC --> Ports
    Ports --> DB & Cache & Messaging & Storage
```

## Infrastructure

The repository includes these infrastructure integrations and tooling:

- **PostgreSQL**: primary relational database.
- **Redis**: cache and token/session support.
- **Kafka**: event publishing.
- **MinIO / S3**: object storage.
- **OpenTelemetry (OTel)**: configuration and support package are present.
- **SQLC**: generates type-safe Go code from SQL queries.
- **GQLGen**: schema and generator config are present for future GraphQL work.
- **Viper**: configuration management from environment variables and optional config files.

## Diagrams

### Use Case Diagram

```mermaid
graph TD
    User((User))
    Admin((Admin))

    subgraph Auth
        User --> Login([Login])
        User --> Refresh([Refresh Token])
        User --> Logout([Logout])
        User --> ChangePassword([Change Password])
    end

    subgraph Task_Management [Task Management]
        User --> CreateTask([Create Task])
        User --> UpdateTask([Update Task])
        User --> ViewTasks([View Tasks])
        User --> SearchTasks([Search Tasks])
        User --> AssignTask([Assign Task])
        User --> CompleteTask([Complete Task])
        User --> ArchiveTask([Archive Task])
        User --> ManageLabels([Manage Labels])
    end

    subgraph Administration
        Admin --> ManageUsers([Manage Users])
        Admin --> ManageRoles([Manage Roles])
        Admin --> ManagePermissions([Manage Permissions])
    end

    Admin -.-> User
```

### Entity Relationship Diagram (ERD)

````mermaid
erDiagram
    USER ||--o{ ROLE : "has"
    ROLE ||--o{ PERMISSION : "contains"
    USER ||--o{ TASK : "creates"
    USER ||--o{ TASK : "is assigned to"
    TASK ||--o{ LABEL : "has"

    USER {
        uuid id PK
        string email
        string password_hash
        string name
        timestamp created_at
        timestamp updated_at
    }

    ROLE {
        uuid id PK
        string name
        string description
        timestamp created_at
        timestamp updated_at
    }

    PERMISSION {
        uuid id PK
        string resource
        string action
        timestamp created_at
        timestamp updated_at
    }

    TASK {
        uuid id PK
        string title
        text description
        string status
        string priority
        timestamp due_date
        uuid creator_id FK
        uuid assignee_id FK
        timestamp created_at
        timestamp updated_at
    }

    LABEL {
        uuid id PK
        string name
        string color
        timestamp created_at
        timestamp updated_at
    }

````

### Request Flow

Example: **Create Task**

```mermaid
sequenceDiagram
    participant Client
    participant REST as REST Handler
    participant DTO as DTO/Validation
    participant UC as CreateTaskUseCase
    participant Domain as Task Entity
    participant Repo as TaskRepository
    participant DB as PostgreSQL
    participant Event as EventPublisher
    participant Kafka

    Client->>REST: POST /tasks (JSON)
    REST->>DTO: Parse and validate
    DTO-->>REST: Valid DTO
    REST->>UC: Execute(ctx, dto)
    UC->>Domain: NewTask(...)
    Domain-->>UC: Task Entity
    UC->>Repo: Save(ctx, task)
    Repo->>DB: INSERT INTO tasks ...
    DB-->>Repo: Success
    UC->>Event: Publish(TaskCreatedEvent)
    Event->>Kafka: Produce message if configured
    UC-->>REST: Task DTO
    REST-->>Client: 201 Created (JSON)
```

## Quick Start

### Prerequisites

- Go 1.26.2+
- Docker and Docker Compose

### Running with Docker

```bash
# Start all services
docker compose --profile prod up -d

# View logs
docker compose logs -f app

# Stop services
docker compose down
```

### Running with Docker Compose Watch

```bash
# Start services and watch application source changes
docker compose --profile watch up --watch

# Or via Makefile
make docker-watch
```

`docker-compose.yml` is the source for both modes using Compose `profiles`. `app` runs under `prod`, while `app-watch` runs under `watch` with `air` and `develop.watch`.

### Running Locally

```bash
# Start infrastructure services
docker compose up -d postgres redis kafka minio minio-init

# Run migrations
docker compose run --rm migrate

# Run the HTTP server
go run ./cmd/server

# Optional: run the gRPC server shell
go run ./cmd/grpc
```

## Available Endpoints

### REST

- Base URL: `http://localhost:8080/api/v1`
- Swagger UI: `http://localhost:8080/swagger/`
- Health check: `GET http://localhost:8080/health`

Auth:
- `POST /auth/login`
- `POST /auth/register` currently returns `501 Not Implemented`
- `POST /auth/refresh`
- `POST /auth/logout`
- `POST /auth/change-password`

Tasks:
- `GET /tasks`
- `GET /tasks/search`
- `GET /tasks/overdue`
- `POST /tasks`
- `GET /tasks/{id}`
- `PUT /tasks/{id}`
- `DELETE /tasks/{id}`
- `POST /tasks/{id}/assign`
- `POST /tasks/{id}/unassign`
- `POST /tasks/{id}/complete`
- `POST /tasks/{id}/archive`
- `POST /tasks/{id}/status`
- `POST /tasks/{id}/labels/{labelId}`
- `DELETE /tasks/{id}/labels/{labelId}`

Users:
- `GET /users`
- `GET /users/me`
- `GET /users/{id}`
- `PUT /users/{id}`
- `DELETE /users/{id}`
- `POST /users/{id}/roles/{roleId}`
- `DELETE /users/{id}/roles/{roleId}`

### Realtime

- WebSocket: `GET /ws`
- SSE: `GET /events`
- Socket.IO: `GET /socket.io/`

### Not Exposed Yet

- GraphQL schema is stored in `internal/transport/graphql/schema.graphqls`, but no GraphQL HTTP endpoint is registered.
- gRPC proto definitions exist in `internal/transport/grpc/proto/task.proto`, but the running gRPC server does not register `TaskService`, `UserService`, or `AuthService` implementations yet.

## Project Structure

```text
├── cmd/
│   ├── server/              # HTTP server entrypoint
│   └── grpc/                # gRPC server entrypoint
├── internal/
│   ├── domain/              # Domain layer
│   │   ├── entity/
│   │   ├── valueobject/
│   │   ├── event/
│   │   └── error/
│   ├── application/         # Use cases, DTOs, validation, and ports
│   ├── infrastructure/      # Database, cache, messaging, storage, logging
│   └── transport/
│       ├── rest/            # REST API handlers
│       ├── grpc/            # gRPC server primitives and proto files
│       ├── graphql/         # GraphQL schema placeholder
│       ├── socketio/        # Socket.IO transport
│       ├── sse/             # Server-Sent Events transport
│       └── websocket/       # WebSocket transport
├── docs/                    # Generated Swagger docs
├── migrations/              # Database migrations
├── pkg/
│   └── config/              # Configuration loader
├── docker-compose.yml
├── Dockerfile
├── Makefile
└── sqlc.yaml
```

## Configuration

Configuration can be provided through environment variables or an optional `config.yaml`.

| Variable | Description | Default |
| --- | --- | --- |
| `SERVER_HOST` | HTTP server host | `0.0.0.0` |
| `SERVER_PORT` | HTTP server port | `8080` |
| `SERVER_READ_TIMEOUT` | HTTP read timeout | `30s` |
| `SERVER_WRITE_TIMEOUT` | HTTP write timeout | `30s` |
| `SERVER_IDLE_TIMEOUT` | HTTP idle timeout | `120s` |
| `GRPC_PORT` | gRPC server port | `9090` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5433` |
| `DB_USER` | PostgreSQL user | `taskmanager` |
| `DB_PASSWORD` | PostgreSQL password | `taskmanager_secret` |
| `DB_NAME` | PostgreSQL database | `taskmanager` |
| `DB_SSLMODE` | PostgreSQL SSL mode | `disable` |
| `DB_MAX_CONNS` | PostgreSQL max connections | `25` |
| `DB_MIN_CONNS` | PostgreSQL min connections | `5` |
| `DB_MAX_CONN_LIFETIME` | PostgreSQL max conn lifetime | `1h` |
| `DB_MAX_CONN_IDLE_TIME` | PostgreSQL max idle conn time | `30m` |
| `REDIS_HOST` | Redis host | `localhost` |
| `REDIS_PORT` | Redis port | `6379` |
| `REDIS_PASSWORD` | Redis password | `redis_secret` |
| `REDIS_DB` | Redis DB index | `0` |
| `REDIS_POOL_SIZE` | Redis pool size | `10` |
| `REDIS_MIN_IDLE_CONNS` | Redis min idle conns | `5` |
| `KAFKA_BROKERS` | Kafka brokers | `localhost:9092` |
| `KAFKA_CONSUMER_GROUP` | Kafka consumer group | `taskmanager` |
| `KAFKA_AUTO_OFFSET_RESET` | Kafka offset reset policy | `earliest` |
| `S3_ENDPOINT` | S3/MinIO endpoint | `http://localhost:9000` |
| `S3_REGION` | S3 region | `us-east-1` |
| `S3_ACCESS_KEY_ID` | S3 access key | `minioadmin` |
| `S3_SECRET_ACCESS_KEY` | S3 secret key | `minioadmin123` |
| `S3_BUCKET` | S3 bucket name | `taskmanager` |
| `S3_USE_PATH_STYLE` | Force path-style S3 addressing | `true` |
| `JWT_SECRET` | JWT signing secret | `your-super-secret-jwt-key-change-in-production` |
| `JWT_ACCESS_TOKEN_TTL` | Access token TTL | `15m` |
| `JWT_REFRESH_TOKEN_TTL` | Refresh token TTL | `168h` |
| `JWT_ISSUER` | JWT issuer | `taskmanager` |
| `OTEL_SERVICE_NAME` | OTel service name | `taskmanager` |
| `OTEL_SERVICE_VERSION` | OTel service version | `1.0.0` |
| `OTEL_EXPORTER_ENDPOINT` | OTel exporter endpoint | `localhost:4317` |
| `OTEL_ENABLED` | Enable OTel exporter | `false` |
| `LOG_LEVEL` | Logger level | `info` |
| `LOG_FORMAT` | Logger format | `json` |

## Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## API Documentation

Once the HTTP server is running, access Swagger at:

- http://localhost:8080/swagger/

Swagger currently covers the REST API only.

## Development

### Hot Reload with Air

```bash
# Install Air
go install github.com/air-verse/air@latest

# Start development server with hot reload
air
```

### SQLC Code Generation

```bash
# Install SQLC
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Generate SQLC code
sqlc generate
```

### Swagger Documentation

```bash
# Install swagger
go install github.com/swaggo/swag/cmd/swag@latest

# Generate Swagger docs
swag init -g cmd/server/main.go -o docs
```

### Database Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create new migration
migrate create -ext sql -dir migrations -seq migration_name

# Run migrations up
migrate -database "postgres://taskmanager:taskmanager_secret@localhost:5433/taskmanager?sslmode=disable" -path migrations up

# Run migrations down
migrate -database "postgres://taskmanager:taskmanager_secret@localhost:5433/taskmanager?sslmode=disable" -path migrations down
```

## License

MIT License
