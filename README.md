# Go Clean Architecture PoC

A Task Management Application built with Hexagonal (Ports & Adapters) Architecture in Go.

## 🏗️ Architecture

This project follows the **Hexagonal Architecture** (also known as Ports & Adapters) pattern, which aims to create a loosely coupled application by isolating the core business logic from external concerns like databases, UI, and external services.

### Core Principles

- **Domain Centric**: The business logic is at the center of the application.
- **Dependency Rule**: Dependencies point inwards. The core (Domain and Application layers) doesn't know anything about the outer layers (Infrastructure and Transport).
- **Independence**: The application is independent of frameworks, UI, databases, or any external agency.
- **Testability**: The core logic can be tested without any external components.

### Layers

1.  **Domain Layer (Core)**: Contains entities, value objects, domain events, and domain errors. It encapsulates the fundamental business rules.
2.  **Application Layer**: Implements use cases (Input Ports) and defines interfaces for external systems (Output Ports). It coordinates the flow of data to and from the domain layer.
3.  **Transport Layer**: Handles incoming requests from various sources (REST, GraphQL, gRPC, WebSocket) and translates them into application use case calls.
4.  **Infrastructure Layer**: Contains concrete implementations of the output ports (e.g., PostgreSQL repository, Kafka publisher, Redis cache).

```mermaid
graph TD
    subgraph Transport_Layer [Transport Layer]
        REST[REST API]
        GraphQL[GraphQL]
        gRPC[gRPC]
        WS[WebSocket]
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

    REST & GraphQL & gRPC & WS --> UC
    UC --> Entities & VO & Events
    UC --> Ports
    Ports --> DB & Cache & Messaging & Storage
```

## 🛠️ Infrastructure

The application relies on several infrastructure components to provide robust features:

- **PostgreSQL**: Primary relational database for persistent storage of users, tasks, roles, and permissions.
- **Redis**: High-performance cache for session management and frequently accessed data.
- **Kafka**: Distributed event streaming platform for handling domain events and asynchronous communication.
- **MinIO / S3**: Object storage for handling file uploads and attachments.
- **OpenTelemetry (OTel)**: Provides observability through distributed tracing, metrics, and logging.
- **GQLGen**: Used for generating GraphQL code from schema.
- **SQLC**: Generates type-safe Go code from SQL queries.
- **Viper**: Manages application configuration from environment variables and files.

## 📊 Diagrams

### Use Case Diagram

```mermaid
graph TD
    User((User))
    Admin((Admin))

    subgraph Auth
        User --> Register([Register])
        User --> Login([Login])
        User --> Logout([Logout])
    end

    subgraph Task_Management [Task Management]
        User --> CreateTask([Create Task])
        User --> UpdateTask([Update Task])
        User --> ViewTasks([View Tasks])
        User --> AssignTask([Assign Task])
        User --> CompleteTask([Complete Task])
        User --> ArchiveTask([Archive Task])
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

### Request Flow (Sequence Diagram)

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
    REST->>DTO: Parse & Validate
    DTO-->>REST: Valid DTO
    REST->>UC: Execute(ctx, dto)
    UC->>Domain: NewTask(title, desc, priority, creatorID)
    Domain-->>UC: Task Entity
    UC->>Repo: Save(ctx, task)
    Repo->>DB: INSERT INTO tasks ...
    DB-->>Repo: Success
    UC->>Event: Publish(TaskCreatedEvent)
    Event->>Kafka: Produce Message
    UC-->>REST: Task DTO
    REST-->>Client: 201 Created (JSON)
```

### Data Processing Flow

```mermaid
graph LR
    subgraph Input_Phase [Input]
        JSON[External JSON/GQL] --> DTO[Application DTO]
    end

    subgraph Business_Phase [Business Logic]
        DTO --> UseCase[UseCase]
        UseCase --> Entity[Domain Entity]
    end

    subgraph Persistence_Phase [Persistence]
        Entity --> SQLC[SQLC/Repository]
        SQLC --> SQL[SQL Query]
        SQL --> DB[(PostgreSQL)]
    end

    subgraph Output_Phase [Output]
        DB --> SQLC
        SQLC --> Entity
        Entity --> DTO
        DTO --> JSON
    end

    style UseCase fill:#f9f,stroke:#333,stroke-width:2px
    style Entity fill:#bbf,stroke:#333,stroke-width:2px
```

## 🚀 Quick Start

### Prerequisites

- Go 1.26.2+
- Docker & Docker Compose

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

`docker-compose.yml` is now the single source for both modes using Compose `profiles`. `app` runs under the `prod` profile with the production image/runtime, while `app-watch` runs under the `watch` profile with `air` and `develop.watch`. Changes under `cmd/`, `internal/`, `pkg/`, `docs/`, and `migrations/` are synced into the container, while changes to `go.mod`, `go.sum`, `.air.toml`, `Dockerfile`, `sqlc.yaml`, and `gqlgen.yml` trigger an image rebuild.

### Running Locally

```bash
# Start infrastructure services
docker compose up -d postgres redis kafka minio

# Run migrations
docker compose run --rm migrate

# Run the application
go run ./cmd/server
```

## 📁 Project Structure

```
├── cmd/
│   └── server/              # Application entrypoint
├── internal/
│   ├── domain/              # Domain Layer (Core Business Logic)
│   │   ├── entity/          # Domain entities
│   │   ├── valueobject/     # Value objects
│   │   ├── event/           # Domain events
│   │   └── error/           # Domain errors
│   ├── application/         # Application Layer
│   │   ├── port/
│   │   │   ├── input/       # Input ports (use case interfaces)
│   │   │   └── output/      # Output ports (repository interfaces)
│   │   ├── usecase/         # Use case implementations
│   │   ├── dto/             # Data transfer objects
│   │   └── validation/      # Input validation
│   ├── infrastructure/      # Infrastructure Layer
│   │   ├── database/        # Database implementations
│   │   ├── cache/           # Cache implementations
│   │   ├── messaging/       # Message broker implementations
│   │   ├── storage/         # File storage implementations
│   │   └── observability/   # Logging, tracing, metrics
│   └── transport/           # Transport Layer
│       ├── rest/            # REST API handlers
│       ├── graphql/         # GraphQL resolvers
│       └── grpc/            # gRPC services
├── migrations/              # Database migrations
├── pkg/                     # Shared packages
│   └── config/              # Configuration
├── Dockerfile
├── docker-compose.yml
└── sqlc.yaml                # SQLC configuration
```

## 🔧 Configuration

Configuration is done via environment variables:

| Variable        | Description        | Default          |
| --------------- | ------------------ | ---------------- |
| `SERVER_HOST`   | Server host        | `0.0.0.0`        |
| `SERVER_PORT`   | Server port        | `8080`           |
| `DB_HOST`       | PostgreSQL host    | `localhost`      |
| `DB_PORT`       | PostgreSQL port    | `5432`           |
| `DB_USER`       | Database user      | `postgres`       |
| `DB_PASSWORD`   | Database password  | `postgres`       |
| `DB_NAME`       | Database name      | `taskmanager`    |
| `REDIS_HOST`    | Redis host         | `localhost`      |
| `REDIS_PORT`    | Redis port         | `6379`           |
| `KAFKA_BROKERS` | Kafka brokers      | `localhost:9092` |
| `S3_ENDPOINT`   | S3/MinIO endpoint  | -                |
| `JWT_SECRET`    | JWT signing secret | -                |

## 🧪 Testing

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with race detection
go test -race ./...
```

## 📚 API Documentation

Once the server is running, access the Swagger documentation at:

- http://localhost:8080/swagger/

## 🛠️ Development

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
swag init -g cmd/server/main.go
```

### Database Migrations

```bash
# Install golang-migrate
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Create new migration
migrate create -ext sql -dir migrations -seq migration_name

# Run migrations up
migrate -database "postgres://user:pass@localhost:5433/taskmanager?sslmode=disable" -path migrations up

# Run migrations down
migrate -database "postgres://user:pass@localhost:5433/taskmanager?sslmode=disable" -path migrations down
```

## 📝 License

MIT License
