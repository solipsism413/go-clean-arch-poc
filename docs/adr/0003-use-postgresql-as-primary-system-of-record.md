# ADR 0003: Use PostgreSQL as the Primary System of Record

## Status

Accepted

## Context

The domain requires durable relational data for users, roles, permissions, tasks, labels, ACL entries, and attachment metadata. The project also benefits from explicit schema management and typed query generation.

## Decision

Use PostgreSQL as the primary source of truth. Manage schema through SQL migrations and use SQLC-backed data access together with repository abstractions.

## Consequences

- The project gets a reliable relational core for task and auth data.
- Schema evolution is explicit through migration files.
- Query safety improves with generated SQLC code.
- Additional infrastructure such as Redis and Kafka remains secondary to PostgreSQL, not a replacement for it.
