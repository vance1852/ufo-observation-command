# UFO Observation Command — Foundation Specification Freeze

Batch: `ufo-20260824-01`  |  Variant: backend  |  Target: 30 independent runtime authoring boundaries

## Business boundary

The service records anomalous aerial observations, assigns them to analysts, validates evidence, coordinates recovery/response work, and preserves an audit trail. It is not a game, social network, ecommerce system, generic CRUD demo, or command-line product. Observer, analyst, dispatcher, and read-only auditor roles have distinct permissions.

## Frozen architecture and invariants

| Area | Frozen decision and verification boundary |
|---|---|
| Persistence | PostgreSQL 16 through pgx pool; no in-memory replacement for production state. |
| Migrations | `migrations/001_initial.sql`, `002_ufo_observation_contract.sql`, `003_console.sql`; idempotent version table and ordered startup migration. |
| Relational model | 20+ related tables including operators, missions, buoys, assignments, jobs, windows, reports, incidents, audit events, users, sessions, commands, and idempotency keys. |
| Transactions | Assignment, evidence acceptance, incident transitions, command dispatch, and audit writes use cross-entity SQL transactions with rollback on failure. |
| State machines | Mission, assignment, recovery job, dive window, report, and incident transitions validate current state and legal next state. |
| Concurrency | PostgreSQL row locks/version checks, bounded pool, deterministic worker claims, and race-tested service coordination. |
| Context | HTTP request context flows through service, repository, migrations, and workers; cancellation and deadlines are preserved. |
| Workers | Assignment, expiration, and safety-alert workers support cancellation, retry/backoff, idempotency, and durable failure status. |
| Errors | Sentinel/domain errors are wrapped with `%w`, mapped to stable JSON envelopes and request IDs at HTTP boundary. |
| HTTP | Health/readiness, authentication, logout/revocation, RBAC, CRUD, pagination/filtering, transitions, batch operations, and audit query endpoints. |
| Identity | PostgreSQL-backed users and hashed passwords; expiring revocable sessions; logout revokes token; observer/analyst/dispatcher/auditor roles enforced in service and HTTP tests. |
| Docker | Multi-stage `golang:1.26`, real `cmd/server` build path, migrations copied to `/app/migrations`, distroless non-root entrypoint. |
| Tests | Domain, service, repository, PostgreSQL integration, HTTP contract, auth/RBAC, transaction rollback, worker cancellation/retry, pagination, and restart recovery tests. |
| Scale | At least 30 production Go files, 10 meaningful packages, 5,000+ production lines and 1,500+ test lines, measured by `measure_project.go -enforce`. |
| Excluded topics | No games, ecommerce/orders, voting/OA/library/medical/CMS, file/password managers, generic dashboards, or benchmark-derived defects. |
| Future capacity | 30+ independent boundaries: auth/session (4), operator registry (4), mission lifecycle (5), assignment/concurrency (4), evidence/report review (4), recovery jobs (3), dive windows (3), incident response (4), audit/idempotency (3), pagination/filtering (2). No candidate, bug, private test, task branch, or answer is created in foundation. |

## Runtime boundary inventory (frozen, no defects designed)

1. login; 2. current identity; 3. logout revocation; 4. session expiry; 5. observer role gate; 6. analyst role gate; 7. operator registration; 8. operator role change; 9. operator listing; 10. mission creation; 11. mission state transition; 12. mission pagination; 13. assignment creation; 14. assignment optimistic locking; 15. assignment completion; 16. recovery job enqueue; 17. recovery retry; 18. recovery cancellation; 19. dive-window creation; 20. window capacity reservation; 21. signal report submission; 22. report validation; 23. report approval; 24. integrity incident creation; 25. incident escalation; 26. incident resolution; 27. command dispatch; 28. idempotency replay; 29. audit-chain query; 30. filtered/paginated audit export; 31. worker graceful shutdown; 32. restart migration recovery.

All boundaries are implemented as normal production behavior and are intentionally free of seeded defects. Candidate design, private tests, `tasks/<task_key>`, red/green branches, and intake payloads are out of scope for this foundation checkpoint.
