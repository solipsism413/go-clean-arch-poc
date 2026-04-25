# AGENTS.md

## Entry Points
- `cmd/server/main.go` is the real composition root. It wires PostgreSQL, Redis, Kafka, S3/MinIO, JWT auth, RBAC/ACL, REST, WebSocket, SSE, and Socket.IO, then seeds system roles on startup.
- `cmd/grpc/main.go` wires the same dependencies as the HTTP server and registers full gRPC services for task, user, auth, and label operations. It starts on `cfg.GRPC.Port`.
- GraphQL is schema/config only: `internal/transport/graphql/schema.graphqls`, `gqlgen.yml`. Do not assume an HTTP GraphQL endpoint exists.

## Architecture Map
- Core boundaries are strict and useful to preserve: `internal/domain` for entities/value objects/events, `internal/application` for ports/use cases/validation, `internal/transport` for protocol adapters, `internal/infrastructure` for concrete adapters.
- PostgreSQL access is split between generated SQLC code in `internal/infrastructure/database/sqlc`, repositories in `internal/infrastructure/database/repository`, and query builders in `internal/infrastructure/database/querybuilder`.
- REST is the only fully active API surface. Start there for behavior changes: `internal/transport/rest/router.go` and the handlers under `internal/transport/rest/handler`.

## Commands
- Required toolchain is Go `1.26.2` (`go.mod` toolchain and CI agree).
- Local infra-first dev flow: `make setup-infra` then `go run ./cmd/server` or `make run-api`.
- `make setup-infra` starts Postgres, Redis, Kafka, MinIO, then runs migrations. Local Postgres is exposed on `localhost:5433`, matching `pkg/config/config.go` defaults.
- Hot reload uses Air: `make dev`. Docker watch mode is `make docker-watch`.
- Full test suite: `make test` (`go test -v -race ./...`).
- Focused Go test: `go test ./internal/transport/rest -run TestName -count=1`.
- Focused package test: `go test ./internal/infrastructure/database/repository -count=1`.

## Codegen And Generated Artifacts
- Run codegen before verification if you change SQL queries, migrations, or Swagger annotations/comments.
- `make generate` runs `sqlc generate` plus Swagger generation only.
- SQLC input is `internal/infrastructure/database/sqlc/queries/` plus `migrations/`; generated output lands in `internal/infrastructure/database/sqlc`.
- Swagger must exist under `docs/api/swagger` because `internal/transport/rest/router.go` imports `github.com/handiism/go-clean-arch-poc/docs/api/swagger`. Prefer `make generate-swagger` or `make generate`.
- `generate-graphql` and `generate-grpc` targets exist. The gRPC runtime path is complete; GraphQL remains schema-only with no active HTTP endpoint.

## Verification Gotchas
- CI effectively does `sqlc generate` + Swagger generation before `go vet`, `staticcheck`, tests, and builds. Mirror that order when touching generated inputs.
- Many tests use Testcontainers (`internal/transport/rest`, database packages, Kafka, S3), so local `go test` needs a working Docker daemon even for some package-level runs.
- REST test helpers execute the SQL files in `migrations/` directly against a containerized Postgres instance; migration changes can break multiple packages at once.

## Docs And ADRs
- Update docs alongside code when behavior, setup, architecture, transport support, or developer workflows change. Start with `README.md`, `docs/status.md`, `docs/index.md`, and `docs/architecture.md` when they are affected.
- Add or update an ADR under `docs/adr/` for changes that introduce, replace, or remove meaningful architectural decisions. Keep ADRs focused on the decision and why it was made.

## Source Priority
- If docs disagree, trust executable sources first: `Makefile`, `go.mod`, `docker-compose.yml`, `.github/workflows/ci.yml`, then the composition roots in `cmd/server` and `cmd/grpc`.
- `docs/status.md` contains some stale statements such as CI being a future item; verify against the actual workflow before repeating it.
