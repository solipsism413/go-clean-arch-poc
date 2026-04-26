# ADR 0004: Redis Token Revocation, Session Invalidation, and Read Caching

## Status

Accepted

## Context

The application already wired Redis via `internal/infrastructure/cache/redis` but only used it for minimal cache invalidation in `TaskUseCase`. The JWT authentication was fully stateless, meaning tokens remained valid until expiry even after logout. There was no way to:

1. Revoke an access token before its natural expiration.
2. Invalidate all sessions for a user (e.g., after password change or role change).
3. Reduce database load on frequently read entities like tasks and users.

## Decision

Expand Redis usage across three areas:

### 1. Token Revocation (Blacklist)

- Store revoked token JTIs in Redis with TTL equal to the remaining token lifetime.
- Key pattern: `app:token:blacklist:{jti}`
- The `AuthUseCase.ValidateToken` method checks the blacklist after JWT signature validation.
- The `AuthUseCase.Logout` method extracts the JTI from the raw access token and blacklists it.
- The `AuthUseCase.RefreshToken` method checks if the refresh token's JTI is blacklisted before issuing a new pair.

### 2. Session Invalidation

- On login/register, store session metadata in Redis under `app:session:{userID}:{jti}` with TTL matching the refresh token lifetime. Two session keys are stored per token pair: one keyed by the access token JTI and one by the refresh token JTI.
- On logout, delete both session keys and blacklist both the access and refresh token JTIs.
- On password change or role assignment/removal, bulk-delete all session keys for the user using `DeletePattern` on `app:session:{userID}:*`.
- `ValidateToken` and `RefreshToken` verify that the corresponding session key still exists. If the session was deleted (via bulk invalidation), the token is rejected even if it has not expired and is not explicitly blacklisted.

### 3. Read Caching (Cache-Aside)

- `GetTask` and `GetUser` cache single entities by ID for 5 minutes.
- `GetUserByEmail` caches by email for 5 minutes.
- `ListTasks` and `ListUsers` cache paginated list results for 2 minutes using a deterministic SHA-256 hash of filter + pagination as the cache key.
- All write operations invalidate the affected entity cache and all list caches (fire-and-forget, errors swallowed).

## Consequences

### Positive

- Logout now actually invalidates both the access and refresh tokens.
- Password changes and role changes force re-login across all sessions, improving security.
- Session existence is validated on every request, so bulk session deletion immediately takes effect.
- Frequently read endpoints (task/user details, lists) will see reduced DB load.
- Cache is optional — the app still starts and functions if Redis is unavailable.

### Negative

- Cache invalidation is coarse for lists (`DeletePattern` on `app:tasks:list:*`), which may evict more entries than strictly necessary.
- Transport layers (REST, gRPC, GraphQL) must now pass the raw JWT token string to `Logout`, requiring context storage of the raw token.
- Tests require additional mock expectations for cache operations.

## Implementation Notes

- `GetJSON` and `SetJSON` were promoted from concrete `redis.CacheRepository` methods to the `output.CacheRepository` interface so all cache implementations (Redis, in-memory, mocks) support them.
- `dto.TokenClaims` now includes `TokenID` (the JTI claim) so middleware and use cases can identify individual tokens.
- `auth.TokenContextKey` was added to carry the raw JWT string through the request context.
- Cache TTL constants are defined locally in each use case package (`entityCacheTTL = 5m`, `listCacheTTL = 2m`).

## References

- `internal/application/usecase/auth/auth_usecase.go`
- `internal/application/usecase/task/task_usecase.go`
- `internal/application/usecase/user/user_usecase.go`
- `internal/application/port/output/cache_repository.go`
- `internal/infrastructure/auth/jwt/token_service.go`
- `internal/auth/middleware.go`
