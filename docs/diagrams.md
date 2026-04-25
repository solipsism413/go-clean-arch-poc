# Diagrams

This document catalogs Mermaid source files used for project documentation.

## Diagram List

### System Architecture

- Purpose: show the high-level runtime structure across clients, transports, application services, domain, and infrastructure adapters.
- Source: `docs/diagrams/01-system-architecture.mmd`

### Create Task Sequence

- Purpose: show the main request path for a task creation flow through REST, use case, repository, database, and event publication.
- Source: `docs/diagrams/02-create-task-sequence.mmd`

### Database ERD

- Purpose: show the initial relational schema defined by the first migration.
- Source: `docs/diagrams/03-database-erd.mmd`

### Authentication and Authorization Flow

- Purpose: show login, bearer token validation, and the RBAC plus ACL path used by protected routes.
- Source: `docs/diagrams/04-authn-authz-flow.mmd`

## Maintenance Note

Keep Mermaid source in `.mmd` files and reference those files from markdown instead of duplicating embedded diagrams across documents.
